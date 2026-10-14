import "@tanstack/react-start/server-only";
import {
  ServerHttpClient,
  type ServerClientOptions,
} from "../../infrastructure/http/server-client";

export function createGoApiClient(options: ServerClientOptions): ServerHttpClient {
  return new ServerHttpClient(options);
}
