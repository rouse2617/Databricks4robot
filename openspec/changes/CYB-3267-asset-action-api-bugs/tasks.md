# CYB-3267 Tasks

## Setup
- [x] Linear CYB-3267
- [x] Branch `fix/CYB-3267-asset-action-api-bugs` from `origin/dev`
- [x] Verify both bugs against live dev (Bug 1 real 500 / UI error; Bug 2 already fixed by CYB-3226 — duration_sec=2 on dev)
- [x] OpenSpec proposal / spec delta / tasks

## Implementation
- [x] [backend] `postgres/actions.go` `scanAction`: add `&a.TaskID,` between `&a.ExternalID` and `&a.TenantID` (Bug 1)
- [x] [backend] `usecase/asset/usecase.go` `Create`: add `DurationMs: (end-start)/1_000_000` (Bug 2, defensive; align with CreateChildAsset)
- [x] [backend] `postgres/actions_test.go` `TestScanAction_ColumnDestinationAlignment`: 22-value scan → `TaskID` set, `tenant_id`/`project_id` not shifted, errors on count mismatch
- [x] [backend] `usecase/asset/usecase_ingest_test.go` `TestCreate_SetsDurationMsFromSpan`: mock-path Create sets `DurationMs=2000` (isolates create-site fix)

## Verification
- [x] [backend] Tier M: gofmt + `go build ./...` + `go vet` + `go test ./internal/postgres/... ./internal/usecase/asset/...` 通过
- [x] [test] `scripts/smoke-asset-actions-dev.sh`: create asset → create action → `GET /assets/:id/actions` 200 + total≥1 (was 500)
- [ ] deploy dev (GHA on merge) → verify: Action 时间轴 tab loads (no 500); `GET /assets/:id/actions` 200
- [ ] **Chrome DevTools MCP**: asset detail → Action 时间轴 renders without the scan error
- [ ] PR → merge → deploy-verify

## API contract sync (bug fix — response shapes unchanged)
- [x] No OpenAPI shape change (task_id already part of the Action schema; behavior-only fix)
- [ ] api-guide: note actions list now returns task_id / no 500 (if a section exists)
- [ ] spec delta Given/When/Then (done)

## 收尾
- [ ] Linear CYB-3267 → Done, commit hash
