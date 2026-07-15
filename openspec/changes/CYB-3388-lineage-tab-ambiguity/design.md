# Design — CYB-3388

## 方案:纯前端 UI 层重构

### 为什么不动数据模型 / 后端
- **A 底层**:raw_mcap 与 mcap_file 共用 id 是 CYB-3376 阶段确认的设计,与其他 asset_type 生态一致,不能改
- **B 底层**:下游数据 `algo_results / deliveries / eval_results` 是 `/lineage` endpoint 返回的完整链路,别的调用方可能仍在用(比如 pipeline-lineage 卡);前端只做**渲染裁剪**,不动 endpoint 结构
- **C 底层**:pipeline 完成回填 `mcap.ingest_state = summarized` 只在 `POST /mcap-files/:id/finalize` 触发。 grace-migration 是外部导入没走这条路径 —— 属于集成侧的规范问题,不由 databrew 后端强制。 前端隐藏字段是最小侵入

### 前端渲染裁剪的关键判断
- `assetType === 'raw_mcap'` → 当前 asset 是"源头":隐藏"上游 mcap_file_id / ingest_state",保留"存储 URI"
- 非 raw_mcap(segment / action / frame / clip …)→ 上游卡照常显示 mcap 元信息 —— 因为 mcap 与自己**不是**同一 id,不构成"自指"
- 下游卡:无论 assetType,只显示 children —— algo/delivery/eval 都归主 Tab

## Layout(D 修)

```tsx
<Row gutter={16}>
  <Col xs={24} md={10}>
    {/* Pipeline provenance 卡(如有)*/}
    {/* 存储 / 上游 卡 */}
  </Col>
  <Col xs={24} md={14}>
    {/* 子资产列表 */}
  </Col>
</Row>
```

`maxWidth: 720` 彻底删除;窄屏 fall back 到单列。

## E:Badge label 语义

现状:
```tsx
<Badge status={badge.status} text={<Text>视频预览</Text>} />
<Button icon={<ExpandOutlined />}>完整预览</Button>
```

改为:
```tsx
<Badge status={badge.status} text={<Text>已就绪</Text>} />
<Button icon={<ExpandOutlined />}>完整预览</Button>
```

「已就绪」明确是状态,绿点 = 状态色。 与「完整预览」按钮语义分离清楚。

## 影响面

**改**
- `Frontend/src/components/asset-detail/LineageTab.tsx` — 主要
- `Frontend/src/components/asset-detail/AssetPreviewHero.tsx` — Badge label 一处
- `Frontend/src/pages/AssetDetailPage.tsx` — 传 assetType

**不改**
- 后端 `/lineage` endpoint / `/mcap-files/:id/finalize` / grace-migration
- 数据结构 `LineageData` interface(前端已定义,字段全保留;仅渲染选择性)
- 血缘 endpoint 后端 SQL

## Rollback

git revert 单 commit 即可。 各 Tab 数据不缺失(algo/delivery/eval 在主 Tab 都能看到)。
