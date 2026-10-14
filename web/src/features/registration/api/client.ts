import { toBrowserRequestError } from "../../../infrastructure/http/browser-error";
import { browserRequest } from "../../../infrastructure/http/browser-client";
import { issueBrowserCsrf } from "../../../infrastructure/session/browser-csrf";
import type { RegistrationFormValues } from "../contracts";

export async function registerAccount(values: RegistrationFormValues): Promise<void> {
  const csrf = await issueBrowserCsrf();
  const response = await browserRequest("/api/register", {
    method: "POST",
    headers: { "content-type": "application/json", "x-csrf-token": csrf },
    body: JSON.stringify(values),
  });
  if (!response.ok) throw await toBrowserRequestError(response);
}
