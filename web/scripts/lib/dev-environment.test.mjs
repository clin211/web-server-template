import { describe, expect, it } from "vitest";
import { resolveDevEnvironment, resolveStartPort, withResolvedPort } from "./dev-environment.mjs";

describe("development environment", () => {
  it("provides local-only defaults for the selected port", () => {
    expect(resolveDevEnvironment({ source: {}, fileEnv: {}, port: 5174 })).toMatchObject({
      BACKEND_API_URL: "http://127.0.0.1:5558",
      CONSOLE_ORIGIN: "http://127.0.0.1:5174",
      CONSOLE_REQUEST_TIMEOUT_MS: "10000",
    });
    expect(
      resolveDevEnvironment({ source: {}, fileEnv: {}, port: 5174 }).CONSOLE_SESSION_SECRET,
    ).toHaveLength(48);
  });

  it("prefers process values over env files and defaults", () => {
    expect(
      resolveDevEnvironment({
        source: { BACKEND_API_URL: "http://shell.test:8080" },
        fileEnv: {
          BACKEND_API_URL: "http://file.test:8080",
          CONSOLE_ORIGIN: "http://file-console.test:5173",
        },
        port: 5173,
      }),
    ).toMatchObject({
      BACKEND_API_URL: "http://shell.test:8080",
      CONSOLE_ORIGIN: "http://file-console.test:5173",
    });
  });

  it.each([
    ["equals syntax", ["--host", "0.0.0.0", "--port=5173"]],
    ["separate value", ["--host", "0.0.0.0", "--port", "5173"]],
  ])("replaces an existing Vite port argument using %s", (_name, args) => {
    expect(withResolvedPort(args, 5180)).toEqual([
      "--host",
      "0.0.0.0",
      "--port",
      "5180",
      "--strictPort",
    ]);
  });

  it.each([
    ["explicit value", ["--port", "6000"], 6000],
    ["equals syntax", ["--port=6001"], 6001],
    ["default", [], 5173],
  ])("uses the %s as the development start port", (_label, args, expected) => {
    expect(resolveStartPort(args)).toBe(expected);
  });
});
