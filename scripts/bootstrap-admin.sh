#!/usr/bin/env bash
# 为已有活跃账号首次授予管理员角色；使用 libpq 标准 PG* 环境变量或 .pgpass。
set -euo pipefail

usage() {
  printf '用法: %s <existing-user-id>\n' "${0##*/}"
  printf '先执行 goose up；连接配置使用 PGHOST、PGPORT、PGDATABASE、PGUSER 和 PGPASSFILE/PGPASSWORD。\n'
}

if [[ $# -eq 1 && ${1:-} == --help ]]; then
  usage
  exit 0
fi
if [[ $# -ne 1 || -z ${1:-} ]]; then
  usage >&2
  exit 2
fi
if ! command -v psql >/dev/null 2>&1; then
  printf '需要 PostgreSQL 客户端 psql，请先安装并加入 PATH。\n' >&2
  exit 1
fi

bootstrap_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
expected_versions=0
for migration in "$bootstrap_dir"/../migrations/[0-9]*_*.sql; do
  if [[ ! -f $migration ]]; then
    printf '未找到版本化 SQL 迁移文件。\n' >&2
    exit 1
  fi
  migration_name=${migration##*/}
  migration_version=${migration_name%%_*}
  expected_versions+=",$((10#$migration_version))"
done

# psql 的 SQL 字面量引用在 DO 块外完成，账号内容不会作为 SQL 执行。
psql --no-psqlrc --no-password --quiet --set=ON_ERROR_STOP=1 \
  --set=bootstrap_user_id="$1" --set=expected_versions="$expected_versions" <<'SQL'
BEGIN;
SET LOCAL lock_timeout = '10s';
SET LOCAL statement_timeout = '30s';
SET LOCAL search_path = public, pg_temp;

CREATE TEMP TABLE bootstrap_admin_input ON COMMIT DROP AS
SELECT :'bootstrap_user_id'::text AS user_id,
       string_to_array(:'expected_versions', ',')::bigint[] AS versions;

DO $bootstrap$
DECLARE
  target_user text;
  expected bigint[];
  applied bigint[];
BEGIN
  -- 与在线角色管理及最后管理员保护共用同一事务锁。
  PERFORM pg_advisory_xact_lock(21120261015);
  SELECT user_id, versions INTO target_user, expected
    FROM pg_temp.bootstrap_admin_input;
  SELECT array_agg(version ORDER BY version) INTO expected
    FROM unnest(expected) AS required(version);
  SELECT array_agg(version_id ORDER BY version_id) INTO applied
    FROM goose_db_version
    WHERE id IN (SELECT max(id) FROM goose_db_version GROUP BY version_id)
      AND is_applied;
  IF applied IS DISTINCT FROM expected THEN
    RAISE EXCEPTION 'database migration version incompatible; run goose up first';
  END IF;
  IF EXISTS (SELECT 1 FROM user_role WHERE role_id = 'role_admin') THEN
    RAISE EXCEPTION 'administrator already initialized; use audited RBAC API';
  END IF;
  PERFORM 1 FROM "user" WHERE user_id = target_user AND status = 0 FOR UPDATE;
  IF NOT FOUND THEN
    RAISE EXCEPTION 'bootstrap account must exist and be active';
  END IF;

  INSERT INTO user_role(user_id, role_id) VALUES (target_user, 'role_admin');
  INSERT INTO audit_log(user_id, action, resource, details, outcome)
    VALUES (target_user, 'rbac.bootstrap', target_user, '{}', 'success');
END
$bootstrap$;
COMMIT;
SQL

printf '首次管理员授权完成，已记录审计。\n'
