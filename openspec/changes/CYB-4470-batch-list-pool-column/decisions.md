# Decisions — CYB-4470

## 2026-07-29 整合两个 review

### 合并 CYB-4471 到 CYB-4470

用户提了 BatchJobList 与 WorkflowExecutionList 两个独立 review，用户回复「都一起整」→ 合并到一个 change。

**做法**：
- Linear: CYB-4471 Canceled，CYB-4470 标题/描述扩展
- OpenSpec: 两个 spec delta（`specs/pipeline/spec.md` + `specs/workflow-execution/spec.md`）共存于一个 change dir
- tasks.md / design-ui.md 双文件改

**理由**：
- 两个页面共享同一套设计语言（Tag palette、Token、空态字符）
- 一次走完 OpenSpec + PR + review 流程
- 后续如需拆分可基于 spec delta 单独抽出

### 资源池着色：按 `target.status` 三态而非按 cluster

**取舍**：原本可能按 cluster（不同集群不同色）更直观，但运维实际关注的是"池子能不能用"（status），不是"在哪跑"。三态配色（绿/红/灰）与现有 Status Tag 语义一致，学习成本低。

### Progress Bar 失败色：暗红而非浅红

**取舍**：用户原意是"降饱和避免红色焦虑"，但完全浅红（`#ffccc7`）会与背景对比度不足，可读性下降。选**暗红 `#cf1322`**——比饱和红低饱和度，但仍醒目。

### 空态统一用 "—" + 弱化样式

**取舍**：用 `—` 比用空字符串更符合中后台惯例（中文用户）；加 `color: #bfbfbf` 弱化视觉权重。

### 复制图标：Hover 显现 vs 点击即复制

**取舍**：用户给了二选一。**默认实现 Hover 显现 + 点击复制**（双保险），hover 解决"看到图标"的视觉噪音，点击仍可触发复制。点击即复制作为备选，需用户后续决策（implementation 阶段可快速切换）。

### 数值右对齐 vs 等宽

**取舍**：用户提到"右对齐或在固定列宽下保持等宽对齐"。**先做右对齐**（实现简单、满足大多数场景）。等宽对齐（tabular-nums）作为后续 P3 优化。

### 微小成本格式化边界值

**取舍**：
- `cost === 0` → `$0.00`（明确"是 0 不是计算失败"）
- `cost > 0 && cost < 0.001` → `<$0.001`（无法显示完整精度，规范化）
- `cost >= 0.001 && cost < 0.01` → 4 位精度
- `cost >= 0.01` → 2 位精度

四档清晰，与行业惯例（Stripe / Datadog / GCP Console）一致。

### 不预留 Batch Action Bar UI，只预留 state

**取舍**：P2 的 Batch Action Bar 需要 `selectedRowKeys` + 浮层渲染，提前在 BatchJobList 加 state 容易污染组件；但**不**在本 change 实现浮层 UI，避免 PR scope creep。

---

## 2026-07-29 用户批准 OpenSpec，开始实现

用户回复「可以的」+「OpenSpec OK，继续」，确认按当前 proposal + design 实施。分支：`feat/CYB-4470-batch-list-pool-column`（基于 `origin/dev`）。

**用户已确认的 8 项设计决策**：
1. 资源池着色按 target.status 三态（绿/红/灰）✓
2. 进度条失败色用暗红 `#cf1322` ✓
3. 空态字符保留 `—` 弱化样式 ✓
4. 复制图标用 Hover 显现 + 点击复制 双保险 ✓
5. 数值右对齐（先做这一步，等宽后续 P3）✓
6. 微小成本格式四档（`$0.00` / `<$0.001` / 4 位 / 2 位）✓
7. Tag 配色两个页面共用 `lib/designTokens.ts` ✓
8. 不实现 Batch Action Bar UI ✓
