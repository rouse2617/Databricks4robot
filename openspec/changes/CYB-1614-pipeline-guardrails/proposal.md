# Proposal — CYB-1614

## Why
Real complex Argo workflow smoke found that DataBrew still accepts pipeline/component shapes that later fail in Argo or produce confusing execution detail UX.

## What Changes

### New Capabilities
- Pipeline authoring validates duplicate fan-in target ports before users submit a run.
- Component authoring normalizes shell script commands into a stable `sh -c <script>` shape.
- Component output behavior is explicit enough that declared output ports do not surprise users with Argo file-output failures.

### Modified Capabilities
- Pipeline run creation/transpilation fails early with clear validation errors for duplicate input bindings.
- Workflow logs render as readable multi-line output in the UI.
- Cost summary UI makes recently syncing or partially available cost data understandable instead of looking like missing data.

## Impact
- **Affected code**: `backend/internal/transpiler`, `backend/internal/usecase/pipeline`, `Frontend/src/pages`, `Frontend/src/components/pipeline`
- **New APIs**: None
- **Dependencies**: None

## Scope
- **In scope**: fan-in target validation, shell command/args normalization, output-port guidance/guardrails, cost syncing UX, workflow log formatting.
- **Out of scope**: exact GCP Billing reconciliation, new pipeline schema migrations, Argo watch-stream replacement, asset-driven run redesign.

## Success Criteria
- [ ] A pipeline with multiple edges targeting the same node input is blocked before Argo submission with a clear error.
- [ ] Components created or edited with `sh -c` store only the script body as args when the command already contains `-c`.
- [ ] Users get a clear warning or backend protection when declared outputs require `/tmp/outputs/<port>`.
- [ ] Recently completed runs show cost sync status clearly while node cost summary is still catching up.
- [ ] Workflow logs render as separate readable lines in the detail UI.
