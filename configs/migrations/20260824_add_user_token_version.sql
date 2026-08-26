BEGIN;

ALTER TABLE "public"."user"
    ADD COLUMN IF NOT EXISTS "token_version" bigint NOT NULL DEFAULT 0;

COMMENT ON COLUMN "public"."user"."token_version"
    IS '令牌版本；递增后使此前签发的令牌失效';

COMMIT;
