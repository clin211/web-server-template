#!/bin/sh
set -eu

server_binary="${1:?server binary path is required}"
http_port="${SMOKE_HTTP_PORT:-5558}"
postgres_port="${INTEGRATION_POSTGRES_PORT:-55432}"
redis_port="${INTEGRATION_REDIS_PORT:-56380}"
kafka_port="${INTEGRATION_KAFKA_PORT:-29092}"

"${server_binary}" \
  --config configs/configs.yaml \
  --http.addr="127.0.0.1:${http_port}" \
  --jwt.secret="smoke-test-jwt-secret-with-at-least-32-characters" \
  --postgresql.addr="127.0.0.1:${postgres_port}" \
  --postgresql.database="enterprise_template_test" \
  --postgresql.password="integration-postgres" \
  --redis.addr="127.0.0.1:${redis_port}" \
  --redis.password="integration-redis" \
  --kafka.brokers="127.0.0.1:${kafka_port}" \
  --otel.output-mode=classic &
server_pid=$!
trap 'kill "${server_pid}" 2>/dev/null || true; wait "${server_pid}" 2>/dev/null || true' EXIT INT TERM

attempt=0
until curl --fail --silent --show-error "http://127.0.0.1:${http_port}/healthz" >/dev/null && \
      curl --fail --silent --show-error "http://127.0.0.1:${http_port}/readyz" >/dev/null; do
  attempt=$((attempt + 1))
  if [ "${attempt}" -ge 30 ]; then
    echo "API smoke test did not become ready" >&2
    exit 1
  fi
  sleep 1
done
