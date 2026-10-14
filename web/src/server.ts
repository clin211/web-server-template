import { createServerEntry } from "@tanstack/react-start/server-entry";
import { createStartHandler, defaultStreamHandler } from "@tanstack/react-start/server";
import { getServerEnv } from "./infrastructure/env";

const startHandler = createStartHandler(defaultStreamHandler);

const fetch = async (...args: Parameters<typeof startHandler>): Promise<Response> => {
  // Validate configuration before serving any request so a misconfigured instance fails closed.
  getServerEnv();
  return startHandler(...args);
};

export default createServerEntry({ fetch });
