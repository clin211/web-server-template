import type { ReactNode } from "react";
import { CircleAlert, Inbox, LockKeyhole, RefreshCw } from "lucide-react";

type TableStateKind = "loading" | "empty" | "error" | "forbidden" | "stale";

const stateContent: Record<TableStateKind, { label: string; icon: typeof Inbox }> = {
  loading: { label: "正在加载数据", icon: RefreshCw },
  empty: { label: "暂无数据", icon: Inbox },
  error: { label: "数据加载失败", icon: CircleAlert },
  forbidden: { label: "无权查看数据", icon: LockKeyhole },
  stale: { label: "数据可能已过期", icon: RefreshCw },
};

export function TableState({
  kind,
  action,
}: Readonly<{ kind: TableStateKind; action?: ReactNode }>) {
  const state = stateContent[kind];
  const Icon = state.icon;
  return (
    <div className="grid min-h-48 place-items-center border border-console-line bg-console-surface p-6 text-center">
      <div>
        <Icon
          aria-hidden="true"
          size={24}
          className={`mx-auto text-console-muted ${kind === "loading" ? "animate-spin" : ""}`}
        />
        <p className="mt-3 text-sm text-console-muted" role={kind === "error" ? "alert" : "status"}>
          {state.label}
        </p>
        {action ? <div className="mt-4">{action}</div> : null}
      </div>
    </div>
  );
}
