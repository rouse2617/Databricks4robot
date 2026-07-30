# UI design — CYB-4477 BatchJobList + WorkflowExecutionList P2

## P2-1: Batch Action Bar

**位置**：表格上方（`<Table>` 上方，header 下方的位置）

**结构**：

```tsx
<BatchActionBar
  selectedJobs={selectedJobs}    // 由 selectedRowKeys 派生
  onAction={(action) => {
    switch (action) {
      case "retry": retryFailedBatchItems(...); break;
      case "export": exportAssetIdsCsv(...); break;
      case "cancel": pauseBatchJob(..., { stopRunning: true }); break;
      case "pause": pauseBatchJob(..., { stopRunning: false }); break;
      case "resume": resumeBatchJob(...); break;
    }
  }}
/>
```

**渲染规则**：
- `selectedJobs.length === 0` → **不渲染**（避免视觉噪音）
- `>= 1` → 渲染 `<Affix offsetTop={120}>` 浮在表格上方
- 内容：
  ```tsx
  <Space>
    <Text>已选 <strong>{n}</strong> 个批次</Text>
    <Button icon={<RedoOutlined />} onClick={retry}>批量重试</Button>
    <Button icon={<DownloadOutlined />} onClick={export}>批量导出</Button>
    <Button icon={<PauseOutlined />} onClick={pause}>批量暂停</Button>
    <Button icon={<PoweroffOutlined />} onClick={cancel} danger>批量取消</Button>
  </Space>
  ```

**视觉**：浅蓝底（`#e6f4ff`）+ 边框，圆角 6px，与 BatchJobList 整体风格一致。

---

## P2-2: 行 Hover 工具栏

**结构**：

```tsx
// HoverActionBar 组件
export function HoverActionBar({ actions, hovered }: { actions: Action[]; hovered: boolean }) {
  return (
    <Space style={{
      opacity: hovered ? 1 : 0,
      transition: "opacity 0.15s",
      pointerEvents: hovered ? "auto" : "none",
    }}>
      {actions.map(a => (
        <Tooltip key={a.key} title={a.label}>
          <Button type="text" size="small" icon={a.icon} onClick={a.onClick} />
        </Tooltip>
      ))}
    </Space>
  );
}
```

**行 hover 状态**：

```tsx
<Table
  onRow={(record) => ({
    onMouseEnter: () => setHoveredId(record.id),
    onMouseLeave: () => setHoveredId(null),
  })}
/>
```

**每个列的内容**（按列类型）：
- "批次名称" 列：hover 时右侧追加 `🔗 查看详情` 按钮
- "操作" 列：替换为 hover-action 组（详细操作）

---

## P2-3: 合并"耗时"列

**现状**：BatchJobList 有两个时间字段：
- `runStartedAt` → `runFinishedAt` = "耗时"（实际执行）
- `createdAt` → `finishedAt` = 含等待

**改后**：合并成单列 "耗时（含等待）"

```tsx
{
  title: (
    <Tooltip title="执行耗时 · 含等待时间（创建→完成减去执行）">
      <span>耗时</span>
    </Tooltip>
  ),
  key: "duration",
  width: 100,
  sorter: (a, b) => batchJobRunDurationSeconds(a) - batchJobRunDurationSeconds(b),
  render: (_, record) => {
    const real = batchJobRunDurationSeconds(record);  // 实际执行
    if (real === null) return "—";
    const wait = batchJobTotalWaitSeconds(record);    // 等待时间
    return (
      <Tooltip title={wait ? `${formatDurationSeconds(real)} (等待 ${formatDurationSeconds(wait)})` : formatDurationSeconds(real)}>
        <span>{formatDurationSeconds(real)}</span>
      </Tooltip>
    );
  },
}
```

**注意**：删除现有独立的"耗时"和"完成时间"列。"完成时间"信息并入"耗时"tooltip。

---

## P2-4: 失败数 / 失败率排序

```tsx
// BatchJobList "进度"列加 sorter
{
  title: "进度",
  key: "progress",
  width: 200,
  sorter: (a, b) => (a.failedCount || 0) - (b.failedCount || 0),
  sortDirections: ["descend", "ascend"],
  render: (_, record) => <BatchProgressCell job={record} />,
}

// BatchJobList "所属用户"列加 sorter
{
  title: "所属用户",
  dataIndex: "createdBy",
  key: "createdBy",
  sorter: (a, b) => (a.createdBy ?? "").localeCompare(b.createdBy ?? "", "zh"),
  ...
}

// WorkflowExecutionList 类似
```

---

## 验收截图清单

- [ ] BatchJobList 选中 3 行 → 顶部浮栏出现
- [ ] BatchJobList hover 某行 → 名称右侧/操作列浮现按钮
- [ ] BatchJobList 耗时列合并后单列展示
- [ ] BatchJobList 进度列点击排序箭头 → 行重新排列