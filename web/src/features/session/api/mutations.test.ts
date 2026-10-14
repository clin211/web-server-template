import { describe, expect, it } from "vitest";
import { loginSessionMutationOptions, logoutSessionMutationOptions } from "./mutations";

describe("session mutation options", () => {
  it.each([
    ["login", loginSessionMutationOptions],
    ["logout", logoutSessionMutationOptions],
  ] as const)("owns the %s mutation policy", (operation, createOptions) => {
    const options = createOptions();
    expect(options.mutationKey).toEqual(["session", operation]);
    expect(options.retry).toBe(false);
    expect(options.mutationFn).toBeTypeOf("function");
  });
});
