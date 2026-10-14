import { describe, expect, it } from "vitest";
import {
  nicknameSchema,
  passwordSchema,
  registrationFormSchema,
  registrationRequestSchema,
  usernameSchema,
} from "./contracts";

describe("registration contracts", () => {
  it.each([
    ["ab", false],
    ["valid_user_20", true],
    ["invalid-user", false],
    ["a".repeat(21), false],
  ])("validates username %s", (value, valid) => {
    expect(usernameSchema.safeParse(value).success).toBe(valid);
  });

  it.each([
    ["abc123", true],
    ["abcdef", false],
    ["123456", false],
    ["a1", false],
  ])("validates password boundary", (value, valid) => {
    expect(passwordSchema.safeParse(value).success).toBe(valid);
  });

  it("counts nickname length by UTF-8 bytes", () => {
    expect(nicknameSchema.safeParse("你".repeat(9)).success).toBe(true);
    expect(nicknameSchema.safeParse("你".repeat(10)).success).toBe(false);
  });

  it("validates confirmation, email, and phone", () => {
    expect(
      registrationFormSchema.safeParse({
        username: "valid_user",
        password: "secret1",
        confirmPassword: "different1",
        nickname: "Ada",
        email: "invalid",
        phone: "123",
      }).success,
    ).toBe(false);
  });

  it("strips the confirmation field from the server payload", () => {
    expect(
      registrationRequestSchema.parse({
        username: "valid_user",
        password: "secret1",
        confirmPassword: "secret1",
        nickname: "Ada",
        email: "ada@example.test",
        phone: "13800138000",
      }),
    ).not.toHaveProperty("confirmPassword");
  });
});
