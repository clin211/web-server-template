import "@tanstack/react-start/server-only";
import { createGoApiClient } from "../../../integrations/go-api/client";
import { parseApiEnvelope } from "../../../integrations/go-api/contracts";
import {
  GoSessionAdapter,
  refreshExpiryFromToken,
} from "../../../integrations/go-api/session-adapter";
import { getServerEnv, type ServerEnv } from "../../../infrastructure/env";
import { TransportError, type TransportErrorKind } from "../../../infrastructure/http/errors";
import { requireWriteGuards } from "../../../infrastructure/session/guards";
import { expireCsrfToken } from "../../../infrastructure/session/csrf";
import { loginFormSchema, loginResponseDataSchema, sessionDataSchema } from "../contracts";

const maxLoginBodyBytes = 8 * 1024;
const noStoreHeaders = { "cache-control": "no-store" };

export async function login(request: Request): Promise<Response> {
  const env = getServerEnv();
  assertJsonRequest(request);
  requireValidWrite(request, env);

  const values = loginFormSchema.safeParse(await readJson(request));
  if (!values.success) {
    return errorResponse(400, "invalid-input", "请输入有效的账号和密码");
  }

  const session = createSession(env);
  const upstream = await createGoApiClient({
    baseUrl: env.BACKEND_API_URL,
    session,
    timeoutMs: env.CONSOLE_REQUEST_TIMEOUT_MS,
  }).request("/v1/auth/login", {
    request,
    policy: { kind: "anonymous" },
    method: "POST",
    body: JSON.stringify(values.data),
  });
  const envelope = await parseApiEnvelope(upstream);

  if ("reason" in envelope) {
    if (upstream.status === 401) return errorResponse(401, "invalid-credentials", "账号或密码错误");
    if (upstream.status === 429)
      return errorResponse(429, "rate-limited", "请求过于频繁，请稍后再试");
    throw mapUpstreamFailure(upstream.status, envelope.reason, upstream);
  }

  const parsed = loginResponseDataSchema.safeParse(envelope.data);
  if (!parsed.success) throw invalidEnvelope(upstream);
  const refreshExpiresAt = refreshExpiryFromToken(parsed.data.refreshToken);
  if (!refreshExpiresAt) throw invalidEnvelope(upstream);

  const response = Response.json({ ok: true }, { headers: noStoreHeaders });
  session.commit(response, {
    accessToken: parsed.data.accessToken,
    refreshToken: parsed.data.refreshToken,
    accessExpiresAt: parsed.data.expireAt,
    refreshExpiresAt,
  });
  return expireCsrfToken(response, isSecure(env));
}

export async function getSession(request: Request): Promise<Response> {
  const env = getServerEnv();
  const session = createSession(env);
  if (!session.readAccess(request)) {
    throw new TransportError("unauthenticated", "Session is missing", { status: 401 });
  }

  const upstream = await createGoApiClient({
    baseUrl: env.BACKEND_API_URL,
    session,
    timeoutMs: env.CONSOLE_REQUEST_TIMEOUT_MS,
  }).request("/v1/auth/info", { request, policy: { kind: "access" } });
  const envelope = await parseApiEnvelope(upstream);
  if ("reason" in envelope) {
    throw mapUpstreamFailure(upstream.status, envelope.reason, upstream);
  }

  const parsed = sessionDataSchema.safeParse(envelope.data);
  if (!parsed.success) throw invalidEnvelope(upstream);
  return Response.json(parsed.data, { headers: noStoreHeaders });
}

export async function refreshSession(request: Request): Promise<Response> {
  const env = getServerEnv();
  requireValidWrite(request, env);
  const session = createSession(env);

  try {
    const upstream = await createGoApiClient({
      baseUrl: env.BACKEND_API_URL,
      session,
      timeoutMs: env.CONSOLE_REQUEST_TIMEOUT_MS,
    }).request("/v1/auth/refresh-token", {
      request,
      policy: { kind: "refresh" },
      method: "PUT",
    });
    const envelope = await parseApiEnvelope(upstream);
    if ("reason" in envelope) {
      const error = mapUpstreamFailure(upstream.status, envelope.reason, upstream);
      if (error.kind === "unauthenticated") return session.clear(publicErrorResponse(error));
      throw error;
    }

    const parsed = loginResponseDataSchema.safeParse(envelope.data);
    const refreshExpiresAt = parsed.success
      ? refreshExpiryFromToken(parsed.data.refreshToken)
      : null;
    if (!parsed.success || !refreshExpiresAt) throw invalidEnvelope(upstream);

    const response = Response.json({ ok: true }, { headers: noStoreHeaders });
    session.commit(response, {
      accessToken: parsed.data.accessToken,
      refreshToken: parsed.data.refreshToken,
      accessExpiresAt: parsed.data.expireAt,
      refreshExpiresAt,
    });
    return expireCsrfToken(response, isSecure(env));
  } catch (error) {
    if (error instanceof Error && !(error instanceof TransportError)) {
      throw new TransportError("unauthenticated", "Session is missing", { status: 401 });
    }
    throw error;
  }
}

export async function logoutSession(request: Request): Promise<Response> {
  const env = getServerEnv();
  requireValidWrite(request, env);
  const session = createSession(env);
  let revocationConfirmed = false;

  try {
    if (session.readAccess(request)) {
      const upstream = await createGoApiClient({
        baseUrl: env.BACKEND_API_URL,
        session,
        timeoutMs: env.CONSOLE_REQUEST_TIMEOUT_MS,
      }).request("/v1/auth/logout", {
        request,
        policy: { kind: "access" },
        method: "POST",
      });
      const envelope = await parseApiEnvelope(upstream);
      if (!("reason" in envelope)) revocationConfirmed = true;
    }
  } catch {
    // Local cookies are cleared even when upstream revocation cannot be confirmed.
  }

  const response = Response.json({ ok: true, revocationConfirmed }, { headers: noStoreHeaders });
  session.clear(response);
  return expireCsrfToken(response, isSecure(env));
}

export function publicErrorResponse(error: TransportError): Response {
  const status =
    error.status && error.status >= 400 ? error.status : error.kind === "forbidden" ? 403 : 502;
  return Response.json(
    { error: { code: error.kind, message: safeMessage(error.kind) } },
    {
      status,
      headers: {
        ...noStoreHeaders,
        ...(error.requestId ? { "x-request-id": error.requestId } : {}),
      },
    },
  );
}

function createSession(env: ServerEnv): GoSessionAdapter {
  return new GoSessionAdapter({ secure: isSecure(env) });
}

function isSecure(env: ServerEnv): boolean {
  return new URL(env.CONSOLE_ORIGIN).protocol === "https:";
}

function requireValidWrite(request: Request, env: ServerEnv): void {
  requireWriteGuards(request, {
    origin: env.CONSOLE_ORIGIN,
    secret: env.CONSOLE_SESSION_SECRET,
    secure: isSecure(env),
  });
}

function mapUpstreamFailure(status: number, _reason: string, response: Response): TransportError {
  const kind: TransportErrorKind =
    status === 401 ? "unauthenticated" : status === 403 ? "forbidden" : "upstream";
  return new TransportError(kind, "Upstream request was rejected", {
    status: status >= 400 ? status : 502,
    requestId: response.headers.get("x-request-id"),
  });
}

function invalidEnvelope(response: Response): TransportError {
  return new TransportError("invalid-envelope", "Upstream response was invalid", {
    status: response.status,
    requestId: response.headers.get("x-request-id"),
  });
}

function safeMessage(kind: TransportErrorKind): string {
  if (kind === "unauthenticated") return "登录状态已失效，请重新登录";
  if (kind === "forbidden") return "安全校验失败，请刷新页面重试";
  if (kind === "invalid-envelope") return "登录服务返回了无法识别的响应";
  if (kind === "timeout") return "登录服务响应超时，请稍后重试";
  return "登录服务暂时不可用，请稍后重试";
}

function assertJsonRequest(request: Request): void {
  const contentType = request.headers.get("content-type")?.split(";", 1)[0].trim().toLowerCase();
  const contentLength = Number(request.headers.get("content-length") ?? "0");
  if (contentType !== "application/json" || contentLength > maxLoginBodyBytes) {
    throw new TransportError("invalid-envelope", "Login request is not acceptable", {
      status: 400,
    });
  }
}

async function readJson(request: Request): Promise<unknown> {
  try {
    const body = await request.text();
    if (new TextEncoder().encode(body).byteLength > maxLoginBodyBytes) {
      throw new TransportError("invalid-envelope", "Login request is too large", { status: 413 });
    }
    return JSON.parse(body) as unknown;
  } catch (error) {
    if (error instanceof TransportError) throw error;
    throw new TransportError("invalid-envelope", "Login request is not valid JSON", {
      status: 400,
    });
  }
}

function errorResponse(status: number, code: string, message: string): Response {
  return Response.json({ error: { code, message } }, { status, headers: noStoreHeaders });
}
