import { useSuspenseQuery } from "@tanstack/react-query";
import { Link, createFileRoute } from "@tanstack/react-router";
import { ArrowRight, IdCard, ShieldCheck, UserX } from "lucide-react";
import { currentSessionQuery } from "../../features/session";

export const Route = createFileRoute("/_console/")({
  head: () => ({ meta: [{ title: "概览 | Gin Console" }] }),
  component: ConsoleOverview,
});

const settingsLinks = [
  { to: "/settings/profile" as const, label: "个人资料", icon: IdCard },
  { to: "/settings/security" as const, label: "安全设置", icon: ShieldCheck },
  { to: "/settings/account" as const, label: "账号管理", icon: UserX },
];

function ConsoleOverview() {
  const { data } = useSuspenseQuery(currentSessionQuery());
  const { user } = data;

  return (
    <main className="mx-auto w-full max-w-[1120px] px-4 py-6 sm:px-8 sm:py-10">
      <header className="border-b border-console-line pb-6">
        <p className="text-sm text-console-muted">当前账号</p>
        <h1 className="mt-1 text-2xl font-semibold text-console-ink">
          {user.nickname || user.username}
        </h1>
      </header>

      <section className="mt-6 border border-console-line bg-console-surface shadow-sm">
        <div className="border-b border-console-line px-5 py-4 sm:px-6">
          <h2 className="text-base font-semibold text-console-ink">账号摘要</h2>
        </div>
        <dl className="grid sm:grid-cols-2">
          <AccountValue label="用户名" value={user.username} />
          <AccountValue label="账号状态" value={user.status === 0 ? "活跃" : "已禁用"} />
          <AccountValue label="邮箱" value={user.email || "未设置"} />
          <AccountValue label="手机号" value={user.phone || "未设置"} />
        </dl>
      </section>

      <section className="mt-8">
        <h2 className="text-base font-semibold text-console-ink">账号设置</h2>
        <div className="mt-4 grid gap-3 sm:grid-cols-3">
          {settingsLinks.map(({ to, label, icon: Icon }) => (
            <Link
              key={to}
              to={to}
              className="flex min-h-20 items-center gap-3 border border-console-line bg-console-surface px-4 py-3 text-console-ink no-underline shadow-sm hover:border-console-accent/50 hover:bg-console-accent-soft"
            >
              <span className="grid size-9 shrink-0 place-items-center bg-console-accent-soft text-console-accent">
                <Icon aria-hidden="true" size={18} />
              </span>
              <span className="min-w-0 flex-1 text-sm font-medium">{label}</span>
              <ArrowRight aria-hidden="true" size={16} className="shrink-0 text-console-muted" />
            </Link>
          ))}
        </div>
      </section>
    </main>
  );
}

function AccountValue({ label, value }: Readonly<{ label: string; value: string }>) {
  return (
    <div className="border-b border-console-line px-5 py-4 last:border-b-0 sm:px-6 sm:[&:nth-last-child(-n+2)]:border-b-0 sm:[&:nth-child(odd)]:border-r">
      <dt className="text-xs text-console-muted">{label}</dt>
      <dd className="mt-1 break-words text-sm font-medium text-console-ink">{value}</dd>
    </div>
  );
}
