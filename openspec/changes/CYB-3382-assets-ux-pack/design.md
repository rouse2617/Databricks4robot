# Design — CYB-3382 Assets 页 UX 优化包

## #1 搜索 DSL 加 `asset_type`

### 现状(`AssetsSearchBar.tsx`)

```tsx
const structuredKeys = [
  { key: "lifecycle_state", ... },
  { key: "algo_status", ... },
  { key: "owner", ... },
  // ← 漏了 asset_type
];
```

Parser 用 `keys` 数组决定哪些 `foo:bar` 被识别为结构化 token,否则整段 fallback 到 `_fulltext`。

### 改动

`structuredKeys` 数组补一项:
```tsx
{ key: "asset_type" }
```

若结构里还有 `values` 建议值,先不做(用户直接输入,不必给下拉 —— 类型列表可以后加)。帮助浮层 `:520` 附近的「可用字段」列表同步补 `asset_type`。

### 边界

- 大小写:parser 现在按字面 key 匹配,大写 `Asset_Type` 不识别;保持一致(不改 parser 语义)
- 空值 `asset_type:`(冒号后无值):parser 现有行为对齐其他字段(应报 chip validation error 或 skip;不改现有行为)

## #2 顶部 facet 折叠

### state

```tsx
const [collapsed, setCollapsed] = useState<boolean>(() => {
  return localStorage.getItem("assets:facet-collapsed") !== "false"; // 默认 true
});
useEffect(() => {
  localStorage.setItem("assets:facet-collapsed", String(collapsed));
}, [collapsed]);
```

### 折叠 UI(灵感)

```
[env=warehouse · algo_status=failed · asset_type=segment · (+1 more)]  [▾ 展开筛选]
```

- 摘要行:遍历当前 active facets/chips,拼一段 pipe-separated 文字(超过 3 项 → `(+N more)`)
- 「展开筛选」按钮触发 `setCollapsed(false)`

展开 UI 是现有 `AssetsFacetSidebar` 组件本身。

### 边界

- 无 active filter 时:摘要行显示「所有资产」或省略行(只留按钮)
- 折叠状态点单个 chip 「叉」应该正常清除该 filter,不用先展开

## #3 KPI 概览卡片

### 组件

新建 `Frontend/src/components/assets/AssetsKpiRow.tsx`:

```tsx
export default function AssetsKpiRow({ data }: { data: AssetsDiscoveryResult }) {
  return (
    <Row gutter={12}>
      <Col span={6}><Statistic title="总数" value={data.total} /></Col>
      <Col span={6}><Statistic title="ready:created" value={ratio(...)} /></Col>
      <Col span={6}><Statistic title="30d 算法失败" value={countAlgoFailed(...)} /></Col>
      <Col span={6}><Statistic title="30d 即将过期" value={countExpiring(...)} /></Col>
    </Row>
  );
}
```

### 数据源

复用 `useAssetsDiscovery`(或其等价 hook)的 fetch 结果:
- 总数 = `result.total`
- ready:created 比例 = `facets.lifecycle_state["ready"] / facets.lifecycle_state["created"]`(整数 or `N/A`)
- 算法失败 = `facets.algo_status["failed"]` 若有(过滤器已加时间窗)
- 即将过期 = 若 `facets.expire_within_30d` 存在则用它,否则 fallback 显示 `—`

### 边界

- fetch 未完成时:card 显示 skeleton loading 而不是 0(避免误导)
- fetch 失败:card 显示 `—`,不 crash
- 不阻塞列表渲染:KPI card 组件失败也要 catch,列表照常显示

### 放置位置

`AssetsPage` layout:
```
[头部 / 搜索栏]
[KPI Row]           ← 新增
[Facet Sidebar]     ← #2 默认折叠
[Results + Preview]
```

## #4 asset_id 副标题

### Cell customRender(表格)

```tsx
{
  title: "Asset ID",
  dataIndex: "asset_id",
  render: (id, record) => (
    <div>
      <Text code copyable>{id}</Text>
      <div>
        <Text type="secondary" style={{ fontSize: 11 }}>
          {record.owner || "—"} · {record.asset_type} · {formatShortDate(record.created_at)}
        </Text>
      </div>
    </div>
  ),
}
```

### 卡片视图

`AssetsCardView` 已有 card 布局;把 asset_id 从纯文字改成上面结构,视觉不再突兀 —— asset_id 仍显眼,但同一区域给了业务上下文。

### 边界

- `owner` 为空时显示 `"—"` 或空 → 不显示占位符
- `created_at` 缺失(理论上应有):fallback `""` 或 `"?"`

## #5 预览侧栏默认态

### Hook

新建 `Frontend/src/hooks/assets/useRecentViewedAssets.ts`:

```tsx
const KEY = "assets:recent-viewed";
const MAX = 5;

export function useRecentViewedAssets() {
  const [recent, setRecent] = useState<string[]>(() => {
    try { return JSON.parse(localStorage.getItem(KEY) || "[]"); }
    catch { return []; }
  });
  const pushRecent = useCallback((id: string) => {
    setRecent(prev => {
      const next = [id, ...prev.filter(x => x !== id)].slice(0, MAX);
      localStorage.setItem(KEY, JSON.stringify(next));
      return next;
    });
  }, []);
  return { recent, pushRecent };
}
```

### 入口

统一在 `AssetQuickPreviewPane` 里 `useEffect` 依赖 `selectedAssetId`:每次 selected id 变化且非 null → `pushRecent(id)`。

其他调用点(如 `/assets/:id` 详情页)可选择也调,但不阻塞本 PR(先小 scope)。

### Empty state UI

```tsx
{!selectedAssetId && (
  <QuickPreviewEmptyState>
    <Empty description="点击行/卡片查看预览" />
    {recent.length > 0 && (
      <>
        <Divider>最近查看</Divider>
        <List>
          {recent.map(id => (
            <List.Item onClick={() => onSelect(id)}>
              <Text code>{id}</Text>
            </List.Item>
          ))}
        </List>
      </>
    )}
  </QuickPreviewEmptyState>
)}
```

### 边界

- `localStorage` 不可用(隐私模式):Hook fallback 到 in-memory `useState`
- `recent` 里的 id 可能已被删除(soft delete):点击后详情页处理不存在,不需要在 Empty state 里 pre-check(避免额外 request)
- 移动端(preview pane 可能不可见):不影响

## 风险 / 回滚

- **风险**:低。5 个改动都是 UI-only,不改数据流 / URL。
- **回滚**:revert 单 commit。localStorage key 不清理(下次再启用可复用状态)。
- CI hard gate:openspec-gate、commitlint、frontend build(全部现有)。

## 不做的通用抽象

- 不建 `KpiCard` 通用组件(用 antd `Statistic` 直接);第二个 KPI 页面出现时再抽
- 不建 `useLocalStorageState` 通用 hook;两处直接读写(#2 collapsed + #5 recent),避免过度工程化
- 不改后端 API
