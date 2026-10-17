-- +goose Up
CREATE TABLE role (id bigserial PRIMARY KEY, role_id varchar(64) UNIQUE NOT NULL, name varchar(80) UNIQUE NOT NULL, builtin boolean NOT NULL DEFAULT false);
CREATE TABLE permission (code varchar(80) PRIMARY KEY);
CREATE TABLE role_permission (role_id varchar(64) NOT NULL REFERENCES role(role_id) ON DELETE CASCADE, permission_code varchar(80) NOT NULL REFERENCES permission(code), PRIMARY KEY(role_id,permission_code));
CREATE TABLE user_role (user_id varchar(32) NOT NULL REFERENCES "user"(user_id) ON DELETE CASCADE, role_id varchar(64) NOT NULL REFERENCES role(role_id) ON DELETE CASCADE, PRIMARY KEY(user_id,role_id));
INSERT INTO role(role_id,name,builtin) VALUES('role_user','user',true),('role_admin','admin',true);
INSERT INTO permission(code) SELECT unnest(ARRAY['task.submit','task.view','task.cancel','task.retry','result.download','schedule.manage','task.view.any','task.cancel.any','task.retry.any','result.download.any','schedule.manage.any','user.manage','rbac.manage','audit.view','dlq.view','dlq.replay']);
INSERT INTO role_permission SELECT 'role_admin',code FROM permission;
INSERT INTO role_permission SELECT 'role_user',code FROM permission WHERE code IN ('task.submit','task.view','task.cancel','task.retry','result.download','schedule.manage');
INSERT INTO user_role SELECT user_id,'role_user' FROM "user";
-- 删除账号不应消除审计证据。
ALTER TABLE audit_log DROP CONSTRAINT fk_audit_log_user_id;
ALTER TABLE audit_log ADD COLUMN outcome varchar(24) NOT NULL DEFAULT 'success';
ALTER TABLE audit_log ADD COLUMN request_id varchar(128);
CREATE TABLE schedule (
 id bigserial PRIMARY KEY, schedule_id varchar(64) NOT NULL UNIQUE, name varchar(80) NOT NULL, task_type varchar(80) NOT NULL,
 params jsonb NOT NULL, params_version integer NOT NULL CHECK(params_version>0), cron varchar(120) NOT NULL, timezone varchar(80) NOT NULL,
 enabled boolean NOT NULL DEFAULT true, archived boolean NOT NULL DEFAULT false, next_run_at timestamptz NOT NULL,
 misfire_policy varchar(16) NOT NULL CHECK(misfire_policy IN ('skip','catch_up')), concurrency_policy varchar(16) NOT NULL CHECK(concurrency_policy IN ('allow','forbid')),
 owner_id varchar(32) NOT NULL, version bigint NOT NULL DEFAULT 1, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_schedule_due ON schedule(next_run_at,id) WHERE enabled AND NOT archived;
CREATE INDEX idx_schedule_owner ON schedule(owner_id,id);
CREATE TABLE task (
 id bigserial PRIMARY KEY, task_id varchar(64) NOT NULL UNIQUE, task_type varchar(80) NOT NULL, params jsonb NOT NULL, params_version integer NOT NULL,
 source varchar(16) NOT NULL CHECK(source IN ('api','schedule','retry')), schedule_id varchar(64) REFERENCES schedule(schedule_id), schedule_version bigint,
 scheduled_at timestamptz, retry_of varchar(64) REFERENCES task(task_id), owner_id varchar(32) NOT NULL,
 status varchar(16) NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','running','retry_wait','succeeded','failed','canceled')),
 priority integer NOT NULL DEFAULT 0, available_at timestamptz NOT NULL DEFAULT now(), timeout_seconds integer NOT NULL DEFAULT 300 CHECK(timeout_seconds BETWEEN 1 AND 3600),
 max_attempts integer NOT NULL DEFAULT 3 CHECK(max_attempts BETWEEN 1 AND 10), attempts integer NOT NULL DEFAULT 0 CHECK(attempts>=0),
 fencing_token bigint NOT NULL DEFAULT 0, worker_id varchar(128), lease_until timestamptz, cancel_requested_at timestamptz,
 idempotency_key varchar(128), request_hash varchar(64) NOT NULL, result_id varchar(64), error_code varchar(80), error_summary varchar(256),
 trace_id varchar(32), traceparent varchar(128), request_id varchar(128), started_at timestamptz, finished_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX uq_task_idempotency ON task(owner_id,task_type,idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE UNIQUE INDEX uq_task_schedule_slot ON task(schedule_id,scheduled_at) WHERE source='schedule';
CREATE INDEX idx_task_claim ON task(priority DESC,available_at,id) WHERE status IN ('pending','retry_wait','running');
CREATE INDEX idx_task_owner ON task(owner_id,id DESC);
CREATE INDEX idx_task_schedule_active ON task(schedule_id,status) WHERE status IN ('pending','running','retry_wait');
CREATE TABLE task_attempt (
 id bigserial PRIMARY KEY, task_id varchar(64) NOT NULL REFERENCES task(task_id) ON DELETE CASCADE, attempt integer NOT NULL,
 worker_id varchar(128) NOT NULL, fencing_token bigint NOT NULL, lease_until timestamptz NOT NULL, heartbeat_at timestamptz NOT NULL,
 started_at timestamptz NOT NULL, finished_at timestamptz, outcome varchar(32), error_code varchar(80), trace_id varchar(32), UNIQUE(task_id,attempt)
);
CREATE TABLE schedule_trigger (
 id bigserial PRIMARY KEY, schedule_id varchar(64) NOT NULL REFERENCES schedule(schedule_id), scheduled_at timestamptz NOT NULL,
 schedule_version bigint NOT NULL, outcome varchar(24) NOT NULL, task_id varchar(64), summary varchar(256), created_at timestamptz NOT NULL DEFAULT now(), UNIQUE(schedule_id,scheduled_at)
);
CREATE TABLE report_result (
 id bigserial PRIMARY KEY, result_id varchar(64) UNIQUE NOT NULL, task_id varchar(64) UNIQUE NOT NULL REFERENCES task(task_id), owner_id varchar(32) NOT NULL,
 content bytea NOT NULL CHECK(octet_length(content)<=10485760), content_type varchar(80) NOT NULL, expires_at timestamptz NOT NULL, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE report_notification (task_id varchar(64) PRIMARY KEY, owner_id varchar(32) NOT NULL, result_id varchar(64) NOT NULL, created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE event_replay (replay_id varchar(64) PRIMARY KEY, dead_letter_id varchar(64) NOT NULL REFERENCES dead_letter_event(dead_letter_id), event_id varchar(64) UNIQUE NOT NULL, actor_id varchar(32) NOT NULL, idempotency_key varchar(128) NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), UNIQUE(actor_id,idempotency_key));
CREATE TABLE runtime_heartbeat (instance_id varchar(128) PRIMARY KEY, role varchar(24) NOT NULL, updated_at timestamptz NOT NULL DEFAULT now());
COMMENT ON TABLE task IS '持久化任务；至少一次执行，终态不重置';
COMMENT ON TABLE task_attempt IS '不可覆盖的任务执行历史；token用于拒绝过期执行者';
COMMENT ON TABLE schedule_trigger IS '计划时点去重及跳过历史';
COMMENT ON TABLE report_result IS '共享数据库报表存储；严格白名单，最大10MiB';
COMMENT ON TABLE event_replay IS '受控死信重放意图及原事件关联';

CREATE INDEX idx_report_result_expiry ON report_result(expires_at,id);
CREATE INDEX idx_task_retention ON task(finished_at,id) WHERE status IN ('succeeded','failed','canceled');
COMMENT ON TABLE schedule IS 'Cron 计划；锁行后原子推进 UTC 游标与创建任务';
COMMENT ON TABLE role IS '内置及自定义 RBAC 角色，内置角色不可修改';
COMMENT ON TABLE permission IS '迁移维护的权限目录，默认拒绝';
COMMENT ON TABLE role_permission IS '角色权限；每次访问实时查询，不信任 JWT 角色缓存';
COMMENT ON TABLE user_role IS '用户角色关系；管理员操作由事务级锁协调';
COMMENT ON TABLE report_notification IS '按 task_id 幂等的报表完成通知';
COMMENT ON TABLE runtime_heartbeat IS '运行实例心跳；用于可用实例数量，非执行租约';
COMMENT ON COLUMN task.fencing_token IS '每次领取递增；提交必须匹配 token、Worker 和未过期租约';
COMMENT ON COLUMN task.idempotency_key IS 'owner/type 作用域内重试安全键，保留期内参数哈希必须一致';
COMMENT ON COLUMN task.traceparent IS 'W3C 上游 TraceContext，仅用于遥测关联';
COMMENT ON COLUMN schedule.next_run_at IS '下一计划时点 UTC；时区转换由固定 Cron 版本计算';
COMMENT ON COLUMN schedule.version IS '乐观配置版本，触发时记录该版本快照';
COMMENT ON COLUMN audit_log.outcome IS '可靠管理审计结果，不使用可采样日志替代';
-- 账号删除或停用不能使受控管理员入口失效，且与角色变更使用同一事务锁。
-- +goose StatementBegin
CREATE FUNCTION protect_last_administrator() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='UPDATE' AND NEW.status=0 THEN RETURN NEW; END IF;
 PERFORM pg_advisory_xact_lock(21120261015);
 IF EXISTS(SELECT 1 FROM user_role WHERE user_id=OLD.user_id AND role_id='role_admin') AND
    NOT EXISTS(SELECT 1 FROM user_role ur JOIN "user" u ON u.user_id=ur.user_id WHERE ur.role_id='role_admin' AND u.status=0 AND u.user_id<>OLD.user_id) THEN
   RAISE EXCEPTION 'last administrator must remain active' USING ERRCODE='23514';
 END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER protect_last_administrator BEFORE DELETE OR UPDATE OF status ON "user" FOR EACH ROW EXECUTE FUNCTION protect_last_administrator();
