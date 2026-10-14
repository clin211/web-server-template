import fs from "node:fs";
import path from "node:path";
import { API } from "typescript/unstable/sync";

const srcDir = path.resolve(new URL("../src", import.meta.url).pathname);
const violations = [];
const sourceFiles = collectSourceFiles(srcDir);
const tsApi = new API({ cwd: path.resolve(new URL("..", import.meta.url).pathname) });
const snapshot = tsApi.updateSnapshot({ openFiles: sourceFiles });

for (const filePath of sourceFiles) {
  const project = snapshot.getDefaultProjectForFile(filePath);
  const sourceFile = project?.program.getSourceFile(filePath);
  if (!sourceFile) {
    throw new Error(`TypeScript could not parse ${relative(filePath)}`);
  }
  const importer = classify(filePath);

  for (const importNode of sourceFile.imports) {
    const specifier = importNode.getText(sourceFile).slice(1, -1);
    if (!specifier.startsWith("@/") && !specifier.startsWith(".")) continue;
    const targetPath = resolveInternalPath(filePath, specifier);
    if (!targetPath) continue;
    const target = classify(targetPath);
    if (!target) continue;

    if (importer === "shared" && target !== "shared") {
      violations.push(`${relative(filePath)} imports ${relative(targetPath)}`);
    }
    if (
      ["shared", "infrastructure", "integrations", "feature"].includes(importer) &&
      target === "composition"
    ) {
      violations.push(`${relative(filePath)} imports composition root ${relative(targetPath)}`);
    }
    if (importer === "infrastructure" && ["routes", "features", "integrations"].includes(target)) {
      violations.push(`${relative(filePath)} imports ${relative(targetPath)}`);
    }
    if (importer === "integrations" && ["routes", "features"].includes(target)) {
      violations.push(`${relative(filePath)} imports ${relative(targetPath)}`);
    }

    if (importer === "feature" && target === "feature") {
      const importerFeature = featureName(filePath);
      const targetFeature = featureName(targetPath);
      if (importerFeature && targetFeature && importerFeature !== targetFeature) {
        violations.push(`${relative(filePath)} imports another feature ${relative(targetPath)}`);
      }
    }
    if (importer === "routes" && target === "feature" && !isFeaturePublicEntry(targetPath)) {
      violations.push(`${relative(filePath)} deep-imports feature ${relative(targetPath)}`);
    }
  }
}

snapshot.dispose();
tsApi.close();

if (violations.length > 0) {
  console.error("Import boundary violations:");
  for (const violation of violations) console.error(`- ${violation}`);
  process.exitCode = 1;
} else {
  console.log("Import boundaries: PASS");
}

function collectSourceFiles(directory) {
  const files = [];
  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    const entryPath = path.join(directory, entry.name);
    if (entry.isDirectory()) files.push(...collectSourceFiles(entryPath));
    else if (/\.(ts|tsx)$/.test(entry.name) && entry.name !== "routeTree.gen.ts")
      files.push(entryPath);
  }
  return files;
}

function resolveInternalPath(importerPath, specifier) {
  const base = specifier.startsWith("@/")
    ? path.resolve(srcDir, specifier.slice(2))
    : path.resolve(path.dirname(importerPath), specifier);
  for (const candidate of [
    base,
    `${base}.ts`,
    `${base}.tsx`,
    path.join(base, "index.ts"),
    path.join(base, "index.tsx"),
  ]) {
    if (fs.existsSync(candidate) && fs.statSync(candidate).isFile()) return candidate;
  }
  return null;
}

function classify(filePath) {
  const relativePath = path.relative(srcDir, filePath);
  const [segment] = relativePath.split(path.sep);
  if (["router.tsx", "providers.tsx", "start.ts", "server.ts"].includes(relativePath)) {
    return "composition";
  }
  if (segment === "shared") return "shared";
  if (segment === "infrastructure") return "infrastructure";
  if (segment === "integrations") return "integrations";
  if (segment === "features") return "feature";
  if (segment === "routes") return "routes";
  return null;
}

function isFeaturePublicEntry(filePath) {
  const parts = path.relative(srcDir, filePath).split(path.sep);
  return (
    parts.length === 3 &&
    parts[0] === "features" &&
    (parts[2] === "index.ts" || parts[2] === "index.tsx" || parts[2] === "server.ts")
  );
}

function featureName(filePath) {
  const relativePath = path.relative(srcDir, filePath).split(path.sep);
  return relativePath[1] ?? null;
}

function relative(filePath) {
  return path.relative(path.resolve(srcDir, ".."), filePath);
}
