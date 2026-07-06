# Design — CYB-3061

## Architecture Context
- **Constraints**:
  - Go 1.25+; Argo Workflows v3.7.14 (`github.com/argoproj/argo-workflows/v3` — verified in `go.mod` / module cache)
  - Kubernetes label values must match `(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?`, max 63 chars — `@` (present in email-format `Owner` values) is illegal
  - GKE Cost Allocation drops **all** labels on a pod if that pod has more than 50 total labels (verified against GCP docs)
- **Goals**: real GCP billing data (via GKE Cost Allocation → BigQuery, once export is configured) sliceable by batch/task-type/owner with no application-side join
- **Non-Goals**: fixing the existing YAML-estimator cost bug; configuring BigQuery export; labeling historical/already-running workflows

## Affected Modules
- `backend/internal/transpiler/transpiler.go` — add `Options.PodLabels map[string]string`; set `wf.Spec.PodMetadata` in `Transpile()` when non-empty
- `backend/internal/usecase/pipeline/usecase.go` — in `Deploy()`, populate `wfOpts.PodLabels` from `DeployOptions.BatchJobID` / `TemplateID` / `Owner` via a new `buildCostTrackingLabels()` + `sanitizeLabelValue()` helper pair

## Architecture Decisions

### Decision 1: Set labels via `WorkflowSpec.PodMetadata`, not per-Template
- **Approach**: use `wf.Spec.PodMetadata = &wfv1.Metadata{Labels: ...}` — verified via the vendored `v1alpha1/workflow_types.go` (`WorkflowSpec.PodMetadata *Metadata`, doc comment: "defines additional metadata that should be applied to workflow pods"). This applies uniformly to every pod Argo creates for every step.
- **Alternative**: follow the existing `TemplateNodeSelector`/`TemplateTolerations` pattern (`applyTemplateSchedulingDefaults`, applied per-`wfv1.Template`)
- **Rationale**: cost-tracking labels must be identical across every step of a run (so BigQuery `GROUP BY` rolls up the whole run/batch); the per-Template pattern exists because node placement legitimately varies per step (GPU vs CPU nodeSelector) — that variability doesn't apply here
- **Trade-off**: none identified — `PodMetadata` is additive and does not interact with per-template scheduling hints

### Decision 2: Sanitize at the `usecase.go` call site, not inside `transpiler.go`
- **Approach**: `buildCostTrackingLabels()`/`sanitizeLabelValue()` live in `usecase/pipeline`, producing an already-valid `map[string]string` that `transpiler.Options.PodLabels` accepts as-is
- **Alternative**: push raw values into `Options.PodLabels` and sanitize inside `Transpile()`
- **Rationale**: `transpiler` package is a pure DAG→Workflow compiler with no domain knowledge of what a "batch job ID" or "owner" is; keeping sanitization at the usecase layer (which already owns `DeployOptions`) avoids leaking domain semantics into the transpiler and keeps `Transpile()`'s contract simple ("labels in, applied verbatim")
- **Risk**: if a future caller passes an already-invalid map directly to `transpiler.Options.PodLabels` bypassing the helper, workflow submission fails at the Kubernetes API layer (fail-fast, not silent) — acceptable, matches existing error-handling posture elsewhere in `Transpile()`
- **Rollback**: revert both call sites; `PodMetadata` unset is a no-op, zero data migration needed

### Decision 3: Skip empty-valued keys rather than emit empty-string labels
- **Approach**: `buildCostTrackingLabels()` omits a key entirely when its source value is empty (e.g. non-batch ad-hoc runs have no `BatchJobID`)
- **Alternative**: always emit all three keys, empty string when unknown
- **Rationale**: an empty-string label value is syntactically legal but semantically useless noise in BigQuery `GROUP BY` output; omitting keeps the label set minimal and every present key meaningful
- **Trade-off**: BigQuery queries checking "does this pod have a batch-job-id label at all" must use `IS NULL`-style absence checks rather than `= ''`

## Data Flow
```
Deploy(ctx, pipelineArg, name, assetIDs, opts DeployOptions)
  └─ buildCostTrackingLabels(opts.BatchJobID, opts.TemplateID, opts.Owner)
       └─ sanitizeLabelValue(...) per field   [strips/escapes illegal chars, caps 63]
  └─ wfOpts := &transpiler.Options{ ..., PodLabels: <sanitized map> }
  └─ transpiler.Transpile(pipe, wfOpts)
       └─ wf.Spec.PodMetadata = &wfv1.Metadata{Labels: opts.PodLabels}   [if non-empty]
  └─ Argo Workflow submitted → every step pod inherits these labels
       └─ (separately, already enabled) GKE Cost Allocation → Cloud Billing BigQuery export
            picks up `k8s-label/cyber-databrew-*` dimensions automatically
```

## Risks / Trade-offs
| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| `Owner`/other field contains characters beyond what `sanitizeLabelValue` anticipates | malformed label value, pod creation rejected by K8s API | unit tests cover email, empty, >63 chars, arbitrary illegal chars; sanitizer strips to the legal charset and trims non-alnum ends rather than passing through unknown input |
| Per-pod label count creeps toward the 50-label GKE Cost Allocation cap over time (future unrelated labels added) | silent loss of **all** cost-allocation label data for that pod | current total is 4 (Argo ×2 + GKE topology ×2) + 3 new = 7; flagged in proposal's Success Criteria as an explicit check, not just an assumption |
| `build_workflows.go` path (component_build/rag_build) never gets these labels | those workflow types stay unattributed in cost breakdowns | explicitly out-of-scope (proposal.md); low cost impact since they are not the GPU-heavy cost driver this session's audit identified |
