import { spawn } from "node:child_process";
import { createServer } from "node:net";
import { fileURLToPath } from "node:url";
import { loadEnv } from "vite";
import {
  resolveDevEnvironment,
  resolveStartPort,
  withResolvedPort,
} from "./lib/dev-environment.mjs";

const DEFAULT_HOST = "127.0.0.1";
const DEFAULT_PORT = 5173;
const MAX_PORT_ATTEMPTS = 100;
const webRoot = fileURLToPath(new URL("..", import.meta.url));
const viteEntry = fileURLToPath(new URL("../node_modules/vite/bin/vite.js", import.meta.url));
const inputArgs = process.argv.slice(2);
const mode = readOption(inputArgs, "--mode") ?? "development";
const port = await findAvailablePort(resolveStartPort(inputArgs, DEFAULT_PORT), DEFAULT_HOST);
const fileEnv = loadEnv(mode, webRoot, "");
const resolvedEnv = resolveDevEnvironment({ source: process.env, fileEnv, port });
const viteArgs = withResolvedPort(withDefaultHost(inputArgs), port);

const child = spawn(process.execPath, [viteEntry, "dev", ...viteArgs], {
  cwd: webRoot,
  env: { ...fileEnv, ...process.env, ...resolvedEnv },
  stdio: "inherit",
});

for (const signal of ["SIGINT", "SIGTERM"]) {
  process.once(signal, () => child.kill(signal));
}

child.on("error", (error) => {
  console.error(`Failed to start the Vite development server: ${error.message}`);
});

const exitCode = await new Promise((resolve) => {
  child.on("exit", (code) => resolve(code ?? 1));
});

process.exitCode = exitCode;

function withDefaultHost(args) {
  return readOption(args, "--host") === undefined ? [...args, "--host", DEFAULT_HOST] : args;
}

function readOption(args, option) {
  for (let index = 0; index < args.length; index += 1) {
    if (args[index] === option) {
      return args[index + 1] ?? "";
    }

    if (args[index].startsWith(`${option}=`)) {
      return args[index].slice(option.length + 1);
    }
  }

  return undefined;
}

async function findAvailablePort(startPort, host) {
  for (let portOffset = 0; portOffset < MAX_PORT_ATTEMPTS; portOffset += 1) {
    const candidate = startPort + portOffset;

    if (await canListen(candidate, host)) {
      return candidate;
    }
  }

  throw new Error(`No available development port found from ${startPort}`);
}

function canListen(port, host) {
  return new Promise((resolve) => {
    const server = createServer();

    server.once("error", () => resolve(false));
    server.listen({ host, port, exclusive: true }, () => {
      server.close((error) => resolve(error === undefined));
    });
  });
}
