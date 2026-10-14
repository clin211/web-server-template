import { describe, expect, it } from "vitest";
import {
  changePasswordFormSchema,
  changePasswordRequestSchema,
  profileUpdateRequestSchema,
} from "./contracts";

describe("account contracts", () => {
  it("requires a non-empty minimal profile update", () => {
    expect(profileUpdateRequestSchema.safeParse({}).success).toBe(false);
    expect(profileUpdateRequestSchema.safeParse({ nickname: "新昵称" }).success).toBe(true);
  });

  it.each([
    [{ oldPassword: "old", newPassword: "secret1", confirmPassword: "secret1" }, true],
    [{ oldPassword: "old", newPassword: "secret1", confirmPassword: "secret2" }, false],
    [{ oldPassword: "", newPassword: "secret1", confirmPassword: "secret1" }, false],
  ])("validates password change %#", (value, valid) => {
    expect(changePasswordFormSchema.safeParse(value).success).toBe(valid);
  });

  it("strips password confirmation from the server payload", () => {
    expect(
      changePasswordRequestSchema.parse({
        oldPassword: "old-password",
        newPassword: "new-password1",
        confirmPassword: "new-password1",
      }),
    ).toEqual({ oldPassword: "old-password", newPassword: "new-password1" });
  });
});
