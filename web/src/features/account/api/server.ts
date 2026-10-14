import "@tanstack/react-start/server-only";
import { z } from "zod";
import { createGoApiClient } from "../../../integrations/go-api/client";
import { parseApiEnvelope } from "../../../integrations/go-api/contracts";
import { GoSessionAdapter } from "../../../integrations/go-api/session-adapter";
import { getServerEnv, type ServerEnv } from "../../../infrastructure/env";
import { TransportError } from "../../../infrastructure/http/errors";
import { readJsonBody } from "../../../infrastructure/http/json-body";
import { expireCsrfToken } from "../../../infrastructure/session/csrf";
import { requireWriteGuards } from "../../../infrastructure/session/guards";
import {
  changePasswordRequestSchema,
  deleteAccountRequestSchema,
  profileUpdateRequestSchema,
} from "../contracts";

const maxAccountBodyBytes = 16 * 1024;
const identitySchema = z.object({
  user: z.object({ userID: z.string().min(1), username: z.string().min(1) }).passthrough(),
});
const noStoreHeaders = { "cache-control": "no-store" };

export async function updateAccount(request: Request): Promise<Response> {
  const env = getServerEnv();
  const session = requireAccountWrite(request, env);
  const values = profileUpdateRequestSchema.safeParse(
    await readJsonBody(request, maxAccountBodyBytes),
  );
  if (!values.success) return errorResponse(400, "invalid-input", "资料更新内容不符合要求");
  const identity = await resolveCurrentIdentity(request, env, session);
  return runAccountMutation({
    request,
    env,
    session,
    path: `/v1/users/${encodeURIComponent(identity.userID)}`,
    method: "PUT",
    body: JSON.stringify(values.data),
    clearSession: false,
  });
}

export async function changeAccountPassword(request: Request): Promise<Response> {
  const env = getServerEnv();
  const session = requireAccountWrite(request, env);
  const values = changePasswordRequestSchema.safeParse(
    await readJsonBody(request, maxAccountBodyBytes),
  );
  if (!values.success) return errorResponse(400, "invalid-input", "密码信息不符合要求");
  const identity = await resolveCurrentIdentity(request, env, session);
  return runAccountMutation({
    request,
    env,
    session,
    path: `/v1/users/${encodeURIComponent(identity.userID)}/change-password`,
    method: "PUT",
    body: JSON.stringify(values.data),
    clearSession: true,
  });
}

export async function deleteAccount(request: Request): Promise<Response> {
  const env = getServerEnv();
  const session = requireAccountWrite(request, env);
  const values = deleteAccountRequestSchema.safeParse(
    await readJsonBody(request, maxAccountBodyBytes),
  );
  if (!values.success) return errorResponse(400, "invalid-confirmation", "请输入当前用户名确认");
  const identity = await resolveCurrentIdentity(request, env, session);
  if (values.data.confirmation !== identity.username) {
    return errorResponse(400, "invalid-confirmation", "用户名确认不匹配");
  }
  return runAccountMutation({
    request,
    env,
    session,
    path: `/v1/users/${encodeURIComponent(identity.userID)}`,
    method: "DELETE",
    clearSession: true,
  });
}

function requireAccountWrite(request: Request, env: ServerEnv): GoSessionAdapter {
  const secure = isSecure(env);
  requireWriteGuards(request, {
    origin: env.CONSOLE_ORIGIN,
    secret: env.CONSOLE_SESSION_SECRET,
    secure,
  });
  return new GoSessionAdapter({ secure });
}

async function resolveCurrentIdentity(
  request: Request,
  env: ServerEnv,
  session: GoSessionAdapter,
): Promise<{ userID: string; username: string }> {
  const client = createGoApiClient({
    baseUrl: env.BACKEND_API_URL,
    session,
    timeoutMs: env.CONSOLE_REQUEST_TIMEOUT_MS,
  });
  const upstream = await client.request("/v1/auth/info", {
    request,
    policy: { kind: "access" },
  });
  const envelope = await parseApiEnvelope(upstream);
  if ("reason" in envelope) throw accountUpstreamError(upstream);
  const parsed = identitySchema.safeParse(envelope.data);
  if (!parsed.success) throw invalidEnvelope(upstream);
  return parsed.data.user;
}

async function runAccountMutation(
  options: Readonly<{
    request: Request;
    env: ServerEnv;
    session: GoSessionAdapter;
    path: string;
    method: "PUT" | "DELETE";
    body?: string;
    clearSession: boolean;
  }>,
): Promise<Response> {
  const client = createGoApiClient({
    baseUrl: options.env.BACKEND_API_URL,
    session: options.session,
    timeoutMs: options.env.CONSOLE_REQUEST_TIMEOUT_MS,
  });
  const upstream = await client.request(options.path, {
    request: options.request,
    policy: { kind: "access" },
    method: options.method,
    body: options.body,
  });
  const envelope = await parseApiEnvelope(upstream);
  if ("reason" in envelope) {
    if (upstream.status === 400) return errorResponse(400, "invalid-input", "提交内容不符合要求");
    if (upstream.status === 409) return errorResponse(409, "conflict", "账号资料与现有记录冲突");
    throw accountUpstreamError(upstream);
  }

  const response = Response.json({ ok: true }, { headers: noStoreHeaders });
  if (options.clearSession) options.session.clear(response);
  return expireCsrfToken(response, isSecure(options.env));
}

function accountUpstreamError(response: Response): TransportError {
  const kind =
    response.status === 401
      ? "unauthenticated"
      : response.status === 403
        ? "forbidden"
        : "upstream";
  return new TransportError(kind, "Account upstream failed", {
    status: response.status >= 400 ? response.status : 502,
    requestId: response.headers.get("x-request-id"),
  });
}

function invalidEnvelope(response: Response): TransportError {
  return new TransportError("invalid-envelope", "Account response was invalid", {
    status: 502,
    requestId: response.headers.get("x-request-id"),
  });
}

function isSecure(env: ServerEnv): boolean {
  return new URL(env.CONSOLE_ORIGIN).protocol === "https:";
}

function errorResponse(status: number, code: string, message: string): Response {
  return Response.json({ error: { code, message } }, { status, headers: noStoreHeaders });
}

export function publicAccountErrorResponse(error: TransportError): Response {
  const status = error.status && error.status >= 400 ? error.status : 502;
  const message =
    error.kind === "unauthenticated"
      ? "登录状态已失效，请重新登录"
      : error.kind === "forbidden"
        ? "当前账号无权执行此操作"
        : error.kind === "timeout"
          ? "账号服务响应超时，请稍后重试"
          : error.kind === "invalid-envelope" && status < 500
            ? "提交内容无法处理"
            : "账号服务暂时不可用，请稍后重试";
  return Response.json(
    { error: { code: error.kind, message } },
    {
      status,
      headers: {
        ...noStoreHeaders,
        ...(error.requestId ? { "x-request-id": error.requestId } : {}),
      },
    },
  );
}
