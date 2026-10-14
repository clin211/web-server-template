import { HttpResponse, http } from "msw";
import { describe, expect, it } from "vitest";
import type { SessionPort } from "../../infrastructure/session/port";
import { fixtureServer } from "../../test/fixture-server";
import { createGoApiClient } from "./client";
import { parseApiEnvelope } from "./contracts";

const session: SessionPort = {
  readAccess: () => "fixture-access-token",
  readRefresh: () => null,
  commit: (response) => response,
  clear: (response) => response,
};

describe("Go API integration port", () => {
  it("executes a credential policy against the fixture backend", async () => {
    let authorization: string | null = null;
    fixtureServer.use(
      http.get("https://api.console.test/v1/fixture", ({ request }) => {
        authorization = request.headers.get("authorization");
        return HttpResponse.json({
          code: 0,
          message: "ok",
          data: { fixture: true },
        });
      }),
    );

    const client = createGoApiClient({ baseUrl: "https://api.console.test", session });
    const response = await client.request("/v1/fixture", {
      request: new Request("https://console.test/api/fixture"),
      policy: { kind: "access" },
    });

    await expect(parseApiEnvelope(response)).resolves.toMatchObject({
      code: 0,
      data: { fixture: true },
    });
    expect(authorization).toBe("Bearer fixture-access-token");
  });
});
