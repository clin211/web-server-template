import fs from "node:fs";
import path from "node:path";

const webRoot = path.resolve(new URL("..", import.meta.url).pathname);
const clientRoot = path.join(webRoot, "dist", "client");

if (!fs.existsSync(clientRoot)) {
  fail(`Client build directory not found: ${clientRoot}`);
}

const files = collectFiles(clientRoot).filter((file) => /\.(js|mjs|map)$/.test(file));
const forbiddenPatterns = [
  /BACKEND_API_URL/,
  /CONSOLE_SESSION_SECRET/,
  /CONSOLE_REQUEST_TIMEOUT_MS/,
  /Bearer\s+access-token/,
  /console-(?:access|refresh)=/,
  /__Host-console-(?:access|refresh)=/,
  /node:crypto/,
];

const violations = [];
for (const filePath of files) {
  const source = fs.readFileSync(filePath, "utf8");
  for (const pattern of forbiddenPatterns) {
    if (pattern.test(source)) {
      violations.push(`${path.relative(webRoot, filePath)} matches ${pattern}`);
    }
  }
}

if (violations.length > 0) {
  console.error("Browser bundle check: FAIL");
  for (const violation of violations) console.error(`- ${violation}`);
  process.exit(1);
}

console.log(`Browser bundle check: PASS (${files.length} client files scanned)`);

function collectFiles(directory) {
  const files = [];
  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    const entryPath = path.join(directory, entry.name);
    if (entry.isDirectory()) files.push(...collectFiles(entryPath));
    else files.push(entryPath);
  }
  return files;
}

function fail(message) {
  console.error(`Browser bundle check: FAIL - ${message}`);
  process.exit(1);
}
