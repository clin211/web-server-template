import { createFileRoute } from "@tanstack/react-router";
import { publicErrorResponse, refreshSession } from "../../../features/session/server";
import { TransportError } from "../../../infrastructure/http/errors";

export const Route = createFileRoute("/api/auth/refresh")({
  server: {
    handlers: {
      POST: async ({ request }) => {
        try {
          return await refreshSession(request);
        } catch (error) {
          if (error instanceof TransportError) return publicErrorResponse(error);
          return Response.json(
            { error: { code: "internal", message: "登录状态暂时无法恢复" } },
            { status: 502, headers: { "cache-control": "no-store" } },
          );
        }
      },
    },
  },
});
