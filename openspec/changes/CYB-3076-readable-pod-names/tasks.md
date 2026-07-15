# Tasks — CYB-3076

Branch: `feat/CYB-3076-readable-pod-names` (from `origin/dev`). Tier **M**.

## 1. Shared key/slug (single source of format)
- [ ] [backend] Add `stepKey(componentName, nodeID) string` + `legacyStepKey(nodeID)` (transpiler pkg). `stepKey` = `step-<slug>-<uuid8>`; slug via `sanitizePodNamePart` truncated ~30; uuid8 = first 8 hex of node UUID (strip `node-`). Empty component → legacy `step-<nodeID>`.
- [ ] [backend] unit tests: slug/sanitize, uuid8, empty-component fallback, RFC1123 validity.

## 2. Transpiler
- [ ] [backend] `templateName` → derive from node (component + id); update callers `transpiler.go:235,488,680,684,729` and `validation.go:145` to pass the node/component.
- [ ] [backend] Fix output-param wiring `transpiler.go:620` to use `nodeTemplates[is.srcNode]` (map) instead of recomputing (no component there). (Scenario: DAG stays internally consistent.)
- [ ] [backend] Collision handling when building the nodeID→template map: on duplicate key, fall back to a longer uuid for the colliding node. (Scenario: duplicate component names still transpile)
- [ ] [backend] transpiler test: readable template + pod-name shape; duplicate-component uniqueness; output-param refs use the new names.

## 3. Backend matchers → dual-format (Scenario: legacy run still works)
- [ ] [backend] backfill `batchNodeOrderFromPipeline`/`normalizeBatchPipelineNodeID`: build candidate keys per definition node (`stepKey(component,id)` + legacy) and match stored `pipeline_node_id`; needs `component.name` (currently only reads `node["id"]`).
- [ ] [backend] workflow `lookupWorkflowNodeRuntimeInfo` (handler.go:377): match via new stepKey AND the existing legacy TrimPrefix path.
- [ ] [backend] tests for both matchers with new + legacy stored ids.

## 4. Frontend
- [ ] [Frontend] `workflowNodeDisplay.ts` `addPipelineNodeLabelVariants`: register the `step-<slug>-<uuid8>` variant (mirror the backend slug logic) so DAG/timeline show the exact component name for new runs; legacy variants kept.
- [ ] [Frontend] test: new-format template name resolves to component label.

## 5. Verify (dev, after deploy)
- [ ] New run: `kubectl get pods` shows `…-step-<component>-<uuid8>-<hash>`; step identifiable.
- [ ] New run: batch node-summary ordered by DAG; run detail shows component names; workflow runtime info present.
- [ ] Legacy run (pre-change): node-summary order / labels / runtime info unchanged.
- [ ] Duplicate-component pipeline transpiles and runs.

## 6. API contract sync
- N/A — no HTTP API change (response field values differ; shapes unchanged). No migration.

## 7. Verification tiers
- [ ] Tier M: `make fmt && make vet`; `go test ./internal/transpiler/... ./internal/usecase/backfill/... ./internal/handlers/workflow/...`; frontend `tsc --noEmit` + related test.
