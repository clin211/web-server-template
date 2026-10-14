import type { ReactNode } from "react";
import type { QueryClient } from "@tanstack/react-query";
import { HeadContent, Outlet, Scripts, createRootRouteWithContext } from "@tanstack/react-router";
import type { SessionCoordinator } from "../features/session";
import { BrowserRequestError } from "../infrastructure/http/browser-error";
import { Providers } from "../providers";
import { RoutePending } from "../shared/components/feedback/RoutePending";
import { StatusPanel } from "../shared/components/feedback/StatusPanel";
import "../styles/globals.css";

type RouterContext = Readonly<{
  queryClient: QueryClient;
  sessionCoordinator: SessionCoordinator;
}>;

export const Route = createRootRouteWithContext<RouterContext>()({
  head: () => ({
    meta: [
      { charSet: "utf-8" },
      { name: "viewport", content: "width=device-width, initial-scale=1" },
      { name: "theme-color", content: "#f5f7fb" },
      { title: "Console" },
    ],
  }),
  pendingComponent: RoutePending,
  errorComponent: RootError,
  notFoundComponent: NotFound,
  shellComponent: RootDocument,
  component: RootContent,
});

function RootDocument({ children }: Readonly<{ children: ReactNode }>) {
  return (
    <html lang="zh-CN">
      <head>
        <HeadContent />
      </head>
      <body>
        {children}
        <Scripts />
      </body>
    </html>
  );
}

function RootContent() {
  const { queryClient } = Route.useRouteContext();
  return (
    <Providers queryClient={queryClient}>
      <Outlet />
    </Providers>
  );
}

function RootError({ error }: Readonly<{ error: unknown }>) {
  const requestId = error instanceof BrowserRequestError ? error.requestId : null;
  return (
    <main className="grid min-h-screen place-items-center bg-console-canvas p-6">
      <div className="w-full max-w-lg">
        <StatusPanel title="页面暂时无法加载" requestId={requestId} headingLevel={1}>
          请稍后重试。如果问题持续，请联系维护人员。
        </StatusPanel>
      </div>
    </main>
  );
}

function NotFound() {
  return (
    <main className="grid min-h-screen place-items-center bg-console-canvas p-6">
      <section className="w-full max-w-lg border border-console-line bg-console-surface p-8 shadow-sm">
        <h1 className="text-2xl font-semibold text-console-ink">页面不存在</h1>
        <p className="mt-3 text-sm leading-6 text-console-muted">
          当前地址没有对应的 Console 页面。
        </p>
      </section>
    </main>
  );
}
