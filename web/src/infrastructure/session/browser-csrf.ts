import { BrowserRequestError, toBrowserRequestError } from "../http/browser-error";
import { browserRequest } from "../http/browser-client";

export async function issueBrowserCsrf(): Promise<string> {
  const response = await browserRequest("/api/auth/csrf");
  if (!response.ok) throw await toBrowserRequestError(response);
  const body = (await response.json()) as { token?: unknown };
  if (typeof body.token !== "string" || body.token.length === 0) {
    throw new BrowserRequestError("invalid-response", 502, "安全校验响应无效");
  }
  return body.token;
}
