import { createFileRoute } from "@tanstack/react-router";
import { getSession, publicErrorResponse } from "../../../features/session/server";
import { TransportError } from "../../../infrastructure/http/errors";

export const Route = createFileRoute("/api/auth/session")({
  server: {
    handlers: {
      GET: async ({ request }) => {
        try {
          return await getSession(request);
        } catch (error) {
          if (error instanceof TransportError) return publicErrorResponse(error);
          return Response.json(
            { error: { code: "internal", message: "当前账号暂时无法读取" } },
            { status: 502, headers: { "cache-control": "no-store" } },
          );
        }
      },
    },
  },
});
