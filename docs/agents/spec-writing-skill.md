# Spec-Writing Skill — OpenSpec 制品质量指南

**触发条件：** 每次创建或修改 `openspec/changes/CYB-*/` 下的文件时自动加载。AI 必须遵循本文要求，不得跳过。

**与 [`spec-driven-workflow.md`](spec-driven-workflow.md) 的分工：** 那边管 **生命周期**（目录、归档、CI gate、豁免标签）；本文管 **写作质量**（proposal / delta / design / tasks 写什么、怎么自检）。

**原则：** OpenSpec 不是「多写一份文档」，而是 **用结构化制品对齐意图与实现**。如果制品不对齐行动，就是废纸。

**本文规范与 `openspec/config.yaml` 的 `rules` 字段保持一致，同时吸收了 [ForceInjection/OpenSpec-practise](https://github.com/ForceInjection/OpenSpec-practise) 社区最佳实践（Given/When/Then、Priority + Rationale、SLO 目标）。**

---

## 一、各制品写法

### 1.1 `proposal.md` — 六段式

**结构参考（仅章节，勿抄电商示例）：** [OpenSpec-practise proposal 模板](https://github.com/ForceInjection/OpenSpec-practise/blob/main/examples/openspec/changes/v1-mvp/proposal.md)。

```markdown
# Proposal — CYB-{id}

## Why
{一句话：用户痛点 / 业务需求 / 线上问题}

## What Changes

### New Capabilities
- {能力 1：用模块名定位，如 asset-management}
- {能力 2}

### Modified Capabilities
- {被修改的已有能力}

## Impact
- **Affected code**: `backend/internal/{package}`, `Frontend/src/components/{path}`
- **New APIs**: {如有，列出路径和方法}
- **Dependencies**: {新增或变更的依赖}

## Scope
- **In scope**: {明确包含的}
- **Out of scope**: {明确不做的}

## Success Criteria
- [ ] {可验证的验收条件 1，如「用户可以 XX」}
- [ ] {可验证的验收条件 2}

## Goals (SLO) — Feature / 性能敏感变更时必写
- **Latency**: {如 p99 < 100ms}
- **Concurrency**: {如 50 RPS}
- **Quality**: {如 测试覆盖率 > 80%}
```

**反模式：**
- ❌ 把 `What Changes` 写成技术实现方案（"新增 POST /api/v1/foo 接口"）
- ❌ 把 `Why` 写成一段长篇叙事
- ❌ Bug fix 不写 `Modified Capabilities` 就直接实现
- ✅ `What Changes` 写行为变化（"用户可以批量导出资产"），技术细节放 `design.md`

---

### 1.2 `specs/<module>/spec.md` — Given/When/Then Delta

**格式：** change delta **必须**用下方模板（`#### Scenario:` + Given/When/Then + Priority + Rationale），与 OpenSpec CLI `validate` 一致。

**基线 spec：** `openspec/specs/*/spec.md` 仍是简版（无 Scenario），**不要照抄**；新需求只写在 `openspec/changes/CYB-*/specs/` delta 里。基线升级见 `spec-driven-workflow.md` 归档步骤。

**核心升级：** 每条 Requirement 必须带 `Priority` + `Rationale` + 至少 1 个 `Given/When/Then` Scenario。

```markdown
## ADDED Requirements

### Requirement: {新行为名称}
The system SHALL {描述新增的系统行为}。

**Priority**: P0 (Critical) / P1 (High) / P2 (Nice-to-have)
**Rationale**: {为什么需要这个行为}

#### Scenario: {happy-path 场景名}
- **Given** {前置条件}
- **When** {触发动作}
- **Then** {预期结果}

#### Scenario: {error-path 场景名}
- **Given** {异常前置条件}
- **When** {触发动作}
- **Then** {错误预期结果}

## MODIFIED Requirements

### Requirement: {已有行为名称}
- **Before**: The system SHALL {旧行为}。
- **After**: The system SHALL {新行为}。
- **Reason**: {为什么改}

#### Scenario: {修改后的典型场景}
- **Given** {前置条件}
- **When** {触发动作}
- **Then** {预期结果}

## REMOVED Requirements

### Requirement: {被移除的行为名称}
- **Was**: The system SHALL {被移除的行为}。
- **Reason**: {为什么移除}
```

**实战示例（资产发现 — asset-management）：**

```markdown
### Requirement: 资产发现页算法状态筛选与列表一致
The system SHALL apply algorithm-status filters using server-side query results
for the asset discovery list, and SHALL show the same rows and total count the
backend returned for the selected filter.

**Priority**: P0 (Critical)
**Rationale**: 客户端二次过滤会导致列表条数与分页 total 不一致，用户误判数据缺失。

#### Scenario: 筛选失败状态后列表与总数一致
- **Given** 后端对「算法失败」筛选返回 12 条资产且 total 为 12
- **When** 用户在资产发现页选择「算法失败」筛选
- **Then** 列表展示 12 条且分页总数显示 12

#### Scenario: 切换筛选后结果随服务端更新
- **Given** 用户当前查看「算法失败」筛选结果
- **When** 用户改为「算法运行中」筛选
- **Then** 列表与总数仅反映「运行中」筛选的后端结果，不保留上一筛选的隐藏行

#### Scenario: 无匹配资产时的空态
- **Given** 后端对当前筛选返回 0 条且 total 为 0
- **When** 用户打开或刷新资产发现页
- **Then** 列表为空且展示合理空态，分页总数显示 0
```

**反模式：**
- ❌ spec delta 里写 API 路径、请求体字段、数据库列名 → 这些放 `design.md`
- ❌ `MODIFIED` 不写 `Reason` → 审查者不知道为什么改
- ❌ 只写 1 个 happy-path Scenario，不写 error-path → AI 实现时猜边界条件
- ❌ 省略 `Priority` 或 `Rationale` → 审查者无法判断哪些是硬性要求
- ✅ 每个 Requirement 至少 1 个 happy-path + 1 个 error-path Scenario

---

### 1.3 `design.md` — 决策三段式 + 架构上下文

**结构参考（仅章节）：** [OpenSpec-practise design 模板](https://github.com/ForceInjection/OpenSpec-practise/blob/main/examples/openspec/changes/v1-mvp/design.md)。

```markdown
# Design — CYB-{id}

## Architecture Context
- **Constraints**: {技术约束，如「Go 1.25+」/「Postgres 唯一数据源」}
- **Goals**: {设计目标}
- **Non-Goals**: {明确不做的}

## Affected Modules
- `backend/internal/{package}` — {改什么}
- `Frontend/src/components/{path}` — {改什么}

## Architecture Decisions

### Decision 1: {决策标题}
- **Approach**: {选了什么方案}
- **Alternative**: {考虑过但没有选的方案}
- **Rationale**: {为什么选这个}
- **Trade-off**: {代价是什么}（optional）
- **Risk**: {已知风险}（optional）
- **Rollback**: {如何回滚}（optional，影响数据/API 时必写）

## Data Flow（涉及跨模块协调时必写）
{用 Mermaid 或 ASCII 描述关键数据流}

## Data Model Changes（涉及 schema 变更时必写）
- **Table**: {表名}
- **Change**: {新增列 / 修改约束 / 新增表}
- **Migration**: {迁移脚本路径}

## Risks / Trade-offs
| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| {风险描述} | {影响范围} | {如何缓解} |
```

**Bug fix 一般不需要 `design.md`**，除非修复涉及架构变更。

**反模式：**
- ❌ 只写「用了 XX 技术」不写为什么不用 YY
- ❌ 把 design.md 写成 API 文档（字段枚举）
- ❌ 缺少 Architecture Context → 审查者不知道技术约束

---

### 1.4 `tasks.md` — 已有模板，注意四点

使用 `openspec/changes/README.md` 中的模板，额外注意：

1. **Implementation 任务和 Scenario 1:1 对齐** — 每个 Given/When/Then 至少 1 个 checkbox
2. **API contract sync 章节** — 若 change 含新/改 HTTP API，复制 `openspec/changes/README.md` 中 API contract 清单，与 [`AI-RULES.md` § API contract sync](AI-RULES.md#api-contract-sync-mandatory) 同步完成
3. **Deploy verification 章节不可省略** — 针对性验证应写明 Scenario 名
4. **标注代码路径** — 每个 checkbox 标注 `[backend]` / `[Frontend]` / `[sdk]`
5. **任务粒度** — 一个 checkbox ≈ 10–30 分钟
6. **`context-files.md` 或 `## Context files`** — 列出实现前必读路径（见 1.4a）

---

### 1.4a `context-files.md` — 必读路径（Trellis 思想，轻量）

在 change 目录放 `context-files.md`（或在 `tasks.md` 里写 `## Context files`）：

```markdown
# Context files — CYB-{id}

Frontend/src/hooks/assets/useAssetsDiscoveryReducer.ts  # 筛选/列表状态
backend/internal/handlers/asset/handler.go              # 资产列表 API
docs/agents/deploy-verification.md               # §6 回归范围
```

**何时写：** proposal `Impact` 列出 ≥2 个路径；或跨模块 bug；或新 agent 会话接手该 change。

**反模式：** 把整个 `src/` 树贴进去；应只列 **与本 change 相关** 的文件。

---

### 1.5 `decisions.md` — 按需追加

触发条件见 `AI-RULES.md` → Decision log。追加式格式：

```markdown
## YYYY-MM-DD — {简短标题}
- **Context**: {发生了什么}
- **Decision**: {做了什么决定}
- **Alternatives**: {还有什么选择}
- **Rationale**: {为什么}
```

---

## 二、质量自检清单（AI 生成后自动过一遍）

### proposal.md
- [ ] Why 是否 ≤ 3 句话
- [ ] What Changes 是否区分 New / Modified Capabilities
- [ ] Scope 是否有 In / Out
- [ ] Impact 是否列出 affected code + new APIs
- [ ] Feature/性能变更是否含 Goals (SLO)
- [ ] Success Criteria 是否可验证

### specs/<module>/spec.md
- [ ] 每条 Requirement 有 Priority (P0/P1/P2) + Rationale
- [ ] 每条 Requirement 用 `The system SHALL` / `MUST` 句式
- [ ] 每个 Requirement ≥ 1 个 Given/When/Then Scenario
- [ ] 复杂行为 ≥ 1 个 error-path Scenario
- [ ] 没有 API 路径、字段名、SQL 出现在 spec 中

### design.md（Feature 时）
- [ ] 有 Architecture Context (Constraints + Goals + Non-Goals)
- [ ] 每条 Decision 有 Approach + Alternative + Rationale
- [ ] 涉及数据/API 变更有 Rollback 路径
- [ ] 跨模块变更有 Data Flow

### tasks.md
- [ ] Implementation 任务和 spec delta Scenario 1:1 对齐
- [ ] 每个 checkbox 标注了代码路径 `[backend]` / `[Frontend]` / `[sdk]`
- [ ] 含 HTTP API 时 **API contract sync** 清单齐全（OpenAPI、api-guide、smoke；SDK/Frontend 按范围）
- [ ] Deploy verification 章节完整填入
- [ ] 有 `context-files.md` 或 `tasks.md` 内 `## Context files`（跨模块 / 多文件变更时）

---

## 三、与 AI-RULES 的对接

| AI-RULES 步骤 | 本文对应章节 |
|---------------|-------------|
| Step 4 — 创建 OpenSpec change | §1.1 ~ 1.5 全部适用 |
| Step 4 — **OpenSpec checkpoint** | 制品写完后 **停下**，等用户确认「OpenSpec OK」再写运行时代码 |
| Bug fix | §1.1 proposal + §1.2 delta + §1.4 tasks（跳过 §1.3 design） |
| Feature | §1.1 ~ 1.5 全部必须 |
| Hotfix（跳过 OpenSpec gate） | 事后补 §1.2 delta（T+2 内回填） |

---

## 四、社区经验：什么情况下 OpenSpec 会失败

| 反模式 | 后果 | 正确做法 |
|--------|------|----------|
| 把 OpenSpec 当成「多写的文档」 | 团队抵触、绕过的比例上升 | 让制品驱动行动：proposal 决定 scope，spec 决定验证，tasks 决定执行顺序 |
| spec delta 写太细（字段级） | 实现时 delta 和代码不一致 | spec 只写行为，API 字段放 design.md |
| 没有 Scenario 或只有 happy-path | 边界 case 在实现阶段才被发现 | 每个 Requirement 配 happy-path + error-path Scenario |
| 没有 Priority | 审查者不知道哪些是硬性要求 | P0=不实现就不能上线，P1=应有，P2=锦上添花 |
| tasks.md 只有代码任务 | 验证缺失，合并后才发现问题 | Deploy verification 不可省略 |
| AI 生成后不审核就实现 | 意图漂移未被发现 | 生成后先跑 §二 的自检清单，**等用户确认 OpenSpec** 再实现 |
| design.md 没有 Alternative | 决策不可审计 | 每条决策必须有对比方案 + 理由 |
