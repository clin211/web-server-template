import { describe, expect, it } from "vitest";
import { TransportError } from "../http/errors";
import { issueCsrfToken } from "./csrf";
import { requireCsrfToken, requireSameOrigin, requireWriteGuards } from "./guards";

describe("request guards", () => {
  it("accepts a matching origin header", () => {
    expect(() =>
      requireSameOrigin(
        new Request("https://console.test/api/users", {
          headers: { origin: "https://console.test" },
        }),
        "https://console.test",
      ),
    ).not.toThrow();
  });

  it.each([
    ["missing origin", {}],
    ["invalid origin", { origin: "not a url" }],
    ["different origin", { origin: "https://evil.test" }],
  ])("rejects %s", (_label, headers) => {
    expect(() =>
      requireSameOrigin(
        new Request("https://console.test/api/users", { headers }),
        "https://console.test",
      ),
    ).toThrowError(TransportError);
  });

  it("requires a matching CSRF token", () => {
    const request = new Request("https://console.test/api/users", {
      headers: { "x-csrf-token": "csrf-token" },
    });
    expect(() => requireCsrfToken(request, "csrf-token")).not.toThrow();
    expect(() => requireCsrfToken(request, "different-token")).toThrowError(TransportError);
  });

  it("validates the signed Cookie/header pair for a write request", () => {
    const secret = "guard-test-secret-with-at-least-32-bytes";
    const token = issueCsrfToken(secret);
    const request = new Request("https://console.test/api/users", {
      headers: {
        origin: "https://console.test",
        cookie: `__Host-console-csrf=${token}`,
        "x-csrf-token": token,
      },
    });
    expect(
      requireWriteGuards(request, {
        origin: "https://console.test",
        secret,
        secure: true,
      }),
    ).toBe(token);
  });

  it.each([
    ["missing Cookie", ""],
    ["invalid signature", "unsigned.invalid"],
  ])("rejects a write request with %s", (_label, token) => {
    const request = new Request("https://console.test/api/users", {
      headers: {
        origin: "https://console.test",
        ...(token ? { cookie: `__Host-console-csrf=${token}`, "x-csrf-token": token } : {}),
      },
    });

    expect(() =>
      requireWriteGuards(request, {
        origin: "https://console.test",
        secret: "guard-test-secret-with-at-least-32-bytes",
        secure: true,
      }),
    ).toThrowError(TransportError);
  });
});
