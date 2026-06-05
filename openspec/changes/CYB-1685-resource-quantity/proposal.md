# Proposal — CYB-1685

## Why
Pipeline component and node resource forms currently accept bare memory or disk values such as `1`, which Kubernetes interprets as bytes instead of the user's intended `512Mi` / `1Gi` style quantity.

## What Changes

### New Capabilities
- Pipeline component editing SHALL validate memory and disk resource quantities before create/update.
- Pipeline node resource overrides SHALL validate memory and disk quantities before saving the node configuration.
- Backend pipeline/component submission paths SHALL reject unsafe bare memory and disk values even if a client bypasses frontend validation.

### Modified Capabilities
- Resource guidance changes from placeholder-only hints to enforced validation with actionable error messages.

## Impact
- **Affected code**: `Frontend/src/pages/ComponentManager.tsx`, `Frontend/src/components/pipeline/NodeConfigPanel.tsx`, pipeline resource validation helpers/tests, backend pipeline component/run validation paths.
- **New APIs**: None.
- **Dependencies**: None.

## Scope
- **In scope**: CPU validation for Kubernetes-style CPU values, memory/disk unit validation requiring binary or decimal units, GPU integer validation, frontend and backend tests for invalid and valid resource values.
- **Out of scope**: Full resource policy, quota management, cluster capacity admission, cost budgeting, or changing GKE/Argo scheduling behavior.

## Success Criteria
- [ ] Creating/updating a component with `memory=1` or `disk=1` is blocked with a clear error.
- [ ] Saving a pipeline node override with `memory=1` or `disk=1` is blocked with a clear error.
- [ ] Valid examples such as `512Mi`, `1Gi`, `20Gi`, `500m`, `1`, and GPU `1` continue to work.
- [ ] Backend rejects unsafe bare memory/disk values so API callers cannot bypass UI validation.
