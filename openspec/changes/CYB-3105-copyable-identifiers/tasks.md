# Tasks — CYB-3105

- [x] `WorkflowExecutionList`: add `copyable` to 名称 cell.
- [x] `WorkflowExecutionList`: add `copyable` to 资产/asset_id column (canonical `AssetIdLink` branch + non-canonical code-text branch; copy icon placed outside the ellipsis so it is not clipped).
- [x] `BatchJobDetailPage`: add `copyable` to 批次 ID and 模板 (header).
- [x] Tier L: `biome check` (touched files clean), related tests 16/16, `npm run build` clean.
- [x] Chrome DevTools MCP (local frontend → Cloud Run dev backend): batch `91ee4db4` header 批次 ID copy puts full UUID on clipboard; 模板 copy icon present; subtask table shows name + asset_id copy icons; main executions list name now copyable; no console errors.
- [ ] PR to dev.
- [ ] Post-deploy confirmation on deployed dev revision.
