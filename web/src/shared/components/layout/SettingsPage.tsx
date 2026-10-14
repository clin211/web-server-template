import type { ReactNode } from "react";

type SettingsPageProps = Readonly<{
  title: string;
  description: string;
  children: ReactNode;
  tone?: "default" | "danger";
}>;

export function SettingsPage({
  title,
  description,
  children,
  tone = "default",
}: SettingsPageProps) {
  return (
    <main className="mx-auto w-full max-w-4xl px-4 py-6 sm:px-8 sm:py-10">
      <header className="border-b border-console-line pb-5">
        <h1 className="text-2xl font-semibold text-console-ink">{title}</h1>
        <p className="mt-2 text-sm leading-6 text-console-muted">{description}</p>
      </header>
      <section
        className={`mt-6 border bg-console-surface p-5 shadow-sm sm:p-6 ${
          tone === "danger" ? "border-red-200" : "border-console-line"
        }`}
      >
        {children}
      </section>
    </main>
  );
}
