import fs from "node:fs";
import { createServer as createHttpServer } from "node:http";
import net from "node:net";
import path from "node:path";
import { spawn } from "node:child_process";

const webRoot = path.resolve(new URL("..", import.meta.url).pathname);
const serverPath = path.join(webRoot, ".output", "server", "index.mjs");
if (!fs.existsSync(serverPath)) fail(`Production server output not found: ${serverPath}`);

await assertMissingEnvironmentFailsFast();

const port = await reservePort();
const fixtureBackend = await startFixtureBackend();
const runtime = spawnServer(validEnvironment(port, fixtureBackend.origin));
let stopped = false;

try {
  const origin = `http://127.0.0.1:${port}`;
  await waitForServer(`${origin}/api/health`, runtime.child);
  const health = await assertResponse(`${origin}/api/health`, 200, "health route", {
    headers: { "x-request-id": "production-smoke-request" },
  });
  if (health.headers.get("x-request-id") !== "production-smoke-request") {
    throw new Error("health route did not preserve the validated request ID");
  }

  await assertResponse(`${origin}/robots.txt`, 200, "static asset");
  await assertResponse(`${origin}/login`, 200, "login route");
  const csrf = await assertResponse(`${origin}/api/auth/csrf`, 200, "CSRF route");
  const csrfCookie = csrf.headers.get("set-cookie") ?? "";
  if (!csrfCookie.includes("console-csrf=") || !csrfCookie.includes("SameSite=Lax")) {
    throw new Error("CSRF route did not set the expected same-site cookie");
  }
  const csrfBody = await csrf.json();
  if (typeof csrfBody.token !== "string" || csrfBody.token.length === 0) {
    throw new Error("CSRF route did not return a usable token");
  }

  const login = await assertResponse(`${origin}/api/auth/login`, 200, "login Server Route", {
    method: "POST",
    headers: {
      "content-type": "application/json",
      cookie: csrfCookie.split(";", 1)[0],
      origin,
      "x-csrf-token": csrfBody.token,
      "x-request-id": "production-login-request",
    },
    body: JSON.stringify({ username: "smoke-user", password: "secret1" }),
  });
  const loginCookies = login.headers.getSetCookie();
  if (
    !loginCookies.some(
      (cookie) =>
        cookie.includes("console-access=fixture-access-token") && cookie.includes("HttpOnly"),
    ) ||
    !loginCookies.some(
      (cookie) => cookie.includes("console-refresh=") && cookie.includes("HttpOnly"),
    ) ||
    !loginCookies.some((cookie) => cookie.startsWith("console-csrf=; Max-Age=0"))
  ) {
    throw new Error("login Server Route did not commit the expected session Cookie set");
  }
  await expectJson(login, { ok: true }, "login Server Route");

  runtime.child.kill("SIGTERM");
  await waitForExit(runtime.child, 5_000, "Production server did not exit after SIGTERM");
  stopped = true;
  console.log(
    "Production smoke: PASS (env fail-fast, Node server, request context, upstream BFF, session Cookie, static asset, login route, SIGTERM)",
  );
} catch (error) {
  console.error(runtime.output());
  throw error;
} finally {
  if (!stopped) await terminate(runtime.child);
  await fixtureBackend.close();
}

// Global fetch connection pools must not keep this one-shot gate alive.
process.exit(0);

async function assertMissingEnvironmentFailsFast() {
  const port = await reservePort();
  const env = validEnvironment(port);
  delete env.CONSOLE_SESSION_SECRET;
  const runtime = spawnServer(env);
  try {
    const result = await waitForExit(
      runtime.child,
      5_000,
      "Production server did not fail fast for a missing CONSOLE_SESSION_SECRET",
    );
    if (result.code === 0) {
      throw new Error("Production server accepted a missing CONSOLE_SESSION_SECRET");
    }
  } catch (error) {
    console.error(runtime.output());
    throw error;
  } finally {
    await terminate(runtime.child);
  }
}

function validEnvironment(port, backendApiUrl = "http://127.0.0.1:5558") {
  return {
    ...process.env,
    BACKEND_API_URL: backendApiUrl,
    CONSOLE_ORIGIN: `http://127.0.0.1:${port}`,
    CONSOLE_SESSION_SECRET: "production-smoke-secret-with-at-least-32-bytes",
    CONSOLE_REQUEST_TIMEOUT_MS: "1000",
    NITRO_HOST: "127.0.0.1",
    NITRO_PORT: String(port),
  };
}

async function expectJson(response, expected, label) {
  const body = await response.json();
  if (JSON.stringify(body) !== JSON.stringify(expected)) {
    throw new Error(`${label} returned an unexpected JSON body`);
  }
}

async function startFixtureBackend() {
  const server = createHttpServer(async (request, response) => {
    try {
      if (request.method !== "POST" || request.url !== "/v1/auth/login") {
        sendJson(response, 404, { error: "unexpected fixture operation" });
        return;
      }
      if (
        request.headers.authorization ||
        request.headers["x-request-id"] !== "production-login-request"
      ) {
        sendJson(response, 400, { error: "unsafe or missing forwarded headers" });
        return;
      }

      let body = "";
      for await (const chunk of request) body += chunk;
      if (body !== JSON.stringify({ username: "smoke-user", password: "secret1" })) {
        sendJson(response, 400, { error: "unexpected fixture request body" });
        return;
      }

      const now = Date.now();
      const refreshPayload = Buffer.from(
        JSON.stringify({ exp: Math.floor(now / 1_000) + 7 * 24 * 60 * 60 }),
      ).toString("base64url");
      sendJson(response, 200, {
        code: 0,
        message: "ok",
        data: {
          accessToken: "fixture-access-token",
          refreshToken: `header.${refreshPayload}.signature`,
          expireAt: new Date(now + 15 * 60 * 1_000).toISOString(),
        },
      });
    } catch (error) {
      sendJson(response, 500, { error: error instanceof Error ? error.name : "fixture failure" });
    }
  });

  const port = await new Promise((resolve, reject) => {
    server.once("error", reject);
    server.listen(0, "127.0.0.1", () => {
      const address = server.address();
      if (!address || typeof address === "string") {
        reject(new Error("Could not start the fixture backend"));
        return;
      }
      resolve(address.port);
    });
  });

  return {
    origin: `http://127.0.0.1:${port}`,
    close: () =>
      new Promise((resolve, reject) =>
        server.close((error) => (error ? reject(error) : resolve())),
      ),
  };
}

function sendJson(response, status, body) {
  response.writeHead(status, { "content-type": "application/json" });
  response.end(JSON.stringify(body));
}

function spawnServer(env) {
  const child = spawn(process.execPath, [serverPath], {
    cwd: webRoot,
    env,
    stdio: ["ignore", "pipe", "pipe"],
  });
  let output = "";
  child.stdout.on("data", (chunk) => {
    output += chunk.toString();
  });
  child.stderr.on("data", (chunk) => {
    output += chunk.toString();
  });
  return { child, output: () => output };
}

async function assertResponse(url, status, label, init) {
  const response = await fetch(url, init);
  if (response.status !== status) {
    throw new Error(`${label} returned HTTP ${response.status}, expected ${status}`);
  }
  return response;
}

async function waitForServer(url, child) {
  const deadline = Date.now() + 15_000;
  while (Date.now() < deadline) {
    if (child.exitCode !== null) throw new Error(`Production server exited with ${child.exitCode}`);
    try {
      const response = await fetch(url);
      if (response.status === 200) return;
    } catch {
      // The server may still be binding its port.
    }
    await new Promise((resolve) => setTimeout(resolve, 100));
  }
  throw new Error(`Timed out waiting for production server at ${url}`);
}

function waitForExit(child, timeoutMs, timeoutMessage) {
  if (child.exitCode !== null || child.signalCode !== null) {
    return Promise.resolve({ code: child.exitCode, signal: child.signalCode });
  }
  return new Promise((resolve, reject) => {
    const timeout = setTimeout(() => reject(new Error(timeoutMessage)), timeoutMs);
    child.once("exit", (code, signal) => {
      clearTimeout(timeout);
      resolve({ code, signal });
    });
  });
}

async function terminate(child) {
  if (child.exitCode !== null || child.signalCode !== null) return;
  child.kill("SIGTERM");
  try {
    await waitForExit(child, 2_000, "Production server cleanup timed out");
  } catch {
    child.kill("SIGKILL");
    await waitForExit(child, 2_000, "Production server could not be terminated");
  }
}

function reservePort() {
  return new Promise((resolve, reject) => {
    const server = net.createServer();
    server.unref();
    server.once("error", reject);
    server.listen(0, "127.0.0.1", () => {
      const address = server.address();
      if (!address || typeof address === "string") {
        server.close();
        reject(new Error("Could not reserve a local TCP port"));
        return;
      }
      server.close((error) => (error ? reject(error) : resolve(address.port)));
    });
  });
}

function fail(message) {
  console.error(`Production smoke: FAIL - ${message}`);
  process.exit(1);
}
