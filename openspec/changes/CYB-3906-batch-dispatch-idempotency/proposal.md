# Proposal — CYB-3906

## Why

Batch subtask materialization persists a placeholder workflow name that differs
from the UUID name used by the actual Argo submission. Automatic recovery can
therefore mint a second attempt after a delayed UID write or concurrent
submission, violating the existing exactly-once dispatch invariant.

## What Changes

### New Capabilities
- None.

### Modified Capabilities
- **pipeline batch dispatch**: one logical item attempt uses one stable run ID
  as its placeholder identity, Argo workflow name, retry identity, and
  AlreadyExists lookup key.
- **automatic submission recovery**: UID-delayed and post-create persistence
  failures reuse the existing attempt; only an explicit operator rerun creates
  a new attempt.
- **legacy placeholder compatibility**: UID-less business-name placeholders
  are normalized to their run UUID before submission without creating a new
  visible attempt.

## Impact
- **Affected code**:
  `backend/internal/usecase/pipeline/batch_subtask.go`,
  `backend/internal/usecase/pipeline/usecase.go`,
  `backend/internal/usecase/backfill/submitter.go`, and focused tests.
- **New APIs**: None.
- **Data model**: No schema or migration changes.
- **Dependencies**: No new dependency; reuse `github.com/google/uuid`.

## Scope
- **In scope**: stable initial attempt identity, automatic retry reuse,
  AlreadyExists convergence, concurrent submitter convergence, legacy
  placeholder normalization, and regression tests.
- **Out of scope**: manual retry semantics beyond preserving the rule that an
  explicit rerun creates a new attempt; cluster fairness, owner propagation,
  backpressure accounting, DLQ reason persistence, and priority API changes.

## Success Criteria
- [ ] Normal materialization creates no abandoned placeholder attempt before
      the first Argo submission.
- [ ] A successful Argo create followed by delayed/failed UID persistence
      converges on the same workflow without creating a second workflow.
- [ ] Two submitter instances selecting the same pending item create at most
      one Argo workflow.
- [ ] AlreadyExists recovery reads the actual UUID-named workflow and
      backfills its UID.
- [ ] Only an explicit operator rerun mints a new run/workflow identity.
