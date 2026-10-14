import { afterEach, describe, expect, it, vi } from "vitest";
import { changeCurrentPassword, deleteCurrentAccount, updateCurrentAccount } from "./client";

afterEach(() => vi.unstubAllGlobals());

describe("account browser client", () => {
  it.each([
    ["update", () => updateCurrentAccount({ nickname: "Ada" }), "/api/account", "PUT"],
    [
      "password",
      () =>
        changeCurrentPassword({
          oldPassword: "old-password",
          newPassword: "new-password1",
          confirmPassword: "new-password1",
        }),
      "/api/account/password",
      "PUT",
    ],
    ["delete", () => deleteCurrentAccount("ada"), "/api/account", "DELETE"],
  ] as const)("sends a guarded %s request", async (_name, action, path, method) => {
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(Response.json({ token: "csrf-token" }))
      .mockResolvedValueOnce(Response.json({ ok: true }));
    vi.stubGlobal("fetch", fetchMock);

    await action();

    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      path,
      expect.objectContaining({ method, credentials: "include" }),
    );
    const headers = new Headers(fetchMock.mock.calls[1]?.[1]?.headers);
    expect(headers.get("x-csrf-token")).toBe("csrf-token");
  });

  it("preserves a public API error", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn<typeof fetch>()
        .mockResolvedValueOnce(Response.json({ token: "csrf-token" }))
        .mockResolvedValueOnce(
          Response.json({ error: { code: "conflict", message: "账号资料冲突" } }, { status: 409 }),
        ),
    );

    await expect(updateCurrentAccount({ nickname: "Ada" })).rejects.toMatchObject({
      code: "conflict",
      status: 409,
      message: "账号资料冲突",
    });
  });
});
