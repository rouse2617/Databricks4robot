# Proposal — CYB-1679

## Why

Pipeline execution records can show impossible estimated cost values, such as a 25-second failed run displaying more than $1M total cost. This makes the product untrustworthy and blocks operators from using estimated cost for triage.

## What Changes

### Modified Capabilities
- Run-level estimated cost will no longer use unsafe pricing fallbacks when node metadata is incomplete.
- Cost estimation will treat Argo `resourcesDuration` conservatively instead of assuming it is direct machine billing time.
- Execution list and run detail will show corrected estimated cost values from the backend.

## Impact

- **Affected code**: `backend/internal/usecase/pipeline/cost.go`, related pipeline usecase tests, execution list/detail cost consumers if formatting needs adjustment
- **New APIs**: none
- **Dependencies**: existing pricing YAML only

## Scope

- **In scope**: backend estimator logic, missing-metadata handling, regression tests, verifying execution list/detail show sane values
- **Out of scope**: actual GCP billing reconciliation, OpenCost integration, historical metrics snapshots, redesigning pricing YAML structure

## Success Criteria

- [ ] Runs without trustworthy instance metadata no longer fall back to GPU pricing by default.
- [ ] Short non-GPU runs no longer display absurd six-figure or seven-figure estimated cost.
- [ ] GPU pricing is applied only when node metadata explicitly indicates GPU resources.
- [ ] Backend tests cover `resourcesDuration` interpretation and missing metadata behavior.
