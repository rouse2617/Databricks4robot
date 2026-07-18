-- CYB-3677: adopt stranded legacy batch jobs into the durable submitter.
--
-- The legacy in-memory dispatch goroutine ran jobs under status 'processing',
-- which the submitter's FindSubmittableJobs never selects. Any backend restart
-- mid-batch therefore stranded the job: items stayed 'pending' forever. Flip
-- exactly those jobs to 'running' so the submitter picks them up and finishes
-- them. Idempotent: re-running matches zero rows.
UPDATE backfill_jobs bj
SET status = 'running', updated_at = now()
WHERE bj.status = 'processing'
  AND EXISTS (
    SELECT 1 FROM backfill_items bi
    WHERE bi.job_id = bj.id AND bi.status = 'pending'
  );
