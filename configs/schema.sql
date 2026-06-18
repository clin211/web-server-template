BEGIN;
-- 清理已存在的表和ID序列
DROP TABLE IF EXISTS "public"."user_login_log";
DROP TABLE IF EXISTS "public"."audit_log";
DROP TABLE IF EXISTS "public"."user";
DROP SEQUENCE IF EXISTS "public"."user_login_log_id_seq";
DROP SEQUENCE IF EXISTS "public"."user_id_seq";
DROP SEQUENCE IF EXISTS "public"."audit_log_id_seq";

-- 创建内部ID序列
CREATE SEQUENCE "public"."user_id_seq" INCREMENT 1 MINVALUE 1 MAXVALUE 9223372036854775807 START 1 CACHE 1;
ALTER SEQUENCE "public"."user_id_seq" OWNER TO "postgres";
COMMENT ON SEQUENCE "public"."user_id_seq" IS '用户表内部ID序列';
CREATE SEQUENCE "public"."user_login_log_id_seq" INCREMENT 1 MINVALUE 1 MAXVALUE 9223372036854775807 START 1 CACHE 1;
ALTER SEQUENCE "public"."user_login_log_id_seq" OWNER TO "postgres";
COMMENT ON SEQUENCE "public"."user_login_log_id_seq" IS '用户登录日志表内部ID序列';
CREATE SEQUENCE "public"."audit_log_id_seq" INCREMENT 1 MINVALUE 1 MAXVALUE 9223372036854775807 START 1 CACHE 1;
ALTER SEQUENCE "public"."audit_log_id_seq" OWNER TO "postgres";
COMMENT ON SEQUENCE "public"."audit_log_id_seq" IS '审计日志表内部ID序列';

-- 用户表
CREATE TABLE "public"."user" (
    "id" bigint NOT NULL DEFAULT nextval('"public"."user_id_seq"'::regclass),
    "user_id" varchar(32) NOT NULL,
    "username" varchar(255) COLLATE "pg_catalog"."default" NOT NULL,
    "password" varchar(255) COLLATE "pg_catalog"."default" NOT NULL,
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
ALTER TABLE "public"."user" OWNER TO "postgres";

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
ALTER TABLE "public"."user_login_log" OWNER TO "postgres";

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
ALTER TABLE "public"."audit_log" OWNER TO "postgres";

-- 索引
CREATE INDEX "idx_user_login_log_username" ON "public"."user_login_log" ("username");
CREATE INDEX "idx_user_login_log_created_at" ON "public"."user_login_log" ("created_at" DESC);
CREATE INDEX "idx_audit_log_user_id" ON "public"."audit_log" ("user_id");
CREATE INDEX "idx_audit_log_action" ON "public"."audit_log" ("action");
CREATE INDEX "idx_audit_log_created_at" ON "public"."audit_log" ("created_at" DESC);

-- 用户表的字段说明
COMMENT ON TABLE "public"."user" IS '用户表';
COMMENT ON COLUMN "public"."user"."id" IS '内部主键ID（自增序列）';
COMMENT ON COLUMN "public"."user"."user_id" IS '用户业务唯一RID';
COMMENT ON COLUMN "public"."user"."username" IS '用户名（唯一，登录用）';
COMMENT ON COLUMN "public"."user"."password" IS '密码哈希（bcrypt加密存储）';
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

-- 设置 id
ALTER SEQUENCE "public"."user_id_seq" OWNED BY "public"."user"."id";
SELECT setval('"public"."user_id_seq"', 1, false);
ALTER SEQUENCE "public"."user_login_log_id_seq" OWNED BY "public"."user_login_log"."id";
SELECT setval('"public"."user_login_log_id_seq"', 1, false);
ALTER SEQUENCE "public"."audit_log_id_seq" OWNED BY "public"."audit_log"."id";
SELECT setval('"public"."audit_log_id_seq"', 1, false);

-- 插入数据
INSERT INTO "public"."user" (
        "username",
        "password",
        "user_id",
        "nickname",
        "gender",
        "status",
        "description"
    )
VALUES (
        'admin',
        '$2a$10$IhKAl47y0wMFlBgYRHVy2uRLBxcuPgOged2Qrk4EXFkzEFztOVLZ2',
        'user-admin',
        -- 123456abcX
        '系统管理员',
        0,
        0,
        '系统初始化默认管理员账号（首次登录后请立即修改密码）'
    );
COMMIT;
