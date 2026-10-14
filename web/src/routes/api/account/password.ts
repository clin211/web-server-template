import { createFileRoute } from "@tanstack/react-router";
import {
  changeAccountPassword,
  publicAccountErrorResponse,
} from "../../../features/account/server";
import { TransportError } from "../../../infrastructure/http/errors";

export const Route = createFileRoute("/api/account/password")({
  server: {
    handlers: {
      PUT: async ({ request }) => {
        try {
          return await changeAccountPassword(request);
        } catch (error) {
          if (error instanceof TransportError) return publicAccountErrorResponse(error);
          return Response.json(
            { error: { code: "internal", message: "密码请求暂时无法处理" } },
            { status: 502, headers: { "cache-control": "no-store" } },
          );
        }
      },
    },
  },
});
