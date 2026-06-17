BEGIN;

ALTER TABLE IF EXISTS "public"."user_config" DROP CONSTRAINT IF EXISTS "user_config_user_id_fkey";

DROP TABLE IF EXISTS "public"."user_login_log";
DROP TABLE IF EXISTS "public"."user_config";
DROP TABLE IF EXISTS "public"."user";
DROP TABLE IF EXISTS "public"."audit_log";

DROP SEQUENCE IF EXISTS "public"."user_login_log_id_seq";
DROP SEQUENCE IF EXISTS "public"."user_config_id_seq";
DROP SEQUENCE IF EXISTS "public"."user_id_seq";
DROP SEQUENCE IF EXISTS "public"."audit_log_id_seq";

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
