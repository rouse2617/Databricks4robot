# Tasks — CYB-3061

## Context files
```
backend/internal/transpiler/transpiler.go       # Transpile(), Options struct — where PodMetadata gets set
backend/internal/usecase/pipeline/usecase.go     # Deploy(), DeployOptions, wfOpts construction (~L3244)
backend/config/gcp_pricing.yaml                  # background only — the existing (buggy) cost estimator this labels-first approach complements
```

## Implementation

- [ ] [backend] `transpiler.go`: add `PodLabels map[string]string` field to `Options` struct
- [ ] [backend] `transpiler.go`: in `Transpile()`, after the `ImagePullSecrets` block, set `wf.Spec.PodMetadata = &wfv1.Metadata{Labels: opts.PodLabels}` when `len(opts.PodLabels) > 0`
- [ ] [backend] `usecase.go`: add `sanitizeLabelValue(raw string) string` — replaces `@` with `-at-`, replaces any other illegal character with `-`, truncates to 63 chars, trims leading/trailing `-_.`
- [ ] [backend] `usecase.go`: add `buildCostTrackingLabels(batchJobID, templateID, owner string) map[string]string` — emits `cyber-databrew/batch-job-id`, `cyber-databrew/template-id`, `cyber-databrew/owner`, skipping any key whose sanitized value is empty
- [ ] [backend] `usecase.go`: in `Deploy()`, set `wfOpts.PodLabels = buildCostTrackingLabels(opts.BatchJobID, opts.TemplateID, opts.Owner)` (aggregate across `opts ...DeployOptions` the same way existing fields on `wfOpts` are sourced)

## Scenario coverage (tests)

- [ ] [backend] Unit test: batch run with known batch-job/template/owner → all three labels present, expected keys — covers *"Batch run submitted with a known batch job, template, and owner"*
- [ ] [backend] Unit test: ad-hoc run with empty `BatchJobID` → no batch-job-id key emitted, other known keys still present — covers *"Ad-hoc run submitted with no batch association"*
- [ ] [backend] Unit test: `sanitizeLabelValue` on an email string (`ruipeng.huang@cyberorigin.ai`) → valid label value, no `@` — covers *"Owner identifier is an email address"*
- [ ] [backend] Unit test: `sanitizeLabelValue` on a >63-char string and a string with illegal characters (e.g. spaces, `/`) → truncated to ≤63 chars, illegal chars replaced, no leading/trailing separator — covers *"Identifier exceeds the Kubernetes label value length limit or contains unexpected characters"*

## API contract sync
N/A — internal-only change. No new/changed HTTP route, handler, request/response shape, or status code.

## Verification (Tier L)
- [ ] `make fmt && make vet`
- [ ] `go test ./internal/transpiler/... ./internal/usecase/pipeline/...`
- [ ] `go test ./...` (full suite, per this session's established practice given prior live-deploy incidents)

## Deploy verification
- [ ] Deploy backend to Cloud Run dev per `deploy-verification.md` §2.0; confirm the serving revision's build ref matches this change's commit
- [ ] Submit one real pipeline run (or reuse an existing small batch) through the deployed dev backend
- [ ] `kubectl -n <target-namespace> get pod -l workflows.argoproj.io/workflow=<new-run's-workflow-name> -o jsonpath='{.metadata.labels}'` — confirm `cyber-databrew/batch-job-id` / `cyber-databrew/template-id` / `cyber-databrew/owner` are present with sanitized values (proves the specific behavioral change is active on the deployed revision, per pitfall P5)
- [ ] Confirm total label count on that pod stays under the 50-label GKE Cost Allocation cap (should be 7: 2 Argo + 2 GKE topology + 3 new)
