import { zodResolver } from "@hookform/resolvers/zod";
import { Button, Input } from "@heroui/react";
import { useMutation } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { BrowserRequestError } from "../../../infrastructure/http/browser-error";
import { FormMessage } from "../../../shared/components/feedback/FormMessage";
import { changeCurrentPasswordMutationOptions } from "../api/mutations";
import { changePasswordFormSchema, type ChangePasswordFormValues } from "../contracts";

export function ChangePasswordForm({ onChanged }: Readonly<{ onChanged: () => void }>) {
  const changePasswordMutation = useMutation(changeCurrentPasswordMutationOptions());
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<ChangePasswordFormValues>({
    resolver: zodResolver(changePasswordFormSchema),
    defaultValues: { oldPassword: "", newPassword: "", confirmPassword: "" },
  });

  async function onSubmit(values: ChangePasswordFormValues) {
    try {
      await changePasswordMutation.mutateAsync(values);
      onChanged();
    } catch (error) {
      setError("root", {
        message:
          error instanceof BrowserRequestError ? error.message : "密码暂时无法更新，请稍后重试",
      });
    }
  }

  return (
    <form className="space-y-5" onSubmit={handleSubmit(onSubmit)} noValidate>
      {errors.root?.message ? <FormMessage>{errors.root.message}</FormMessage> : null}
      <PasswordField label="当前密码" id="current-password" error={errors.oldPassword?.message}>
        <Input
          {...register("oldPassword")}
          id="current-password"
          type="password"
          autoComplete="current-password"
        />
      </PasswordField>
      <PasswordField label="新密码" id="new-password" error={errors.newPassword?.message}>
        <Input
          {...register("newPassword")}
          id="new-password"
          type="password"
          autoComplete="new-password"
        />
      </PasswordField>
      <PasswordField
        label="确认新密码"
        id="confirm-password"
        error={errors.confirmPassword?.message}
      >
        <Input
          {...register("confirmPassword")}
          id="confirm-password"
          type="password"
          autoComplete="new-password"
        />
      </PasswordField>

      <div className="flex justify-end border-t border-console-line pt-5">
        <Button
          type="submit"
          variant="primary"
          isDisabled={isSubmitting || changePasswordMutation.isPending}
        >
          {isSubmitting ? "更新中..." : "更新密码"}
        </Button>
      </div>
    </form>
  );
}

function PasswordField({
  label,
  id,
  error,
  children,
}: Readonly<{ label: string; id: string; error?: string; children: React.ReactNode }>) {
  return (
    <div>
      <label className="mb-2 block text-sm font-medium text-console-ink" htmlFor={id}>
        {label}
      </label>
      {children}
      {error ? <p className="mt-1.5 text-xs text-red-700">{error}</p> : null}
    </div>
  );
}
