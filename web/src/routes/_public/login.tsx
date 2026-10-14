import { createFileRoute, redirect } from "@tanstack/react-router";
import { ShieldCheck } from "lucide-react";
import {
  BrowserApiError,
  LoginForm,
  currentSessionQuery,
  safeReturnTo,
} from "../../features/session";
import { AuthLayout } from "../../shared/components/layout/AuthLayout";

const loginSearchSchema = {
  parse(value: Record<string, unknown>): {
    returnTo?: string;
    registered?: boolean;
    reason?: string;
  } {
    const search: { returnTo?: string; registered?: boolean; reason?: string } = {};
    if (value.returnTo !== undefined) search.returnTo = safeReturnTo(value.returnTo);
    if (value.registered === "1" || value.registered === true) search.registered = true;
    if (typeof value.reason === "string" && value.reason.length > 0) search.reason = value.reason;
    return search;
  },
};

export const Route = createFileRoute("/_public/login")({
  validateSearch: loginSearchSchema,
  beforeLoad: async ({ context }) => {
    if (typeof window === "undefined") return;
    try {
      await context.queryClient.ensureQueryData(currentSessionQuery());
    } catch (error) {
      if (error instanceof BrowserApiError && error.status === 401) return;
      throw error;
    }
    throw redirect({ to: "/" });
  },
  head: () => ({ meta: [{ title: "登录 | Gin Console" }] }),
  component: LoginPage,
});

function LoginPage() {
  const search = Route.useSearch();
  return (
    <AuthLayout eyebrow="身份验证" title="账号登录" description="使用企业账号继续访问 Gin Console">
      <LoginForm
        returnTo={safeReturnTo(search.returnTo)}
        registered={search.registered ?? false}
        reason={search.reason}
      />
      <div className="mt-10 flex items-center justify-center gap-2 border-t border-console-line pt-5 text-xs leading-5 text-console-muted">
        <ShieldCheck aria-hidden="true" size={15} strokeWidth={1.75} />
        <span>受保护的企业工作区，仅限授权成员访问</span>
      </div>
    </AuthLayout>
  );
}
