# Spec delta — WorkflowExecutionList

## ADDED Requirements

### Requirement: 复制图标 Hover 显现
`Frontend/src/pages/WorkflowExecutionList.tsx` 表格中所有"复制"图标 SHALL 默认隐藏，鼠标悬停到单元格时才浮现。

**Priority**: P0 (Critical)
**Rationale**: 当前每行 ID / 用户 / 资产 ID 旁常驻 `<CopyOutlined />`，数十个图标形成视觉噪音，遮挡文字快速扫读。

#### Scenario: 默认不显示复制图标
- **Given** 表格处于正常渲染状态（鼠标未 hover）
- **When** 用户扫视列表
- **Then** 看不到任何 `<CopyOutlined />` 图标

#### Scenario: Hover 单元格浮现复制图标
- **Given** 鼠标 hover 到包含可复制内容的单元格
- **When** 停留 ≥ 100ms
- **Then** 复制图标浮现（opacity 0 → 1 或 display none → inline-flex）

#### Scenario: 点击复制成功反馈
- **Given** 复制图标已浮现
- **When** 用户点击
- **Then** 调用 `navigator.clipboard.writeText(value)`，弹出 message.success("已复制")

---

### Requirement: 资产 ID 单元格去除 `asset_id=` 前缀
`Frontend/src/pages/WorkflowExecutionList.tsx` 的资产 ID 渲染 SHALL 仅显示 ID Hash，不再携带 `asset_id=` 前缀。

**Priority**: P0 (Critical)
**Rationale**: 列头已叫"资产 ID"，单元格里再写 `asset_id=...` 是信息冗余；前缀占用约 60px 横向空间。

#### Scenario: 资产 ID 单元格仅显示 Hash
- **Given** 一条 run 含 `labels.asset_id = "asset_id=019dbeda...5146d"`
- **When** 表格渲染该行的"资产 ID"列
- **Then** 单元格仅显示 `019dbeda...5146d`，不包含 `asset_id=` 前缀

---

### Requirement: 时间列降行高（相对时间为主，Hover Tooltip 弹完整）
`Frontend/src/pages/WorkflowExecutionList.tsx` 的"创建时间"/"完成时间" SHALL 默认仅显示相对时间，hover 时 Tooltip 弹出完整时间。

**Priority**: P1 (High)
**Rationale**: 当前 `renderTimestamp` 在 line 188-197 用 `fromNow()` + 绝对时间双行堆叠，整张表格 row height 拉高，一屏可见数据量下降。

#### Scenario: 时间列默认单行
- **Given** 一条 run 含 `createdAt = "2026-07-29T22:33:12Z"`
- **When** 表格渲染该行的"创建时间"列
- **Then** 仅显示一行文本（如 "11 分钟前"）

#### Scenario: Hover 时间列弹出完整时间
- **Given** 时间单元格可见
- **When** 用户 hover 该单元格
- **Then** Tooltip 显示 `2026-07-29 22:33:12`（含秒）

---

### Requirement: Tag 样式统一（浅色背景 + 文字色）
`Frontend/src/pages/WorkflowExecutionList.tsx` 中所有 `<Tag>` SHALL 使用统一"浅色背景 + 文字色"语义样式，与 BatchJobList 共享 token 表。

**Priority**: P1 (High)
**Rationale**: 当前 4 种 Tag 设计（实色圆角、浅蓝无边框、线框白底、灰色代码块）混用，视觉语言不一致。

#### Scenario: 版本/系统 Tag 使用蓝色 token
- **Given** 一个版本 Tag（如 `模板 v10`）
- **When** 渲染
- **Then** 使用蓝色浅底（`#e6f4ff` / 文字 `#1677ff`）

#### Scenario: 资源/失败 Tag 使用橙红色 token
- **Given** 一个资源池或失败 Tag（如 `vpp-cpu`、`排队中`）
- **When** 渲染
- **Then** 使用橙/红浅底（`#fff2e8` 文字 `#d4380d`，或 `#fff1f0` 文字 `#cf1322`）

#### Scenario: 辅助标记 Tag 使用灰色 token
- **Given** 一个辅助信息 Tag（如 `Dev 草稿`）
- **When** 渲染
- **Then** 使用灰色浅底（`#fafafa` 文字 `#8c8c8c`）

---

### Requirement: 数值字段右对齐
`Frontend/src/pages/WorkflowExecutionList.tsx` 中耗时 / 视频时长 / 总成本等数值字段 SHALL 右对齐。

**Priority**: P2 (Nice-to-have)
**Rationale**: 数据对齐规范——文本类左对齐，数值类右对齐，方便用户上下扫视数值大小。

#### Scenario: 成本列右对齐
- **Given** 一条 run 含 `estimatedCostUsd = 0.1234`
- **When** 渲染"总成本"列
- **Then** 数字右对齐（`text-align: right` 或 `align: 'right'` 在 column config）

---

### Requirement: 微小成本格式化
`Frontend/src/pages/WorkflowExecutionList.tsx::renderEstimatedCost` SHALL 把 `< $0.001` 的成本显示为 `<$0.001`，把 `= 0` 显示为 `$0.00`。

**Priority**: P2 (Nice-to-have)
**Rationale**: `$0.0000` 让用户产生"是免费还是计算失败？"的疑虑；规范化格式消除歧义。

#### Scenario: 微小成本显示为 `<$0.001`
- **Given** `cost = 0.0001`
- **When** 渲染
- **Then** 显示为 `<$0.001`

#### Scenario: 零成本显示为 `$0.00`
- **Given** `cost = 0`
- **When** 渲染
- **Then** 显示为 `$0.00`

#### Scenario: 正常成本保持 4 位精度
- **Given** `cost = 0.0055`
- **When** 渲染
- **Then** 显示为 `$0.0055`

---

### Requirement: 操作列层级清理
`Frontend/src/pages/WorkflowExecutionList.tsx` 操作列 SHALL 仅保留"操作 ⁝"下拉菜单，"批次名称"可点击查看详情。

**Priority**: P2 (Nice-to-have)
**Rationale**: 当前"模板 / 查看 / 操作 ⁝"三按钮横向并排，右侧操作列视觉拥挤；让批次名称承担主入口，操作 ⁝ 收纳次要动作。

#### Scenario: 批次名称点击跳转
- **Given** 用户点击"批次名称"列的文本
- **When** click 事件触发
- **Then** 跳转到 `/runs/{runId}` 或 `/pipeline/executions/{name}`

#### Scenario: 操作列只显示"操作 ⁝"
- **Given** 表格渲染某行
- **When** 渲染"操作"列
- **Then** 仅看到一个 `<Dropdown>` 触发按钮（"操作 ⁝"），不再有"模板"和"查看"按钮

---

### Requirement: 「对比选中」按钮反馈
`Frontend/src/pages/WorkflowExecutionList.tsx` 顶部的"对比选中"按钮 SHALL 在勾选 < 2 项时 Hover 提示，≥ 2 项时高亮并显示数量。

**Priority**: P2 (Nice-to-have)
**Rationale**: 当前按钮始终禁用或灰态，用户不知道触发条件；明确反馈降低试错成本。

#### Scenario: < 2 项时禁用 + Hover 提示
- **Given** `selectedRowKeys.length = 1`
- **When** 用户 hover "对比选中"按钮
- **Then** Tooltip 显示"请至少勾选 2 条记录进行对比"，按钮 disabled

#### Scenario: ≥ 2 项时激活并显示数量
- **Given** `selectedRowKeys.length = 3`
- **When** 渲染"对比选中"按钮
- **Then** 按钮高亮激活，文字显示"对比选中 (3)"
