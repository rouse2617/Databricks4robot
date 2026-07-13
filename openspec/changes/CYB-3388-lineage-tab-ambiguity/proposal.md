# CYB-3388 — 资产详情 血缘 Tab 语义澄清

## Why
用户走查资产详情页(以 `qrZQGxpm` 这个 raw_mcap 资产为例)提了 5 个歧义,3 个是硬伤:

### A. 自身 ID = 上游 MCAP File ID(信任杀手)
Header `qrZQGxpm` = 「上游 MCAP File ID: qrZQGxpm」—— 自己指向自己。 底层原因:`raw_mcap` 类型 asset 与 `mcap_file` 共用同一 id(设计如此,CYB-3376 走过这条路径),不是 bug。 但 UI 直接展示"上游 = 自己"让研究员认为"血缘出错了"。

### B. 一级 Tab 与「血缘·下游」重复
一级 Tab 已有「算法处理 (1)」「交付历史」「评测与指标 (0)」;「血缘」Tab 的下游卡里又列了「算法处理 / 交付记录 / 评测结果」。 用户困惑:该点顶部 Tab 还是血缘卡?两处一致吗?**血缘下游本应只是实体子资产**(那 7 个 segment)。

### C. 状态自相矛盾 —— 上游 `pending` vs 下游 `algo ok`
`data.upstream.ingest_state` 是 `mcap_files.ingest_state` 字段,值 `pending`。 但下游已有 `transcode@1.0.0 ok` + 7 个 segment。 从数据管道看这违反时间顺序。

实测 dev raw_mcap 分布:**79 条 mcap 中 22 条 pending 但已有 segment 子资产**(28% 概率触发)。 Root cause 是 external migration(grace-migration import_batch)插入 mcap 时**跳过了 `POST /mcap-files/:id/finalize` 端点**,`ingest_state` 停在默认 `pending`,后续也没被 backfill。

### D. 布局失衡
LineageTab 写死 `maxWidth: 720`,页面右侧 60% 空白,像"没加载完"。

### E. 视觉语义模糊
视频容器下方 `<Badge status="success" text="视频预览">` + `<Button>完整预览</Button>` 视觉上像并排 tab,但绿点其实是"视频已加载"状态指示,与"完整预览"按钮无联动关系。

## What Changes

### 前端 `LineageTab.tsx`

1. **A 修**:接受 `assetType` prop。 当 `assetType === 'raw_mcap'`(即当前 asset 就是 mcap 源头):
   - 上游卡改标题为「**存储**」(而不是「上游」)
   - 不显示「MCAP File ID」行(冗余,顶部 Header 已有)
   - 不显示「入库状态」行(顺便消除 C)
   - 保留「存储 URI」(GCS 路径,用户实际关心)

2. **B 修**:下游卡 **只保留 `children`(子资产)** 一节,标题改为「**子资产**」;移除 `algo_results` / `deliveries` / `eval_results` 三节 —— 一级 Tab 已承担。

3. **D 修**:删除 `maxWidth: 720`,用 antd `<Row gutter>` 两栏布局。 「Pipeline 血缘」+「存储」左侧 Col span=10;「子资产」右侧 Col span=14。 mobile(`isNarrow`)自适应 100%。

### 前端 `AssetPreviewHero.tsx`

4. **E 修**:`Badge status="success" text="视频预览"` → `text="已就绪"`(明确是状态,不误认作 tab)。 「完整预览」按钮不变。

### 数据存量(C 存量补救)

5. **不改代码**:dev 22 条 pending 但已有 segment 的 mcap 属于历史脏数据。 后续如需清理,附一个 SQL 脚本:
   ```sql
   UPDATE mcap_files SET ingest_state = 'summarized', updated_at = now()
   WHERE ingest_state = 'pending'
     AND mcap_file_id IN (SELECT DISTINCT mcap_file_id FROM assets WHERE asset_type = 'segment' AND mcap_file_id IS NOT NULL);
   ```
   由运维手动 dev 上执行(需 PG psql 直连)。 A 修生效后 UI 不再露出 ingest_state,这步只是数据卫生,非必需。

## Impact

- **面向用户**:资产详情页血缘 Tab 显示与主 Tab 组的语义清晰分离;raw_mcap 类型不再"自指";入库状态错误提示不再出现
- **无后端 API 改动** — /lineage endpoint 数据字段不变,前端选择性渲染
- **无 grace-migration 逻辑改动** — 存量 22 条 dev 数据可选 SQL 清理
- **前端 test**:LineageTab 单测补 raw_mcap / 非 raw_mcap 两分支

## 验证
- Tier L:tsc + biome + LineageTab test 全绿
- dev 回归:
  - Chrome MCP 打开 `/assets/qrZQGxpm`(raw_mcap)→ 上游卡标题「存储」,只显示 URI;下游卡只有子资产 7 条
  - 打开一个 segment 类型 asset → 上游卡显示 MCAP File ID(仍展示,因为 segment ≠ 源头);下游卡只有子资产
  - 布局占满宽度,右侧无 60% 空白
  - AssetPreviewHero「已就绪」+ 绿点显示
