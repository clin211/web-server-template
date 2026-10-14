import { describe, expect, it } from "vitest";
import { registerAccountMutationOptions } from "./mutations";

describe("registration mutation options", () => {
  it("owns registration mutation identity and retry policy", () => {
    const options = registerAccountMutationOptions();
    expect(options.mutationKey).toEqual(["registration", "create"]);
    expect(options.retry).toBe(false);
    expect(options.mutationFn).toBeTypeOf("function");
  });
});
