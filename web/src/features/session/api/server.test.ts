import { describe, expect, it, vi } from "vitest";
import { TransportError } from "../../../infrastructure/http/errors";
import { issueCsrfToken } from "../../../infrastructure/session/csrf";
import { getSession, login, logoutSession, publicErrorResponse, refreshSession } from "./server";

const secret = "session-server-test-secret-with-at-least-32-bytes";
const refreshToken = `header.${Buffer.from(
  JSON.stringify({ exp: Math.floor(Date.now() / 1_000) + 7 * 24 * 60 * 60 }),
).toString("base64url")}.signature`;
const user = {
  userID: "user-1",
  username: "ada",
  nickname: "Ada",
  email: "ada@example.test",
  phone: "13800138000",
  postCount: "0",
  createdAt: "1700000000",
  updatedAt: "1700000000",
  status: 0,
  gender: 0,
  avatar: "",
  description: "",
  lastLoginAt: "1700000000",
};

async function withServerEnv<T>(callback: () => Promise<T>): Promise<T> {
  vi.stubGlobal("window", undefined);
  process.env.BACKEND_API_URL = "http://api.console.test";
  process.env.CONSOLE_ORIGIN = "https://console.test";
  process.env.CONSOLE_SESSION_SECRET = secret;
  process.env.CONSOLE_REQUEST_TIMEOUT_MS = "1000";
  try {
    return await callback();
  } finally {
    vi.unstubAllGlobals();
  }
}

function csrfCookie() {
  const token = issueCsrfToken(secret);
  return { token, cookie: `__Host-console-csrf=${token}` };
}

function request(path: string, init: RequestInit = {}) {
  return new Request(`https://console.test${path}`, {
    ...init,
    headers: {
      origin: "https://console.test",
      ...init.headers,
    },
  });
}

describe("session server adapters", () => {
  it("logs in through the anonymous operation and commits both credentials", async () => {
    await withServerEnv(async () => {
      const csrf = csrfCookie();
      const fetchMock = vi.fn<typeof fetch>(async (input, init) => {
        expect(String(input)).toBe("http://api.console.test/v1/auth/login");
        expect(init?.method).toBe("POST");
        expect(new Headers(init?.headers).get("authorization")).toBeNull();
        return Response.json({
          code: 0,
          message: "ok",
          data: {
            accessToken: "access-token",
            refreshToken,
            expireAt: "2030-01-01T00:02:00.000Z",
          },
        });
      });
      vi.stubGlobal("fetch", fetchMock);

      const response = await login(
        request("/api/auth/login", {
          method: "POST",
          headers: {
            cookie: csrf.cookie,
            "x-csrf-token": csrf.token,
            "content-type": "application/json",
          },
          body: JSON.stringify({ username: "ada", password: "secret1" }),
        }),
      );
      expect(response.status).toBe(200);
      expect(response.headers.getSetCookie()).toHaveLength(3);
      await expect(response.json()).resolves.toEqual({ ok: true });
    });
  });

  it.each([
    [401, "invalid-credentials"],
    [429, "rate-limited"],
  ])("maps login HTTP %i to %s", async (status, code) => {
    await withServerEnv(async () => {
      const csrf = csrfCookie();
      vi.stubGlobal(
        "fetch",
        vi.fn<typeof fetch>(async () =>
          Response.json({ code: status, reason: "rejected", message: "private" }, { status }),
        ),
      );
      const response = await login(
        request("/api/auth/login", {
          method: "POST",
          headers: {
            cookie: csrf.cookie,
            "x-csrf-token": csrf.token,
            "content-type": "application/json",
          },
          body: JSON.stringify({ username: "ada", password: "secret1" }),
        }),
      );
      expect(response.status).toBe(status);
      await expect(response.json()).resolves.toMatchObject({ error: { code } });
    });
  });

  it("rejects malformed login input before calling upstream", async () => {
    await withServerEnv(async () => {
      const csrf = csrfCookie();
      const fetchMock = vi.fn<typeof fetch>();
      vi.stubGlobal("fetch", fetchMock);
      const response = await login(
        request("/api/auth/login", {
          method: "POST",
          headers: {
            cookie: csrf.cookie,
            "x-csrf-token": csrf.token,
            "content-type": "application/json",
          },
          body: JSON.stringify({ username: "", password: "" }),
        }),
      );
      expect(response.status).toBe(400);
      expect(fetchMock).not.toHaveBeenCalled();
    });
  });

  it("reads the current user through the access credential and validates data", async () => {
    await withServerEnv(async () => {
      const fetchMock = vi.fn<typeof fetch>(async (input, init) => {
        expect(String(input)).toBe("http://api.console.test/v1/auth/info");
        expect(new Headers(init?.headers).get("authorization")).toBe("Bearer access-token");
        return Response.json({ code: 0, message: "ok", data: { user } });
      });
      vi.stubGlobal("fetch", fetchMock);

      const response = await getSession(
        request("/api/auth/session", {
          headers: { cookie: "__Host-console-access=access-token" },
        }),
      );
      expect(response.status).toBe(200);
      await expect(response.json()).resolves.toEqual({ user });
    });
  });

  it("refreshes tokens only with a valid origin and CSRF pair", async () => {
    await withServerEnv(async () => {
      const csrf = csrfCookie();
      vi.stubGlobal(
        "fetch",
        vi.fn<typeof fetch>(async (input, init) => {
          expect(String(input)).toBe("http://api.console.test/v1/auth/refresh-token");
          expect(init?.method).toBe("PUT");
          expect(new Headers(init?.headers).get("authorization")).toBe("Bearer refresh-token");
          return Response.json({
            code: 0,
            message: "ok",
            data: {
              accessToken: "next-access",
              refreshToken,
              expireAt: "2030-01-01T00:02:00.000Z",
            },
          });
        }),
      );

      const response = await refreshSession(
        request("/api/auth/refresh", {
          method: "POST",
          headers: {
            cookie: `__Host-console-refresh=refresh-token; ${csrf.cookie}`,
            "x-csrf-token": csrf.token,
          },
        }),
      );
      expect(response.status).toBe(200);
      expect(response.headers.getSetCookie()).toHaveLength(3);
      await expect(response.json()).resolves.toEqual({ ok: true });
    });
  });

  it("clears credentials when refresh is rejected as unauthenticated", async () => {
    await withServerEnv(async () => {
      const csrf = csrfCookie();
      vi.stubGlobal(
        "fetch",
        vi.fn<typeof fetch>(async () =>
          Response.json(
            { code: 401, reason: "Auth.TokenExpired", message: "expired" },
            { status: 401 },
          ),
        ),
      );
      const response = await refreshSession(
        request("/api/auth/refresh", {
          method: "POST",
          headers: {
            cookie: `__Host-console-refresh=refresh-token; ${csrf.cookie}`,
            "x-csrf-token": csrf.token,
          },
        }),
      );
      expect(response.status).toBe(401);
      expect(response.headers.getSetCookie()).toHaveLength(2);
    });
  });

  it("always clears local cookies when logout upstream is unavailable", async () => {
    await withServerEnv(async () => {
      const csrf = csrfCookie();
      vi.stubGlobal(
        "fetch",
        vi.fn<typeof fetch>(async () => {
          throw new Error("upstream unavailable");
        }),
      );
      const response = await logoutSession(
        request("/api/auth/logout", {
          method: "POST",
          headers: {
            cookie: `__Host-console-access=access-token; __Host-console-refresh=refresh-token; ${csrf.cookie}`,
            "x-csrf-token": csrf.token,
          },
        }),
      );
      expect(response.status).toBe(200);
      expect(response.headers.getSetCookie()).toEqual([
        "__Host-console-access=; Max-Age=0; HttpOnly; Secure; SameSite=Lax; Path=/",
        "__Host-console-refresh=; Max-Age=0; HttpOnly; Secure; SameSite=Lax; Path=/",
        "__Host-console-csrf=; Max-Age=0; Secure; SameSite=Lax; Path=/",
      ]);
      await expect(response.json()).resolves.toEqual({ ok: true, revocationConfirmed: false });
    });
  });

  it("confirms upstream logout when the operation succeeds", async () => {
    await withServerEnv(async () => {
      const csrf = csrfCookie();
      vi.stubGlobal(
        "fetch",
        vi.fn<typeof fetch>(async () => Response.json({ code: 0, message: "ok", data: {} })),
      );
      const response = await logoutSession(
        request("/api/auth/logout", {
          method: "POST",
          headers: {
            cookie: `__Host-console-access=access-token; ${csrf.cookie}`,
            "x-csrf-token": csrf.token,
          },
        }),
      );
      await expect(response.json()).resolves.toEqual({ ok: true, revocationConfirmed: true });
    });
  });

  it("rejects a missing access session before calling upstream", async () => {
    await withServerEnv(async () => {
      const fetchMock = vi.fn<typeof fetch>();
      vi.stubGlobal("fetch", fetchMock);
      await expect(getSession(request("/api/auth/session"))).rejects.toMatchObject({
        kind: "unauthenticated",
        status: 401,
      });
      expect(fetchMock).not.toHaveBeenCalled();
    });
  });

  it.each([
    ["unauthenticated", 401, "登录状态已失效，请重新登录"],
    ["forbidden", 403, "安全校验失败，请刷新页面重试"],
    ["invalid-envelope", 502, "登录服务返回了无法识别的响应"],
    ["timeout", 504, "登录服务响应超时，请稍后重试"],
    ["upstream", 502, "登录服务暂时不可用，请稍后重试"],
  ] as const)("publishes a safe %s error", async (kind, status, message) => {
    const response = publicErrorResponse(
      new TransportError(kind, "private", { status, requestId: "request-1" }),
    );
    expect(response.status).toBe(status);
    expect(response.headers.get("x-request-id")).toBe("request-1");
    await expect(response.json()).resolves.toEqual({ error: { code: kind, message } });
  });
});
