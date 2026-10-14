import { zodResolver } from "@hookform/resolvers/zod";
import { Button, Input } from "@heroui/react";
import { useMutation } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { BrowserRequestError } from "../../../infrastructure/http/browser-error";
import { FormMessage } from "../../../shared/components/feedback/FormMessage";
import { deleteCurrentAccountMutationOptions } from "../api/mutations";
import { deleteAccountRequestSchema, type DeleteAccountFormValues } from "../contracts";

type DeleteAccountFormProps = Readonly<{
  username: string;
  onDeleted: () => void;
}>;

export function DeleteAccountForm({ username, onDeleted }: DeleteAccountFormProps) {
  const deleteMutation = useMutation(deleteCurrentAccountMutationOptions());
  const {
    register,
    handleSubmit,
    watch,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<DeleteAccountFormValues>({
    resolver: zodResolver(deleteAccountRequestSchema),
    defaultValues: { confirmation: "" },
  });
  const confirmation = watch("confirmation");

  async function onSubmit(values: DeleteAccountFormValues) {
    try {
      await deleteMutation.mutateAsync(values.confirmation);
      onDeleted();
    } catch (error) {
      setError("root", {
        message:
          error instanceof BrowserRequestError ? error.message : "账号暂时无法注销，请稍后重试",
      });
    }
  }

  return (
    <form className="space-y-5" onSubmit={handleSubmit(onSubmit)} noValidate>
      {errors.root?.message ? <FormMessage>{errors.root.message}</FormMessage> : null}
      <FormMessage tone="info">注销确认用户名：{username}</FormMessage>
      <div>
        <label className="mb-2 block text-sm font-medium text-console-ink" htmlFor="confirmation">
          用户名确认
        </label>
        <Input
          {...register("confirmation")}
          id="confirmation"
          autoComplete="off"
          aria-invalid={errors.confirmation ? "true" : undefined}
        />
        {errors.confirmation?.message ? (
          <p className="mt-1.5 text-xs text-red-700">{errors.confirmation.message}</p>
        ) : null}
      </div>

      <div className="flex justify-end border-t border-red-200 pt-5">
        <Button
          type="submit"
          variant="danger"
          isDisabled={isSubmitting || deleteMutation.isPending || confirmation !== username}
        >
          {isSubmitting ? "注销中..." : "永久注销账号"}
        </Button>
      </div>
    </form>
  );
}
