# CYB-3297 Phase D — raw_mcap 实时入索引

## Why
mcap 入库时创建的占位 raw_mcap 资产只发了 `mcap_file_created` 事件（`AggregateType=mcap_file`、无 `asset_id`），ES 订阅者对空 `AssetID` 的事件直接丢弃（`es_subscriber.go:122`）。→ 新入库的 raw_mcap 在**下次全量 reindex 前搜不到**（现存 74 个是靠周期 reindex 补的）。

## What Changes
`internal/handlers/mcap/handler.go` `createFileTx`：插入占位 raw_mcap 后，**额外发一个 asset-scoped `asset_created` 事件**（`AssetID=占位资产ID`、`AggregateType=asset`），与 usecase 里普通资产创建的发事件方式一致 → 订阅者立即 Build 并索引该 raw_mcap。

## Impact
- Affected code: `internal/handlers/mcap/handler.go`（handler，非 off-limits）+ 新增 `rawmcap_index_test.go`
- 不动 `internal/outbox/` 内部（只是通过 eventRepo.Append 写一条 outbox 行，属正常应用代码）。
- 无 schema 变更。
- **验证不受 CYB-3298(b) 影响**：无论新/旧 builder，订阅者都会索引 raw_mcap（本变更只保证事件被发出、不被丢弃）。

## 验证（dev）
- 通过 API 新建一个 mcap-file → 立即在搜索里出现对应 raw_mcap（无需等 reindex）。
