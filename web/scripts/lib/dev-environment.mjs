const DEVELOPMENT_DEFAULTS = Object.freeze({
  BACKEND_API_URL: "http://127.0.0.1:5558",
  CONSOLE_REQUEST_TIMEOUT_MS: "10000",
  CONSOLE_SESSION_SECRET: "local-development-session-secret-000000000000000",
});

export function resolveDevEnvironment({ source, fileEnv, port }) {
  const defaults = {
    ...DEVELOPMENT_DEFAULTS,
    CONSOLE_ORIGIN: `http://127.0.0.1:${port}`,
  };

  return Object.fromEntries(
    Object.keys(defaults).map((name) => [name, source[name] ?? fileEnv[name] ?? defaults[name]]),
  );
}

export function resolveStartPort(args, fallback = 5173) {
  const value = readOption(args, "--port");
  if (value === undefined || !/^\d+$/.test(value)) return fallback;
  const port = Number(value);
  return port > 0 && port <= 65_535 ? port : fallback;
}

export function withResolvedPort(args, port) {
  const filteredArgs = [];

  for (let index = 0; index < args.length; index += 1) {
    const argument = args[index];

    if (argument === "--port") {
      index += 1;
      continue;
    }

    if (argument.startsWith("--port=") || argument === "--strictPort") {
      continue;
    }

    filteredArgs.push(argument);
  }

  return [...filteredArgs, "--port", String(port), "--strictPort"];
}

function readOption(args, option) {
  for (let index = 0; index < args.length; index += 1) {
    if (args[index] === option) return args[index + 1];
    if (args[index].startsWith(`${option}=`)) return args[index].slice(option.length + 1);
  }
  return undefined;
}
