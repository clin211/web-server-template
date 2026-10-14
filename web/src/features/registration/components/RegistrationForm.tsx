import { zodResolver } from "@hookform/resolvers/zod";
import { Button, Input } from "@heroui/react";
import { useMutation } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { useForm } from "react-hook-form";
import { BrowserRequestError } from "../../../infrastructure/http/browser-error";
import { registerAccountMutationOptions } from "../api/mutations";
import { registrationFormSchema, type RegistrationFormValues } from "../contracts";

export function RegistrationForm() {
  const registrationMutation = useMutation(registerAccountMutationOptions());
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<RegistrationFormValues>({
    resolver: zodResolver(registrationFormSchema),
    defaultValues: {
      username: "",
      password: "",
      confirmPassword: "",
      nickname: "",
      email: "",
      phone: "",
    },
  });

  async function onSubmit(values: RegistrationFormValues) {
    try {
      await registrationMutation.mutateAsync(values);
      window.location.assign("/login?registered=1");
    } catch (error) {
      const message =
        error instanceof BrowserRequestError && error.code === "already-exists"
          ? "用户名、邮箱或手机号已被使用"
          : error instanceof BrowserRequestError
            ? error.message
            : "注册服务暂时不可用，请稍后重试";
      setError("root", { message });
    }
  }

  return (
    <form className="mt-8 space-y-4" onSubmit={handleSubmit(onSubmit)} noValidate>
      {errors.root?.message ? (
        <div
          className="border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700"
          role="alert"
        >
          {errors.root.message}
        </div>
      ) : null}

      <Field label="用户名" error={errors.username?.message}>
        <Input {...register("username")} id="register-username" autoComplete="username" />
      </Field>
      <div className="grid gap-4 sm:grid-cols-2">
        <Field label="昵称" error={errors.nickname?.message}>
          <Input {...register("nickname")} id="register-nickname" autoComplete="nickname" />
        </Field>
        <Field label="手机号" error={errors.phone?.message}>
          <Input {...register("phone")} id="register-phone" autoComplete="tel" inputMode="tel" />
        </Field>
      </div>
      <Field label="邮箱" error={errors.email?.message}>
        <Input {...register("email")} id="register-email" type="email" autoComplete="email" />
      </Field>
      <div className="grid gap-4 sm:grid-cols-2">
        <Field label="密码" error={errors.password?.message}>
          <Input
            {...register("password")}
            id="register-password"
            type="password"
            autoComplete="new-password"
          />
        </Field>
        <Field label="确认密码" error={errors.confirmPassword?.message}>
          <Input
            {...register("confirmPassword")}
            id="register-confirm-password"
            type="password"
            autoComplete="new-password"
          />
        </Field>
      </div>

      <Button
        type="submit"
        variant="primary"
        size="lg"
        className="w-full"
        isDisabled={isSubmitting || registrationMutation.isPending}
      >
        {isSubmitting ? "提交中..." : "创建账号"}
      </Button>
      <p className="text-center text-sm text-console-muted">
        已有账号？
        <Link
          to="/login"
          search={{}}
          className="ml-1 font-medium text-console-accent no-underline hover:underline"
        >
          返回登录
        </Link>
      </p>
    </form>
  );
}

function Field({
  label,
  error,
  children,
}: Readonly<{ label: string; error?: string; children: React.ReactElement }>) {
  const input = children.props as { id?: string };
  return (
    <div>
      <label className="mb-2 block text-sm font-medium text-console-ink" htmlFor={input.id}>
        {label}
      </label>
      {children}
      {error ? <p className="mt-1.5 text-xs text-red-700">{error}</p> : null}
    </div>
  );
}
