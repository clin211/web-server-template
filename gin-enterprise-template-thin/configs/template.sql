/*
 Navicat Premium Dump SQL

 Source Server         : local-postgresql
 Source Server Type    : PostgreSQL
 Source Server Version : 170007 (170007)
 Source Host           : localhost:5432
 Source Catalog        : template
 Source Schema         : public

 Target Server Type    : PostgreSQL
 Target Server Version : 170007 (170007)
 File Encoding         : 65001

 Date: 10/01/2026 16:47:03
*/


-- ----------------------------
-- Type structure for menu_type
-- ----------------------------
DROP TYPE IF EXISTS "public"."menu_type";
CREATE TYPE "public"."menu_type" AS ENUM (
  'menu',
  'page'
);
ALTER TYPE "public"."menu_type" OWNER TO "postgres";
COMMENT ON TYPE "public"."menu_type" IS '菜单类型枚举：menu=目录, page=页面';

-- ----------------------------
-- Type structure for resource_type
-- ----------------------------
DROP TYPE IF EXISTS "public"."resource_type";
CREATE TYPE "public"."resource_type" AS ENUM (
  'menu',
  'button'
);
ALTER TYPE "public"."resource_type" OWNER TO "postgres";
COMMENT ON TYPE "public"."resource_type" IS '资源类型枚举：menu=菜单, button=按钮';

-- ----------------------------
-- Sequence structure for audit_log_id_seq
-- ----------------------------
DROP SEQUENCE IF EXISTS "public"."audit_log_id_seq";
CREATE SEQUENCE "public"."audit_log_id_seq" 
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;
ALTER SEQUENCE "public"."audit_log_id_seq" OWNER TO "postgres";
COMMENT ON SEQUENCE "public"."audit_log_id_seq" IS '审计日志表内部ID序列';

-- ----------------------------
-- Sequence structure for casbin_rule_id_seq
-- ----------------------------
DROP SEQUENCE IF EXISTS "public"."casbin_rule_id_seq";
CREATE SEQUENCE "public"."casbin_rule_id_seq" 
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;
ALTER SEQUENCE "public"."casbin_rule_id_seq" OWNER TO "postgres";
COMMENT ON SEQUENCE "public"."casbin_rule_id_seq" IS 'Casbin规则表内部ID序列';

-- ----------------------------
-- Sequence structure for menu_id_seq
-- ----------------------------
DROP SEQUENCE IF EXISTS "public"."menu_id_seq";
CREATE SEQUENCE "public"."menu_id_seq" 
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;
ALTER SEQUENCE "public"."menu_id_seq" OWNER TO "postgres";
COMMENT ON SEQUENCE "public"."menu_id_seq" IS '菜单表内部ID序列';

-- ----------------------------
-- Sequence structure for permission_id_seq
-- ----------------------------
DROP SEQUENCE IF EXISTS "public"."permission_id_seq";
CREATE SEQUENCE "public"."permission_id_seq" 
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;
ALTER SEQUENCE "public"."permission_id_seq" OWNER TO "postgres";
COMMENT ON SEQUENCE "public"."permission_id_seq" IS '权限表内部ID序列';

-- ----------------------------
-- Sequence structure for role_id_seq
-- ----------------------------
DROP SEQUENCE IF EXISTS "public"."role_id_seq";
CREATE SEQUENCE "public"."role_id_seq" 
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;
ALTER SEQUENCE "public"."role_id_seq" OWNER TO "postgres";
COMMENT ON SEQUENCE "public"."role_id_seq" IS '角色表内部ID序列';

-- ----------------------------
-- Sequence structure for role_permission_id_seq
-- ----------------------------
DROP SEQUENCE IF EXISTS "public"."role_permission_id_seq";
CREATE SEQUENCE "public"."role_permission_id_seq" 
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;
ALTER SEQUENCE "public"."role_permission_id_seq" OWNER TO "postgres";
COMMENT ON SEQUENCE "public"."role_permission_id_seq" IS '角色权限关联表内部ID序列';

-- ----------------------------
-- Sequence structure for user_config_id_seq
-- ----------------------------
DROP SEQUENCE IF EXISTS "public"."user_config_id_seq";
CREATE SEQUENCE "public"."user_config_id_seq" 
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;
ALTER SEQUENCE "public"."user_config_id_seq" OWNER TO "postgres";
COMMENT ON SEQUENCE "public"."user_config_id_seq" IS '用户配置表内部ID序列';

-- ----------------------------
-- Sequence structure for user_id_seq
-- ----------------------------
DROP SEQUENCE IF EXISTS "public"."user_id_seq";
CREATE SEQUENCE "public"."user_id_seq" 
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;
ALTER SEQUENCE "public"."user_id_seq" OWNER TO "postgres";
COMMENT ON SEQUENCE "public"."user_id_seq" IS '用户表内部ID序列';

-- ----------------------------
-- Sequence structure for user_login_log_id_seq
-- ----------------------------
DROP SEQUENCE IF EXISTS "public"."user_login_log_id_seq";
CREATE SEQUENCE "public"."user_login_log_id_seq" 
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;
ALTER SEQUENCE "public"."user_login_log_id_seq" OWNER TO "postgres";
COMMENT ON SEQUENCE "public"."user_login_log_id_seq" IS '用户登录日志表内部ID序列';

-- ----------------------------
-- Sequence structure for user_role_id_seq
-- ----------------------------
DROP SEQUENCE IF EXISTS "public"."user_role_id_seq";
CREATE SEQUENCE "public"."user_role_id_seq" 
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;
ALTER SEQUENCE "public"."user_role_id_seq" OWNER TO "postgres";
COMMENT ON SEQUENCE "public"."user_role_id_seq" IS '用户角色关联表内部ID序列';

-- ----------------------------
-- Table structure for audit_log
-- ----------------------------
DROP TABLE IF EXISTS "public"."audit_log";
CREATE TABLE "public"."audit_log" (
  "id" int8 NOT NULL DEFAULT nextval('audit_log_id_seq'::regclass),
  "user_id" uuid NOT NULL,
  "action" varchar(50) COLLATE "pg_catalog"."default" NOT NULL,
  "resource" varchar(200) COLLATE "pg_catalog"."default",
  "details" jsonb,
  "created_at" timestamptz(6) NOT NULL DEFAULT CURRENT_TIMESTAMP
)
;
ALTER TABLE "public"."audit_log" OWNER TO "postgres";
COMMENT ON COLUMN "public"."audit_log"."id" IS '内部主键ID（自增序列）';
COMMENT ON COLUMN "public"."audit_log"."user_id" IS '操作用户UUID';
COMMENT ON COLUMN "public"."audit_log"."action" IS '操作类型（如role_assign、permission_deny）';
COMMENT ON COLUMN "public"."audit_log"."resource" IS '操作的资源';
COMMENT ON COLUMN "public"."audit_log"."details" IS '操作详情（JSONB格式，记录变更前后数据）';
COMMENT ON COLUMN "public"."audit_log"."created_at" IS '操作时间';
COMMENT ON TABLE "public"."audit_log" IS '审计日志表，记录权限相关操作历史';

-- ----------------------------
-- Table structure for casbin_rule
-- ----------------------------
DROP TABLE IF EXISTS "public"."casbin_rule";
CREATE TABLE "public"."casbin_rule" (
  "id" int8 NOT NULL DEFAULT nextval('casbin_rule_id_seq'::regclass),
  "ptype" varchar(100) COLLATE "pg_catalog"."default" NOT NULL,
  "v0" varchar(100) COLLATE "pg_catalog"."default",
  "v1" varchar(100) COLLATE "pg_catalog"."default",
  "v2" varchar(100) COLLATE "pg_catalog"."default",
  "v3" varchar(100) COLLATE "pg_catalog"."default",
  "v4" varchar(100) COLLATE "pg_catalog"."default",
  "v5" varchar(100) COLLATE "pg_catalog"."default"
)
;
ALTER TABLE "public"."casbin_rule" OWNER TO "postgres";
COMMENT ON COLUMN "public"."casbin_rule"."id" IS '内部主键ID（自增序列）';
COMMENT ON COLUMN "public"."casbin_rule"."ptype" IS '规则类型（p=权限, g=角色继承）';
COMMENT ON COLUMN "public"."casbin_rule"."v0" IS '主体（用户/角色）';
COMMENT ON COLUMN "public"."casbin_rule"."v1" IS '资源（对象）';
COMMENT ON COLUMN "public"."casbin_rule"."v2" IS '动作（读/写等）';
COMMENT ON COLUMN "public"."casbin_rule"."v3" IS '扩展字段1（条件等）';
COMMENT ON COLUMN "public"."casbin_rule"."v4" IS '扩展字段2';
COMMENT ON COLUMN "public"."casbin_rule"."v5" IS '扩展字段3';
COMMENT ON TABLE "public"."casbin_rule" IS 'Casbin权限规则表，作为系统中唯一的权限控制机制，支持RBAC和ABAC策略';

-- ----------------------------
-- Table structure for menu
-- ----------------------------
DROP TABLE IF EXISTS "public"."menu";
CREATE TABLE "public"."menu" (
  "id" int8 NOT NULL DEFAULT nextval('menu_id_seq'::regclass),
  "menu_id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "parent_id" uuid,
  "menu_name" varchar(50) COLLATE "pg_catalog"."default" NOT NULL,
  "menu_code" varchar(50) COLLATE "pg_catalog"."default" NOT NULL,
  "menu_type" "public"."menu_type" NOT NULL,
  "icon" varchar(50) COLLATE "pg_catalog"."default",
  "path" varchar(200) COLLATE "pg_catalog"."default",
  "component" varchar(200) COLLATE "pg_catalog"."default",
  "permission_id" uuid,
  "sort_order" int4 NOT NULL DEFAULT 0,
  "visible" int2 NOT NULL DEFAULT 1,
  "status" int2 NOT NULL DEFAULT 0,
  "created_at" timestamptz(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "deleted_at" timestamptz(6)
)
;
ALTER TABLE "public"."menu" OWNER TO "postgres";
COMMENT ON COLUMN "public"."menu"."id" IS '内部主键ID（自增序列）';
COMMENT ON COLUMN "public"."menu"."menu_id" IS '菜单业务唯一UUID';
COMMENT ON COLUMN "public"."menu"."parent_id" IS '父菜单UUID（用于构建菜单树）';
COMMENT ON COLUMN "public"."menu"."menu_name" IS '菜单名称';
COMMENT ON COLUMN "public"."menu"."menu_code" IS '菜单编码（唯一标识）';
COMMENT ON COLUMN "public"."menu"."menu_type" IS '菜单类型（menu=目录, page=页面）';
COMMENT ON COLUMN "public"."menu"."icon" IS '菜单图标';
COMMENT ON COLUMN "public"."menu"."path" IS '路由路径';
COMMENT ON COLUMN "public"."menu"."component" IS '前端组件路径（兼容vue-pure-admin）';
COMMENT ON COLUMN "public"."menu"."permission_id" IS '关联权限UUID（外键）';
COMMENT ON COLUMN "public"."menu"."sort_order" IS '排序序号（支持拖拽排序）';
COMMENT ON COLUMN "public"."menu"."visible" IS '是否可见（0=隐藏,1=显示）';
COMMENT ON COLUMN "public"."menu"."status" IS '菜单状态（0=启用,1=禁用）';
COMMENT ON COLUMN "public"."menu"."created_at" IS '创建时间';
COMMENT ON COLUMN "public"."menu"."updated_at" IS '更新时间';
COMMENT ON COLUMN "public"."menu"."deleted_at" IS '软删除时间（NULL=未删除）';
COMMENT ON TABLE "public"."menu" IS '菜单表，存储前端菜单和页面配置信息';

-- ----------------------------
-- Table structure for permission
-- ----------------------------
DROP TABLE IF EXISTS "public"."permission";
CREATE TABLE "public"."permission" (
  "id" int8 NOT NULL DEFAULT nextval('permission_id_seq'::regclass),
  "permission_id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "permission_name" varchar(100) COLLATE "pg_catalog"."default" NOT NULL,
  "permission_code" varchar(100) COLLATE "pg_catalog"."default" NOT NULL,
  "resource_type" "public"."resource_type" NOT NULL,
  "resource_path" varchar(200) COLLATE "pg_catalog"."default",
  "action" varchar(20) COLLATE "pg_catalog"."default" NOT NULL,
  "description" varchar(200) COLLATE "pg_catalog"."default",
  "parent_id" uuid,
  "path" varchar(500) COLLATE "pg_catalog"."default",
  "status" int2 NOT NULL DEFAULT 0,
  "created_at" timestamptz(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "deleted_at" timestamptz(6)
)
;
ALTER TABLE "public"."permission" OWNER TO "postgres";
COMMENT ON COLUMN "public"."permission"."id" IS '内部主键ID（自增序列）';
COMMENT ON COLUMN "public"."permission"."permission_id" IS '权限业务唯一UUID';
COMMENT ON COLUMN "public"."permission"."permission_name" IS '权限名称';
COMMENT ON COLUMN "public"."permission"."permission_code" IS '权限编码（唯一标识）';
COMMENT ON COLUMN "public"."permission"."resource_type" IS '资源类型（menu=菜单, button=按钮）';
COMMENT ON COLUMN "public"."permission"."resource_path" IS '资源路径（如 /system/user/list）';
COMMENT ON COLUMN "public"."permission"."action" IS 'HTTP动词或自定义操作（GET/POST/export等）';
COMMENT ON COLUMN "public"."permission"."description" IS '权限描述';
COMMENT ON COLUMN "public"."permission"."parent_id" IS '父权限UUID（用于构建权限树）';
COMMENT ON COLUMN "public"."permission"."path" IS '全路径（用于树形查询优化）';
COMMENT ON COLUMN "public"."permission"."status" IS '权限状态（0=启用,1=禁用）';
COMMENT ON COLUMN "public"."permission"."created_at" IS '创建时间';
COMMENT ON COLUMN "public"."permission"."updated_at" IS '更新时间';
COMMENT ON COLUMN "public"."permission"."deleted_at" IS '软删除时间（NULL=未删除）';
COMMENT ON TABLE "public"."permission" IS '权限表，存储系统资源和操作权限信息';

-- ----------------------------
-- Table structure for role
-- ----------------------------
DROP TABLE IF EXISTS "public"."role";
CREATE TABLE "public"."role" (
  "id" int8 NOT NULL DEFAULT nextval('role_id_seq'::regclass),
  "role_id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "role_name" varchar(50) COLLATE "pg_catalog"."default" NOT NULL,
  "role_code" varchar(50) COLLATE "pg_catalog"."default" NOT NULL,
  "description" varchar(200) COLLATE "pg_catalog"."default",
  "status" int2 NOT NULL DEFAULT 0,
  "sort_order" int4 NOT NULL DEFAULT 0,
  "created_at" timestamptz(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "deleted_at" timestamptz(6)
)
;
ALTER TABLE "public"."role" OWNER TO "postgres";
COMMENT ON COLUMN "public"."role"."id" IS '内部主键ID（自增序列）';
COMMENT ON COLUMN "public"."role"."role_id" IS '角色业务唯一UUID';
COMMENT ON COLUMN "public"."role"."role_name" IS '角色名称';
COMMENT ON COLUMN "public"."role"."role_code" IS '角色编码（唯一标识，如super_admin、admin）';
COMMENT ON COLUMN "public"."role"."description" IS '角色描述';
COMMENT ON COLUMN "public"."role"."status" IS '角色状态（0=启用,1=禁用）';
COMMENT ON COLUMN "public"."role"."sort_order" IS '排序序号';
COMMENT ON COLUMN "public"."role"."created_at" IS '创建时间';
COMMENT ON COLUMN "public"."role"."updated_at" IS '更新时间';
COMMENT ON COLUMN "public"."role"."deleted_at" IS '软删除时间（NULL=未删除）';
COMMENT ON TABLE "public"."role" IS '角色表，存储系统角色信息';

-- ----------------------------
-- Table structure for role_permission
-- ----------------------------
DROP TABLE IF EXISTS "public"."role_permission";
CREATE TABLE "public"."role_permission" (
  "id" int8 NOT NULL DEFAULT nextval('role_permission_id_seq'::regclass),
  "role_id" uuid NOT NULL,
  "permission_id" uuid NOT NULL,
  "version" int4 NOT NULL DEFAULT 1,
  "created_at" timestamptz(6) NOT NULL DEFAULT CURRENT_TIMESTAMP
)
;
ALTER TABLE "public"."role_permission" OWNER TO "postgres";
COMMENT ON COLUMN "public"."role_permission"."id" IS '内部主键ID（自增序列）';
COMMENT ON COLUMN "public"."role_permission"."role_id" IS '角色UUID（外键）';
COMMENT ON COLUMN "public"."role_permission"."permission_id" IS '权限UUID（外键）';
COMMENT ON COLUMN "public"."role_permission"."version" IS '乐观锁版本号';
COMMENT ON COLUMN "public"."role_permission"."created_at" IS '创建时间';
COMMENT ON TABLE "public"."role_permission" IS '角色权限关联表，实现角色与权限的多对多关系';

-- ----------------------------
-- Table structure for user
-- ----------------------------
DROP TABLE IF EXISTS "public"."user";
CREATE TABLE "public"."user" (
  "id" int8 NOT NULL DEFAULT nextval('user_id_seq'::regclass),
  "user_id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "username" varchar(50) COLLATE "pg_catalog"."default" NOT NULL,
  "password" varchar(255) COLLATE "pg_catalog"."default" NOT NULL,
  "email" varchar(255) COLLATE "pg_catalog"."default",
  "phone" varchar(20) COLLATE "pg_catalog"."default",
  "avatar" varchar(500) COLLATE "pg_catalog"."default",
  "nickname" varchar(100) COLLATE "pg_catalog"."default" NOT NULL DEFAULT ''::character varying,
  "gender" int2 NOT NULL DEFAULT 0,
  "status" int2 NOT NULL DEFAULT 0,
  "last_login_at" timestamptz(6),
  "created_at" timestamptz(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "description" text COLLATE "pg_catalog"."default"
)
;
ALTER TABLE "public"."user" OWNER TO "postgres";
COMMENT ON COLUMN "public"."user"."id" IS '内部主键ID（自增序列）';
COMMENT ON COLUMN "public"."user"."user_id" IS '用户业务唯一UUID';
COMMENT ON COLUMN "public"."user"."username" IS '用户名（唯一，登录用）';
COMMENT ON COLUMN "public"."user"."password" IS '密码哈希（bcrypt加密存储）';
COMMENT ON COLUMN "public"."user"."email" IS '电子邮箱（唯一）';
COMMENT ON COLUMN "public"."user"."phone" IS '手机号（唯一）';
COMMENT ON COLUMN "public"."user"."avatar" IS '头像URL';
COMMENT ON COLUMN "public"."user"."nickname" IS '用户昵称';
COMMENT ON COLUMN "public"."user"."gender" IS '性别（0=未知,1=男,2=女）';
COMMENT ON COLUMN "public"."user"."status" IS '用户状态（0=活跃,1=禁用）';
COMMENT ON COLUMN "public"."user"."last_login_at" IS '最后登录时间';
COMMENT ON COLUMN "public"."user"."created_at" IS '创建时间';
COMMENT ON COLUMN "public"."user"."updated_at" IS '更新时间';
COMMENT ON COLUMN "public"."user"."description" IS '用户描述/简介';
COMMENT ON TABLE "public"."user" IS '用户表，存储用户认证信息、基本资料和应用扩展';

-- ----------------------------
-- Table structure for user_config
-- ----------------------------
DROP TABLE IF EXISTS "public"."user_config";
CREATE TABLE "public"."user_config" (
  "id" int8 NOT NULL DEFAULT nextval('user_config_id_seq'::regclass),
  "user_id" uuid NOT NULL,
  "config_key" varchar(100) COLLATE "pg_catalog"."default" NOT NULL,
  "config_value" jsonb NOT NULL,
  "created_at" timestamptz(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz(6) NOT NULL DEFAULT CURRENT_TIMESTAMP
)
;
ALTER TABLE "public"."user_config" OWNER TO "postgres";
COMMENT ON COLUMN "public"."user_config"."id" IS '内部主键ID（自增序列）';
COMMENT ON COLUMN "public"."user_config"."user_id" IS '用户UUID（外键）';
COMMENT ON COLUMN "public"."user_config"."config_key" IS '配置键名（唯一组合）';
COMMENT ON COLUMN "public"."user_config"."config_value" IS '配置值（JSONB格式）';
COMMENT ON COLUMN "public"."user_config"."created_at" IS '创建时间';
COMMENT ON COLUMN "public"."user_config"."updated_at" IS '更新时间';
COMMENT ON TABLE "public"."user_config" IS '用户个人配置表，存储用户偏好设置';

-- ----------------------------
-- Table structure for user_login_log
-- ----------------------------
DROP TABLE IF EXISTS "public"."user_login_log";
CREATE TABLE "public"."user_login_log" (
  "id" int8 NOT NULL DEFAULT nextval('user_login_log_id_seq'::regclass),
  "username" varchar(50) COLLATE "pg_catalog"."default",
  "ip_address" inet,
  "user_agent" varchar(1000) COLLATE "pg_catalog"."default",
  "status" bool NOT NULL,
  "error_message" text COLLATE "pg_catalog"."default",
  "created_at" timestamptz(6) NOT NULL DEFAULT CURRENT_TIMESTAMP
)
;
ALTER TABLE "public"."user_login_log" OWNER TO "postgres";
COMMENT ON COLUMN "public"."user_login_log"."id" IS '内部主键ID（自增序列）';
COMMENT ON COLUMN "public"."user_login_log"."username" IS '登录用户名';
COMMENT ON COLUMN "public"."user_login_log"."ip_address" IS '登录IP地址';
COMMENT ON COLUMN "public"."user_login_log"."user_agent" IS '用户代理字符串';
COMMENT ON COLUMN "public"."user_login_log"."status" IS '登录状态（true=成功, false=失败）';
COMMENT ON COLUMN "public"."user_login_log"."error_message" IS '错误消息（失败时）';
COMMENT ON COLUMN "public"."user_login_log"."created_at" IS '登录尝试时间';
COMMENT ON TABLE "public"."user_login_log" IS '用户登录日志表，记录登录尝试和安全信息';

-- ----------------------------
-- Table structure for user_role
-- ----------------------------
DROP TABLE IF EXISTS "public"."user_role";
CREATE TABLE "public"."user_role" (
  "id" int8 NOT NULL DEFAULT nextval('user_role_id_seq'::regclass),
  "user_id" uuid NOT NULL,
  "role_id" uuid NOT NULL,
  "assigned_at" timestamptz(6) NOT NULL DEFAULT CURRENT_TIMESTAMP
)
;
ALTER TABLE "public"."user_role" OWNER TO "postgres";
COMMENT ON COLUMN "public"."user_role"."id" IS '内部主键ID（自增序列）';
COMMENT ON COLUMN "public"."user_role"."user_id" IS '用户UUID（外键）';
COMMENT ON COLUMN "public"."user_role"."role_id" IS '角色UUID（外键）';
COMMENT ON COLUMN "public"."user_role"."assigned_at" IS '分配时间';
COMMENT ON TABLE "public"."user_role" IS '用户角色关联表，实现用户与角色的多对多关系';

-- ----------------------------
-- Function structure for uuid_generate_v1
-- ----------------------------
DROP FUNCTION IF EXISTS "public"."uuid_generate_v1"();
CREATE FUNCTION "public"."uuid_generate_v1"()
  RETURNS "pg_catalog"."uuid" AS '$libdir/uuid-ossp', 'uuid_generate_v1'
  LANGUAGE c VOLATILE STRICT
  COST 1;
ALTER FUNCTION "public"."uuid_generate_v1"() OWNER TO "postgres";

-- ----------------------------
-- Function structure for uuid_generate_v1mc
-- ----------------------------
DROP FUNCTION IF EXISTS "public"."uuid_generate_v1mc"();
CREATE FUNCTION "public"."uuid_generate_v1mc"()
  RETURNS "pg_catalog"."uuid" AS '$libdir/uuid-ossp', 'uuid_generate_v1mc'
  LANGUAGE c VOLATILE STRICT
  COST 1;
ALTER FUNCTION "public"."uuid_generate_v1mc"() OWNER TO "postgres";

-- ----------------------------
-- Function structure for uuid_generate_v3
-- ----------------------------
DROP FUNCTION IF EXISTS "public"."uuid_generate_v3"("namespace" uuid, "name" text);
CREATE FUNCTION "public"."uuid_generate_v3"("namespace" uuid, "name" text)
  RETURNS "pg_catalog"."uuid" AS '$libdir/uuid-ossp', 'uuid_generate_v3'
  LANGUAGE c IMMUTABLE STRICT
  COST 1;
ALTER FUNCTION "public"."uuid_generate_v3"("namespace" uuid, "name" text) OWNER TO "postgres";

-- ----------------------------
-- Function structure for uuid_generate_v4
-- ----------------------------
DROP FUNCTION IF EXISTS "public"."uuid_generate_v4"();
CREATE FUNCTION "public"."uuid_generate_v4"()
  RETURNS "pg_catalog"."uuid" AS '$libdir/uuid-ossp', 'uuid_generate_v4'
  LANGUAGE c VOLATILE STRICT
  COST 1;
ALTER FUNCTION "public"."uuid_generate_v4"() OWNER TO "postgres";

-- ----------------------------
-- Function structure for uuid_generate_v5
-- ----------------------------
DROP FUNCTION IF EXISTS "public"."uuid_generate_v5"("namespace" uuid, "name" text);
CREATE FUNCTION "public"."uuid_generate_v5"("namespace" uuid, "name" text)
  RETURNS "pg_catalog"."uuid" AS '$libdir/uuid-ossp', 'uuid_generate_v5'
  LANGUAGE c IMMUTABLE STRICT
  COST 1;
ALTER FUNCTION "public"."uuid_generate_v5"("namespace" uuid, "name" text) OWNER TO "postgres";

-- ----------------------------
-- Function structure for uuid_nil
-- ----------------------------
DROP FUNCTION IF EXISTS "public"."uuid_nil"();
CREATE FUNCTION "public"."uuid_nil"()
  RETURNS "pg_catalog"."uuid" AS '$libdir/uuid-ossp', 'uuid_nil'
  LANGUAGE c IMMUTABLE STRICT
  COST 1;
ALTER FUNCTION "public"."uuid_nil"() OWNER TO "postgres";

-- ----------------------------
-- Function structure for uuid_ns_dns
-- ----------------------------
DROP FUNCTION IF EXISTS "public"."uuid_ns_dns"();
CREATE FUNCTION "public"."uuid_ns_dns"()
  RETURNS "pg_catalog"."uuid" AS '$libdir/uuid-ossp', 'uuid_ns_dns'
  LANGUAGE c IMMUTABLE STRICT
  COST 1;
ALTER FUNCTION "public"."uuid_ns_dns"() OWNER TO "postgres";

-- ----------------------------
-- Function structure for uuid_ns_oid
-- ----------------------------
DROP FUNCTION IF EXISTS "public"."uuid_ns_oid"();
CREATE FUNCTION "public"."uuid_ns_oid"()
  RETURNS "pg_catalog"."uuid" AS '$libdir/uuid-ossp', 'uuid_ns_oid'
  LANGUAGE c IMMUTABLE STRICT
  COST 1;
ALTER FUNCTION "public"."uuid_ns_oid"() OWNER TO "postgres";

-- ----------------------------
-- Function structure for uuid_ns_url
-- ----------------------------
DROP FUNCTION IF EXISTS "public"."uuid_ns_url"();
CREATE FUNCTION "public"."uuid_ns_url"()
  RETURNS "pg_catalog"."uuid" AS '$libdir/uuid-ossp', 'uuid_ns_url'
  LANGUAGE c IMMUTABLE STRICT
  COST 1;
ALTER FUNCTION "public"."uuid_ns_url"() OWNER TO "postgres";

-- ----------------------------
-- Function structure for uuid_ns_x500
-- ----------------------------
DROP FUNCTION IF EXISTS "public"."uuid_ns_x500"();
CREATE FUNCTION "public"."uuid_ns_x500"()
  RETURNS "pg_catalog"."uuid" AS '$libdir/uuid-ossp', 'uuid_ns_x500'
  LANGUAGE c IMMUTABLE STRICT
  COST 1;
ALTER FUNCTION "public"."uuid_ns_x500"() OWNER TO "postgres";

-- ----------------------------
-- Alter sequences owned by
-- ----------------------------
ALTER SEQUENCE "public"."audit_log_id_seq"
OWNED BY "public"."audit_log"."id";
SELECT setval('"public"."audit_log_id_seq"', 1, false);

-- ----------------------------
-- Alter sequences owned by
-- ----------------------------
ALTER SEQUENCE "public"."casbin_rule_id_seq"
OWNED BY "public"."casbin_rule"."id";
SELECT setval('"public"."casbin_rule_id_seq"', 1, false);

-- ----------------------------
-- Alter sequences owned by
-- ----------------------------
ALTER SEQUENCE "public"."menu_id_seq"
OWNED BY "public"."menu"."id";
SELECT setval('"public"."menu_id_seq"', 1, false);

-- ----------------------------
-- Alter sequences owned by
-- ----------------------------
ALTER SEQUENCE "public"."permission_id_seq"
OWNED BY "public"."permission"."id";
SELECT setval('"public"."permission_id_seq"', 1, false);

-- ----------------------------
-- Alter sequences owned by
-- ----------------------------
ALTER SEQUENCE "public"."role_id_seq"
OWNED BY "public"."role"."id";
SELECT setval('"public"."role_id_seq"', 1, false);

-- ----------------------------
-- Alter sequences owned by
-- ----------------------------
ALTER SEQUENCE "public"."role_permission_id_seq"
OWNED BY "public"."role_permission"."id";
SELECT setval('"public"."role_permission_id_seq"', 1, false);

-- ----------------------------
-- Alter sequences owned by
-- ----------------------------
ALTER SEQUENCE "public"."user_config_id_seq"
OWNED BY "public"."user_config"."id";
SELECT setval('"public"."user_config_id_seq"', 1, false);

-- ----------------------------
-- Alter sequences owned by
-- ----------------------------
ALTER SEQUENCE "public"."user_id_seq"
OWNED BY "public"."user"."id";
SELECT setval('"public"."user_id_seq"', 1, false);

-- ----------------------------
-- Alter sequences owned by
-- ----------------------------
ALTER SEQUENCE "public"."user_login_log_id_seq"
OWNED BY "public"."user_login_log"."id";
SELECT setval('"public"."user_login_log_id_seq"', 1, false);

-- ----------------------------
-- Alter sequences owned by
-- ----------------------------
ALTER SEQUENCE "public"."user_role_id_seq"
OWNED BY "public"."user_role"."id";
SELECT setval('"public"."user_role_id_seq"', 1, false);

-- ----------------------------
-- Indexes structure for table audit_log
-- ----------------------------
CREATE INDEX "idx_audit_log_created_at" ON "public"."audit_log" USING btree (
  "created_at" "pg_catalog"."timestamptz_ops" DESC NULLS FIRST
);
CREATE INDEX "idx_audit_log_user_id" ON "public"."audit_log" USING btree (
  "user_id" "pg_catalog"."uuid_ops" ASC NULLS LAST
);

-- ----------------------------
-- Primary Key structure for table audit_log
-- ----------------------------
ALTER TABLE "public"."audit_log" ADD CONSTRAINT "audit_log_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table casbin_rule
-- ----------------------------
CREATE INDEX "idx_casbin_rule_g_v0" ON "public"."casbin_rule" USING btree (
  "ptype" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "v0" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE ptype::text = 'g'::text;
CREATE INDEX "idx_casbin_rule_ptype" ON "public"."casbin_rule" USING btree (
  "ptype" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE INDEX "idx_casbin_rule_ptype_v0_v1" ON "public"."casbin_rule" USING btree (
  "ptype" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "v0" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "v1" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);

-- ----------------------------
-- Primary Key structure for table casbin_rule
-- ----------------------------
ALTER TABLE "public"."casbin_rule" ADD CONSTRAINT "casbin_rule_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table menu
-- ----------------------------
CREATE INDEX "idx_menu_active" ON "public"."menu" USING btree (
  "created_at" "pg_catalog"."timestamptz_ops" DESC NULLS FIRST
) WHERE deleted_at IS NULL;
CREATE INDEX "idx_menu_code" ON "public"."menu" USING btree (
  "menu_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE deleted_at IS NULL;
CREATE INDEX "idx_menu_parent_id" ON "public"."menu" USING btree (
  "parent_id" "pg_catalog"."uuid_ops" ASC NULLS LAST
);
CREATE INDEX "idx_menu_parent_sort" ON "public"."menu" USING btree (
  "parent_id" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "sort_order" "pg_catalog"."int4_ops" ASC NULLS LAST
) WHERE deleted_at IS NULL;
CREATE INDEX "idx_menu_path" ON "public"."menu" USING btree (
  "path" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE INDEX "idx_menu_permission_id" ON "public"."menu" USING btree (
  "permission_id" "pg_catalog"."uuid_ops" ASC NULLS LAST
);
CREATE INDEX "idx_menu_status" ON "public"."menu" USING btree (
  "status" "pg_catalog"."int2_ops" ASC NULLS LAST
);
CREATE INDEX "idx_menu_type" ON "public"."menu" USING btree (
  "menu_type" "pg_catalog"."enum_ops" ASC NULLS LAST
);
CREATE INDEX "idx_menu_visible" ON "public"."menu" USING btree (
  "visible" "pg_catalog"."int2_ops" ASC NULLS LAST
);

-- ----------------------------
-- Uniques structure for table menu
-- ----------------------------
ALTER TABLE "public"."menu" ADD CONSTRAINT "menu_menu_id_key" UNIQUE ("menu_id");
ALTER TABLE "public"."menu" ADD CONSTRAINT "menu_menu_code_key" UNIQUE ("menu_code");

-- ----------------------------
-- Primary Key structure for table menu
-- ----------------------------
ALTER TABLE "public"."menu" ADD CONSTRAINT "menu_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table permission
-- ----------------------------
CREATE INDEX "idx_permission_active" ON "public"."permission" USING btree (
  "created_at" "pg_catalog"."timestamptz_ops" DESC NULLS FIRST
) WHERE deleted_at IS NULL;
CREATE INDEX "idx_permission_code" ON "public"."permission" USING btree (
  "permission_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE deleted_at IS NULL;
CREATE INDEX "idx_permission_parent_id" ON "public"."permission" USING btree (
  "parent_id" "pg_catalog"."uuid_ops" ASC NULLS LAST
);
CREATE INDEX "idx_permission_path" ON "public"."permission" USING btree (
  "path" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE INDEX "idx_permission_resource_type" ON "public"."permission" USING btree (
  "resource_type" "pg_catalog"."enum_ops" ASC NULLS LAST
);
CREATE INDEX "idx_permission_status" ON "public"."permission" USING btree (
  "status" "pg_catalog"."int2_ops" ASC NULLS LAST
);

-- ----------------------------
-- Uniques structure for table permission
-- ----------------------------
ALTER TABLE "public"."permission" ADD CONSTRAINT "permission_permission_id_key" UNIQUE ("permission_id");
ALTER TABLE "public"."permission" ADD CONSTRAINT "permission_permission_code_key" UNIQUE ("permission_code");

-- ----------------------------
-- Primary Key structure for table permission
-- ----------------------------
ALTER TABLE "public"."permission" ADD CONSTRAINT "permission_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table role
-- ----------------------------
CREATE INDEX "idx_role_active" ON "public"."role" USING btree (
  "created_at" "pg_catalog"."timestamptz_ops" DESC NULLS FIRST
) WHERE deleted_at IS NULL;
CREATE INDEX "idx_role_code" ON "public"."role" USING btree (
  "role_code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE deleted_at IS NULL;
CREATE INDEX "idx_role_status" ON "public"."role" USING btree (
  "status" "pg_catalog"."int2_ops" ASC NULLS LAST
);

-- ----------------------------
-- Uniques structure for table role
-- ----------------------------
ALTER TABLE "public"."role" ADD CONSTRAINT "role_role_id_key" UNIQUE ("role_id");
ALTER TABLE "public"."role" ADD CONSTRAINT "role_role_code_key" UNIQUE ("role_code");

-- ----------------------------
-- Primary Key structure for table role
-- ----------------------------
ALTER TABLE "public"."role" ADD CONSTRAINT "role_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table role_permission
-- ----------------------------
CREATE INDEX "idx_role_permission_permission_id" ON "public"."role_permission" USING btree (
  "permission_id" "pg_catalog"."uuid_ops" ASC NULLS LAST
);
CREATE INDEX "idx_role_permission_role_id" ON "public"."role_permission" USING btree (
  "role_id" "pg_catalog"."uuid_ops" ASC NULLS LAST
);

-- ----------------------------
-- Uniques structure for table role_permission
-- ----------------------------
ALTER TABLE "public"."role_permission" ADD CONSTRAINT "role_permission_role_id_permission_id_key" UNIQUE ("role_id", "permission_id");

-- ----------------------------
-- Primary Key structure for table role_permission
-- ----------------------------
ALTER TABLE "public"."role_permission" ADD CONSTRAINT "role_permission_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table user
-- ----------------------------
CREATE INDEX "idx_user_email" ON "public"."user" USING btree (
  "email" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE email IS NOT NULL;
CREATE INDEX "idx_user_phone" ON "public"."user" USING btree (
  "phone" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE phone IS NOT NULL;
CREATE INDEX "idx_user_status" ON "public"."user" USING btree (
  "status" "pg_catalog"."int2_ops" ASC NULLS LAST
);
CREATE INDEX "idx_user_status_last_login" ON "public"."user" USING btree (
  "status" "pg_catalog"."int2_ops" ASC NULLS LAST,
  "last_login_at" "pg_catalog"."timestamptz_ops" DESC NULLS FIRST
);

-- ----------------------------
-- Uniques structure for table user
-- ----------------------------
ALTER TABLE "public"."user" ADD CONSTRAINT "uni_user_user_id" UNIQUE ("user_id");
ALTER TABLE "public"."user" ADD CONSTRAINT "user_username_key" UNIQUE ("username");
ALTER TABLE "public"."user" ADD CONSTRAINT "user_email_key" UNIQUE ("email");
ALTER TABLE "public"."user" ADD CONSTRAINT "user_phone_key" UNIQUE ("phone");

-- ----------------------------
-- Primary Key structure for table user
-- ----------------------------
ALTER TABLE "public"."user" ADD CONSTRAINT "user_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table user_config
-- ----------------------------
CREATE INDEX "idx_user_config_user" ON "public"."user_config" USING btree (
  "user_id" "pg_catalog"."uuid_ops" ASC NULLS LAST
);

-- ----------------------------
-- Uniques structure for table user_config
-- ----------------------------
ALTER TABLE "public"."user_config" ADD CONSTRAINT "user_config_user_id_config_key_key" UNIQUE ("user_id", "config_key");

-- ----------------------------
-- Primary Key structure for table user_config
-- ----------------------------
ALTER TABLE "public"."user_config" ADD CONSTRAINT "user_config_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table user_login_log
-- ----------------------------
CREATE INDEX "idx_user_login_log_created" ON "public"."user_login_log" USING btree (
  "created_at" "pg_catalog"."timestamptz_ops" DESC NULLS FIRST
);
CREATE INDEX "idx_user_login_log_status_created" ON "public"."user_login_log" USING btree (
  "status" "pg_catalog"."bool_ops" ASC NULLS LAST,
  "created_at" "pg_catalog"."timestamptz_ops" DESC NULLS FIRST
);
CREATE INDEX "idx_user_login_log_username" ON "public"."user_login_log" USING btree (
  "username" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
) WHERE username IS NOT NULL;

-- ----------------------------
-- Primary Key structure for table user_login_log
-- ----------------------------
ALTER TABLE "public"."user_login_log" ADD CONSTRAINT "user_login_log_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table user_role
-- ----------------------------
CREATE INDEX "idx_user_role_role_id" ON "public"."user_role" USING btree (
  "role_id" "pg_catalog"."uuid_ops" ASC NULLS LAST
);
CREATE INDEX "idx_user_role_user_id" ON "public"."user_role" USING btree (
  "user_id" "pg_catalog"."uuid_ops" ASC NULLS LAST
);

-- ----------------------------
-- Uniques structure for table user_role
-- ----------------------------
ALTER TABLE "public"."user_role" ADD CONSTRAINT "user_role_user_id_role_id_key" UNIQUE ("user_id", "role_id");

-- ----------------------------
-- Primary Key structure for table user_role
-- ----------------------------
ALTER TABLE "public"."user_role" ADD CONSTRAINT "user_role_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Foreign Keys structure for table menu
-- ----------------------------
ALTER TABLE "public"."menu" ADD CONSTRAINT "menu_parent_id_fkey" FOREIGN KEY ("parent_id") REFERENCES "public"."menu" ("menu_id") ON DELETE CASCADE ON UPDATE NO ACTION;
ALTER TABLE "public"."menu" ADD CONSTRAINT "menu_permission_id_fkey" FOREIGN KEY ("permission_id") REFERENCES "public"."permission" ("permission_id") ON DELETE SET NULL ON UPDATE NO ACTION;

-- ----------------------------
-- Foreign Keys structure for table role_permission
-- ----------------------------
ALTER TABLE "public"."role_permission" ADD CONSTRAINT "role_permission_permission_id_fkey" FOREIGN KEY ("permission_id") REFERENCES "public"."permission" ("permission_id") ON DELETE CASCADE ON UPDATE NO ACTION;
ALTER TABLE "public"."role_permission" ADD CONSTRAINT "role_permission_role_id_fkey" FOREIGN KEY ("role_id") REFERENCES "public"."role" ("role_id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- ----------------------------
-- Foreign Keys structure for table user_config
-- ----------------------------
ALTER TABLE "public"."user_config" ADD CONSTRAINT "user_config_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."user" ("user_id") ON DELETE CASCADE ON UPDATE NO ACTION;

-- ----------------------------
-- Foreign Keys structure for table user_role
-- ----------------------------
ALTER TABLE "public"."user_role" ADD CONSTRAINT "user_role_role_id_fkey" FOREIGN KEY ("role_id") REFERENCES "public"."role" ("role_id") ON DELETE CASCADE ON UPDATE NO ACTION;
ALTER TABLE "public"."user_role" ADD CONSTRAINT "user_role_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."user" ("user_id") ON DELETE CASCADE ON UPDATE NO ACTION;
