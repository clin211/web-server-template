import { ShieldX } from "lucide-react";

export function ForbiddenState() {
  return (
    <main className="grid min-h-[calc(100vh-4rem)] place-items-center p-6">
      <section className="w-full max-w-md border border-console-line bg-console-surface p-6 text-center shadow-sm">
        <ShieldX aria-hidden="true" className="mx-auto text-console-muted" size={28} />
        <h1 className="mt-4 text-lg font-semibold text-console-ink">无权访问</h1>
        <p className="mt-2 text-sm text-console-muted">当前账号没有访问此页面的权限。</p>
      </section>
    </main>
  );
}
