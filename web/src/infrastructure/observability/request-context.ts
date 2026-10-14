import { requestIdSchema } from "../../shared/validation/common";

export type RequestContext = {
  requestId: string;
  startedAt: number;
  traceId: string | null;
};

export type TraceContext = Readonly<{
  traceId: string;
  traceparent: string;
  tracestate?: string;
}>;

const traceparentPattern = /^00-([0-9a-f]{32})-([0-9a-f]{16})-([0-9a-f]{2})$/;
const simpleTracestateKeyPattern = /^[a-z][a-z0-9_*/-]{0,255}$/;
const tenantTracestateKeyPattern = /^[a-z0-9][a-z0-9_*/-]{0,240}@[a-z][a-z0-9_*/-]{0,13}$/;

export function getRequestId(request: Request): string {
  const candidate = request.headers.get("x-request-id");
  const parsed = candidate ? requestIdSchema.safeParse(candidate) : null;
  const requestId = parsed?.success ? parsed.data : crypto.randomUUID();
  request.headers.set("x-request-id", requestId);
  return requestId;
}

export function getTraceContext(request: Request): TraceContext | null {
  const traceparent = request.headers.get("traceparent");
  if (!traceparent) return null;
  const match = traceparent.match(traceparentPattern);
  if (!match || isAllZero(match[1]) || isAllZero(match[2])) return null;

  const tracestate = normalizeTracestate(request.headers.get("tracestate"));
  return {
    traceId: match[1],
    traceparent,
    ...(tracestate ? { tracestate } : {}),
  };
}

function normalizeTracestate(value: string | null): string | null {
  if (!value || value.length > 512) return null;
  const members = value.split(",").map((member) => member.trim());
  if (members.length === 0 || members.length > 32) return null;

  const keys = new Set<string>();
  for (const member of members) {
    const separator = member.indexOf("=");
    if (separator <= 0 || member.indexOf("=", separator + 1) !== -1) return null;
    const key = member.slice(0, separator);
    const memberValue = member.slice(separator + 1);
    if (!isValidTracestateKey(key) || !isValidTracestateValue(memberValue) || keys.has(key)) {
      return null;
    }
    keys.add(key);
  }

  return members.join(",");
}

function isValidTracestateKey(key: string): boolean {
  return simpleTracestateKeyPattern.test(key) || tenantTracestateKeyPattern.test(key);
}

function isValidTracestateValue(value: string): boolean {
  if (value.length === 0 || value.length > 256 || value.startsWith(" ") || value.endsWith(" ")) {
    return false;
  }
  return [...value].every((character) => {
    const code = character.charCodeAt(0);
    return code >= 0x20 && code <= 0x7e && character !== "," && character !== "=";
  });
}

function isAllZero(value: string | undefined): boolean {
  return value === undefined || /^0+$/.test(value);
}
