import type { ReactNode } from "react";
import { QueryClientProvider, type QueryClient } from "@tanstack/react-query";
import { NuqsAdapter } from "nuqs/adapters/tanstack-router";
import { MotionConfig } from "motion/react";

export function Providers({
  children,
  queryClient,
}: Readonly<{ children: ReactNode; queryClient: QueryClient }>) {
  return (
    <MotionConfig reducedMotion="user">
      <QueryClientProvider client={queryClient}>
        <NuqsAdapter>
          <div data-theme="light" className="min-h-screen">
            {children}
          </div>
        </NuqsAdapter>
      </QueryClientProvider>
    </MotionConfig>
  );
}
