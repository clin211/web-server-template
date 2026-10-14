export const approvedPrereleases = new Set([
  // Reviewed transitive versions currently required by the stable Nitro toolchain.
  "h3@2.0.1-rc.20",
  "unenv@2.0.0-rc.24",
  // Babel build-time transitive dependency; it is not shipped as application runtime code.
  "gensync@1.0.0-beta.2",
]);

export function findUnapprovedPrereleases(lockfile) {
  if (!lockfile || typeof lockfile !== "object" || !isRecord(lockfile.packages)) {
    throw new Error("pnpm lockfile must contain a packages map");
  }

  return Object.keys(lockfile.packages)
    .filter((identifier) => isPrereleasePackage(identifier))
    .filter((identifier) => !approvedPrereleases.has(withoutPeerSuffix(identifier)))
    .sort();
}

export function findDirectPrereleases(manifest) {
  if (!manifest || typeof manifest !== "object") throw new Error("package manifest is invalid");
  const violations = [];
  for (const section of ["dependencies", "devDependencies", "optionalDependencies"]) {
    const entries = manifest[section];
    if (entries === undefined) continue;
    if (!isRecord(entries)) throw new Error(`package manifest ${section} must be an object`);
    for (const [name, version] of Object.entries(entries)) {
      if (typeof version !== "string") throw new Error(`${section}.${name} must be a string`);
      if (isPrereleaseVersion(version)) violations.push(`${name}@${version}`);
    }
  }
  return violations.sort();
}

function isPrereleasePackage(identifier) {
  const normalized = withoutPeerSuffix(identifier);
  const match = normalized.match(/^(@[^/]+\/[^@]+|[^@]+)@(.+)$/);
  return match ? isPrereleaseVersion(match[2]) : false;
}

function isPrereleaseVersion(version) {
  return /^\d+\.\d+\.\d+-[0-9A-Za-z][0-9A-Za-z.-]*(?:\+[0-9A-Za-z.-]+)?$/.test(version);
}

function withoutPeerSuffix(identifier) {
  const peerStart = identifier.indexOf("(");
  return peerStart === -1 ? identifier : identifier.slice(0, peerStart);
}

function isRecord(value) {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}
