import fs from "node:fs";
import path from "node:path";
import { parse } from "yaml";
import {
  approvedPrereleases,
  findDirectPrereleases,
  findUnapprovedPrereleases,
} from "./lib/dependency-policy.mjs";

const webRoot = path.resolve(new URL("..", import.meta.url).pathname);

try {
  const manifest = JSON.parse(fs.readFileSync(path.join(webRoot, "package.json"), "utf8"));
  const lockfile = parse(fs.readFileSync(path.join(webRoot, "pnpm-lock.yaml"), "utf8"));
  const violations = [...findDirectPrereleases(manifest), ...findUnapprovedPrereleases(lockfile)];
  if (violations.length > 0) {
    throw new Error(`unreviewed prerelease dependencies: ${violations.join(", ")}`);
  }
  console.log(
    `Dependency policy: PASS (${approvedPrereleases.size} reviewed transitive prereleases)`,
  );
} catch (error) {
  console.error(
    `Dependency policy: FAIL - ${error instanceof Error ? error.message : String(error)}`,
  );
  process.exitCode = 1;
}
