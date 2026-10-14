import { toBrowserRequestError } from "../../../infrastructure/http/browser-error";
import { browserRequest } from "../../../infrastructure/http/browser-client";
import { issueBrowserCsrf } from "../../../infrastructure/session/browser-csrf";
import type { ProfileUpdateRequest } from "../contracts";

export async function updateCurrentAccount(values: ProfileUpdateRequest): Promise<void> {
  await accountWrite("/api/account", "PUT", values);
}

export async function changeCurrentPassword(
  values: Readonly<{
    oldPassword: string;
    newPassword: string;
    confirmPassword: string;
  }>,
): Promise<void> {
  await accountWrite("/api/account/password", "PUT", values);
}

export async function deleteCurrentAccount(confirmation: string): Promise<void> {
  await accountWrite("/api/account", "DELETE", { confirmation });
}

async function accountWrite(path: string, method: "PUT" | "DELETE", body: unknown): Promise<void> {
  const csrf = await issueBrowserCsrf();
  const response = await browserRequest(path, {
    method,
    headers: { "content-type": "application/json", "x-csrf-token": csrf },
    body: JSON.stringify(body),
  });
  if (!response.ok) throw await toBrowserRequestError(response);
}
