import { useState, type ReactNode } from "react";
import { Button, Tooltip } from "@heroui/react";
import { Link } from "@tanstack/react-router";
import { IdCard, LayoutDashboard, LogOut, ShieldCheck, UserRound, UserX } from "lucide-react";

type ConsoleShellProps = Readonly<{
  children: ReactNode;
  displayName: string;
  username: string;
  onLogout: () => Promise<void>;
}>;

const navigation = [
  { to: "/" as const, label: "概览", icon: LayoutDashboard },
  { to: "/settings/profile" as const, label: "个人资料", icon: IdCard },
  { to: "/settings/security" as const, label: "安全", icon: ShieldCheck },
  { to: "/settings/account" as const, label: "账号", icon: UserX },
];

export function ConsoleShell({ children, displayName, username, onLogout }: ConsoleShellProps) {
  const [isLoggingOut, setIsLoggingOut] = useState(false);
  const [logoutError, setLogoutError] = useState(false);

  async function handleLogout() {
    setIsLoggingOut(true);
    setLogoutError(false);
    try {
      await onLogout();
    } catch {
      setLogoutError(true);
      setIsLoggingOut(false);
    }
  }

  return (
    <div className="min-h-screen bg-console-canvas text-console-ink">
      <header className="border-b border-console-line bg-console-surface">
        <div className="mx-auto flex min-h-16 w-full max-w-[1440px] items-center justify-between gap-4 px-4 sm:px-8">
          <Link
            to="/"
            className="flex min-w-0 items-center gap-3 text-console-ink no-underline"
            aria-label="返回 Console 概览"
          >
            <span className="grid size-9 shrink-0 place-items-center bg-console-accent text-sm font-bold text-white">
              GC
            </span>
            <span className="truncate text-sm font-semibold">Gin Console</span>
          </Link>

          <nav aria-label="主导航" className="hidden items-center gap-1 md:flex">
            {navigation.map((item) => (
              <NavItem key={item.to} {...item} />
            ))}
          </nav>

          <div className="flex min-w-0 items-center gap-2">
            <UserRound aria-hidden="true" size={17} className="shrink-0 text-console-muted" />
            <div className="hidden min-w-0 text-right sm:block">
              <p className="truncate text-xs font-medium text-console-ink">{displayName}</p>
              <p className="truncate text-xs text-console-muted">{username}</p>
            </div>
            <Tooltip>
              <Button
                variant="ghost"
                size="sm"
                isIconOnly
                isDisabled={isLoggingOut}
                onPress={handleLogout}
                aria-label="退出登录"
              >
                <LogOut aria-hidden="true" size={17} />
              </Button>
              <Tooltip.Content>{logoutError ? "退出失败，请重试" : "退出登录"}</Tooltip.Content>
            </Tooltip>
          </div>
        </div>

        <nav
          aria-label="移动端主导航"
          className="mx-auto flex w-full max-w-[1440px] gap-1 overflow-x-auto border-t border-console-line px-4 py-2 md:hidden"
        >
          {navigation.map((item) => (
            <NavItem key={item.to} {...item} />
          ))}
        </nav>
      </header>
      {logoutError ? (
        <p
          className="border-b border-red-200 bg-red-50 px-4 py-2 text-center text-xs text-red-700"
          role="alert"
        >
          退出请求未完成，请重试。
        </p>
      ) : null}
      <div data-testid="console-shell">{children}</div>
    </div>
  );
}

type NavigationItem = (typeof navigation)[number];

function NavItem({ to, label, icon: Icon }: NavigationItem) {
  return (
    <Link
      to={to}
      activeOptions={{ exact: to === "/" }}
      activeProps={{ className: "bg-console-accent-soft text-console-accent" }}
      className="flex shrink-0 items-center gap-2 px-3 py-2 text-sm text-console-muted no-underline hover:bg-console-canvas hover:text-console-ink"
    >
      <Icon aria-hidden="true" size={16} />
      {label}
    </Link>
  );
}
