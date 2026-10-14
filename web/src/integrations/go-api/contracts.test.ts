import { describe, expect, it } from "vitest";
import { TransportError } from "../../infrastructure/http/errors";
import { parseApiEnvelope } from "./contracts";

describe("Go API envelopes", () => {
  it("accepts a successful response only when HTTP status is successful", async () => {
    await expect(
      parseApiEnvelope(Response.json({ code: 0, message: "ok", data: { id: "user-1" } })),
    ).resolves.toMatchObject({ code: 0, data: { id: "user-1" } });
  });

  it("returns a typed failure envelope for an upstream error", async () => {
    await expect(
      parseApiEnvelope(
        Response.json(
          { code: 10001, reason: "unauthenticated", message: "session expired" },
          { status: 401 },
        ),
      ),
    ).resolves.toMatchObject({ reason: "unauthenticated" });
  });

  it.each([
    ["invalid JSON", new Response("not json", { status: 502 })],
    ["invalid envelope", Response.json({ code: 0, data: null })],
  ])("fails closed for %s", async (_label, response) => {
    const error = await parseApiEnvelope(response).catch((value: unknown) => value);
    expect(error).toBeInstanceOf(TransportError);
    expect(error).toMatchObject({ kind: "invalid-envelope" });
  });
});
