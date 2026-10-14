import { describe, expect, it } from "vitest";
import {
  changeCurrentPasswordMutationOptions,
  deleteCurrentAccountMutationOptions,
  updateCurrentAccountMutationOptions,
} from "./mutations";

describe("account mutation options", () => {
  it.each([
    ["update", updateCurrentAccountMutationOptions],
    ["change-password", changeCurrentPasswordMutationOptions],
    ["delete", deleteCurrentAccountMutationOptions],
  ] as const)("owns the %s mutation policy", (operation, createOptions) => {
    const options = createOptions();
    expect(options.mutationKey).toEqual(["account", operation]);
    expect(options.retry).toBe(false);
    expect(options.mutationFn).toBeTypeOf("function");
  });
});
