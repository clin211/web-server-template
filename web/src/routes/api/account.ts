import { createFileRoute } from "@tanstack/react-router";
import {
  deleteAccount,
  publicAccountErrorResponse,
  updateAccount,
} from "../../features/account/server";
import { TransportError } from "../../infrastructure/http/errors";

export const Route = createFileRoute("/api/account")({
  server: {
    handlers: {
      PUT: ({ request }) => handleAccountRequest(() => updateAccount(request)),
      DELETE: ({ request }) => handleAccountRequest(() => deleteAccount(request)),
    },
  },
});

async function handleAccountRequest(action: () => Promise<Response>): Promise<Response> {
  try {
    return await action();
  } catch (error) {
    if (error instanceof TransportError) return publicAccountErrorResponse(error);
    return Response.json(
      { error: { code: "internal", message: "账号请求暂时无法处理" } },
      { status: 502, headers: { "cache-control": "no-store" } },
    );
  }
}
