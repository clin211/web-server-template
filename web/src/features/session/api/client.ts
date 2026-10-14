import { browserRequest } from "../../../infrastructure/http/browser-client";
import {
  BrowserRequestError,
  toBrowserRequestError,
} from "../../../infrastructure/http/browser-error";
import { issueBrowserCsrf } from "../../../infrastructure/session/browser-csrf";
import { sessionDataSchema, type LoginFormValues, type SessionData } from "../contracts";

export { BrowserRequestError as BrowserApiError };
export { issueBrowserCsrf };

export async function fetchSession(): Promise<SessionData> {
  const response = await browserRequest("/api/auth/session");
  if (!response.ok) throw await toBrowserRequestError(response);
  return sessionDataSchema.parse(await response.json());
}

export async function loginBrowserSession(
  input: Readonly<{ values: LoginFormValues; csrfToken: string }>,
): Promise<void> {
  const response = await browserRequest("/api/auth/login", {
    method: "POST",
    headers: {
      "content-type": "application/json",
      "x-csrf-token": input.csrfToken,
    },
    body: JSON.stringify(input.values),
  });
  if (!response.ok) throw await toBrowserRequestError(response);
}

export async function refreshBrowserSession(): Promise<void> {
  const csrf = await issueBrowserCsrf();
  const response = await browserRequest("/api/auth/refresh", {
    method: "POST",
    headers: { "x-csrf-token": csrf },
  });
  if (!response.ok) throw await toBrowserRequestError(response);
}

export async function logoutBrowserSession(): Promise<{ revocationConfirmed: boolean }> {
  const csrf = await issueBrowserCsrf();
  const response = await browserRequest("/api/auth/logout", {
    method: "POST",
    headers: { "x-csrf-token": csrf },
  });
  if (!response.ok) throw await toBrowserRequestError(response);
  const body = (await response.json()) as { revocationConfirmed?: unknown };
  return { revocationConfirmed: body.revocationConfirmed === true };
}
