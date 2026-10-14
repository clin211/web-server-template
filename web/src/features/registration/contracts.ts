import { z } from "zod";

export const usernameSchema = z
  .string()
  .trim()
  .min(3, "用户名至少 3 个字符")
  .max(20, "用户名不能超过 20 个字符")
  .regex(/^[A-Za-z0-9_]+$/, "用户名只能包含字母、数字和下划线");

export const passwordSchema = z
  .string()
  .min(6, "密码至少 6 个字符")
  .regex(/[A-Za-z]/, "密码至少包含一个字母")
  .regex(/\d/, "密码至少包含一个数字");

export const nicknameSchema = z
  .string()
  .trim()
  .min(1, "请输入昵称")
  .refine((value) => new TextEncoder().encode(value).byteLength < 30, "昵称不能超过 29 字节");

export const registrationFormSchema = z
  .object({
    username: usernameSchema,
    password: passwordSchema,
    confirmPassword: z.string(),
    nickname: nicknameSchema,
    email: z.email("请输入有效的邮箱地址"),
    phone: z.string().regex(/^1[3-9]\d{9}$/, "请输入有效的中国大陆手机号"),
  })
  .refine((value) => value.password === value.confirmPassword, {
    message: "两次输入的密码不一致",
    path: ["confirmPassword"],
  });

export const registrationRequestSchema = registrationFormSchema.transform(
  ({ confirmPassword: _confirmPassword, ...values }) => values,
);

export const registrationResponseSchema = z.object({ userID: z.string().min(1) });

export type RegistrationFormValues = z.input<typeof registrationFormSchema>;
