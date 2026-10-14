import { describe, expect, it } from "vitest";
import { csrfCookieName, expireCsrfToken, issueCsrfToken, verifyCsrfToken } from "./csrf";

const secret = "0123456789abcdef0123456789abcdef";

describe("CSRF token", () => {
  it("issues and verifies a signed token", () => {
    const token = issueCsrfToken(secret);
    expect(token).toContain(".");
    expect(verifyCsrfToken(token, secret)).toBe(true);
  });

  it.each([
    "missing-signature",
    ".signature",
    "nonce.",
    "nonce.invalid",
    "nonce.invalid.extra",
    `${"a".repeat(40)}.${"b".repeat(43)}`,
  ])("rejects invalid token %s", (token) => {
    expect(verifyCsrfToken(token, secret)).toBe(false);
  });

  it("uses host-prefixed cookies only for secure origins", () => {
    expect(csrfCookieName(true)).toBe("__Host-console-csrf");
    expect(csrfCookieName(false)).toBe("console-csrf");
  });

  it("expires the double-submit cookie after a security boundary change", () => {
    const response = expireCsrfToken(new Response(null), true);
    expect(response.headers.getSetCookie()).toEqual([
      "__Host-console-csrf=; Max-Age=0; Secure; SameSite=Lax; Path=/",
    ]);
  });
});
