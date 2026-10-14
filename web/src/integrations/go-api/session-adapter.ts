import "@tanstack/react-start/server-only";
import type { SessionPort, TokenSet } from "../../infrastructure/session/port";
import { appendSetCookie, readCookie, serializeCookie } from "../../infrastructure/session/cookies";

const maximumRefreshTokenLifetimeMs = 8 * 24 * 60 * 60 * 1_000;

type SessionCookieConfig = Readonly<{
  secure: boolean;
  accessName?: string;
  refreshName?: string;
}>;

export class GoSessionAdapter implements SessionPort {
  private readonly accessName: string;
  private readonly refreshName: string;
  private readonly secure: boolean;

  constructor(config: SessionCookieConfig) {
    this.secure = config.secure;
    this.accessName =
      config.accessName ?? (config.secure ? "__Host-console-access" : "console-access");
    this.refreshName =
      config.refreshName ?? (config.secure ? "__Host-console-refresh" : "console-refresh");
  }

  readAccess(request: Request): string | null {
    return readCookie(request, this.accessName);
  }

  readRefresh(request: Request): string | null {
    return readCookie(request, this.refreshName);
  }

  commit(response: Response, tokens: TokenSet): Response {
    appendSetCookie(
      response,
      serializeCookie(this.accessName, tokens.accessToken, {
        httpOnly: true,
        secure: this.secure,
        sameSite: "Lax",
        maxAge: secondsUntil(tokens.accessExpiresAt),
      }),
    );
    appendSetCookie(
      response,
      serializeCookie(this.refreshName, tokens.refreshToken, {
        httpOnly: true,
        secure: this.secure,
        sameSite: "Lax",
        maxAge: secondsUntil(tokens.refreshExpiresAt),
      }),
    );
    return response;
  }

  clear(response: Response): Response {
    appendSetCookie(
      response,
      serializeCookie(this.accessName, "", {
        httpOnly: true,
        secure: this.secure,
        sameSite: "Lax",
        maxAge: 0,
      }),
    );
    appendSetCookie(
      response,
      serializeCookie(this.refreshName, "", {
        httpOnly: true,
        secure: this.secure,
        sameSite: "Lax",
        maxAge: 0,
      }),
    );
    return response;
  }
}

export function refreshExpiryFromToken(token: string): string | null {
  const [, encodedPayload] = token.split(".");
  if (!encodedPayload) return null;

  try {
    const payload = JSON.parse(decodeBase64Url(encodedPayload)) as { exp?: unknown };
    if (typeof payload.exp !== "number" || !Number.isFinite(payload.exp) || payload.exp <= 0) {
      return null;
    }
    const expiresAt = payload.exp * 1_000;
    const now = Date.now();
    if (expiresAt <= now || expiresAt > now + maximumRefreshTokenLifetimeMs) return null;
    return new Date(expiresAt).toISOString();
  } catch {
    return null;
  }
}

function secondsUntil(value: string): number {
  const milliseconds = Date.parse(value) - Date.now();
  return Math.max(0, Math.floor(milliseconds / 1000));
}

function decodeBase64Url(value: string): string {
  const normalized = value.replaceAll("-", "+").replaceAll("_", "/");
  const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, "=");
  return Buffer.from(padded, "base64").toString("utf8");
}
