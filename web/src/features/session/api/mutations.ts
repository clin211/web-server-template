import { mutationOptions } from "@tanstack/react-query";
import { loginBrowserSession, logoutBrowserSession } from "./client";

export const sessionMutationKeys = {
  login: () => ["session", "login"] as const,
  logout: () => ["session", "logout"] as const,
};

export const loginSessionMutationOptions = () =>
  mutationOptions({
    mutationKey: sessionMutationKeys.login(),
    mutationFn: loginBrowserSession,
    retry: false,
  });

export const logoutSessionMutationOptions = () =>
  mutationOptions({
    mutationKey: sessionMutationKeys.logout(),
    mutationFn: logoutBrowserSession,
    retry: false,
  });
