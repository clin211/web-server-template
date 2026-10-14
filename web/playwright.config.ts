import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 2 : 0,
  reporter: process.env.CI ? "line" : "list",
  use: {
    baseURL: "http://127.0.0.1:5173",
    trace: "on-first-retry",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
  webServer: {
    command: "node ./node_modules/vite/bin/vite.js dev --host 127.0.0.1",
    env: {
      ...process.env,
      BACKEND_API_URL: "http://127.0.0.1:5558",
      CONSOLE_ORIGIN: "http://127.0.0.1:5173",
      CONSOLE_SESSION_SECRET: "playwright-only-session-secret-32-bytes",
    },
    url: "http://127.0.0.1:5173/login",
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
  },
});
