import { afterEach, describe, expect, it, vi } from "vitest";
import {
  fetchSession,
  issueBrowserCsrf,
  loginBrowserSession,
  logoutBrowserSession,
  refreshBrowserSession,
} from "./client";

afterEach(() => vi.unstubAllGlobals());

describe("session browser client", () => {
  it("parses canonical session data", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        Response.json({
          user: {
            userID: "user-1",
            username: "ada",
            nickname: "Ada",
            email: "ada@example.test",
            phone: "13800138000",
            postCount: "0",
            createdAt: "1",
            updatedAt: "1",
            status: 0,
            gender: 0,
            avatar: "",
            description: "",
            lastLoginAt: "1",
          },
        }),
      ),
    );
    await expect(fetchSession()).resolves.toMatchObject({ user: { username: "ada" } });
  });

  it("fails closed for an invalid CSRF response", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => Response.json({ token: null })),
    );
    await expect(issueBrowserCsrf()).rejects.toMatchObject({ code: "invalid-response" });
  });

  it("maps a rejected CSRF request to a safe browser error", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        Response.json({ error: { code: "forbidden", message: "安全校验失败" } }, { status: 403 }),
      ),
    );
    await expect(issueBrowserCsrf()).rejects.toMatchObject({ code: "forbidden", status: 403 });
  });

  it("uses the issued CSRF token when refreshing", async () => {
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(Response.json({ token: "csrf-token" }))
      .mockResolvedValueOnce(Response.json({ ok: true }));
    vi.stubGlobal("fetch", fetchMock);
    await refreshBrowserSession();
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "/api/auth/refresh",
      expect.objectContaining({
        method: "POST",
        credentials: "include",
        headers: expect.any(Headers),
      }),
    );
    const headers = new Headers(fetchMock.mock.calls[1]?.[1]?.headers);
    expect(headers.get("x-csrf-token")).toBe("csrf-token");
  });

  it("submits login credentials with the prepared CSRF token", async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(Response.json({ ok: true }));
    vi.stubGlobal("fetch", fetchMock);

    await loginBrowserSession({
      values: { username: "ada", password: "secret1" },
      csrfToken: "csrf-token",
    });

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/auth/login",
      expect.objectContaining({ method: "POST", credentials: "include" }),
    );
    const options = fetchMock.mock.calls[0]?.[1];
    expect(new Headers(options?.headers).get("x-csrf-token")).toBe("csrf-token");
    expect(JSON.parse(String(options?.body))).toEqual({ username: "ada", password: "secret1" });
  });

  it("reports whether upstream logout revocation was confirmed", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn<typeof fetch>()
        .mockResolvedValueOnce(Response.json({ token: "csrf-token" }))
        .mockResolvedValueOnce(Response.json({ revocationConfirmed: true })),
    );

    await expect(logoutBrowserSession()).resolves.toEqual({ revocationConfirmed: true });
  });
});
