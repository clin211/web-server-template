import { Link } from "@tanstack/react-router";
import { LogIn } from "lucide-react";

export function UnauthorizedState() {
  return (
    <main className="grid min-h-[100dvh] place-items-center bg-console-canvas p-6">
      <section className="w-full max-w-md border border-console-line bg-console-surface p-6 text-center shadow-sm">
        <LogIn aria-hidden="true" className="mx-auto text-console-muted" size={28} />
        <h1 className="mt-4 text-lg font-semibold text-console-ink">需要登录</h1>
        <p className="mt-2 text-sm text-console-muted">登录状态无效，请重新登录。</p>
        <Link
          to="/login"
          search={{ reason: "session-expired" }}
          className="mt-5 inline-flex min-h-10 items-center bg-console-accent px-4 text-sm font-medium text-white no-underline"
        >
          返回登录
        </Link>
      </section>
    </main>
  );
}
