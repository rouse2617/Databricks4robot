# CYB-3305 — assets UI/UX 巡检:一致性 cleanups

## Why

2026-07-11 逐页 E2E 巡检 dev 前端(revision `dev#19a7167e`,即 origin/dev HEAD)发现的一批**中/小等级一致性问题**。所有 console 无 error/warn,不阻断使用,但会误导用户判断和让 UX 掉档。汇总为一个 bundled PR,最小改动收敛。

完整巡检报告见 `docs/review/e2e-2026-07-11.md`(本 PR 一起提交)。

## Root cause 概览

- **facet 类型不全**:`AssetsFacetSidebar.tsx` `ASSET_TYPE_OPTIONS` 硬编码 4 类,但 PG 里实际存在 `raw_mcap` / `action` / `frame` 三类;`AddFilterPopover.tsx` 重复硬编码。前端不消费 backend facets 里未列在硬编码的 bucket,facet 侧栏永远看不到这三类。
- **AppLayout `resolveSelectedKey`** 缺 `/api-keys` 分支,进 `/api-keys` 高亮却是「资产管理」(fallback 到默认)。`menuItems` 已包含 `/api-keys`,仅缺 selectedKey 匹配。
- **Dashboard 主力事件类型 KPI 卡** 用超大字号显示 `pipeline_processing` 而没有字号自适应/截断,长值溢出。
- **Dashboard 事件分布图例** 第 4 项与分页箭头相互挤压截断。
- **资产详情标签来源 badge** 一律显示「system」,未按 `source_type` 区分 human/algo/system。
- **MCAP 文件大小列** 已汇总(.mp4)条目显示「—」——只有 raw .mcap 记了 size_bytes。
- **注册中心 · 资源池** 表格 CPU/MEM 全「—」但文案承诺「受 ResourceQuota 约束」——K8s 命名空间未设 quota。
- **Dashboard(湖仓 Iceberg 236)vs `/assets` 列表(PG 230)** 数字差 6,数据源无 UI 说明,用户体感对不上账;Settings 页 PG↔ES Gap 6 也是同一批。

## What Changes

**Frontend**(主要):
1. **单一真相源常量** `ASSET_TYPES`(sidebar 导出),`AddFilterPopover` 从 sidebar `import`。加入 `raw_mcap` / `action` / `frame`。
2. **兜底渲染**:facet 侧栏「资产类型」组遍历 `backend counts` 里的所有 bucket,并集显示;硬编码列表只保证空 backend 时也可用。
3. `AppLayout.resolveSelectedKey` 加 `startsWith("/api-keys")` 分支。
4. Dashboard「主力事件类型」KPI 卡:文字加 `max-width` + `ellipsis` + tooltip 完整值;字号夹到合理范围。
5. Dashboard「事件分布」图例:ECharts `legend.type = "scroll"`,拉宽或缩短第 4 项显示名,避免与分页控件挤压。
6. 资产详情标签 tab 来源 badge:按 `source_type` 三档渲染(`human` / `algo` / `system`),不再一律 system。
7. `McapFilesPage`「大小」列:汇总条目回读 `size_bytes` 若有则展示;否则显示「未记录」而非「—」。
8. 注册中心 · 资源池:若表格无实际 quota 数据,文案改为「未配置 ResourceQuota」,不再承诺约束。
9. `DashboardPage` KPI 面板上方加一行副标题(源 = 湖仓 Iceberg / 按日聚合);`/assets` 列表工具条加副标注(源 = PG 实时)。

**Backend**(最小):
10. `queryexec/elasticsearch` + `queryir` facet 生成:资产类型 bucket 白名单加 `raw_mcap` / `action` / `frame`;如果 backend 之前就是「不限白名单」,则本项零改动、只在前端合并 backend 返回的 bucket。

## Impact

- 纯 UX 一致性 + facet 完整性修复,行为向后兼容。
- 无 schema/migration/auth 改动。
- 前端 `AssetsFacetSidebar.test.tsx` 需更新(常量断言 & 合并渲染断言)。
- 后端若碰 facet compile,加 1 处 `go test` 覆盖(可能仅 config-level 白名单常量)。

## 验收

见 `tasks.md` 的 verification checklist,Chrome DevTools MCP 在 dev 上逐条验证 + 截图。

## Out of scope(另开 issue)

- 🔴 P0:湖仓 Bronze CDC 停摆 10.6 天(80,926 事件积压)—— 由数据/湖仓 owner 单独开 CYB。
- 🟠 `/runs` 视频时长列全空 → $/秒 无法算 —— 单独开 CYB(需接 Grace API 时长)。
- 🟠 算法运行「运行中」孤儿 run 45 天未终结 —— 单独开 CYB(前端 stuck 标识 + 后端超时收敛)。
- 🟡 流水线 342 条 dev 脏数据(镜像 URL 当名字、拼写等) —— 手工清理,非代码 issue。
