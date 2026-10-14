import type { ReactNode } from "react";

type StatusPanelProps = Readonly<{
  title: string;
  children: ReactNode;
  requestId?: string | null;
  headingLevel?: 1 | 2;
}>;

export function StatusPanel({ title, children, requestId, headingLevel = 2 }: StatusPanelProps) {
  const Heading = headingLevel === 1 ? "h1" : "h2";
  return (
    <section className="border border-console-line bg-console-surface p-6 shadow-sm">
      <Heading className="text-base font-semibold text-console-ink">{title}</Heading>
      <div className="mt-2 text-sm leading-6 text-console-muted">{children}</div>
      {requestId ? (
        <p className="mt-4 border-t border-console-line pt-3 text-xs text-console-muted">
          请求 ID：<code className="font-mono text-console-ink">{requestId}</code>
        </p>
      ) : null}
    </section>
  );
}
