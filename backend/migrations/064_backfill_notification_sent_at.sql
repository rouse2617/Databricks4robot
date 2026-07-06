-- CYB-3071: track whether the batch-job-completion Feishu notification has
-- already been claimed/sent for a job, so concurrent backend instances
-- observing the same terminal transition do not send it twice.
ALTER TABLE backfill_jobs
  ADD COLUMN IF NOT EXISTS notification_sent_at TIMESTAMPTZ;
