import type { ReactNode } from "react";

type FormMessageProps = Readonly<{
  children: ReactNode;
  tone?: "error" | "success" | "info";
}>;

const toneClasses = {
  error: "border-red-200 bg-red-50 text-red-700",
  success: "border-console-success/30 bg-console-success/10 text-console-success",
  info: "border-console-accent/30 bg-console-accent-soft text-console-accent",
} as const;

export function FormMessage({ children, tone = "error" }: FormMessageProps) {
  return (
    <div
      className={`border px-4 py-3 text-sm ${toneClasses[tone]}`}
      role={tone === "error" ? "alert" : "status"}
    >
      {children}
    </div>
  );
}
