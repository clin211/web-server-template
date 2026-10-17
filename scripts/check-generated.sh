#!/usr/bin/env bash
set -euo pipefail
root="$(pwd)"
check_dir="$(mktemp -d "${TMPDIR:-/tmp}/template-generated.XXXXXX")"
trap 'rm -rf "$check_dir"' EXIT
# Use the actual worktree, including unstaged source and new Proto files.
tar --exclude=.git --exclude=_output --exclude=data -cf - . | tar -xf - -C "$check_dir"
(cd "$check_dir"; make protoc; go generate ./internal/apiserver ./internal/pkg/dbschema)
for path in pkg/api/apiserver/v1 api/openapi internal/apiserver/wire_gen.go internal/pkg/dbschema/manifest_gen.go; do
  diff -ru "$root/$path" "$check_dir/$path"
done
