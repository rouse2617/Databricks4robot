# Tasks — CYB-1018 UI (CYB-1031)

- [x] Linear [CYB-1031](https://linear.app/cyberorigin/issue/CYB-1031) + OpenSpec proposal
- [x] Design doc (ui-ux-pro-max) → `docs/review/unified-asset-catalog/design/algo-runs-ui.md`
- [x] `api/algoRuns.ts` + `RunIdLink` + `lib/runId.ts`
- [x] Wire AlgoTab, VersionProvenanceTab, AlgoStatusPopover
- [x] Unit test + `npm run build`
- [x] Deploy frontend dev + Chrome verify

## Deploy record — CYB-1031

| Service | Image tag | Cloud Run revision | URL |
|---------|-----------|-------------------|-----|
| frontend-dev | `cyber-databrew-frontend:0be76c2` | `cyber-databrew-frontend-dev-00185-jbf` | https://cyber-databrew-frontend-dev-wtttm6suaq-uc.a.run.app |

## Chrome verify (2026-05-23)

| Step | Expected | Actual |
|------|----------|--------|
| 资产 `9KnuP7F3` → 算法处理 Tab | 列「来自 run」 | OK；无 run 显示 `—` |
| 同上，`body_tracking` 带 16 位 run_id | 可点击短链 + Popover | OK `LRdl…tE7` → hand_track@2.0 ok |
| 版本与溯源 | legacy `cyb1013-verify` 纯文本 | OK（非 16 位不链） |
| Console | 无 error | OK |

截图：`deploy-verify-algo-tab.png`, `deploy-verify-run-popover.png`

- [ ] PR → `dev`; Linear CYB-1031 → Done
