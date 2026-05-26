-- 015_widen_deliveries_customer_id.sql — align deliveries.customer_id with
-- schemas/pg-phase0.sql (TEXT). Fresh installs get TEXT from 001_init.sql;
-- existing DBs may still have VARCHAR(64) from older 001.

ALTER TABLE deliveries
  ALTER COLUMN customer_id TYPE TEXT;
