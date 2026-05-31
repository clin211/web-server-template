-- 清理脚本：删除所有表、字段（类型/序列）、索引等对象
-- 执行顺序很重要：先删除外键约束，再删除索引/唯一约束，再删除表，最后删除序列和类型

BEGIN;

-- ----------------------------
-- 1. 删除外键约束（通过删除表自动清理，但显式处理以确保顺序）
-- ----------------------------
ALTER TABLE IF EXISTS "public"."menu" DROP CONSTRAINT IF EXISTS "menu_parent_id_fkey";
ALTER TABLE IF EXISTS "public"."menu" DROP CONSTRAINT IF EXISTS "menu_permission_id_fkey";
ALTER TABLE IF EXISTS "public"."role_permission" DROP CONSTRAINT IF EXISTS "role_permission_permission_id_fkey";
ALTER TABLE IF EXISTS "public"."role_permission" DROP CONSTRAINT IF EXISTS "role_permission_role_id_fkey";
ALTER TABLE IF EXISTS "public"."user_config" DROP CONSTRAINT IF EXISTS "user_config_user_id_fkey";
ALTER TABLE IF EXISTS "public"."user_role" DROP CONSTRAINT IF EXISTS "user_role_role_id_fkey";
ALTER TABLE IF EXISTS "public"."user_role" DROP CONSTRAINT IF EXISTS "user_role_user_id_fkey";

-- ----------------------------
-- 2. 删除表（按依赖顺序，先删子表）
-- ----------------------------
DROP TABLE IF EXISTS "public"."scheduled_task_execution";
DROP TABLE IF EXISTS "public"."scheduled_task";
DROP TABLE IF EXISTS "public"."user_role";
DROP TABLE IF EXISTS "public"."user_login_log";
DROP TABLE IF EXISTS "public"."user_config";
DROP TABLE IF EXISTS "public"."user";
DROP TABLE IF EXISTS "public"."role_permission";
DROP TABLE IF EXISTS "public"."role";
DROP TABLE IF EXISTS "public"."permission";
DROP TABLE IF EXISTS "public"."menu";
DROP TABLE IF EXISTS "public"."casbin_rule";
DROP TABLE IF EXISTS "public"."audit_log";

-- ----------------------------
-- 3. 删除序列
-- ----------------------------
DROP SEQUENCE IF EXISTS "public"."scheduled_task_execution_id_seq";
DROP SEQUENCE IF EXISTS "public"."scheduled_task_id_seq";
DROP SEQUENCE IF EXISTS "public"."user_role_id_seq";
DROP SEQUENCE IF EXISTS "public"."user_login_log_id_seq";
DROP SEQUENCE IF EXISTS "public"."user_config_id_seq";
DROP SEQUENCE IF EXISTS "public"."user_id_seq";
DROP SEQUENCE IF EXISTS "public"."role_permission_id_seq";
DROP SEQUENCE IF EXISTS "public"."role_id_seq";
DROP SEQUENCE IF EXISTS "public"."permission_id_seq";
DROP SEQUENCE IF EXISTS "public"."menu_id_seq";
DROP SEQUENCE IF EXISTS "public"."casbin_rule_id_seq";
DROP SEQUENCE IF EXISTS "public"."audit_log_id_seq";

-- ----------------------------
-- 4. 删除类型
-- ----------------------------
DROP TYPE IF EXISTS "public"."resource_type";
DROP TYPE IF EXISTS "public"."menu_type";

-- ----------------------------
-- 5. 删除 UUID 函数（可选，取决于是否需要清理）
-- ----------------------------
DROP FUNCTION IF EXISTS "public"."uuid_ns_x500"();
DROP FUNCTION IF EXISTS "public"."uuid_ns_url"();
DROP FUNCTION IF EXISTS "public"."uuid_ns_oid"();
DROP FUNCTION IF EXISTS "public"."uuid_ns_dns"();
DROP FUNCTION IF EXISTS "public"."uuid_nil"();
DROP FUNCTION IF EXISTS "public"."uuid_generate_v5"("namespace" uuid, "name" text);
DROP FUNCTION IF EXISTS "public"."uuid_generate_v4"();
DROP FUNCTION IF EXISTS "public"."uuid_generate_v3"("namespace" uuid, "name" text);
DROP FUNCTION IF EXISTS "public"."uuid_generate_v1mc"();
DROP FUNCTION IF EXISTS "public"."uuid_generate_v1"();

COMMIT;