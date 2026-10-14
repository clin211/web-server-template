import { describe, expect, it } from "vitest";
import { BrowserRequestError, toBrowserRequestError } from "./browser-error";

describe("browser request error", () => {
  it("preserves only a validated request ID", async () => {
    const safe = await toBrowserRequestError(
      Response.json(
        { error: { code: "conflict", message: "请求冲突" } },
        { status: 409, headers: { "x-request-id": "request-123" } },
      ),
    );
    const unsafe = new BrowserRequestError("failed", 500, "请求失败", "contains spaces");

    expect(safe.requestId).toBe("request-123");
    expect(unsafe.requestId).toBeNull();
  });

  it("uses public defaults when the response has no safe error body", async () => {
    const error = await toBrowserRequestError(
      new Response("private upstream stack", { status: 502 }),
    );

    expect(error).toMatchObject({
      code: "request-failed",
      message: "请求暂时无法完成",
      requestId: null,
      status: 502,
    });
  });
});
