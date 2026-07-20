-- CYB-3678: durable transient-retry counter for the submitter's DLQ cap.
-- A poison item stops burning cycles after N transient failures and lands in
-- the DLQ (status=failed); the counter survives restarts so the cap holds
-- across redeploys. Reset on explicit DLQ retry.
ALTER TABLE "backfill_items" ADD COLUMN "submit_attempts" integer NOT NULL DEFAULT 0;
