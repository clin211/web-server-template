import { useState } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, Input } from "@heroui/react";
import { useMutation } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { BrowserRequestError } from "../../../infrastructure/http/browser-error";
import { FormMessage } from "../../../shared/components/feedback/FormMessage";
import { updateCurrentAccountMutationOptions } from "../api/mutations";
import { profileFormSchema, type ProfileFormValues, type ProfileUpdateRequest } from "../contracts";

type ProfileFormProps = Readonly<{
  profile: ProfileFormValues;
  onUpdated: () => Promise<void>;
}>;

export function ProfileForm({ profile, onUpdated }: ProfileFormProps) {
  const [saved, setSaved] = useState(false);
  const updateMutation = useMutation(updateCurrentAccountMutationOptions());
  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { dirtyFields, errors, isDirty, isSubmitting },
  } = useForm<ProfileFormValues>({
    resolver: zodResolver(profileFormSchema),
    defaultValues: profile,
  });

  async function onSubmit(values: ProfileFormValues) {
    setSaved(false);
    const payload: ProfileUpdateRequest = {};
    for (const key of Object.keys(dirtyFields) as Array<keyof ProfileFormValues>) {
      if (dirtyFields[key]) payload[key] = values[key];
    }

    try {
      await updateMutation.mutateAsync(payload);
      await onUpdated();
      reset(values);
      setSaved(true);
    } catch (error) {
      setError("root", {
        message:
          error instanceof BrowserRequestError ? error.message : "账号资料暂时无法保存，请稍后重试",
      });
    }
  }

  return (
    <form className="space-y-5" onSubmit={handleSubmit(onSubmit)} noValidate>
      {errors.root?.message ? <FormMessage>{errors.root.message}</FormMessage> : null}
      {saved && !errors.root ? <FormMessage tone="success">账号资料已更新。</FormMessage> : null}

      <div className="grid gap-5 sm:grid-cols-2">
        <Field label="用户名" id="profile-username" error={errors.username?.message}>
          <Input
            {...register("username")}
            id="profile-username"
            autoComplete="username"
            aria-invalid={errors.username ? "true" : undefined}
          />
        </Field>
        <Field label="昵称" id="profile-nickname" error={errors.nickname?.message}>
          <Input
            {...register("nickname")}
            id="profile-nickname"
            autoComplete="nickname"
            aria-invalid={errors.nickname ? "true" : undefined}
          />
        </Field>
        <Field label="邮箱" id="profile-email" error={errors.email?.message}>
          <Input
            {...register("email")}
            id="profile-email"
            type="email"
            autoComplete="email"
            aria-invalid={errors.email ? "true" : undefined}
          />
        </Field>
        <Field label="手机号" id="profile-phone" error={errors.phone?.message}>
          <Input
            {...register("phone")}
            id="profile-phone"
            autoComplete="tel"
            inputMode="tel"
            aria-invalid={errors.phone ? "true" : undefined}
          />
        </Field>
      </div>

      <div className="flex justify-end border-t border-console-line pt-5">
        <Button
          type="submit"
          variant="primary"
          isDisabled={isSubmitting || updateMutation.isPending || !isDirty}
        >
          {isSubmitting ? "保存中..." : "保存资料"}
        </Button>
      </div>
    </form>
  );
}

function Field({
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
      {error ? (
        <p className="mt-1.5 text-xs text-red-700" id={`${id}-error`}>
          {error}
        </p>
      ) : null}
    </div>
  );
}
