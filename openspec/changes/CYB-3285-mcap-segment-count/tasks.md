# Tasks — CYB-3285

## Context files
- `backend/internal/postgres/repos.go:1035-1130` — `McapFileRepo.List`(SELECT + scan);`:833` — `Get` 的 SELECT(同步加)
- `backend/internal/models/` — `McapFile` struct(加字段)
- `Frontend/src/pages/McapFilesPage.tsx:192-205` — 通道数 / 分块数 列
- `Frontend/src/api/types.ts:89` — `McapFile` type
- `scripts/api-guide-smoke.sh` — mcap-files 端点回归

## Implementation
### Backend
- [ ] [backend] `models` — `McapFile` 加 `SegmentCount int64 \`json:"segment_count"\``
- [ ] [backend] `repos.go` `McapFileRepo.List` — SELECT 末尾加 `(SELECT COUNT(*) FROM assets a WHERE a.mcap_file_id = mcap_files.mcap_file_id AND a.asset_type='segment' AND a.is_deleted=FALSE)`;scan 加 `&f.SegmentCount`
- [ ] [backend] `repos.go` `McapFileRepo.Get`(:833)— 同样加子查询 + scan(避免 Get 返回恒 0)
### Frontend
- [ ] [frontend] `api/types.ts` — `McapFile` 加 `segment_count?: number`
- [ ] [frontend] `McapFilesPage.tsx` — 删 channels + chunks 两列;加「子 segment 数」列(`dataIndex: "segment_count"`,`render: v => v ?? 0`);清理 `COLUMN_LABELS.channels/chunks` 若不再引用

## API contract sync (mandatory — mcap list response 加字段)
- [ ] `api/openapi.yaml` — `McapFile` schema 加 `segment_count: {type: integer}`
- [ ] `docs/review/api-guide.md` — mcap-files 段注明 `segment_count`(子 segment 数)
- [ ] `scripts/api-guide-smoke.sh` — mcap-files 检查项确认响应含 `segment_count`(如脚本已覆盖该端点)
- [ ] SDK:无需改(dict passthrough;在 PR 描述注明)

## Local verification (Tier L — API 契约字段 + 前端列)
- [ ] `cd backend && make fmt && make vet && go build ./... && go test ./internal/postgres/... ./internal/handlers/mcap/...`
- [ ] `cd Frontend && npm run lint && npm run build`

## Deploy verification
- [ ] backend + frontend 部署 dev(PR → CICD deploy-dev)
- [ ] backend smoke:`curl $BASE/api/v1/mcap-files` 确认每条含 `segment_count`;对已知有 segment 的 mcap(如 6EDE33F6)确认 count 正确(与 `queries/run` 数 segment 对得上)
- [ ] [frontend] Chrome DevTools MCP:MCAP 文件页 → 列头无「通道数/分块数」、有「子 segment 数」且有值;Console 无 error;截图存 change dir
- [ ] 回归:资产列表(#2)/详情(#3)不受影响

## PR
- [ ] 标题:`feat(backend): mcap list child-segment count, drop channels/chunks columns (cyb-3285)`
- [ ] body:Linear CYB-3285 / OpenSpec / API contract sync 勾项 / 部署验证证据;注明 SDK 无需改、无 migration
