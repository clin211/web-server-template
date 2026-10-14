import { describe, expect, it, vi } from "vitest";
import { TransportError } from "../../../infrastructure/http/errors";
import { issueCsrfToken } from "../../../infrastructure/session/csrf";
import { registerAccount } from "./server";

const secret = "registration-test-secret-with-at-least-32-bytes";

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

function validRequest() {
  const csrf = issueCsrfToken(secret);
  return new Request("https://console.test/api/register", {
    method: "POST",
    headers: {
      origin: "https://console.test",
      cookie: `__Host-console-csrf=${csrf}`,
      "x-csrf-token": csrf,
      "content-type": "application/json",
    },
    body: JSON.stringify({
      username: "valid_user",
      password: "secret1",
      confirmPassword: "secret1",
      nickname: "Ada",
      email: "ada@example.test",
      phone: "13800138000",
    }),
  });
}

describe("registration server adapter", () => {
  it("maps a valid registration to the anonymous create operation", async () => {
    await withServerEnv(async () => {
      const fetchMock = vi.fn<typeof fetch>(async (input, init) => {
        expect(String(input)).toBe("http://api.console.test/v1/users");
        expect(new Headers(init?.headers).get("authorization")).toBeNull();
        expect(JSON.parse(String(init?.body))).not.toHaveProperty("confirmPassword");
        return Response.json({ code: 0, message: "ok", data: { userID: "user-1" } });
      });
      vi.stubGlobal("fetch", fetchMock);
      const response = await registerAccount(validRequest());
      expect(response.status).toBe(200);
      expect(response.headers.getSetCookie()).toContain(
        "__Host-console-csrf=; Max-Age=0; Secure; SameSite=Lax; Path=/",
      );
    });
  });

  it("maps an upstream conflict without leaking its message", async () => {
    await withServerEnv(async () => {
      vi.stubGlobal(
        "fetch",
        vi.fn<typeof fetch>(async () =>
          Response.json(
            { code: 40901, reason: "User.AlreadyExists", message: "private detail" },
            { status: 409 },
          ),
        ),
      );
      const response = await registerAccount(validRequest());
      expect(response.status).toBe(409);
      await expect(response.json()).resolves.toEqual({
        error: { code: "already-exists", message: "用户名、邮箱或手机号已被使用" },
      });
    });
  });

  it.each([
    [400, "invalid-input"],
    [429, "rate-limited"],
  ])("maps upstream HTTP %i to %s", async (status, code) => {
    await withServerEnv(async () => {
      vi.stubGlobal(
        "fetch",
        vi.fn<typeof fetch>(async () =>
          Response.json({ code: status, reason: "rejected", message: "private" }, { status }),
        ),
      );
      const response = await registerAccount(validRequest());
      expect(response.status).toBe(status);
      await expect(response.json()).resolves.toMatchObject({ error: { code } });
    });
  });

  it("rejects invalid registration before calling upstream", async () => {
    await withServerEnv(async () => {
      const csrf = issueCsrfToken(secret);
      const fetchMock = vi.fn<typeof fetch>();
      vi.stubGlobal("fetch", fetchMock);
      const response = await registerAccount(
        new Request("https://console.test/api/register", {
          method: "POST",
          headers: {
            origin: "https://console.test",
            cookie: `__Host-console-csrf=${csrf}`,
            "x-csrf-token": csrf,
            "content-type": "application/json",
          },
          body: JSON.stringify({ username: "invalid" }),
        }),
      );
      expect(response.status).toBe(400);
      expect(fetchMock).not.toHaveBeenCalled();
    });
  });

  it.each([
    [Response.json({ code: 0, message: "ok", data: {} }), "invalid-envelope"],
    [
      Response.json(
        { code: 500, reason: "Service.Unavailable", message: "private" },
        { status: 500, headers: { "x-request-id": "request-1" } },
      ),
      "upstream",
    ],
  ])("fails closed for an unusable upstream response", async (upstream, kind) => {
    await withServerEnv(async () => {
      vi.stubGlobal(
        "fetch",
        vi.fn<typeof fetch>(async () => upstream.clone()),
      );
      const error = await registerAccount(validRequest()).catch((value: unknown) => value);
      expect(error).toBeInstanceOf(TransportError);
      expect(error).toMatchObject({ kind, status: 502 });
    });
  });
});
