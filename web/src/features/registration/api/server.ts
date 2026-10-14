import "@tanstack/react-start/server-only";
import { createGoApiClient } from "../../../integrations/go-api/client";
import { parseApiEnvelope } from "../../../integrations/go-api/contracts";
import { GoSessionAdapter } from "../../../integrations/go-api/session-adapter";
import { getServerEnv } from "../../../infrastructure/env";
import { TransportError } from "../../../infrastructure/http/errors";
import { readJsonBody } from "../../../infrastructure/http/json-body";
import { expireCsrfToken } from "../../../infrastructure/session/csrf";
import { requireWriteGuards } from "../../../infrastructure/session/guards";
import { registrationRequestSchema, registrationResponseSchema } from "../contracts";

const maxRegistrationBodyBytes = 16 * 1024;
const noStoreHeaders = { "cache-control": "no-store" };

export async function registerAccount(request: Request): Promise<Response> {
  const env = getServerEnv();
  const secure = new URL(env.CONSOLE_ORIGIN).protocol === "https:";
  requireWriteGuards(request, {
    origin: env.CONSOLE_ORIGIN,
    secret: env.CONSOLE_SESSION_SECRET,
    secure,
  });

  const parsed = registrationRequestSchema.safeParse(
    await readJsonBody(request, maxRegistrationBodyBytes),
  );
  if (!parsed.success) return errorResponse(400, "invalid-input", "注册信息不符合要求");

  const client = createGoApiClient({
    baseUrl: env.BACKEND_API_URL,
    session: new GoSessionAdapter({ secure }),
    timeoutMs: env.CONSOLE_REQUEST_TIMEOUT_MS,
  });
  const upstream = await client.request("/v1/users", {
    request,
    policy: { kind: "anonymous" },
    method: "POST",
    body: JSON.stringify(parsed.data),
  });
  const envelope = await parseApiEnvelope(upstream);
  if ("reason" in envelope) {
    if (upstream.status === 409)
      return errorResponse(409, "already-exists", "用户名、邮箱或手机号已被使用");
    if (upstream.status === 429)
      return errorResponse(429, "rate-limited", "请求过于频繁，请稍后再试");
    if (upstream.status === 400) return errorResponse(400, "invalid-input", "注册信息不符合要求");
    throw new TransportError("upstream", "Registration upstream failed", {
      status: 502,
      requestId: upstream.headers.get("x-request-id"),
    });
  }

  const data = registrationResponseSchema.safeParse(envelope.data);
  if (!data.success) {
    throw new TransportError("invalid-envelope", "Registration response was invalid", {
      status: 502,
      requestId: upstream.headers.get("x-request-id"),
    });
  }

  const response = Response.json({ ok: true }, { headers: noStoreHeaders });
  return expireCsrfToken(response, secure);
}

function errorResponse(status: number, code: string, message: string): Response {
  return Response.json({ error: { code, message } }, { status, headers: noStoreHeaders });
}
