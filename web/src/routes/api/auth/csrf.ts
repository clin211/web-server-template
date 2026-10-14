import { createFileRoute } from "@tanstack/react-router";
import { getServerEnv } from "../../../infrastructure/env";
import { readCookie, serializeCookie } from "../../../infrastructure/session/cookies";
import {
  csrfCookieName,
  issueCsrfToken,
  verifyCsrfToken,
} from "../../../infrastructure/session/csrf";

export const Route = createFileRoute("/api/auth/csrf")({
  server: {
    handlers: {
      GET: async ({ request }) => {
        const env = getServerEnv();
        const secure = new URL(env.CONSOLE_ORIGIN).protocol === "https:";
        const cookieName = csrfCookieName(secure);
        const current = readCookie(request, cookieName);
        const token =
          current && verifyCsrfToken(current, env.CONSOLE_SESSION_SECRET)
            ? current
            : issueCsrfToken(env.CONSOLE_SESSION_SECRET);
        const response = Response.json({ token }, { headers: { "cache-control": "no-store" } });
        response.headers.append(
          "set-cookie",
          serializeCookie(cookieName, token, {
            secure,
            sameSite: "Lax",
            maxAge: 60 * 60,
          }),
        );
        return response;
      },
    },
  },
});
