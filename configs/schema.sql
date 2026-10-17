-- 参考快照；运行数据库使用官方 goose -dir migrations up。
BEGIN;


-- 本文件只负责在空数据库中创建 schema，不执行任何破坏性清理。
-- 如需重建本地开发数据库，必须显式执行 configs/cleanup.sql 后再执行本文件。

-- 创建内部ID序列
CREATE SEQUENCE "public"."user_id_seq" INCREMENT 1 MINVALUE 1 MAXVALUE 9223372036854775807 START 1 CACHE 1;
COMMENT ON SEQUENCE "public"."user_id_seq" IS '用户表内部ID序列';
CREATE SEQUENCE "public"."user_login_log_id_seq" INCREMENT 1 MINVALUE 1 MAXVALUE 9223372036854775807 START 1 CACHE 1;
COMMENT ON SEQUENCE "public"."user_login_log_id_seq" IS '用户登录日志表内部ID序列';
CREATE SEQUENCE "public"."audit_log_id_seq" INCREMENT 1 MINVALUE 1 MAXVALUE 9223372036854775807 START 1 CACHE 1;
COMMENT ON SEQUENCE "public"."audit_log_id_seq" IS '审计日志表内部ID序列';
CREATE SEQUENCE "public"."outbox_event_id_seq" INCREMENT 1 MINVALUE 1 MAXVALUE 9223372036854775807 START 1 CACHE 1;
COMMENT ON SEQUENCE "public"."outbox_event_id_seq" IS '事务性Outbox事件表内部ID序列';
CREATE SEQUENCE "public"."inbox_event_id_seq" INCREMENT 1 MINVALUE 1 MAXVALUE 9223372036854775807 START 1 CACHE 1;
COMMENT ON SEQUENCE "public"."inbox_event_id_seq" IS '幂等Inbox事件表内部ID序列';
CREATE SEQUENCE "public"."dead_letter_event_id_seq" INCREMENT 1 MINVALUE 1 MAXVALUE 9223372036854775807 START 1 CACHE 1;
COMMENT ON SEQUENCE "public"."dead_letter_event_id_seq" IS '消费死信事件表内部ID序列';

-- 用户表
CREATE TABLE "public"."user" (
    "id" bigint NOT NULL DEFAULT nextval('"public"."user_id_seq"'::regclass),
    "user_id" varchar(32) NOT NULL,
    "username" varchar(255) COLLATE "pg_catalog"."default" NOT NULL,
    "password" varchar(255) COLLATE "pg_catalog"."default" NOT NULL,
    "token_version" bigint NOT NULL DEFAULT 0,
    "email" varchar(255) COLLATE "pg_catalog"."default",
    "phone" varchar(32) COLLATE "pg_catalog"."default",
    "avatar" text COLLATE "pg_catalog"."default",
    "nickname" varchar(255) COLLATE "pg_catalog"."default" NOT NULL,
    "gender" smallint NOT NULL DEFAULT 0,
    "status" smallint NOT NULL DEFAULT 0,
    "last_login_at" timestamptz(6),
    "created_at" timestamptz(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "description" text COLLATE "pg_catalog"."default",
    CONSTRAINT "user_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "user_user_id_key" UNIQUE ("user_id"),
    CONSTRAINT "user_username_key" UNIQUE ("username"),
    CONSTRAINT "user_email_key" UNIQUE ("email"),
    CONSTRAINT "user_phone_key" UNIQUE ("phone"),
    CONSTRAINT "chk_user_gender" CHECK ("gender" IN (0, 1, 2)),
    CONSTRAINT "chk_user_status" CHECK ("status" IN (0, 1))
);

-- 用户登录日志
CREATE TABLE "public"."user_login_log" (
    "id" bigint NOT NULL DEFAULT nextval('"public"."user_login_log_id_seq"'::regclass),
    "username" varchar(255) COLLATE "pg_catalog"."default",
    "ip_address" varchar(64) COLLATE "pg_catalog"."default",
    "user_agent" text COLLATE "pg_catalog"."default",
    "status" boolean NOT NULL,
    "error_message" text COLLATE "pg_catalog"."default",
    "created_at" timestamptz(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "user_login_log_pkey" PRIMARY KEY ("id")
);

-- 审计日志
CREATE TABLE "public"."audit_log" (
    "id" bigint NOT NULL DEFAULT nextval('"public"."audit_log_id_seq"'::regclass),
    "user_id" varchar(32) NOT NULL,
    "action" varchar(255) COLLATE "pg_catalog"."default" NOT NULL,
    "resource" varchar(255) COLLATE "pg_catalog"."default",
    "details" text COLLATE "pg_catalog"."default",
    "created_at" timestamptz(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "audit_log_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "fk_audit_log_user_id" FOREIGN KEY ("user_id") REFERENCES "public"."user" ("user_id") ON DELETE CASCADE ON UPDATE NO ACTION
);

-- 事务性 Outbox 事件
CREATE TABLE "public"."outbox_event" (
    "id" bigint NOT NULL DEFAULT nextval('"public"."outbox_event_id_seq"'::regclass),
    "event_id" varchar(64) NOT NULL,
    "event_type" varchar(128) NOT NULL,
    "aggregate_type" varchar(64) NOT NULL,
    "aggregate_id" varchar(64) NOT NULL,
    "payload" jsonb NOT NULL,
    "trace_id" varchar(32),
    "status" varchar(16) NOT NULL DEFAULT 'pending',
    "attempts" integer NOT NULL DEFAULT 0,
    "available_at" timestamptz(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "published_at" timestamptz(6),
    "last_error" text,
    "created_at" timestamptz(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "outbox_event_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "outbox_event_event_id_key" UNIQUE ("event_id"),
    CONSTRAINT "chk_outbox_event_status" CHECK ("status" IN ('pending', 'processing', 'published', 'failed', 'dead'))
);

-- 幂等 Inbox 事件
CREATE TABLE "public"."inbox_event" (
    "id" bigint NOT NULL DEFAULT nextval('"public"."inbox_event_id_seq"'::regclass),
    "consumer_name" varchar(128) NOT NULL,
    "event_id" varchar(64) NOT NULL,
    "event_type" varchar(128) NOT NULL,
    "source_topic" varchar(249) NOT NULL,
    "source_partition" integer NOT NULL,
    "source_offset" bigint NOT NULL,
    "payload_hash" bytea NOT NULL,
    "status" varchar(16) NOT NULL DEFAULT 'processing',
    "attempts" integer NOT NULL DEFAULT 1,
    "available_at" timestamptz(6) NOT NULL,
    "processed_at" timestamptz(6),
    "last_error" text,
    "created_at" timestamptz(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "inbox_event_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "inbox_event_consumer_event_key" UNIQUE ("consumer_name", "event_id"),
    CONSTRAINT "inbox_event_source_position_key" UNIQUE ("consumer_name", "source_topic", "source_partition", "source_offset"),
    CONSTRAINT "chk_inbox_event_status" CHECK ("status" IN ('processing', 'failed', 'completed', 'dead_lettered')),
    CONSTRAINT "chk_inbox_event_source_partition" CHECK ("source_partition" >= 0),
    CONSTRAINT "chk_inbox_event_source_offset" CHECK ("source_offset" >= 0),
    CONSTRAINT "chk_inbox_event_attempts" CHECK ("attempts" >= 1),
    CONSTRAINT "chk_inbox_event_payload_hash" CHECK (octet_length("payload_hash") = 32)
);

-- 消费死信事件
CREATE TABLE "public"."dead_letter_event" (
    "id" bigint NOT NULL DEFAULT nextval('"public"."dead_letter_event_id_seq"'::regclass),
    "dead_letter_id" varchar(64) NOT NULL,
    "consumer_name" varchar(128) NOT NULL,
    "original_event_id" varchar(64),
    "original_event_type" varchar(128),
    "source_topic" varchar(249) NOT NULL,
    "source_partition" integer NOT NULL,
    "source_offset" bigint NOT NULL,
    "message_key" bytea,
    "headers" jsonb NOT NULL,
    "payload" bytea NOT NULL,
    "trace_id" varchar(32),
    "failure_kind" varchar(32) NOT NULL,
    "consume_attempts" integer NOT NULL,
    "last_error" text NOT NULL,
    "dlq_topic" varchar(249) NOT NULL,
    "status" varchar(16) NOT NULL DEFAULT 'pending',
    "publish_attempts" integer NOT NULL DEFAULT 0,
    "available_at" timestamptz(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "published_at" timestamptz(6),
    "created_at" timestamptz(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "dead_letter_event_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "dead_letter_event_dead_letter_id_key" UNIQUE ("dead_letter_id"),
    CONSTRAINT "dead_letter_event_source_position_key" UNIQUE ("consumer_name", "source_topic", "source_partition", "source_offset"),
    CONSTRAINT "chk_dead_letter_event_status" CHECK ("status" IN ('pending', 'processing', 'published', 'failed', 'dead')),
    CONSTRAINT "chk_dead_letter_event_failure_kind" CHECK ("failure_kind" IN ('malformed', 'unsupported', 'permanent', 'retry_exhausted', 'identity_conflict')),
    CONSTRAINT "chk_dead_letter_event_source_partition" CHECK ("source_partition" >= 0),
    CONSTRAINT "chk_dead_letter_event_source_offset" CHECK ("source_offset" >= 0),
    CONSTRAINT "chk_dead_letter_event_consume_attempts" CHECK ("consume_attempts" >= 1),
    CONSTRAINT "chk_dead_letter_event_publish_attempts" CHECK ("publish_attempts" >= 0)
);

-- 索引
CREATE INDEX "idx_user_login_log_username" ON "public"."user_login_log" ("username");
CREATE INDEX "idx_user_login_log_created_at" ON "public"."user_login_log" ("created_at" DESC);
CREATE INDEX "idx_audit_log_user_id" ON "public"."audit_log" ("user_id");
CREATE INDEX "idx_audit_log_action" ON "public"."audit_log" ("action");
CREATE INDEX "idx_audit_log_created_at" ON "public"."audit_log" ("created_at" DESC);
CREATE INDEX "idx_outbox_event_relay" ON "public"."outbox_event" ("status", "available_at", "id") WHERE "status" IN ('pending', 'processing', 'failed');
CREATE INDEX "idx_outbox_event_aggregate" ON "public"."outbox_event" ("aggregate_type", "aggregate_id", "id");
CREATE INDEX "idx_inbox_event_retry" ON "public"."inbox_event" ("status", "available_at", "id") WHERE "status" IN ('processing', 'failed');
CREATE INDEX "idx_inbox_event_created_at" ON "public"."inbox_event" ("created_at" DESC);
CREATE INDEX "idx_dead_letter_event_relay" ON "public"."dead_letter_event" ("status", "available_at", "id") WHERE "status" IN ('pending', 'processing', 'failed');
CREATE INDEX "idx_dead_letter_event_original" ON "public"."dead_letter_event" ("consumer_name", "original_event_id", "id");

-- 用户表的字段说明
COMMENT ON TABLE "public"."user" IS '用户表';
COMMENT ON COLUMN "public"."user"."id" IS '内部主键ID（自增序列）';
COMMENT ON COLUMN "public"."user"."user_id" IS '用户业务唯一RID';
COMMENT ON COLUMN "public"."user"."username" IS '用户名（唯一，登录用）';
COMMENT ON COLUMN "public"."user"."password" IS '密码哈希（bcrypt加密存储）';
COMMENT ON COLUMN "public"."user"."token_version" IS '令牌版本；递增后使此前签发的令牌失效';
COMMENT ON COLUMN "public"."user"."email" IS '电子邮箱（唯一）';
COMMENT ON COLUMN "public"."user"."phone" IS '手机号（唯一）';
COMMENT ON COLUMN "public"."user"."avatar" IS '头像URL';
COMMENT ON COLUMN "public"."user"."nickname" IS '用户昵称';
COMMENT ON COLUMN "public"."user"."gender" IS '性别（0=未知,1=男,2=女，默认0）';
COMMENT ON COLUMN "public"."user"."status" IS '用户状态（0=活跃,1=禁用，默认0）';
COMMENT ON COLUMN "public"."user"."last_login_at" IS '最后登录时间';
COMMENT ON COLUMN "public"."user"."created_at" IS '创建时间';
COMMENT ON COLUMN "public"."user"."updated_at" IS '更新时间';
COMMENT ON COLUMN "public"."user"."description" IS '用户描述/简介';

-- 登录日志表的字段说明
COMMENT ON TABLE "public"."user_login_log" IS '用户登录日志表：记录登录成功与失败尝试';
COMMENT ON COLUMN "public"."user_login_log"."id" IS '内部主键ID（自增序列）';
COMMENT ON COLUMN "public"."user_login_log"."username" IS '登录用户名';
COMMENT ON COLUMN "public"."user_login_log"."ip_address" IS '登录IP地址';
COMMENT ON COLUMN "public"."user_login_log"."user_agent" IS '用户代理字符串';
COMMENT ON COLUMN "public"."user_login_log"."status" IS '登录状态（true=成功, false=失败）';
COMMENT ON COLUMN "public"."user_login_log"."error_message" IS '错误消息（失败时）';
COMMENT ON COLUMN "public"."user_login_log"."created_at" IS '登录尝试时间';

-- 日志审计表的字段说明
COMMENT ON TABLE "public"."audit_log" IS '审计日志表：记录用户对系统资源的操作审计';
COMMENT ON COLUMN "public"."audit_log"."id" IS '内部主键ID（自增序列）';
COMMENT ON COLUMN "public"."audit_log"."user_id" IS '操作用户RID，引用user(user_id)';
COMMENT ON COLUMN "public"."audit_log"."action" IS '操作类型（如role_assign、permission_deny）';
COMMENT ON COLUMN "public"."audit_log"."resource" IS '操作的资源';
COMMENT ON COLUMN "public"."audit_log"."details" IS '操作详情（JSON字符串格式，记录变更前后数据）';
COMMENT ON COLUMN "public"."audit_log"."created_at" IS '操作时间';

COMMENT ON TABLE "public"."outbox_event" IS '事务性Outbox事件表：与业务数据同事务写入并由Relay可靠发布';
COMMENT ON COLUMN "public"."outbox_event"."event_id" IS '事件全局唯一标识';
COMMENT ON COLUMN "public"."outbox_event"."event_type" IS '带版本的事件类型';
COMMENT ON COLUMN "public"."outbox_event"."aggregate_type" IS '业务聚合类型';
COMMENT ON COLUMN "public"."outbox_event"."aggregate_id" IS '业务聚合标识';
COMMENT ON COLUMN "public"."outbox_event"."payload" IS '完整事件envelope';
COMMENT ON COLUMN "public"."outbox_event"."trace_id" IS '产生事件时的追踪标识';
COMMENT ON COLUMN "public"."outbox_event"."status" IS 'pending、processing、published、failed或dead；dead表示达到发布重试上限';
COMMENT ON COLUMN "public"."outbox_event"."attempts" IS '已领取发布次数';
COMMENT ON COLUMN "public"."outbox_event"."available_at" IS '事件可被Relay领取的时间或租约过期时间';
COMMENT ON COLUMN "public"."outbox_event"."published_at" IS '成功发布时间';
COMMENT ON COLUMN "public"."outbox_event"."last_error" IS '最近一次发布错误摘要';

COMMENT ON TABLE "public"."inbox_event" IS '幂等Inbox事件表：按逻辑消费者隔离事件处理状态和租约';
COMMENT ON COLUMN "public"."inbox_event"."consumer_name" IS '稳定逻辑消费者名称，通常为Kafka consumer group';
COMMENT ON COLUMN "public"."inbox_event"."event_id" IS '事件全局唯一标识，在consumer_name范围内唯一';
COMMENT ON COLUMN "public"."inbox_event"."event_type" IS '带版本的事件类型';
COMMENT ON COLUMN "public"."inbox_event"."source_topic" IS '来源Kafka topic';
COMMENT ON COLUMN "public"."inbox_event"."source_partition" IS '来源Kafka partition';
COMMENT ON COLUMN "public"."inbox_event"."source_offset" IS '来源Kafka offset';
COMMENT ON COLUMN "public"."inbox_event"."payload_hash" IS '完整消息载荷SHA-256，用于检测event_id冲突';
COMMENT ON COLUMN "public"."inbox_event"."status" IS 'processing、failed、completed或dead_lettered';
COMMENT ON COLUMN "public"."inbox_event"."attempts" IS '已领取处理次数，同时作为fencing token';
COMMENT ON COLUMN "public"."inbox_event"."available_at" IS '处理租约过期时间或下次可重试时间';
COMMENT ON COLUMN "public"."inbox_event"."processed_at" IS '成功处理或可靠写入DLQ的时间';
COMMENT ON COLUMN "public"."inbox_event"."last_error" IS '最近一次处理错误摘要';

COMMENT ON TABLE "public"."dead_letter_event" IS '消费死信持久队列表：先与Inbox终态原子落库，再由Relay发布Kafka DLQ';
COMMENT ON COLUMN "public"."dead_letter_event"."dead_letter_id" IS '稳定死信唯一标识，由消费者和来源位置生成';
COMMENT ON COLUMN "public"."dead_letter_event"."consumer_name" IS '产生死信的逻辑消费者名称';
COMMENT ON COLUMN "public"."dead_letter_event"."original_event_id" IS '原事件ID，畸形消息可为空';
COMMENT ON COLUMN "public"."dead_letter_event"."original_event_type" IS '原事件类型，畸形消息可为空';
COMMENT ON COLUMN "public"."dead_letter_event"."source_topic" IS '原消息Kafka topic';
COMMENT ON COLUMN "public"."dead_letter_event"."source_partition" IS '原消息Kafka partition';
COMMENT ON COLUMN "public"."dead_letter_event"."source_offset" IS '原消息Kafka offset';
COMMENT ON COLUMN "public"."dead_letter_event"."message_key" IS '原始Kafka message key';
COMMENT ON COLUMN "public"."dead_letter_event"."headers" IS '原始Kafka headers JSON数组，保留顺序和重复键';
COMMENT ON COLUMN "public"."dead_letter_event"."payload" IS '原始消息字节，可包含畸形JSON';
COMMENT ON COLUMN "public"."dead_letter_event"."trace_id" IS '原消息追踪标识';
COMMENT ON COLUMN "public"."dead_letter_event"."failure_kind" IS 'malformed、unsupported、permanent、retry_exhausted或identity_conflict';
COMMENT ON COLUMN "public"."dead_letter_event"."consume_attempts" IS '原消息处理尝试次数';
COMMENT ON COLUMN "public"."dead_letter_event"."last_error" IS '最终处理错误摘要';
COMMENT ON COLUMN "public"."dead_letter_event"."dlq_topic" IS '目标Kafka DLQ topic';
COMMENT ON COLUMN "public"."dead_letter_event"."status" IS 'pending、processing、published、failed或dead；dead表示达到发布重试上限';
COMMENT ON COLUMN "public"."dead_letter_event"."publish_attempts" IS 'DLQ发布领取次数，同时作为fencing token';

-- 设置 id
ALTER SEQUENCE "public"."user_id_seq" OWNED BY "public"."user"."id";
SELECT setval('"public"."user_id_seq"', 1, false);
ALTER SEQUENCE "public"."user_login_log_id_seq" OWNED BY "public"."user_login_log"."id";
SELECT setval('"public"."user_login_log_id_seq"', 1, false);
ALTER SEQUENCE "public"."audit_log_id_seq" OWNED BY "public"."audit_log"."id";
SELECT setval('"public"."audit_log_id_seq"', 1, false);
ALTER SEQUENCE "public"."outbox_event_id_seq" OWNED BY "public"."outbox_event"."id";
SELECT setval('"public"."outbox_event_id_seq"', 1, false);
ALTER SEQUENCE "public"."inbox_event_id_seq" OWNED BY "public"."inbox_event"."id";
SELECT setval('"public"."inbox_event_id_seq"', 1, false);
ALTER SEQUENCE "public"."dead_letter_event_id_seq" OWNED BY "public"."dead_letter_event"."id";
SELECT setval('"public"."dead_letter_event_id_seq"', 1, false);

-- 不创建固定管理员。首个账号必须通过受控 bootstrap 流程或部署脚本注入，
-- 禁止在可复用模板中分发已知用户名和密码哈希。
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
CREATE TRIGGER protect_last_administrator BEFORE DELETE OR UPDATE OF status ON "user" FOR EACH ROW EXECUTE FUNCTION protect_last_administrator();

COMMIT;
