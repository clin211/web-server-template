import { describe, expect, it } from "vitest";
import {
  assertAllowedMethod,
  assertBrowserApiPath,
  assertRelativePath,
  getCredentialHeader,
} from "./request-policy";

describe("request policy", () => {
  it.each([
    ["GET", "GET"],
    ["post", "POST"],
    ["Delete", "DELETE"],
  ])("normalizes allowed method %s", (input, expected) => {
    expect(assertAllowedMethod(input)).toBe(expected);
  });

  it.each(["CONNECT", "TRACE", "OPTIONS"])("rejects method %s", (method) => {
    expect(() => assertAllowedMethod(method)).toThrow("not allowed");
  });

  it.each(["/v1/users", "/v1/users?page=2"])("accepts relative path %s", (pathname) => {
    expect(assertRelativePath(pathname)).toBe(pathname);
  });

  it.each([
    "https://evil.example/api/users",
    "//evil.example/api/users",
    "/\\\\evil.example/api/users",
    "api/users",
  ])("rejects unsafe path %s", (pathname) => {
    expect(() => assertRelativePath(pathname)).toThrow();
  });

  it("requires browser paths to stay under /api/", () => {
    expect(assertBrowserApiPath("/api/health")).toBe("/api/health");
    expect(() => assertBrowserApiPath("/v1/users")).toThrow("/api/");
  });

  it("only creates authorization for credentialed policies", () => {
    expect(getCredentialHeader({ kind: "anonymous" }, null)).toBeNull();
    expect(getCredentialHeader({ kind: "access" }, "access-token")).toBe("Bearer access-token");
    expect(() => getCredentialHeader({ kind: "refresh" }, null)).toThrow("refresh");
  });
});
