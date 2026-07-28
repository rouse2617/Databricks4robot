# CYB-4350 decisions

## Why on-demand aggregate, not a denormalized column

- Zero migration cost.
- Zero write-time surface change — everything is computed from `assets.duration_ms` which is already correct.
- Single-batch aggregate is fast enough (~10-50ms even with tens of thousands of children — indexed on `pipeline_runs.batch_job_id`, GIN on `asset_ids` array element).
- List page shows 20 batches per page — one aggregate query over ~20 batch ids is trivially fast.
- If this becomes hot at scale, splitting into a materialized column is a follow-up, but until we see load evidence it's premature.

## Why no dedup across runs

If a run has `asset_ids = ['A', 'B']` and another has `asset_ids = ['A', 'C']`, A counts twice. Rationale: the notification text reads "总时长" as "how much processing time worth of data was in this batch's runs" — which is what per-run-multiplicity gives. Dedup would answer "how many distinct assets" which is a different question and can be derived from the count columns already.

Also matches the accounting semantics of `pipeline_runs.asset_ids` as authored — a run "owns" its input assets for the purpose of that run.

## Why exclude soft-deleted assets

Otherwise a stale row lingering in `assets` after deletion would inflate the total. The list already respects `is_deleted` in the assets discovery endpoints; keep this consistent.

## Why omit `总时长` line when 0

`总时长:0s` or `总时长:—` is noise. Batches that legitimately have 0 duration (e.g. a pure-metadata batch with no video) should just skip the line. Callers reading the notification learn nothing from a zero total.

## Why extend `formatBatchJobNotificationText` signature (not read total inside)

The formatter is a pure function of a `BackfillJob` + summary. It doesn't own DB access. Passing the precomputed value keeps the layers clean and lets tests assert output shape without a repo mock.

## Why extract `formatDurationMs` to `lib/batchJobs.ts`

Two callers need it now (durations preset + batch job list column) and more will show up (costs preset already reuses this shape, lineage preset just added). One canonical duration formatter avoids the drift where the batch list shows `2h 34m` while the preset shows `9240000ms`.

## What we're NOT doing

- Not adding total duration to the batch detail page — that page has its own node summary + can be a follow-up if user asks
- Not real-time updating the total as children finish — snapshot only; refresh to see updates
- Not exposing distinct-asset-count separately — the count columns already surface that
