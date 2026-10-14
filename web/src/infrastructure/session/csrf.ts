import "@tanstack/react-start/server-only";
import { createHmac, randomBytes, timingSafeEqual } from "node:crypto";
import { appendSetCookie, serializeCookie } from "./cookies";

export function csrfCookieName(secure: boolean): string {
  return secure ? "__Host-console-csrf" : "console-csrf";
}

export function issueCsrfToken(secret: string): string {
  const nonce = randomBytes(32).toString("base64url");
  return `${nonce}.${signature(secret, nonce)}`;
}

export function expireCsrfToken(response: Response, secure: boolean): Response {
  return appendSetCookie(
    response,
    serializeCookie(csrfCookieName(secure), "", {
      secure,
      sameSite: "Lax",
      maxAge: 0,
    }),
  );
}

export function verifyCsrfToken(token: string, secret: string): boolean {
  const parts = token.split(".");
  if (parts.length !== 2) return false;
  const [nonce, provided] = parts;
  if (!nonce || !provided) return false;

  const expected = signature(secret, nonce);
  const providedBytes = Buffer.from(provided);
  const expectedBytes = Buffer.from(expected);
  return (
    providedBytes.length === expectedBytes.length && timingSafeEqual(providedBytes, expectedBytes)
  );
}

function signature(secret: string, nonce: string): string {
  return createHmac("sha256", secret).update(nonce).digest("base64url");
}
