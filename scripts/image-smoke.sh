#!/usr/bin/env bash
set -euo pipefail
image="${1:?immutable image ID is required}"
run_name="template-smoke-${GITHUB_RUN_ID:-local}-$$"
common=(--network host -e APP_POSTGRESQL_ADDR=127.0.0.1:55432 -e APP_POSTGRESQL_DATABASE=enterprise_template_test -e APP_POSTGRESQL_PASSWORD=integration-postgres -e APP_POSTGRESQL_USERNAME=postgres -e APP_POSTGRESQL_SSL_MODE=disable -e APP_REDIS_ADDR=127.0.0.1:56380 -e APP_REDIS_PASSWORD=integration-redis -e APP_JWT_SECRET=smoke-test-jwt-secret-with-at-least-32-characters -e APP_KAFKA_BROKERS=127.0.0.1:29092 -e APP_OTEL_OUTPUT_MODE=classic)
cleanup() {
  for role in apiserver worker scheduler; do
    docker logs "$run_name-$role" > "${TMPDIR:-/tmp}/$run_name-$role.log" 2>&1 || true
    docker stop --time 45 "$run_name-$role" >/dev/null 2>&1 || true
    docker rm "$run_name-$role" >/dev/null 2>&1 || true
  done
}
trap cleanup EXIT
# Migration is an explicit release step, using the very same image.
docker run --rm --network host -e GOOSE_DRIVER=postgres -e 'GOOSE_DBSTRING=postgres://postgres:integration-postgres@127.0.0.1:55432/enterprise_template_test?sslmode=disable' --entrypoint /app/goose "$image" -env=none -dir /app/migrations up
docker run -d --name "$run_name-apiserver" "${common[@]}" "$image" >/dev/null
docker run -d --name "$run_name-worker" "${common[@]}" -e APP_OUTBOX_ENABLED=true -e APP_CONSUMER_ENABLED=true --entrypoint /app/gin-enterprise-template-worker "$image" --config /app/configs/configs.yaml >/dev/null
docker run -d --name "$run_name-scheduler" "${common[@]}" --entrypoint /app/gin-enterprise-template-scheduler "$image" --config /app/configs/configs.yaml >/dev/null
for port in 5558 5560 5561; do
  for attempt in {1..30}; do
    if curl --fail --silent "http://127.0.0.1:$port/readyz" >/dev/null; then break; fi
    sleep 1
  done
  curl --fail --silent "http://127.0.0.1:$port/readyz" >/dev/null
done
python3 scripts/report-smoke.py
