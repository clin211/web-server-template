import { createFileRoute } from "@tanstack/react-router";
import { registerAccount } from "../../features/registration/server";
import { TransportError } from "../../infrastructure/http/errors";

export const Route = createFileRoute("/api/register")({
  server: {
    handlers: {
      POST: async ({ request }) => {
        try {
          return await registerAccount(request);
        } catch (error) {
          const requestId = error instanceof TransportError ? error.requestId : null;
          return Response.json(
            { error: { code: "register-failed", message: "注册服务暂时不可用，请稍后重试" } },
            {
              status: 502,
              headers: {
                "cache-control": "no-store",
                ...(requestId ? { "x-request-id": requestId } : {}),
              },
            },
          );
        }
      },
    },
  },
});
