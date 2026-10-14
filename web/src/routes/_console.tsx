import { useMutation, useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { Outlet, createFileRoute, redirect } from "@tanstack/react-router";
import {
  BrowserApiError,
  currentSessionQuery,
  logoutSessionMutationOptions,
} from "../features/session";
import { ConsoleShell } from "../shared/components/layout/ConsoleShell";
import { RoutePending } from "../shared/components/feedback/RoutePending";
import { ForbiddenState } from "../shared/components/feedback/ForbiddenState";
import { StatusPanel } from "../shared/components/feedback/StatusPanel";

export const Route = createFileRoute("/_console")({
  beforeLoad: async ({ location, context }) => {
    try {
      if (typeof window === "undefined") return {};
      await context.queryClient.ensureQueryData(currentSessionQuery());
      return {};
    } catch (error) {
      if (error instanceof BrowserApiError && error.status === 401) {
        throw redirect({
          to: "/session-refresh",
          search: { returnTo: location.href },
        });
      }
      throw error;
    }
  },
  pendingComponent: RoutePending,
  errorComponent: ConsoleError,
  component: ConsoleLayout,
});

function ConsoleError({ error }: Readonly<{ error: unknown }>) {
  if (error instanceof BrowserApiError && error.status === 403) return <ForbiddenState />;
  return (
    <main className="mx-auto w-full max-w-2xl p-6 sm:p-10">
      <StatusPanel
        title="页面暂时无法加载"
        requestId={error instanceof BrowserApiError ? error.requestId : null}
      >
        当前请求未完成，请稍后重试。
      </StatusPanel>
    </main>
  );
}

function ConsoleLayout() {
  const queryClient = useQueryClient();
  const logoutMutation = useMutation(logoutSessionMutationOptions());
  const { data } = useSuspenseQuery(currentSessionQuery());

  return (
    <ConsoleShell
      displayName={data.user.nickname || data.user.username}
      username={data.user.username}
      onLogout={async () => {
        await logoutMutation.mutateAsync();
        queryClient.clear();
        window.location.assign("/login");
      }}
    >
      <Outlet />
    </ConsoleShell>
  );
}
