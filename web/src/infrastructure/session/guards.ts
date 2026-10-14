import "@tanstack/react-start/server-only";
import { timingSafeEqual } from "node:crypto";
import { TransportError } from "../http/errors";
import { readCookie } from "./cookies";
import { csrfCookieName, verifyCsrfToken } from "./csrf";

export function requireSameOrigin(request: Request, expectedOrigin: string): void {
  const origin = request.headers.get("origin") ?? request.headers.get("referer");
  if (!origin || !sameOrigin(origin, expectedOrigin)) {
    throw new TransportError("forbidden", "Request origin is not trusted", { status: 403 });
  }
}

export function requireCsrfToken(request: Request, expectedToken: string | null): void {
  const provided = request.headers.get("x-csrf-token");
  if (!provided || !expectedToken || !safeEqual(provided, expectedToken)) {
    throw new TransportError("forbidden", "CSRF validation failed", { status: 403 });
  }
}

export function requireWriteGuards(
  request: Request,
  options: Readonly<{ origin: string; secret: string; secure: boolean }>,
): string {
  requireSameOrigin(request, options.origin);
  const token = readCookie(request, csrfCookieName(options.secure));
  if (!token || !verifyCsrfToken(token, options.secret)) {
    throw new TransportError("forbidden", "CSRF validation failed", { status: 403 });
  }
  requireCsrfToken(request, token);
  return token;
}

function sameOrigin(actual: string, expected: string): boolean {
  try {
    return new URL(actual).origin === new URL(expected).origin;
  } catch {
    return false;
  }
}

function safeEqual(actual: string, expected: string): boolean {
  const actualBytes = Buffer.from(actual);
  const expectedBytes = Buffer.from(expected);
  return actualBytes.length === expectedBytes.length && timingSafeEqual(actualBytes, expectedBytes);
}
