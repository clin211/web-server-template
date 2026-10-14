import type { CredentialPolicy } from "../session/port";

const allowedMethods = new Set(["GET", "POST", "PUT", "PATCH", "DELETE"]);

export function assertRelativePath(pathname: string): string {
  if (!pathname.startsWith("/") || pathname.startsWith("//")) {
    throw new Error("Only absolute paths relative to the configured origin are allowed");
  }

  const parsed = new URL(pathname, "http://console.invalid");
  if (parsed.origin !== "http://console.invalid" || !parsed.pathname.startsWith("/")) {
    throw new Error("The upstream path must be relative");
  }

  return `${parsed.pathname}${parsed.search}`;
}

export function assertAllowedMethod(method: string): string {
  const normalized = method.toUpperCase();
  if (!allowedMethods.has(normalized)) {
    throw new Error(`HTTP method ${method} is not allowed`);
  }
  return normalized;
}

export function assertBrowserApiPath(pathname: string): string {
  const normalized = assertRelativePath(pathname);
  if (!normalized.startsWith("/api/")) {
    throw new Error("Browser requests must target a same-origin /api/ route");
  }
  return normalized;
}

export function getCredentialHeader(policy: CredentialPolicy, token: string | null): string | null {
  if (policy.kind === "anonymous") {
    return null;
  }
  if (!token) {
    throw new Error(`Missing ${policy.kind} credential for upstream request`);
  }
  return `Bearer ${token}`;
}
