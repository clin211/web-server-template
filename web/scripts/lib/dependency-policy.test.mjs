import { describe, expect, it } from "vitest";
import { findDirectPrereleases, findUnapprovedPrereleases } from "./dependency-policy.mjs";

describe("dependency prerelease policy", () => {
  it("rejects a new prerelease while preserving the reviewed allowlist", () => {
    const lockfile = {
      packages: {
        "stable-package@1.2.3": {},
        "h3@2.0.1-rc.20": {},
        "new-runtime@3.0.0-beta.1": {},
      },
    };

    expect(findUnapprovedPrereleases(lockfile)).toEqual(["new-runtime@3.0.0-beta.1"]);
  });

  it("fails closed when the lockfile package map is missing", () => {
    expect(() => findUnapprovedPrereleases({})).toThrow("packages");
  });

  it("rejects direct prerelease dependencies", () => {
    expect(
      findDirectPrereleases({
        dependencies: { stable: "1.0.0", preview: "2.0.0-rc.1" },
        devDependencies: { nightly: "3.0.0-nightly.2" },
      }),
    ).toEqual(["nightly@3.0.0-nightly.2", "preview@2.0.0-rc.1"]);
  });
});
