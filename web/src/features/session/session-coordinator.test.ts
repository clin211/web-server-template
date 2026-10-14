import { afterEach, describe, expect, it, vi } from "vitest";
import { createSessionCoordinator } from "./session-coordinator";

afterEach(() => vi.unstubAllGlobals());

describe("session coordinator", () => {
  it("refreshes once and replays the session request once", async () => {
    const user = {
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
    };
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(
        Response.json({ error: { code: "unauthenticated", message: "expired" } }, { status: 401 }),
      )
      .mockResolvedValueOnce(Response.json({ token: "csrf-token" }))
      .mockResolvedValueOnce(Response.json({ ok: true }))
      .mockResolvedValueOnce(Response.json({ user }));
    vi.stubGlobal("fetch", fetchMock);

    const coordinator = createSessionCoordinator();
    await expect(coordinator.requireSession()).resolves.toEqual({ user });
    expect(fetchMock).toHaveBeenCalledTimes(4);
  });

  it("does not retry a non-session failure", async () => {
    const fetchMock = vi.fn<typeof fetch>(async () =>
      Response.json({ error: { code: "upstream", message: "down" } }, { status: 502 }),
    );
    vi.stubGlobal("fetch", fetchMock);
    const coordinator = createSessionCoordinator();
    await expect(coordinator.requireSession()).rejects.toMatchObject({ status: 502 });
    expect(fetchMock).toHaveBeenCalledOnce();
  });

  it("shares one refresh operation across concurrent callers", async () => {
    let completeRefresh = () => {};
    const refresh = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          completeRefresh = resolve;
        }),
    );
    const coordinator = createSessionCoordinator({
      fetchSession: vi.fn(),
      refreshSession: refresh,
    });

    const first = coordinator.refreshOnce();
    const second = coordinator.refreshOnce();
    expect(refresh).toHaveBeenCalledOnce();

    completeRefresh();
    await expect(Promise.all([first, second])).resolves.toEqual([undefined, undefined]);
  });
});
