import { z } from "zod";

export const loginFormSchema = z.object({
  username: z.string().trim().min(1, "请输入用户名").max(20, "用户名不能超过 20 个字符"),
  password: z.string().min(1, "请输入密码"),
});

export const loginResponseDataSchema = z.object({
  accessToken: z.string().min(1),
  refreshToken: z.string().min(1),
  expireAt: z.string().datetime({ offset: true }),
});

const int64StringSchema = z.union([z.string(), z.number().int()]).transform(String);

export const userSchema = z.object({
  userID: z.string().min(1),
  username: z.string(),
  nickname: z.string(),
  email: z.string(),
  phone: z.string(),
  postCount: int64StringSchema,
  createdAt: int64StringSchema,
  updatedAt: int64StringSchema,
  status: z.coerce.number().int(),
  gender: z.coerce.number().int(),
  avatar: z.string(),
  description: z.string(),
  lastLoginAt: int64StringSchema,
});

export const sessionDataSchema = z.object({ user: userSchema });
export const logoutResponseDataSchema = z.object({});

export type LoginFormValues = z.infer<typeof loginFormSchema>;
export type LoginResponseData = z.infer<typeof loginResponseDataSchema>;
export type User = z.infer<typeof userSchema>;
export type SessionData = z.infer<typeof sessionDataSchema>;
