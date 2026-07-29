# Design — CYB-4445

## 匹配语义：精确 8 位字母数字

`mcap_file_id` 在 schema 层有两个事实：

1. `assets_asset_id_check` / `chk_mcap_file_id_format`（schema CHECK）规定合法形状为 `^[A-Za-z0-9]{8}$`，DB 层兜底。
2. `Frontend/src/components/CmdKSearch.tsx:35` 已使用 `isAssetId = /^[A-Za-z0-9]{8}$/.test(query.trim())` 来直接跳详情。

为与上述两端已经形成的「精确 8 位」契约一致，本次也只接受精确 8 位匹配 —— 不再做 `ILIKE '%<q>%'` 子串模糊。原因：

- 8 位 ID 长度太短，子串模糊容易撞库（例如 `a` 命中 `a0000001` / `00a00000` / `0000a001` 全部），给排查具体文件 ID 的人带来噪声。
- 服务端 fallback 返回 `ILIKE` 还会引入走 trgm gin 或 seq scan 的可能，破坏主键恒定时间特性的预期。
- 客户端补 8 位之前不发送参数，避免「打字到一半就回零结果」的体验。

最后这一点也是为什么前端用 debounce + 强校验，而不是「自动切换」：输入 7 位就跟未过滤等价，输入 8 位立刻收敛到 0/1 行。逻辑无歧义。

## 接口签名变更

`McapFileRepository.List` 当前签名是

```go
List(ctx context.Context, page, pageSize int, ingestState, owner string) (...)
```

AI-RULES.md §6 的 "new interface methods" 条款要求：接口变更时所有实现一次性更新。我们的实现只有 `postgres.McapFileRepo`，测试桩有两处 — `handler_test.go:21` 的 `mockMcapRepo`、`searchindex/builder_test.go:97` 的 `stubMcapRepo` 必须同 commit 改签名。新签名：

```go
List(ctx context.Context, page, pageSize int, ingestState, owner, mcapFileID string) (...)
```

参数尾位 + 不变（不传 `nil` / `*string`），避免改调用方动态行为。

## WHERE 子句拼装

沿用现有 arg-index counter 模式（`repos.go:1159`）。新加一段：

```go
if mcapFileID != "" {
    where += fmt.Sprintf(" AND mcap_file_id = $%d", argIdx)
    args = append(args, mcapFileID)
    argIdx++
}
```

不需要额外索引：PK 已是主键 `mcap_files.mcap_file_id`，恒定时间。

## 服务端校验

`Handler.ListFiles` 在调 `repo.List` 之前对非空 `mcap_file_id` 用同一正则校验；失败时复用一个已经在用的错误体（已经处理过 `mcap_file_id` 的创建/分配场景，`handler.go:260` 的 `CodeInvalidArgument` 与文案一致）：

```go
httpresp.BadRequest(c, httpresp.CodeInvalidArgument,
    "mcap_file_id must be exactly 8 alphanumeric characters", nil)
```

注意：handler 也要支持空字符串（视为不过滤），所以 `if mcapFileID != "" && !valid.RegexMatch(...)` 才返回 400。

## 前端 debounce + 校验

跟现有 `ownerFilter` 同 300ms debounce 节奏（`McapFilesPage.tsx:79`）：

```ts
const [idFilter, setIdFilter] = useState("");
const [debouncedIdFilter, setDebouncedIdFilter] = useState("");
useEffect(() => {
  const t = setTimeout(() => {
    const v = idFilter.trim();
    setDebouncedIdFilter(/^[A-Za-z0-9]{8}$/.test(v) ? v : "");
  }, 300);
  return () => clearTimeout(t);
}, [idFilter]);
```

未达到 8 位时 `debouncedIdFilter` 强制为 `""`，等同未过滤。`mcap_file_id` 改变重置页码到 1（与 owner 同模式）。

`maxLength={8}` 在输入控件侧硬限，避免粘贴超过 8 位后半段被丢弃看不见。

## URL 直跳详情抽屉不变

`McapFilesPage.tsx:66-164` 的 `?mcap_file_id=...` 流程是另一个路径：URL 参数 → 触发单条 `mcapFilesApi.get(id)` → 打开抽屉。本次 list filter 完全不影响它，但保留了作为「已知 ID 也能用 URL 直达」的入口。

## 部署与验证

- 后端：`deploy/cloudrun/backend-dev.sh` 重建 + 部署；migration 列表无变化（无 schema 改动）。
- 前端：`wrangler deploy --env dev`。
- Chrome DevTools MCP 实测：打开 `/mcap-files` → 在新输入框输入 `LEMpjOmB` → 截图确认表格 1 行 → 输入 `LEMpjO` 7 位 → 表格回到原列表。

---

## 决定追溯

| 决策 | 备选 | 理由 |
|------|------|------|
| 精确 8 位匹配 | 子串模糊 `ILIKE '%q%'` / 自动切换 | 与 `CmdKSearch.isAssetId` 一致；主键恒定时间；避免半输入误命中 |
| 服务端也校验 8 位 | 仅前端校验 | 给 curl / SDK 用户一个一致契约；防止任意字符串被下推到 SQL |
| 接口签名末尾追加 `string` | 用 options struct | 当前接口只 5 参；保留调用方静态可读性；后续参数再涨时再统一 |
| 客户端 8 位字符前不下推 | 输入即下推（500ms 内） | 「未达 8 位 = 未过滤」对操作者更直观，避免半输入导致的「结果怎么没了」 |
