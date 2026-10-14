import { mutationOptions } from "@tanstack/react-query";
import { registerAccount } from "./client";

export const registrationMutationKeys = {
  all: ["registration"] as const,
  create: () => [...registrationMutationKeys.all, "create"] as const,
};

export const registerAccountMutationOptions = () =>
  mutationOptions({
    mutationKey: registrationMutationKeys.create(),
    mutationFn: registerAccount,
    retry: false,
  });
