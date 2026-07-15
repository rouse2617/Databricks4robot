# Decisions — CYB-3076

## 2026-07-07 — Final pod name = <run-id>-step-<component>[-<uuid8>]-<argoHash>
- **workflow name = run id (depID)**: change `wfName` from `<pipeName>-<uuid8>` to the run id (depID, known at transpile; PreallocatedRunID honored). So `pipeline_runs.workflow_name == run.id`, and the pod-name prefix IS the run id → from any pod you get the run URL `/runs/<prefix>` directly. Confirmed safe: only `pod_name.go:79` parses the wf name and it just strips `workflowName+"-"` (no pipeName-format assumption). A bare UUID is a valid DNS-1123 name. Trade-off (accepted by user): Argo UI lists workflows by UUID, not `grace-sync-<ts>`.
- **template name = `step-<component-slug>`**, appending `-<uuid8>` ONLY when the slug is duplicated within the workflow (keeps clean names for the common unique-component case; guarantees template-name uniqueness which Argo requires — the Argo hash only dedups pods, not templates). Keep the `step-` prefix (marker + exit-notify exclusion + matching).
- **随机id** = Argo's auto-appended pod hash (not ours).
- **dual-format matching** (candidate keys {new stepKey, legacy step-<nodeID>}) so historical runs' summary/labels/runtime-info don't degrade. No migration.
- run-id pod label NOT added — the run id is now in the pod name itself.

## 2026-07-07 — Centralized, no scattered branches
- Single dup-aware builder `buildStepTemplateNames(nodes) map[nodeID]name` in the transpiler is the one source of the format; all template builders + validation consume the map (not recompute). Fix the output-param wiring at transpiler.go:620 to use the map. Backend matchers + frontend replicate the same slug via a shared/mirrored helper. Legacy branch removable once old runs age out.
