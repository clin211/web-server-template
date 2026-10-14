import { afterEach, describe, expect, it, vi } from "vitest";
import { registerAccount } from "./client";

afterEach(() => vi.unstubAllGlobals());

describe("registration browser client", () => {
  it("submits registration through the guarded same-origin route", async () => {
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(Response.json({ token: "csrf-token" }))
      .mockResolvedValueOnce(Response.json({ ok: true }));
    vi.stubGlobal("fetch", fetchMock);
    const values = {
      username: "ada_user",
      password: "secret1",
      confirmPassword: "secret1",
      nickname: "Ada",
      email: "ada@example.test",
      phone: "13800138000",
    };

    await registerAccount(values);

    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "/api/register",
      expect.objectContaining({ method: "POST", credentials: "include" }),
    );
    const options = fetchMock.mock.calls[1]?.[1];
    expect(new Headers(options?.headers).get("x-csrf-token")).toBe("csrf-token");
    expect(JSON.parse(String(options?.body))).toEqual(values);
  });
});
