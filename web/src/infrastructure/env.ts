import "@tanstack/react-start/server-only";
import { z } from "zod";

const serverEnvSchema = z.object({
  BACKEND_API_URL: httpUrlSchema(),
  CONSOLE_ORIGIN: httpUrlSchema(),
  CONSOLE_SESSION_SECRET: z.string().min(32),
  CONSOLE_REQUEST_TIMEOUT_MS: z.coerce.number().int().positive().default(10_000),
});

function httpUrlSchema() {
  return z
    .string()
    .url()
    .refine((value) => {
      const protocol = new URL(value).protocol;
      return protocol === "http:" || protocol === "https:";
    }, "URL must use http or https");
}

export type ServerEnv = z.infer<typeof serverEnvSchema>;

export function getServerEnv(source: NodeJS.ProcessEnv = process.env): ServerEnv {
  if (typeof window !== "undefined") {
    throw new Error("Server environment can only be read in the server runtime");
  }

  return serverEnvSchema.parse(source);
}
