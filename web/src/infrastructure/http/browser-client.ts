import { assertAllowedMethod, assertBrowserApiPath } from "./request-policy";

export async function browserRequest(
  pathname: string,
  options: Readonly<RequestInit> = {},
): Promise<Response> {
  const headers = new Headers(options.headers);
  headers.delete("authorization");
  headers.delete("cookie");
  headers.delete("host");

  return fetch(assertBrowserApiPath(pathname), {
    ...options,
    method: assertAllowedMethod(options.method ?? "GET"),
    headers,
    credentials: "include",
  });
}
