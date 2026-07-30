# Spec delta — CYB-4470

## ADDED Requirements

### Requirement: BatchJobList 暴露资源池列
`Frontend/src/pages/BatchJobList.tsx` 的 columns 数组 SHALL 在"状态"列之后、"优先级"列之前新增"资源池"列。

**Priority**: P0 (Critical)
**Rationale**: 用户在 1920px+ 屏无法一眼分辨批次所属资源池；现状必须逐行点开才能确认。

#### Scenario: 资源池列渲染目标池名
- **Given** 一条 batch 的 `filterJson` 含 `targetId` 且 `targetMap` 有对应 target
- **When** 表格渲染该行
- **Then** "资源池"列显示 `<Tag color="success">{target.name}</Tag>`（available + enabled=true）

#### Scenario: 资源池列渲染 unavailable 池
- **Given** 一条 batch 指向 `target.status='unavailable'`
- **When** 表格渲染该行
- **Then** "资源池"列显示 `<Tag color="error">{target.name}</Tag>`

#### Scenario: 资源池列渲染 disabled 池
- **Given** 一条 batch 指向 `target.enabled=false`
- **When** 表格渲染该行
- **Then** "资源池"列显示 `<Tag color="default">{target.name}</Tag>`

#### Scenario: 资源池列空态
- **Given** 一条 batch 没有 targetId（filterJson 缺字段）
- **When** 表格渲染该行
- **Then** "资源池"列显示 `<span class="text-muted">N/A</span>`（不显示 "—" 或空字符串）

#### Scenario: 资源池 Tooltip 显示集群信息
- **Given** 资源池 Tag 被 hover
- **When** 鼠标悬停 ≥ 100ms
- **Then** 弹出 Tooltip 显示 `cluster={target.cluster} · ns={target.namespace}`

#### Scenario: 资源池列过滤
- **Given** 用户点击"资源池"列表头的过滤图标
- **When** 从下拉框勾选一个或多个 target.name
- **Then** 列表只显示 `targetId` 在所选集合内的行

---

### Requirement: Status Tag 配色统一规范
所有出现在 BatchJobList 的状态 Tag SHALL 使用 `lib/batchJobs.ts::STATUS_TAG_PALETTE` 中定义的语义色。

**Priority**: P1 (High)
**Rationale**: 当前"失败红 + 部分失败橙 + 暂停黄 + 完成绿 + 优先级蓝/灰"五色齐飞，认知负担大；统一为 success/error/warning/paused/neutral 五档。

#### Scenario: 完成 Tag 使用 success 配色
- **Given** `derived='completed'`
- **When** 渲染状态 Tag
- **Then** Tag 使用 `#f6ffed / #b7eb8f / #389e0d`（或 antd `color="success"`）

#### Scenario: 失败 Tag 使用 error 配色
- **Given** `derived='failure'`（completed_count=0 且 failed_count>0）
- **When** 渲染状态 Tag
- **Then** Tag 使用 `#fff1f0 / #ffa39e / #cf1322`（或 antd `color="error"`）

---

### Requirement: Progress Bar 失败段降饱和
`Frontend/src/components/pipeline/BatchProgressCell.tsx` 渲染失败进度条 SHALL 使用暗红（`#cf1322`），不再使用饱和红（`#f5222d`）。

**Priority**: P1 (High)
**Rationale**: 整屏饱和红块触发"红色焦虑"，运维无法快速判断系统整体状态。

#### Scenario: 全失败批次进度条降饱和
- **Given** `derived='failure'`（completed_count=0）
- **When** 渲染 Progress
- **Then** 失败段使用 `#cf1322`（暗红），非 `#f5222d`

#### Scenario: 部分失败进度条保持橙色
- **Given** `derived='partial_failure'`
- **When** 渲染 Progress
- **Then** 失败段使用 `#fa8c16`（橙色，与部分失败 Tag 配色一致）

---

### Requirement: 所有 ellipsis 列必须配套 Tooltip
所有设置了 `ellipsis: true` 的列 SHALL 配套 Tooltip 显示完整内容。

**Priority**: P0 (Critical)
**Rationale**: 用户在 1920px+ 屏被截断文本后无法看到完整 ID/模板名/用户邮箱，操作受阻。

#### Scenario: 所属用户列 Tooltip
- **Given** "所属用户"列存在较长邮箱被截断
- **When** 用户 hover 该单元格
- **Then** Tooltip 显示完整邮箱地址
