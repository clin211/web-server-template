import { describe, expect, it } from "vitest";
import { getRequestId, getTraceContext } from "./request-context";

describe("request context", () => {
  it("does not create trace context when traceparent is absent", () => {
    expect(getTraceContext(new Request("https://console.test"))).toBeNull();
  });

  it.each([
    ["request-123", "request-123"],
    ["trace:abc_01", "trace:abc_01"],
  ])("accepts a safe request ID", (header, expected) => {
    expect(
      getRequestId(
        new Request("https://console.test", {
          headers: { "x-request-id": header },
        }),
      ),
    ).toBe(expected);
  });

  it("replaces missing or unsafe request IDs", () => {
    const generatedRequest = new Request("https://console.test");
    const generated = getRequestId(generatedRequest);
    const unsafeRequest = new Request("https://console.test", {
      headers: { "x-request-id": "contains spaces" },
    });
    const replaced = getRequestId(unsafeRequest);

    expect(generated).toMatch(/^[0-9a-f-]{36}$/);
    expect(replaced).toMatch(/^[0-9a-f-]{36}$/);
    expect(getRequestId(generatedRequest)).toBe(generated);
    expect(generatedRequest.headers.get("x-request-id")).toBe(generated);
    expect(unsafeRequest.headers.get("x-request-id")).toBe(replaced);
  });

  it("accepts and normalizes a valid W3C trace context", () => {
    const context = getTraceContext(
      new Request("https://console.test", {
        headers: {
          traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
          tracestate: "vendor=value, tenant@system=opaque",
        },
      }),
    );

    expect(context).toEqual({
      traceId: "4bf92f3577b34da6a3ce929d0e0e4736",
      traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
      tracestate: "vendor=value,tenant@system=opaque",
    });
  });

  it.each([
    "00-00000000000000000000000000000000-00f067aa0ba902b7-01",
    "00-4bf92f3577b34da6a3ce929d0e0e4736-0000000000000000-01",
    "00-4BF92F3577B34DA6A3CE929D0E0E4736-00f067aa0ba902b7-01",
    "ff-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
  ])("rejects unsafe traceparent %s", (traceparent) => {
    expect(
      getTraceContext(
        new Request("https://console.test", {
          headers: { traceparent, tracestate: "vendor=value" },
        }),
      ),
    ).toBeNull();
  });

  it("drops an invalid tracestate while preserving a valid traceparent", () => {
    expect(
      getTraceContext(
        new Request("https://console.test", {
          headers: {
            traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
            tracestate: "Vendor=uppercase-key",
          },
        }),
      ),
    ).toEqual({
      traceId: "4bf92f3577b34da6a3ce929d0e0e4736",
      traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
    });
  });

  it.each([
    ["missing tracestate", undefined],
    [
      "too many members",
      Array.from({ length: 33 }, (_, index) => `vendor${index}=value`).join(","),
    ],
    ["missing separator", "vendor"],
    ["empty value", "vendor="],
  ])("drops %s while preserving a valid traceparent", (_label, tracestate) => {
    const headers = new Headers({
      traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
    });
    if (tracestate !== undefined) headers.set("tracestate", tracestate);

    expect(getTraceContext(new Request("https://console.test", { headers }))).toEqual({
      traceId: "4bf92f3577b34da6a3ce929d0e0e4736",
      traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
    });
  });
});
