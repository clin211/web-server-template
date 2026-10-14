import path from "node:path";
import { loadApiContract, validateApiContract } from "./lib/api-contract.mjs";

const webRoot = path.resolve(new URL("..", import.meta.url).pathname);
const repoRoot = path.resolve(webRoot, "..");
const swaggerPath = path.join(repoRoot, "api/openapi/apiserver.swagger.json");

try {
  const { document, manifests } = loadApiContract(
    swaggerPath,
    path.join(webRoot, "src", "features"),
  );
  const operationCount = validateApiContract(document, manifests);
  console.log(
    `API contract: PASS (${Object.keys(document.paths).length} upstream paths, ${operationCount} feature operations)`,
  );
} catch (error) {
  console.error(`API contract: FAIL - ${error instanceof Error ? error.message : String(error)}`);
  process.exitCode = 1;
}
