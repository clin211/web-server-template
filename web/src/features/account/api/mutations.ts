import { mutationOptions } from "@tanstack/react-query";
import { changeCurrentPassword, deleteCurrentAccount, updateCurrentAccount } from "./client";

export const accountMutationKeys = {
  all: ["account"] as const,
  update: () => [...accountMutationKeys.all, "update"] as const,
  changePassword: () => [...accountMutationKeys.all, "change-password"] as const,
  delete: () => [...accountMutationKeys.all, "delete"] as const,
};

export const updateCurrentAccountMutationOptions = () =>
  mutationOptions({
    mutationKey: accountMutationKeys.update(),
    mutationFn: updateCurrentAccount,
    retry: false,
  });

export const changeCurrentPasswordMutationOptions = () =>
  mutationOptions({
    mutationKey: accountMutationKeys.changePassword(),
    mutationFn: changeCurrentPassword,
    retry: false,
  });

export const deleteCurrentAccountMutationOptions = () =>
  mutationOptions({
    mutationKey: accountMutationKeys.delete(),
    mutationFn: deleteCurrentAccount,
    retry: false,
  });
