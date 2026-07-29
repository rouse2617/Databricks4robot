# UI design — CYB-4470 BatchJobList + WorkflowExecutionList

## 0. 共享 token

两个页面统一使用：

```ts
// 提议新建：Frontend/src/lib/designTokens.ts
export const STATUS_TAG_PALETTE = {
  success:    { bg: "#f6ffed", border: "#b7eb8f", text: "#389e0d" },
  error:      { bg: "#fff1f0", border: "#ffa39e", text: "#cf1322" },
  warning:    { bg: "#fff2e8", border: "#ffbb96", text: "#d4380d" },
  paused:     { bg: "#fafafa", border: "#d9d9d9", text: "#8c8c8c" },
  neutral:    { bg: "#fafafa", border: "#d9d9d9", text: "#8c8c8c" },
  // 系统/版本类
  system:     { bg: "#e6f4ff", border: "#91caff", text: "#1677ff" },
} as const;

export const PROGRESS_COLORS = {
  failure:         "#cf1322",  // 暗红（不再是饱和红）
  partial_failure: "#fa8c16",  // 橙
  completed:       "#52c41a",  // 绿
  running:         "#1677ff",  // 蓝（动态）
  default:         "#d9d9d9",  // 灰
} as const;
```

---

## A. BatchJobList

### A1. 资源池列（同 CYB-4470 原 design-ui）

```tsx
{
  title: "资源池",
  key: "target",
  width: 160,
  filterMultiple: true,
  filters: targetFilterOptions,
  onFilter: (value, record) => batchTargetId(record.filterJson) === value,
  sorter: (a, b) => targetNameOf(a).localeCompare(targetNameOf(b)),
  render: (_, record) => {
    const tid = batchTargetId(record.filterJson);
    const target = tid ? targetMap.get(tid) : undefined;
    if (!target) return <span className="text-muted">N/A</span>;
    const color =
      target.status === "unavailable" ? "error" :
      target.enabled === false ? "default" :
      "success";
    return (
      <Tooltip title={`cluster=${target.cluster} · ns=${target.namespace}`}>
        <Tag color={color}>{target.name}</Tag>
      </Tooltip>
    );
  },
}
```

### A2. 列宽重平衡

| 列 | 当前 | 调整 |
|---|---|---|
| 批次名称 | 320 | 340 |
| 模板 | 260 | 260 |
| 资源池 | — | 160 |
| 耗时 | 100 | 80 |
| 所属用户 | 160 | 140 |

### A3. 空态字符

```tsx
<span style={{ color: "#bfbfbf", fontStyle: "italic" }}>—</span>
```

### A4. Status Tag 配色

| 状态 | Tag color | 含义 |
|---|---|---|
| completed | success | 全部完成 |
| failure | error | 全部失败 |
| partial_failure | warning | 部分失败 |
| paused | paused | 暂停/挂起 |
| 普通 | neutral | 默认 |

### A5. Progress Bar 失败色

`derived='failure'` → `#cf1322`（暗红）
`derived='partial_failure'` → `#fa8c16`（橙）

---

## B. WorkflowExecutionList

### B1. 复制图标 Hover 显现

```tsx
// 抽出 <CopyableCell> 组件
export function CopyableCell({ value }: { value: string }) {
  return (
    <span
      style={{
        display: "inline-flex",
        alignItems: "center",
        gap: 4,
      }}
    >
      <span>{value}</span>
      <CopyOutlined
        style={{
          opacity: 0,
          cursor: "pointer",
          fontSize: 12,
          transition: "opacity 0.15s",
        }}
        onClick={(e) => {
          e.stopPropagation();
          void navigator.clipboard
            .writeText(value)
            .then(() => message.success("已复制"))
            .catch(() => message.error("复制失败"));
        }}
        className="copy-on-hover-target"   // CSS: 父 hover 时 opacity: 1
      />
    </span>
  );
}
```

```css
/* Frontend/src/index.css 或同位置追加 */
.copy-cell:hover .copy-on-hover-target {
  opacity: 1 !important;
}
```

**备选方案**（点击即复制 + Toast）：直接 `onClick` 整段文本复制，无需图标。

### B2. 资产 ID 去前缀

```tsx
// line 396 渲染时
const assetIdValue = (labels.asset_id ?? "").replace(/^asset_id=/, "");
return <CopyableCell value={assetIdValue} />;
```

列宽 `width: 280 → 220`（释放 60px）。

### B3. 时间列降行高

```tsx
const renderTimestamp = (value?: string) => {
  if (!value) return "-";
  const parsed = dayjs(value);
  if (!parsed.isValid()) return new Date(value).toLocaleString();
  return (
    <Tooltip title={parsed.format("YYYY-MM-DD HH:mm:ss")}>
      <span>{parsed.fromNow()}</span>
    </Tooltip>
  );
};
```

**对比**：
- 改前：双行堆叠，行高 ~52px
- 改后：单行，行高 ~32px（行高缩减 38%，一屏可见数据量提升）

### B4. Tag 统一

按 `STATUS_TAG_PALETTE` 选择对应档：

```tsx
<Tag color={pickPalette(record.status, "success" | "error" | "warning" | "paused")}>
  {label}
</Tag>
```

| 当前 line | 现状 | 改后 |
|---|---|---|
| 355 | `<Tag>` 状态 | 改用 palette |
| 1090 | `<Tag>` labels | 改用 `system` |
| 1316 | `<Tag color="blue">模板 v{version}</Tag>` | 改用 `system` |
| 1319 | `<Tag color="green" ...>` 草稿 | 改用 `paused` |
| 1323 | `<Tag color="blue" ...>` | 改用 `system` |
| 1331 | `<Tag>` 资源池 | 改用 `warning` |
| 1373 | `<Tag>` 排队/警告 | 改用 `warning` |
| 1389 | `<Tag color="gold">排队中</Tag>` | 改用 `warning` |
| 1396 | `<Tag color="warning">疑似停滞</Tag>` | 已是 warning，调底色 |
| 1529 | `<Tag color={nsColor} ...>` namespace | 改用 `system` |

### B5. 数值右对齐

```tsx
{
  title: "耗时",
  dataIndex: "duration",
  align: 'right',
  ...
}
{
  title: "总成本",
  align: 'right',
  ...
}
```

CSS fallback：`th.ant-table-cell, td.ant-table-cell { text-align: right; }` 仅作用于该列。

### B6. 微小成本格式化

```tsx
const renderEstimatedCost = (_: unknown, record: WorkflowSummary) => {
  const cost = getWorkflowEstimatedCost(record);
  if (cost == null) {
    return (
      <Tooltip title={getEstimatedCostTooltip(record)}>
        <Typography.Text type="secondary">—</Typography.Text>
      </Tooltip>
    );
  }
  const display =
    cost === 0 ? "$0.00" :
    cost < 0.001 ? "<$0.001" :
    cost < 0.01 ? `$${cost.toFixed(4)}` :
    `$${cost.toFixed(2)}`;
  return (
    <Tooltip title="估算总成本，非 GCP Billing 最终对账金额">
      <Typography.Text strong>{display}</Typography.Text>
    </Tooltip>
  );
};
```

### B7. 操作列清理

```tsx
{
  title: "操作",
  key: "actions",
  width: 100,                                // 220 → 100
  render: (_, record) => (
    <Dropdown menu={{ items: actionMenuItems(record) }} trigger={['click']}>
      <Button type="text">操作 ⁝</Button>
    </Dropdown>
  ),
}

// 同时让批次名称承担查看入口
{
  title: "名称",
  dataIndex: "name",
  render: (name, record) => (
    <a onClick={() => navigate(`/runs/${encodeURIComponent(record.runId)}`)}>
      {name}
    </a>
  ),
}
```

### B8. "对比选中"按钮反馈

```tsx
<Button
  disabled={selectedRowKeys.length < 2}
  type={selectedRowKeys.length >= 2 ? "primary" : "default"}
>
  对比选中{selectedRowKeys.length >= 2 && ` (${selectedRowKeys.length})`}
</Button>

{/* 配合 Tooltip 包装 */}
<Tooltip
  title={selectedRowKeys.length < 2 ? "请至少勾选 2 条记录进行对比" : ""}
>
  <Button ... />
</Tooltip>
```

---

## 验收截图清单（Chrome DevTools MCP）

- [ ] BatchJobList 资源池列（彩色 Tag + Tooltip）
- [ ] BatchJobList 失败批次进度条（暗红而非饱和红）
- [ ] BatchJobList 状态 Tag 配色统一
- [ ] WorkflowExecutionList 复制图标 hover 才出现
- [ ] WorkflowExecutionList 资产 ID 无 `asset_id=` 前缀
- [ ] WorkflowExecutionList 时间列单行 + hover 弹完整
- [ ] WorkflowExecutionList 数值列右对齐
- [ ] WorkflowExecutionList "对比选中"按钮 < 2 项时禁用 + Tooltip
- [ ] 两个页面 Tag 配色一致（截图对比）
