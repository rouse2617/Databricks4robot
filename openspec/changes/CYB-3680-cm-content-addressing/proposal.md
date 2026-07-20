# Proposal — CYB-3680

## Why

Every run with a runtime config creates its own `runtime-config-<run-id>`
ConfigMap even when the bytes are identical across the whole batch (verified
byte-level identical in production): a 100k batch mints 100k redundant etcd
objects. Worse, the CM was created AFTER the Workflow (its owner ref needed
the server-assigned UID), leaving a crash window where pods mount a ConfigMap
that will never exist — the field-observed `Init:0/1` FailedMount class stuck
for hours (design v1.5 §06, problem P2 + the original stuck-pod incident).

## What Changes

### Modified Capabilities
- **pipeline** (runtime config): content-addressed ConfigMaps —
  `runtime-config-<sha256(content)[:32]>` — shared across runs, created
  **before** the Workflow (idempotent; AlreadyExists = reuse), owner-less.
  A batch's CM count collapses from O(runs) to O(distinct configs).
- Sliding-reference lifecycle: CMs carry `cyberorigin.ai/content-addressed`
  + `cyberorigin.ai/last-referenced`; every ensure touches the timestamp
  (rate-limited to one PATCH per hash per hour); a janitor reclaims CMs whose
  reference age exceeds `RUNTIME_CONFIG_TTL_DAYS` (default 35 > workflow TTL,
  so Argo-native retries never re-mount a reclaimed CM).
- DB blob source of truth: `runtime_config_blobs(hash, files)` upserted on
  deploy (best-effort) — deleting a CM is never data loss; self-heal rebuild
  layers (CYB-3681) read from here.
- Legacy per-run CMs keep their owner-cascade lifecycle untouched (dual-rail
  transition; the janitor's label selector excludes them).

## Impact
- `usecase/pipeline/usecase.go` (hashing, ordering flip, blob wiring; the
  owner-lookup + delete-workflow compensations are deleted), `k8s/runtime_config.go`
  (+janitor), `postgres/runtime_config_blob_repo.go`, config, core wiring,
  one migration. API surface unchanged. Transpiler untouched.

## Success Criteria
- [ ] Two runs sharing config → exactly one CM, hash-named, labeled,
      annotated, owner-less; blob row present.
- [ ] CM ensure failure aborts Deploy BEFORE any Workflow exists.
- [ ] Janitor deletes only expired content-addressed CMs; legacy/fresh/
      ambiguous kept.
- [ ] Legacy projections keep owner cascade byte-for-byte.
