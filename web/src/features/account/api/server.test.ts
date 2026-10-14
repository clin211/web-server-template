import { describe, expect, it, vi } from "vitest";
import { TransportError } from "../../../infrastructure/http/errors";
import { issueCsrfToken } from "../../../infrastructure/session/csrf";
import {
  changeAccountPassword,
  deleteAccount,
  publicAccountErrorResponse,
  updateAccount,
} from "./server";

const secret = "account-server-test-secret-with-at-least-32-bytes";

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

function writeRequest(path: string, body: unknown) {
  const csrf = issueCsrfToken(secret);
  return new Request(`https://console.test${path}`, {
    method: "POST",
    headers: {
      origin: "https://console.test",
      cookie: `__Host-console-access=access-token; __Host-console-refresh=refresh-token; __Host-console-csrf=${csrf}`,
      "x-csrf-token": csrf,
      "content-type": "application/json",
    },
    body: JSON.stringify(body),
  });
}

function identityResponse() {
  return Response.json({
    code: 0,
    message: "ok",
    data: { user: { userID: "user/1", username: "ada" } },
  });
}

describe("account server adapters", () => {
  it("derives the update target from the current session", async () => {
    await withServerEnv(async () => {
      const fetchMock = vi
        .fn<typeof fetch>()
        .mockResolvedValueOnce(identityResponse())
        .mockImplementationOnce(async (input, init) => {
          expect(String(input)).toBe("http://api.console.test/v1/users/user%2F1");
          expect(JSON.parse(String(init?.body))).toEqual({ nickname: "New Name" });
          expect(JSON.parse(String(init?.body))).not.toHaveProperty("userID");
          return Response.json({ code: 0, message: "ok", data: {} });
        });
      vi.stubGlobal("fetch", fetchMock);
      const response = await updateAccount(
        writeRequest("/api/account", { nickname: "New Name", userID: "attacker-target" }),
      );
      expect(response.status).toBe(200);
    });
  });

  it("clears both session cookies after changing the password", async () => {
    await withServerEnv(async () => {
      vi.stubGlobal(
        "fetch",
        vi
          .fn<typeof fetch>()
          .mockResolvedValueOnce(identityResponse())
          .mockResolvedValueOnce(Response.json({ code: 0, message: "ok", data: {} })),
      );
      const response = await changeAccountPassword(
        writeRequest("/api/account/password", {
          oldPassword: "old-password",
          newPassword: "new-password1",
          confirmPassword: "new-password1",
        }),
      );
      expect(response.headers.getSetCookie()).toEqual([
        "__Host-console-access=; Max-Age=0; HttpOnly; Secure; SameSite=Lax; Path=/",
        "__Host-console-refresh=; Max-Age=0; HttpOnly; Secure; SameSite=Lax; Path=/",
        "__Host-console-csrf=; Max-Age=0; Secure; SameSite=Lax; Path=/",
      ]);
    });
  });

  it("rejects account deletion when the username confirmation does not match", async () => {
    await withServerEnv(async () => {
      const fetchMock = vi.fn<typeof fetch>(async () => identityResponse());
      vi.stubGlobal("fetch", fetchMock);
      const response = await deleteAccount(
        writeRequest("/api/account", { confirmation: "different-user" }),
      );
      expect(response.status).toBe(400);
      expect(fetchMock).toHaveBeenCalledOnce();
    });
  });

  it("deletes only the resolved current account and clears credentials", async () => {
    await withServerEnv(async () => {
      const fetchMock = vi
        .fn<typeof fetch>()
        .mockResolvedValueOnce(identityResponse())
        .mockImplementationOnce(async (input, init) => {
          expect(String(input)).toBe("http://api.console.test/v1/users/user%2F1");
          expect(init?.method).toBe("DELETE");
          return Response.json({ code: 0, message: "ok", data: {} });
        });
      vi.stubGlobal("fetch", fetchMock);
      const response = await deleteAccount(writeRequest("/api/account", { confirmation: "ada" }));
      expect(response.status).toBe(200);
      expect(response.headers.getSetCookie()).toHaveLength(3);
    });
  });

  it("rejects an empty profile update before resolving identity", async () => {
    await withServerEnv(async () => {
      const fetchMock = vi.fn<typeof fetch>();
      vi.stubGlobal("fetch", fetchMock);
      const response = await updateAccount(writeRequest("/api/account", { userID: "target" }));
      expect(response.status).toBe(400);
      expect(fetchMock).not.toHaveBeenCalled();
    });
  });

  it.each([
    [400, "invalid-input"],
    [409, "conflict"],
  ])("maps account mutation HTTP %i to %s", async (status, code) => {
    await withServerEnv(async () => {
      vi.stubGlobal(
        "fetch",
        vi
          .fn<typeof fetch>()
          .mockResolvedValueOnce(identityResponse())
          .mockResolvedValueOnce(
            Response.json({ code: status, reason: "rejected", message: "private" }, { status }),
          ),
      );
      const response = await updateAccount(writeRequest("/api/account", { nickname: "New Name" }));
      expect(response.status).toBe(status);
      await expect(response.json()).resolves.toMatchObject({ error: { code } });
    });
  });

  it.each([
    [401, "unauthenticated"],
    [403, "forbidden"],
    [500, "upstream"],
  ])("maps identity HTTP %i to %s", async (status, kind) => {
    await withServerEnv(async () => {
      vi.stubGlobal(
        "fetch",
        vi.fn<typeof fetch>(async () =>
          Response.json({ code: status, reason: "rejected", message: "private" }, { status }),
        ),
      );
      await expect(
        updateAccount(writeRequest("/api/account", { nickname: "New Name" })),
      ).rejects.toMatchObject({ kind, status });
    });
  });

  it.each([
    ["unauthenticated", 401, "登录状态已失效，请重新登录"],
    ["forbidden", 403, "当前账号无权执行此操作"],
    ["timeout", 504, "账号服务响应超时，请稍后重试"],
    ["invalid-envelope", 400, "提交内容无法处理"],
    ["upstream", 502, "账号服务暂时不可用，请稍后重试"],
  ] as const)("publishes a safe %s account error", async (kind, status, message) => {
    const response = publicAccountErrorResponse(
      new TransportError(kind, "private", { status, requestId: "request-1" }),
    );
    expect(response.status).toBe(status);
    expect(response.headers.get("x-request-id")).toBe("request-1");
    await expect(response.json()).resolves.toEqual({ error: { code: kind, message } });
  });
});
