import { useEffect } from "react";
import { createFileRoute } from "@tanstack/react-router";
import { safeReturnTo, sessionKeys } from "../../features/session";

export const Route = createFileRoute("/_public/session-refresh")({
  validateSearch: {
    parse(value: Record<string, unknown>) {
      return { returnTo: safeReturnTo(value.returnTo) };
    },
  },
  head: () => ({ meta: [{ title: "恢复登录状态 | Gin Console" }] }),
  component: SessionRefreshPage,
});

function SessionRefreshPage() {
  const { returnTo } = Route.useSearch();
  const { queryClient, sessionCoordinator } = Route.useRouteContext();

  useEffect(() => {
    void sessionCoordinator
      .refreshOnce()
      .then(async () => {
        await queryClient.invalidateQueries({ queryKey: sessionKeys.all });
        window.location.assign(returnTo);
      })
      .catch(() => {
        queryClient.clear();
        window.location.assign(
          `/login?returnTo=${encodeURIComponent(returnTo)}&reason=session-expired`,
        );
      });
  }, [queryClient, returnTo, sessionCoordinator]);

  return (
    <main className="grid min-h-[100dvh] place-items-center bg-console-canvas p-6">
      <div className="text-center" role="status" aria-live="polite">
        <div className="mx-auto size-8 animate-spin rounded-full border-2 border-console-line border-t-console-accent motion-reduce:animate-none" />
        <p className="mt-4 text-sm font-medium text-console-ink">正在恢复登录状态</p>
      </div>
    </main>
  );
}
