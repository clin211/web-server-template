import { createFileRoute } from "@tanstack/react-router";
import { logoutSession, publicErrorResponse } from "../../../features/session/server";
import { TransportError } from "../../../infrastructure/http/errors";

export const Route = createFileRoute("/api/auth/logout")({
  server: {
    handlers: {
      POST: async ({ request }) => {
        try {
          return await logoutSession(request);
        } catch (error) {
          if (error instanceof TransportError) return publicErrorResponse(error);
          return Response.json(
            { error: { code: "internal", message: "退出请求暂时无法处理" } },
            { status: 502, headers: { "cache-control": "no-store" } },
          );
        }
      },
    },
  },
});
