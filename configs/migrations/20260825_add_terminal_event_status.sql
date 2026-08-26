BEGIN;

ALTER TABLE "public"."outbox_event"
    DROP CONSTRAINT IF EXISTS "chk_outbox_event_status";
ALTER TABLE "public"."outbox_event"
    ADD CONSTRAINT "chk_outbox_event_status"
    CHECK ("status" IN ('pending', 'processing', 'published', 'failed', 'dead'));

ALTER TABLE "public"."dead_letter_event"
    DROP CONSTRAINT IF EXISTS "chk_dead_letter_event_status";
ALTER TABLE "public"."dead_letter_event"
    ADD CONSTRAINT "chk_dead_letter_event_status"
    CHECK ("status" IN ('pending', 'processing', 'published', 'failed', 'dead'));

COMMENT ON COLUMN "public"."outbox_event"."status"
    IS 'pending、processing、published、failed或dead；dead表示达到发布重试上限';
COMMENT ON COLUMN "public"."dead_letter_event"."status"
    IS 'pending、processing、published、failed或dead；dead表示达到发布重试上限';

COMMIT;
