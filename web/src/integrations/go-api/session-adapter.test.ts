import { describe, expect, it, vi } from "vitest";
import { GoSessionAdapter, refreshExpiryFromToken } from "./session-adapter";

describe("refresh token expiry", () => {
  it("reads the standard exp claim without treating it as signature verification", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2030-01-01T00:00:00.000Z"));
    const expiry = Date.parse("2030-01-08T00:00:00.000Z") / 1000;
    const payload = Buffer.from(JSON.stringify({ exp: expiry })).toString("base64url");
    expect(refreshExpiryFromToken(`header.${payload}.signature`)).toBe("2030-01-08T00:00:00.000Z");
    vi.useRealTimers();
  });

  it.each([
    ["expired", "2029-12-31T23:59:59.000Z"],
    ["beyond the maximum lifetime", "2030-01-10T00:00:00.000Z"],
  ])("rejects an %s expiry", (_label, expiry) => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2030-01-01T00:00:00.000Z"));
    const payload = Buffer.from(JSON.stringify({ exp: Date.parse(expiry) / 1000 })).toString(
      "base64url",
    );
    expect(refreshExpiryFromToken(`header.${payload}.signature`)).toBeNull();
    vi.useRealTimers();
  });

  it.each(["invalid", "header.invalid.signature", "header.e30.signature"])(
    "rejects malformed token %s",
    (token) => {
      expect(refreshExpiryFromToken(token)).toBeNull();
    },
  );

  it("reads, commits, and clears local HTTP session cookies", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2030-01-01T00:00:00.000Z"));
    const adapter = new GoSessionAdapter({ secure: false });
    const request = new Request("http://console.test", {
      headers: { cookie: "console-access=access; console-refresh=refresh" },
    });
    expect(adapter.readAccess(request)).toBe("access");
    expect(adapter.readRefresh(request)).toBe("refresh");

    const committed = adapter.commit(new Response(null), {
      accessToken: "new-access",
      refreshToken: "new-refresh",
      accessExpiresAt: "2030-01-01T00:02:00.000Z",
      refreshExpiresAt: "2030-01-01T01:00:00.000Z",
    });
    expect(committed.headers.getSetCookie()).toEqual([
      "console-access=new-access; Max-Age=120; HttpOnly; SameSite=Lax; Path=/",
      "console-refresh=new-refresh; Max-Age=3600; HttpOnly; SameSite=Lax; Path=/",
    ]);

    const cleared = adapter.clear(new Response(null));
    expect(cleared.headers.getSetCookie()).toEqual([
      "console-access=; Max-Age=0; HttpOnly; SameSite=Lax; Path=/",
      "console-refresh=; Max-Age=0; HttpOnly; SameSite=Lax; Path=/",
    ]);
    vi.useRealTimers();
  });

  it("uses host-prefixed secure cookies and clamps expired sessions", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2030-01-01T00:00:00.000Z"));
    const adapter = new GoSessionAdapter({ secure: true });
    const response = adapter.commit(new Response(null), {
      accessToken: "access",
      refreshToken: "refresh",
      accessExpiresAt: "2029-12-31T23:59:00.000Z",
      refreshExpiresAt: "2029-12-31T23:59:00.000Z",
    });
    expect(response.headers.getSetCookie()).toEqual([
      "__Host-console-access=access; Max-Age=0; HttpOnly; Secure; SameSite=Lax; Path=/",
      "__Host-console-refresh=refresh; Max-Age=0; HttpOnly; Secure; SameSite=Lax; Path=/",
    ]);
    vi.useRealTimers();
  });
});
