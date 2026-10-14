import fs from "node:fs";
import path from "node:path";

const requiredOperationFields = [
  "name",
  "method",
  "path",
  "upstreamSecurity",
  "credentialPolicy",
  "successRef",
  "errorRef",
];

export function loadApiContract(swaggerPath, featuresPath) {
  if (!fs.existsSync(swaggerPath)) {
    throw new Error(`OpenAPI contract not found: ${swaggerPath}`);
  }
  const document = JSON.parse(fs.readFileSync(swaggerPath, "utf8"));
  const manifests = collectManifests(featuresPath).map((manifestPath) => ({
    path: manifestPath,
    value: JSON.parse(fs.readFileSync(manifestPath, "utf8")),
  }));
  return { document, manifests };
}

export function validateApiContract(document, manifests) {
  if (document?.swagger !== "2.0" || !document.paths || !document.definitions) {
    throw new Error("OpenAPI contract is missing swagger 2.0 paths or definitions");
  }
  if (!Array.isArray(manifests) || manifests.length === 0) {
    throw new Error("No feature operation manifests were found");
  }

  let operationCount = 0;
  const operationNames = new Set();
  for (const { path: manifestPath, value: manifest } of manifests) {
    if (
      typeof manifest?.feature !== "string" ||
      manifest.feature.length === 0 ||
      !Array.isArray(manifest.operations) ||
      manifest.operations.length === 0
    ) {
      throw new Error(`${manifestPath}: feature and non-empty operations are required`);
    }

    for (const operation of manifest.operations) {
      operationCount += 1;
      const operationKey = `${manifest.feature}:${operation?.name}`;
      if (operationNames.has(operationKey)) {
        throw new Error(`${manifestPath}: duplicate operation name ${operation?.name}`);
      }
      operationNames.add(operationKey);
      validateOperation(document, manifestPath, operation);
    }
  }
  return operationCount;
}

export function validateOperation(document, manifestPath, operation) {
  for (const key of requiredOperationFields) {
    if (typeof operation?.[key] !== "string" || operation[key].length === 0) {
      throw new Error(`${manifestPath}: operation.${key} is required`);
    }
  }

  const method = operation.method.toLowerCase();
  if (!["get", "post", "put", "delete", "patch"].includes(method)) {
    throw new Error(`${manifestPath}: ${operation.name} method is invalid`);
  }
  const upstream = document.paths[operation.path]?.[method];
  if (!upstream) {
    throw new Error(`${manifestPath}: ${operation.method} ${operation.path} is not in Swagger`);
  }

  const actualSecurity =
    Array.isArray(upstream.security) && upstream.security.length > 0 ? "bearer" : "anonymous";
  if (operation.upstreamSecurity !== actualSecurity) {
    throw new Error(`${manifestPath}: ${operation.name} security does not match Swagger`);
  }
  if (!["anonymous", "access", "refresh"].includes(operation.credentialPolicy)) {
    throw new Error(`${manifestPath}: ${operation.name} credentialPolicy is invalid`);
  }
  if (
    (actualSecurity === "anonymous" && operation.credentialPolicy !== "anonymous") ||
    (actualSecurity === "bearer" && operation.credentialPolicy === "anonymous")
  ) {
    throw new Error(`${manifestPath}: ${operation.name} credentialPolicy does not match security`);
  }

  const actualSuccessRef = upstream.responses?.["200"]?.schema?.$ref;
  const actualErrorRef = upstream.responses?.default?.schema?.$ref;
  assertOperationRef(document, manifestPath, operation.name, "successRef", operation.successRef);
  assertOperationRef(document, manifestPath, operation.name, "errorRef", operation.errorRef);
  if (operation.successRef !== actualSuccessRef) {
    throw new Error(`${manifestPath}: ${operation.name} successRef does not match Swagger`);
  }
  if (operation.errorRef !== actualErrorRef) {
    throw new Error(`${manifestPath}: ${operation.name} errorRef does not match Swagger`);
  }
}

function assertOperationRef(document, manifestPath, operationName, field, reference) {
  const prefix = "#/definitions/";
  if (!reference.startsWith(prefix) || !document.definitions[reference.slice(prefix.length)]) {
    throw new Error(`${manifestPath}: ${operationName} ${field} references unknown ${reference}`);
  }
}

function collectManifests(directory) {
  if (!fs.existsSync(directory)) return [];
  const files = [];
  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    const entryPath = path.join(directory, entry.name);
    if (entry.isDirectory()) files.push(...collectManifests(entryPath));
    else if (entry.name.endsWith(".operation-manifest.json")) files.push(entryPath);
  }
  return files.sort();
}
