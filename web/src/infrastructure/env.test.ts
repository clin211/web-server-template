import { describe, expect, it, vi } from "vitest";
import { getServerEnv } from "./env";

const validEnv = {
  BACKEND_API_URL: "http://api.console.test",
  CONSOLE_ORIGIN: "https://console.test",
  CONSOLE_SESSION_SECRET: "a-secret-with-at-least-32-characters-long",
  CONSOLE_REQUEST_TIMEOUT_MS: "2500",
};

describe("server environment", () => {
  it("parses and defaults validated server settings", () => {
    vi.stubGlobal("window", undefined);
    try {
      expect(getServerEnv({ ...validEnv, CONSOLE_REQUEST_TIMEOUT_MS: undefined })).toMatchObject({
        BACKEND_API_URL: "http://api.console.test",
        CONSOLE_ORIGIN: "https://console.test",
        CONSOLE_REQUEST_TIMEOUT_MS: 10_000,
      });
    } finally {
      vi.unstubAllGlobals();
    }
  });

  it.each([
    ["missing backend URL", { ...validEnv, BACKEND_API_URL: undefined }],
    ["short session secret", { ...validEnv, CONSOLE_SESSION_SECRET: "short" }],
    ["unsupported URL protocol", { ...validEnv, CONSOLE_ORIGIN: "ftp://console.test" }],
    ["invalid timeout", { ...validEnv, CONSOLE_REQUEST_TIMEOUT_MS: "0" }],
  ])("fails closed for %s", (_label, source) => {
    vi.stubGlobal("window", undefined);
    try {
      expect(() => getServerEnv(source)).toThrow();
    } finally {
      vi.unstubAllGlobals();
    }
  });
});
