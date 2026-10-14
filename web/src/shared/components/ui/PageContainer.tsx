import type { ReactNode } from "react";

export function PageContainer({ children }: Readonly<{ children: ReactNode }>) {
  return <div className="mx-auto w-full max-w-[1280px] px-4 py-6 sm:px-8 sm:py-10">{children}</div>;
}
