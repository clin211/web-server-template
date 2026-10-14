import { z } from "zod";

const usernameSchema = z
  .string()
  .trim()
  .min(3, "用户名至少 3 个字符")
  .max(20, "用户名不能超过 20 个字符")
  .regex(/^[A-Za-z0-9_]+$/, "用户名只能包含字母、数字和下划线");

const nicknameSchema = z
  .string()
  .trim()
  .min(1, "请输入昵称")
  .refine((value) => new TextEncoder().encode(value).byteLength < 30, "昵称不能超过 29 字节");

const passwordSchema = z
  .string()
  .min(6, "密码至少 6 个字符")
  .regex(/[A-Za-z]/, "密码至少包含一个字母")
  .regex(/\d/, "密码至少包含一个数字");

export const profileFormSchema = z.object({
  username: usernameSchema,
  nickname: nicknameSchema,
  email: z.email("请输入有效的邮箱地址"),
  phone: z.string().regex(/^1[3-9]\d{9}$/, "请输入有效的中国大陆手机号"),
});

export const profileUpdateRequestSchema = z
  .object({
    username: usernameSchema.optional(),
    nickname: nicknameSchema.optional(),
    email: z.email("请输入有效的邮箱地址").optional(),
    phone: z
      .string()
      .regex(/^1[3-9]\d{9}$/, "请输入有效的中国大陆手机号")
      .optional(),
  })
  .refine((value) => Object.keys(value).length > 0, "没有需要更新的字段");

export const changePasswordFormSchema = z
  .object({
    oldPassword: z.string().min(1, "请输入当前密码"),
    newPassword: passwordSchema,
    confirmPassword: z.string(),
  })
  .refine((value) => value.newPassword === value.confirmPassword, {
    message: "两次输入的新密码不一致",
    path: ["confirmPassword"],
  });

export const changePasswordRequestSchema = changePasswordFormSchema.transform(
  ({ confirmPassword: _confirmPassword, ...values }) => values,
);

export const deleteAccountRequestSchema = z.object({ confirmation: z.string().min(1) });

export type ProfileFormValues = z.infer<typeof profileFormSchema>;
export type ProfileUpdateRequest = z.infer<typeof profileUpdateRequestSchema>;
export type ChangePasswordFormValues = z.input<typeof changePasswordFormSchema>;
export type DeleteAccountFormValues = z.infer<typeof deleteAccountRequestSchema>;
