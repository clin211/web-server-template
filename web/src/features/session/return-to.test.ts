import { describe, expect, it } from "vitest";
import { safeReturnTo } from "./return-to";

describe("login return target", () => {
  it.each([
    ["/", "/"],
    ["/settings/profile?tab=basic#contact", "/settings/profile?tab=basic#contact"],
  ])("accepts internal path %s", (value, expected) => {
    expect(safeReturnTo(value)).toBe(expected);
  });

  it.each([
    undefined,
    "https://evil.test",
    "//evil.test/account",
    "/\\\\evil.test/account",
    "javascript:alert(1)",
    "settings/profile",
  ])("rejects external or malformed target %s", (value) => {
    expect(safeReturnTo(value)).toBe("/");
  });
});
