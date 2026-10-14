import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { nitroV2Plugin } from "@tanstack/nitro-v2-vite-plugin";
import { tanstackStart } from "@tanstack/react-start/plugin/vite";

const envStartupPlugin = new URL("./src/infrastructure/env-startup-plugin.ts", import.meta.url)
  .pathname;

export default defineConfig({
  plugins: [
    tailwindcss(),
    tanstackStart(),
    nitroV2Plugin({
      compatibilityDate: "2026-08-29",
      plugins: [envStartupPlugin],
    }),
    react(),
  ],
});
