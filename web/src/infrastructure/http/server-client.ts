import "@tanstack/react-start/server-only";
import { TransportError } from "./errors";
import { assertAllowedMethod, assertRelativePath, getCredentialHeader } from "./request-policy";
import { getRequestId, getTraceContext } from "../observability/request-context";
import type { CredentialPolicy, SessionPort } from "../session/port";

const forbiddenRequestHeaders = [
  "authorization",
  "connection",
  "cookie",
  "host",
  "keep-alive",
  "proxy-connection",
  "te",
  "traceparent",
  "tracestate",
  "trailer",
  "transfer-encoding",
  "upgrade",
] as const;

export type ServerClientOptions = Readonly<{
  baseUrl: string;
  session: SessionPort;
  timeoutMs?: number;
  fetchFn?: typeof fetch;
}>;

export type ServerRequestOptions = Readonly<{
  request: Request;
  policy: CredentialPolicy;
  method?: string;
  headers?: HeadersInit;
  body?: BodyInit | null;
  signal?: AbortSignal;
}>;

export class ServerHttpClient {
  private readonly baseUrl: URL;
  private readonly session: SessionPort;
  private readonly timeoutMs: number;
  private readonly fetchFn: typeof fetch;

  constructor(options: ServerClientOptions) {
    this.baseUrl = new URL(options.baseUrl);
    this.session = options.session;
    this.timeoutMs = options.timeoutMs ?? 10_000;
    this.fetchFn = options.fetchFn ?? fetch;
  }

  async request(pathname: string, options: ServerRequestOptions): Promise<Response> {
    const method = assertAllowedMethod(options.method ?? "GET");
    const path = assertRelativePath(pathname);
    const url = new URL(path, this.baseUrl);
    const headers = new Headers(options.headers);
    const token =
      options.policy.kind === "access"
        ? this.session.readAccess(options.request)
        : options.policy.kind === "refresh"
          ? this.session.readRefresh(options.request)
          : null;
    const authorization = getCredentialHeader(options.policy, token);

    for (const name of forbiddenRequestHeaders) headers.delete(name);
    if (authorization) {
      headers.set("authorization", authorization);
    }
    headers.set("x-request-id", getRequestId(options.request));
    const traceContext = getTraceContext(options.request);
    if (traceContext) {
      headers.set("traceparent", traceContext.traceparent);
      if (traceContext.tracestate) headers.set("tracestate", traceContext.tracestate);
    }
    if (options.body && !headers.has("content-type")) {
      headers.set("content-type", "application/json");
    }

    const timeoutSignal = AbortSignal.timeout(this.timeoutMs);
    const signal = options.signal
      ? AbortSignal.any([options.signal, timeoutSignal])
      : timeoutSignal;

    try {
      return await this.fetchFn(url, {
        method,
        headers,
        body: options.body,
        signal,
      });
    } catch {
      if (signal.aborted) {
        throw new TransportError("timeout", "Upstream request timed out");
      }
      throw new TransportError("network", "Upstream request failed");
    }
  }
}
