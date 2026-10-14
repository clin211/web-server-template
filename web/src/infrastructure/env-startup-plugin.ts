import { getServerEnv } from "./env";

export default function validateServerEnvironmentAtStartup(): void {
  getServerEnv();
}
