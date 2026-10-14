import { useEffect, useState } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, Input } from "@heroui/react";
import { Eye, EyeOff } from "lucide-react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { useForm } from "react-hook-form";
import { BrowserRequestError } from "../../../infrastructure/http/browser-error";
import { issueBrowserCsrf } from "../api/client";
import { currentSessionQuery } from "../api/queries";
import { loginSessionMutationOptions } from "../api/mutations";
import { loginFormSchema, type LoginFormValues } from "../contracts";

type LoginFormProps = Readonly<{
  returnTo: string;
  registered: boolean;
  reason?: string;
}>;

export function LoginForm({ returnTo, registered, reason }: LoginFormProps) {
  const queryClient = useQueryClient();
  const loginMutation = useMutation(loginSessionMutationOptions());
  const [csrfToken, setCsrfToken] = useState<string | null>(null);
  const [csrfError, setCsrfError] = useState<string | null>(null);
  const [formMessage, setFormMessage] = useState<string | null>(null);
  const [showPassword, setShowPassword] = useState(false);
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<LoginFormValues>({
    resolver: zodResolver(loginFormSchema),
    defaultValues: { username: "", password: "" },
  });

  useEffect(() => {
    let cancelled = false;
    void issueBrowserCsrf()
      .then((token) => {
        if (!cancelled) setCsrfToken(token);
      })
      .catch(() => {
        if (!cancelled) setCsrfError("安全校验暂不可用，请刷新页面重试");
      });

    return () => {
      cancelled = true;
    };
  }, []);

  async function onSubmit(values: LoginFormValues) {
    setFormMessage(null);
    if (!csrfToken) {
      setFormMessage(csrfError ?? "正在准备安全校验，请稍后重试");
      return;
    }

    try {
      await loginMutation.mutateAsync({ values, csrfToken });
      await queryClient.prefetchQuery(currentSessionQuery());
      window.location.assign(returnTo);
    } catch (error) {
      if (error instanceof BrowserRequestError && error.code === "invalid-credentials") {
        setFormMessage("账号或密码错误，请重新输入");
      } else if (error instanceof BrowserRequestError && error.code === "csrf-invalid") {
        setFormMessage("安全校验已失效，请刷新页面重试");
      } else {
        setFormMessage("登录服务暂时不可用，请稍后重试");
      }
    }
  }

  return (
    <form className="mt-8 space-y-5" onSubmit={handleSubmit(onSubmit)} noValidate>
      {registered ? (
        <div
          className="border border-console-success/30 bg-console-success/10 px-4 py-3 text-sm text-console-success"
          role="status"
        >
          注册成功，请使用新账号登录。
        </div>
      ) : null}
      {reason === "session-expired" ? (
        <div
          className="border border-console-accent/30 bg-console-accent-soft px-4 py-3 text-sm text-console-accent"
          role="status"
        >
          登录状态已过期，请重新验证身份。
        </div>
      ) : null}
      {reason === "password-changed" ? (
        <div
          className="border border-console-accent/30 bg-console-accent-soft px-4 py-3 text-sm text-console-accent"
          role="status"
        >
          密码已更新，请使用新密码登录。
        </div>
      ) : null}
      {formMessage ? (
        <div
          className="border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700"
          role="alert"
        >
          {formMessage}
        </div>
      ) : null}

      <div>
        <label className="block text-sm font-medium text-console-ink" htmlFor="username">
          账号
        </label>
        <Input
          {...register("username")}
          id="username"
          className="mt-2 w-full"
          placeholder="请输入用户名"
          autoComplete="username"
          aria-invalid={errors.username ? "true" : undefined}
          aria-describedby={errors.username ? "username-error" : undefined}
        />
        {errors.username ? (
          <p id="username-error" className="mt-1.5 text-xs text-red-700">
            {errors.username.message}
          </p>
        ) : null}
      </div>

      <div>
        <label className="block text-sm font-medium text-console-ink" htmlFor="password">
          密码
        </label>
        <div className="relative mt-2">
          <Input
            {...register("password")}
            id="password"
            className="w-full pr-11"
            type={showPassword ? "text" : "password"}
            placeholder="请输入密码"
            autoComplete="current-password"
            aria-invalid={errors.password ? "true" : undefined}
            aria-describedby={errors.password ? "password-error" : undefined}
          />
          <button
            type="button"
            className="absolute right-2 top-1/2 grid size-8 -translate-y-1/2 place-items-center text-console-muted hover:text-console-ink"
            onClick={() => setShowPassword((visible) => !visible)}
            aria-label={showPassword ? "隐藏输入内容" : "显示输入内容"}
            title={showPassword ? "隐藏输入内容" : "显示输入内容"}
          >
            {showPassword ? (
              <EyeOff aria-hidden="true" size={17} />
            ) : (
              <Eye aria-hidden="true" size={17} />
            )}
          </button>
        </div>
        {errors.password ? (
          <p id="password-error" className="mt-1.5 text-xs text-red-700">
            {errors.password.message}
          </p>
        ) : null}
      </div>

      <Button
        type="submit"
        variant="primary"
        size="lg"
        className="w-full"
        isDisabled={isSubmitting || loginMutation.isPending}
      >
        {isSubmitting ? "登录中..." : "登录"}
      </Button>

      <p className="text-center text-sm text-console-muted">
        还没有账号？
        <Link
          to="/register"
          className="ml-1 font-medium text-console-accent no-underline hover:underline"
        >
          创建账号
        </Link>
      </p>
    </form>
  );
}
