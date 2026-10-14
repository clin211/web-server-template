import { describe, expect, it } from "vitest";
import { getRouter } from "./router";

describe("router factory", () => {
  it("isolates router state for every runtime instance", () => {
    const first = getRouter();
    const second = getRouter();

    expect(first).not.toBe(second);
    expect(first.options.context.queryClient).not.toBe(second.options.context.queryClient);
    expect(first.options.context.sessionCoordinator).not.toBe(
      second.options.context.sessionCoordinator,
    );
  });
});
