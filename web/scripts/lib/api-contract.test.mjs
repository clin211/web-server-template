import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { describe, expect, it } from "vitest";
import { loadApiContract, validateApiContract } from "./api-contract.mjs";

const successRef = "#/definitions/v1LoginResponseEnvelope";
const errorRef = "#/definitions/v1APIError";
const document = {
  swagger: "2.0",
  paths: {
    "/v1/auth/login": {
      post: {
        responses: {
          200: { schema: { $ref: successRef } },
          default: { schema: { $ref: errorRef } },
        },
      },
    },
  },
  definitions: {
    v1LoginResponseEnvelope: {},
    v1APIError: {},
  },
};

function validManifest() {
  return {
    path: "session.operation-manifest.json",
    value: {
      feature: "session",
      operations: [
        {
          name: "login",
          method: "POST",
          path: "/v1/auth/login",
          upstreamSecurity: "anonymous",
          credentialPolicy: "anonymous",
          successRef,
          errorRef,
        },
      ],
    },
  };
}

function cloneManifest() {
  return structuredClone(validManifest());
}

describe("API contract gate", () => {
  it("accepts a manifest aligned with the Swagger operation", () => {
    expect(validateApiContract(document, [validManifest()])).toBe(1);
  });

  it("fails closed when a required field is deleted", () => {
    const manifest = validManifest();
    delete manifest.value.operations[0].credentialPolicy;
    expect(() => validateApiContract(document, [manifest])).toThrow(
      "operation.credentialPolicy is required",
    );
  });

  it("fails closed when the success model belongs to another operation", () => {
    const manifest = validManifest();
    manifest.value.operations[0].successRef = errorRef;
    expect(() => validateApiContract(document, [manifest])).toThrow(
      "successRef does not match Swagger",
    );
  });

  it.each([
    ["invalid document", () => validateApiContract({}, [validManifest()]), "missing swagger 2.0"],
    ["no manifests", () => validateApiContract(document, []), "No feature operation manifests"],
    [
      "empty feature operations",
      () =>
        validateApiContract(document, [
          { path: "empty.json", value: { feature: "empty", operations: [] } },
        ]),
      "non-empty operations",
    ],
    [
      "duplicate operation",
      () => {
        const manifest = cloneManifest();
        manifest.value.operations.push(structuredClone(manifest.value.operations[0]));
        return validateApiContract(document, [manifest]);
      },
      "duplicate operation name",
    ],
  ])("rejects %s", (_label, action, message) => {
    expect(action).toThrow(message);
  });

  it.each([
    ["invalid method", (operation) => (operation.method = "TRACE"), "method is invalid"],
    ["unknown path", (operation) => (operation.path = "/unknown"), "is not in Swagger"],
    [
      "security mismatch",
      (operation) => (operation.upstreamSecurity = "bearer"),
      "security does not match Swagger",
    ],
    [
      "invalid credential policy",
      (operation) => (operation.credentialPolicy = "cookie"),
      "credentialPolicy is invalid",
    ],
    [
      "credential mismatch",
      (operation) => (operation.credentialPolicy = "access"),
      "credentialPolicy does not match security",
    ],
    [
      "unknown definition",
      (operation) => (operation.errorRef = "#/definitions/Unknown"),
      "references unknown",
    ],
    [
      "wrong error model",
      (operation) => (operation.errorRef = successRef),
      "errorRef does not match Swagger",
    ],
  ])("rejects an operation with %s", (_label, mutate, message) => {
    const manifest = cloneManifest();
    mutate(manifest.value.operations[0]);
    expect(() => validateApiContract(document, [manifest])).toThrow(message);
  });

  it("loads nested feature manifests from disk", () => {
    const directory = fs.mkdtempSync(path.join(os.tmpdir(), "console-contract-"));
    try {
      const swaggerPath = path.join(directory, "swagger.json");
      const featuresPath = path.join(directory, "features");
      const nestedPath = path.join(featuresPath, "session");
      fs.mkdirSync(nestedPath, { recursive: true });
      fs.writeFileSync(swaggerPath, JSON.stringify(document));
      fs.writeFileSync(
        path.join(nestedPath, "session.operation-manifest.json"),
        JSON.stringify(validManifest().value),
      );
      const loaded = loadApiContract(swaggerPath, featuresPath);
      expect(loaded.document.swagger).toBe("2.0");
      expect(loaded.manifests).toHaveLength(1);
    } finally {
      fs.rmSync(directory, { recursive: true, force: true });
    }
  });

  it("fails when the OpenAPI file is missing", () => {
    expect(() => loadApiContract("/missing/swagger.json", "/missing/features")).toThrow(
      "OpenAPI contract not found",
    );
  });
});
