import { defineConfig } from "vitest/config";

export default defineConfig({
  resolve: {
    alias: {
      "@": new URL("./src", import.meta.url).pathname,
    },
  },
  test: {
    environment: "jsdom",
    setupFiles: ["./src/test/setup.ts"],
    include: ["src/**/*.test.{ts,tsx}", "scripts/**/*.test.mjs"],
    coverage: {
      provider: "v8",
      reporter: ["text", "html"],
      exclude: [
        "src/routes/**",
        "src/routeTree.gen.ts",
        "src/providers.tsx",
        "src/router.tsx",
        "src/shared/components/**",
        "src/features/**/components/**",
        "src/features/**/index.ts",
        "src/features/*/server.ts",
        "src/features/_template/**",
        "src/integrations/_template/**",
      ],
      thresholds: {
        lines: 80,
        statements: 80,
        functions: 80,
        branches: 75,
        "src/features/session/return-to.ts": { branches: 100 },
        "src/infrastructure/http/{browser-error,json-body,request-policy,server-client}.ts": {
          branches: 100,
        },
        "src/infrastructure/observability/request-context.ts": { branches: 100 },
        "src/infrastructure/session/*.ts": { branches: 100 },
        "src/integrations/go-api/session-adapter.ts": { branches: 100 },
      },
    },
  },
});
