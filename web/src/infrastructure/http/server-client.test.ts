import { describe, expect, it } from "vitest";
import { ServerHttpClient } from "./server-client";
import type { SessionPort } from "../session/port";

const session: SessionPort = {
  readAccess: () => "access-token",
  readRefresh: () => "refresh-token",
  commit: (response) => response,
  clear: (response) => response,
};

describe("server HTTP client", () => {
  it("injects only the selected credential and strips browser credentials", async () => {
    let capturedUrl = "";
    let capturedInit: RequestInit | undefined;
    const client = new ServerHttpClient({
      baseUrl: "https://api.console.test",
      session,
      fetchFn: async (input, init) => {
        capturedUrl = String(input);
        capturedInit = init;
        return new Response(null, { status: 204 });
      },
    });

    await client.request("/v1/users", {
      request: new Request("https://console.test/api/users", {
        headers: {
          "x-request-id": "request-123",
          traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
          tracestate: "vendor=value",
        },
      }),
      policy: { kind: "access" },
      method: "post",
      headers: {
        authorization: "Bearer browser-token",
        connection: "keep-alive",
        cookie: "console-access=browser-token",
        host: "evil.test",
        upgrade: "websocket",
      },
      body: JSON.stringify({ name: "Ada" }),
    });

    expect(capturedUrl).toBe("https://api.console.test/v1/users");
    const headers = new Headers(capturedInit?.headers);
    expect(headers.get("authorization")).toBe("Bearer access-token");
    expect(headers.get("cookie")).toBeNull();
    expect(headers.get("host")).toBeNull();
    expect(headers.get("connection")).toBeNull();
    expect(headers.get("upgrade")).toBeNull();
    expect(headers.get("content-type")).toBe("application/json");
    expect(headers.get("x-request-id")).toBe("request-123");
    expect(headers.get("traceparent")).toBe(
      "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
    );
    expect(headers.get("tracestate")).toBe("vendor=value");
    expect(capturedInit?.method).toBe("POST");
  });

  it("drops caller-supplied and invalid trace headers", async () => {
    let capturedInit: RequestInit | undefined;
    const client = new ServerHttpClient({
      baseUrl: "https://api.console.test",
      session,
      fetchFn: async (_input, init) => {
        capturedInit = init;
        return new Response(null, { status: 204 });
      },
    });

    await client.request("/v1/users", {
      request: new Request("https://console.test/api/users", {
        headers: {
          traceparent: "00-00000000000000000000000000000000-00f067aa0ba902b7-01",
          tracestate: "vendor=inbound",
        },
      }),
      policy: { kind: "anonymous" },
      headers: {
        traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
        tracestate: "vendor=caller",
      },
    });

    const headers = new Headers(capturedInit?.headers);
    expect(headers.get("traceparent")).toBeNull();
    expect(headers.get("tracestate")).toBeNull();
  });

  it("propagates traceparent without tracestate and combines an abort signal", async () => {
    let capturedInit: RequestInit | undefined;
    const controller = new AbortController();
    const client = new ServerHttpClient({
      baseUrl: "https://api.console.test",
      session,
      fetchFn: async (_input, init) => {
        capturedInit = init;
        return new Response(null, { status: 204 });
      },
    });

    await client.request("/v1/users", {
      request: new Request("https://console.test/api/users", {
        headers: {
          traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
        },
      }),
      policy: { kind: "anonymous" },
      signal: controller.signal,
    });

    const headers = new Headers(capturedInit?.headers);
    expect(headers.get("traceparent")).toBe(
      "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
    );
    expect(headers.get("tracestate")).toBeNull();
    controller.abort();
    expect(capturedInit?.signal?.aborted).toBe(true);
  });

  it("normalizes upstream network errors without exposing internals", async () => {
    const client = new ServerHttpClient({
      baseUrl: "https://api.console.test",
      session,
      fetchFn: async () => {
        throw new Error("socket path and private host");
      },
    });

    await expect(
      client.request("/v1/users", {
        request: new Request("https://console.test/api/users"),
        policy: { kind: "anonymous" },
      }),
    ).rejects.toMatchObject({
      kind: "network",
      message: "Upstream request failed",
    });
  });

  it("normalizes an aborted upstream request as a timeout", async () => {
    const client = new ServerHttpClient({
      baseUrl: "https://api.console.test",
      session,
      timeoutMs: 1,
      fetchFn: async (_input, init) => {
        await new Promise((resolve, reject) => {
          init?.signal?.addEventListener("abort", () => reject(new Error("aborted")), {
            once: true,
          });
          setTimeout(resolve, 20);
        });
        return new Response(null, { status: 204 });
      },
    });

    await expect(
      client.request("/v1/users", {
        request: new Request("https://console.test/api/users"),
        policy: { kind: "anonymous" },
      }),
    ).rejects.toMatchObject({ kind: "timeout", message: "Upstream request timed out" });
  });
});
