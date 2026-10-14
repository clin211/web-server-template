import { createFileRoute } from "@tanstack/react-router";
import { login, publicErrorResponse } from "../../../features/session/server";
import { TransportError } from "../../../infrastructure/http/errors";

export const Route = createFileRoute("/api/auth/login")({
  server: {
    handlers: {
      POST: async ({ request }) => {
        try {
          return await login(request);
        } catch (error) {
          if (error instanceof TransportError) {
            return publicErrorResponse(error);
          }
          return Response.json(
            { error: { code: "internal", message: "登录服务暂时不可用，请稍后重试" } },
            { status: 502, headers: { "cache-control": "no-store" } },
          );
        }
      },
    },
  },
});
