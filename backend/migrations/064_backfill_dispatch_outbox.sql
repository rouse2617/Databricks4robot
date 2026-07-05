-- Phase 4 dispatcher (outbox engine) — see
-- openspec/changes/CYB-RUN-DIAGNOSIS-REFACTOR/PHASE4-DESIGN.md.
--
-- This migration extends backfill_items with the outbox state machine
-- columns. The legacy in-memory goroutine dispatch path (runItems /
-- executeItem / ClaimNextItem) continues to work after this migration
-- because:
--   * New columns have safe defaults (dispatch_state='pending').
--   * Existing rows are immediately stamped dispatch_state='legacy_skip'
--     so the new dispatcher does NOT pick them up; they continue
--     draining through the legacy path.
--   * Migration is additive; no existing column is altered.
--
-- The dispatcher entry-point switch and legacy path deletion are
-- handled in follow-up commits (Phase 4 Commit B/C).

-- 1. Schema additions ----------------------------------------------------

ALTER TABLE backfill_items
    ADD COLUMN dispatch_state              VARCHAR(16) NOT NULL DEFAULT 'pending',
        -- Outbox state machine:
        --   pending       ready to be claimed
        --   claimed       ticker SKIP-LOCKED-locked this row; lease active
        --   submitting    worker has refreshed lease; submit in flight
        --   failed        retryable; lease holds the backoff window
        --   submitted     ABSORBING — successful, never re-claim
        --   dead          ABSORBING — MaxAttempts reached, never re-claim
        --   legacy_skip   pre-migration row, dispatcher ignores
    ADD COLUMN dispatch_generation         BIGINT      NOT NULL DEFAULT 1,
        -- Bumped by PrepareItemsForRerun. Part of the deterministic
        -- workflow_name_planned identity (Phase 1) — a rerun must
        -- legitimately produce a NEW workflow, not silently re-adopt
        -- an old one via 409 + AlreadyExists.
    ADD COLUMN workflow_name_planned       VARCHAR(253),
        -- The deterministic Argo workflow name computed at claim time
        -- and used verbatim in DeployOptions.PreallocatedWorkflowName.
    ADD COLUMN dispatch_lease_expires_at   TIMESTAMPTZ,
        -- Active lease wall-clock. NULL when not held. Stale rows
        -- (lease in the past) are re-claimable. Also serves as the
        -- exponential-backoff window for retryable failures
        -- (Decision B, "no-perception backoff lock"): the lease field
        -- doubles as the retry gate via ClaimNextDispatch's WHERE clause.
    ADD COLUMN dispatch_last_error         TEXT,
        -- Failure attempt counter. Incremented on each claim; transitions
        -- to 'dead' when attempts >= MaxAttempts (set by dispatcher).
    ADD COLUMN attempts                    INT         NOT NULL DEFAULT 0;

-- Optional bigserial — strictly for cold-archive water-mark. The plan
-- has it as the single-source cursor for the legacy archive sweep;
-- it MUST NOT be used as the cursor for active claiming (see
-- "游标铁律": nextval() commits independently of transactions, so a
-- persistent cursor that consumes it can permanently miss a row whose
-- smaller seq was allocated before the cursor was last bumped).
ALTER TABLE backfill_items
    ADD COLUMN dispatch_seq BIGSERIAL;

-- 2. Indexes ------------------------------------------------------------
--
-- Partial index for the hot dispatcher claim path. The REDUCED index
-- (excluding 'legacy_skip', 'submitted', 'dead') keeps the index
-- self-healing: rows that transition out of claimable states shrink
-- the index naturally without VACUUM REINDEX.

CREATE INDEX idx_backfill_dispatch_hot
    ON backfill_items (job_id, dispatch_lease_expires_at)
    WHERE dispatch_state IN ('pending', 'failed', 'claimed', 'submitting');

-- 3. Data migration ------------------------------------------------------
--
-- Stamp existing rows to 'legacy_skip' so the new dispatcher doesn't
-- accidentally pick them up on Commit A deployment. The legacy
-- ClaimNextItem path keeps draining them. Once Commit B turns on the
-- new dispatcher for entry points, NEW rows will be 'pending' and
-- drain via the dispatcher; the legacy_skip rows continue to drain
-- via the legacy path until Commit C deletes the legacy path.

UPDATE backfill_items
    SET dispatch_state='legacy_skip'
    WHERE dispatch_state='pending'
      AND status IN ('running', 'pending');
-- Note: rows currently 'completed' / 'failed' keep dispatch_state='pending'
-- (from column default). They are already terminal in the legacy
-- path and the new dispatcher will skip them (its claim filter only
-- matches 'pending' and 'failed', and 'completed' isn't one of them).

-- 4. Dispatch-relevant row maintenance ----------------------------------
--
-- Reaper-equivalent via WHERE only (no UPDATE) — also exclude the
-- legacy_skip sentinel from any future maintenance paths. This is
-- documentation in DB-form.

-- 5. Table-level autovacuum + fillfactor for write-heavy dispatch
--    column churn. Per-plan §"游标铁律".

ALTER TABLE backfill_items SET (
    fillfactor                         = 80,
    -- autovacuum_vacuum_scale_factor: trigger autovacuum when ~2% dead tuples
    -- (default 20%). With aggressive claim/submit churn and many
    -- state transitions, default 20% lets the table bloat before
    -- vacuuming.
    autovacuum_vacuum_scale_factor     = 0.02,
    autovacuum_vacuum_threshold        = 200,
    -- Increase per-round work budget so autovacuum can actually
    -- catch up under flood-grade batch sizes.
    autovacuum_vacuum_cost_limit       = 3000,
    autovacuum_vacuum_cost_delay       = 1
);

-- 6. REPLICA IDENTITY protection -----------------------------------------
--
-- We do NOT enable REPLICA IDENTITY FULL — it serialises the entire
-- row (including any blob/large text) into the WAL on UPDATE, which
-- would multiply WAL volume and CDC bandwidth on this high-churn
-- table. The default REPLICA IDENTITY DEFAULT (which is effectively
-- NOTHING for republish) is correct here; we're not exposing
-- backfill_items to logical replication.

-- End of migration 064.
