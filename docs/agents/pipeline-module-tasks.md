# 流水线模块任务清单（工程 + 产品 + 规模化）

> 生成时间：2026-06-14  
> 更新时间：2026-06-14（明确总目标 + 架构容量 10k/100k + C 系列任务）  
> 项目：`cyber-databrew`（Frontend + Backend）  
> 主页面：`http://127.0.0.1:5176/pipeline?tab=pipelines`  
> Batch 详情：`/pipeline/batch/:id`

---

## 文档用途

本文件是流水线模块的**单一事实来源（SSOT）**：把 QA 回归结论、产品缺口、大批量场景、架构容量约束收敛到**可排期、可验收**的任务项。供产品拍优先级、研发拆 sprint、评审时对照「做没做对」。

---

## 总目标（North Star）

**让数据团队能安全、可观测、可恢复地对海量资产跑流水线——而不是只能「全量盲跑、出错全废」。**

拆成四个可衡量支柱：

| 支柱 | 用户问题 | 目标状态（完成定义） |
|------|----------|----------------------|
| **G1 安全与信任** | prod 会不会被误删？只读是不是真的只读？ | prod 不可批量删；只读无编辑入口；关键操作有确认 |
| **G2 日常效率** | 96 条模板怎么管？搜不到、选不对？ | 分页/搜索/筛选可用；列表语义清晰；保存/编辑不丢上下文 |
| **G3 大批量可观测** | 1 万条跑起来，哪个节点在挂？ | Batch 详情 **<3s** 看到 10 节点成败分布；可下钻到资产 |
| **G4 大批量可恢复** | 跑一半发现节点错了，要全重跑吗？ | 可暂停；可**指定范围/换版**重跑；可选 Pilot 试跑 |

---

## 规模里程碑（明确边界）

不同规模对应**不同架构目标**，避免「一次做到 100k」导致过度设计或承诺过度。

| 里程碑 | 资产规模 | 架构定位 | 本清单对应阶段 | 完成定义（摘要） |
|--------|----------|----------|----------------|------------------|
| **M0 现状** | ≤10,000 / 批 | 硬上限已落地 | — | 创建被拒 ≥10001；5 并发下发；无节点热力图 |
| **M1 可运营** | 10,000 / 批 | 单批可看清、可止血、可局部重跑 | Phase A～B+ | B-1/B-2/B-3 验收通过；21 天内可跑完依赖集群而非 API |
| **M2 可扩展** | 10,000～30,000 / 批 | Worker 化、懒物化、可调并发 | Phase C+（C-1～C-5） | API 重启不丢批；物化不预建全量 ledger；并发可配置 |
| **M3 十万级** | 100,000（逻辑任务） | 协调器拆批 + 队列 + 限流 | Phase D（C-6～C-10） | 用户提交 1 个 10 万任务；系统拆 N 子批；统一进度 |

**非目标（本清单不承诺）：**

- 单批 inline 提交 100,000 个 `assetIds`（请求体与物化模型均不适合）
- 替代 K8s/Argo 集群扩容（平台不负责帮你买机器）
- 自动检测「逻辑错但 Succeeded」的业务正确性（需 Pilot + 校验节点/人工抽样）
- 持久化日志全量分页（见 CYB-1535，另案）

---

## 当前基线（2026-06-14）

| 项 | 状态 |
|----|------|
| 流水线列表分页/搜索/筛选/prod 保护 | ✅ 已上线 |
| 单批最大资产数 | **10,000**（`MaxBackfillAssetCount`） |
| 批内下发并发 | **5**（`maxConcurrentBatchItems`） |
| 批次节点级概览 | ❌ |
| 指定子任务批量重试 | ❌ |
| 换版重跑 completed | ❌ |
| 独立 Backfill Worker | ❌（API 进程内 goroutine） |

---

## 文档结构

| 部分 | 内容 | 主要读者 |
|------|------|----------|
| **第一部分** | 工程 / QA 任务（#1～#12） | 前端、QA |
| **第二部分** | 产品 Backlog（P-1～P-26） | 产品、设计 |
| **第三部分** | 大批量专项（B-1～B-4，1 万/批场景） | 全栈 |
| **第四部分** | 架构容量与规模化（C-1～C-10，10k→100k） | 后端、SRE、架构 |

---

# 第一部分：工程 / QA 任务

---

## 已完成（不必重复做）

- [x] 服务端分页 / 搜索 / 筛选 / 总数角标
- [x] prod 保护（无 checkbox、不可批量删）
- [x] 批量删除 Popconfirm + 数字确认 Modal
- [x] 发布二次确认
- [x] prod「查看定义」只读模式（画布 + 组件面板 disabled）
- [x] dev **编辑** URL 保留 `tab=design`
- [ ] dev **保存** URL 仍可能丢 `tab=design`（见产品 P-1）
- [x] 筛选变更只 refetch pipelines（不重拉 execution-targets）
- [x] `Promise.allSettled` 部分失败降级
- [x] `DeployPanel` 单元测试 16/16 通过

---

## Sprint 1 — 小改、高收益（建议先做）

### 1. 只读模式禁用「配置节点」

| 项 | 内容 |
|----|------|
| 优先级 | P0 |
| 问题 | prod 只读查看时，右侧「配置节点」仍可打开 `NodeConfigPanel` 并改本地状态 |
| 改动 | `readOnlyMode` 时隐藏按钮，或面板只读 + 禁用 `onSave` |
| 文件 | `Frontend/src/pages/PipelinePage.tsx`、`Frontend/src/components/pipeline/NodeConfigPanel.tsx` |
| 验收 | prod 只读路径下无法编辑节点配置 |

### 2. 「全选」文案改为「全选当前页」

| 项 | 内容 |
|----|------|
| 优先级 | P0 |
| 问题 | 当前「全选」仅选当前已加载页的可删 dev 项，易被误解为全量 |
| 改动 | Checkbox 文案改为「全选当前页」；必要时 tooltip 说明范围 |
| 文件 | `Frontend/src/components/pipeline/DeployPanel.tsx`、`DeployPanel.test.tsx` |
| 验收 | 文案与行为一致，单测更新 |

### 3. Workflow 日志 Modal 回归修复

| 项 | 内容 |
|----|------|
| 优先级 | P0 |
| 问题 | 首轮 QA：节点详情「查看日志」长时间 loading；部分 Tab 切换异常 |
| 改动 | 回归 `WorkflowDetailPage` + `useWorkflowDetail` 日志拉取 / follow 链路 |
| 文件 | `Frontend/src/pages/WorkflowDetailPage.tsx`、`Frontend/src/pages/useWorkflowDetail.ts` |
| 验收 | 日志 Modal 正常加载；Tab 可切换；失败有明确错误提示 |

---

## Sprint 2 — 体验增强

### 4. 搜索体验优化

| 项 | 内容 |
|----|------|
| 优先级 | P1 |
| 问题 | 搜索需 Enter 或点搜索按钮，用户可能不知道 |
| 方案 A | 300ms debounce 即时搜索 |
| 方案 B | placeholder 写「输入后按 Enter 搜索」 |
| 文件 | `Frontend/src/components/pipeline/DeployPanel.tsx` |

### 5. 批量删除失败明细

| 项 | 内容 |
|----|------|
| 优先级 | P1 |
| 问题 | 部分失败仅 toast「N 条失败」，不知道是哪几条 |
| 改动 | 列出失败流水线名称（或 id） |
| 文件 | `Frontend/src/components/pipeline/DeployPanel.tsx` |

### 6. 发布前 diff 预览（可选）

| 项 | 内容 |
|----|------|
| 优先级 | P1（可选） |
| 问题 | 发布仅有文字 confirm，看不到变更内容 |
| 改动 | 发布 confirm 前调用 `getPipelineDiff`，展示 diff 摘要 |
| 文件 | `DeployPanel.tsx`、复用 `VersionHistoryDrawer` diff 逻辑 |

### 7. 执行记录 → 模板跳转（可选）

| 项 | 内容 |
|----|------|
| 优先级 | P1（可选） |
| 问题 | 执行记录与模板管理 Tab 割裂，排查需多次跳转 |
| 改动 | 执行记录行增加「查看模板」链到设计页或流水线卡片 |
| 文件 | `WorkflowExecutionList.tsx`、`DeployPanel` 路由 |

---

## Sprint 3 — 工程与质量

### 8. 手动刷新时缓存 execution-targets

| 项 | 内容 |
|----|------|
| 优先级 | P2 |
| 问题 | 点「刷新」走 `refreshAll()`，每次都重拉 targets |
| 改动 | targets 内存缓存或 TTL；刷新按钮默认只更新 templates / deployments |
| 文件 | `Frontend/src/components/pipeline/DeployPanel.tsx` |

### 9. listPipelines 无 total 时分页 fallback

| 项 | 内容 |
|----|------|
| 优先级 | P2 |
| 问题 | 后端无 `total` 时 `items.length` 当 total，「加载更多」可能不准 |
| 改动 | 检测 fallback 模式时隐藏「加载更多」或 Alert 提示 |
| 文件 | `Frontend/src/api/pipelineApi.ts`、`DeployPanel.tsx` |

### 10. Playwright 冒烟 E2E

| 项 | 内容 |
|----|------|
| 优先级 | P2 |
| 用例 1 | 全选当前页 → 批量删除 → Popconfirm → 输入数字 → 取消 |
| 用例 2 | prod「查看定义」→ 只读 Alert + 组件面板 disabled |
| 用例 3 | dev「发布」→ confirm → 取消 |
| 文件 | 新建 `e2e/pipeline-*.spec.ts`（或项目现有 e2e 目录） |

---

## Backlog — 按需排期

### 11. 「全选筛选结果」跨页批量删除（可选）

| 项 | 内容 |
|----|------|
| 优先级 | P3 |
| 场景 | 管理员一次性清理大量 dev 模板 |
| 要求 | 更强确认（数量 + 名称列表 + 二次输入） |

### 12. auth/me StrictMode 双请求去重

| 项 | 内容 |
|----|------|
| 优先级 | P3 |
| 说明 | 应用层 StrictMode 双 mount，与流水线模块无关，影响极小 |

---

## 建议执行顺序

```
1 → 2 → 3 → 10 → 4 → 5 → 8 → 9
```

可选项（6、7、11、12）按产品需求排期。

---

## 关键文件索引

| 文件 | 职责 |
|------|------|
| `Frontend/src/components/pipeline/DeployPanel.tsx` | 流水线列表、分页、搜索、批量操作 |
| `Frontend/src/api/pipelineApi.ts` | `listPipelines` 及 API 兼容 |
| `Frontend/src/pages/PipelinePage.tsx` | 设计器、只读模式 |
| `Frontend/src/components/pipeline/ComponentPalette.tsx` | 组件面板 disabled |
| `Frontend/src/components/pipeline/DeployPanel.test.tsx` | 单元测试 |
| `Frontend/src/pages/WorkflowDetailPage.tsx` | Workflow 详情、日志 Modal |
| `Frontend/src/pages/useWorkflowDetail.ts` | 日志 state / follow |
| `Frontend/src/pages/WorkflowExecutionList.tsx` | 执行记录列表 |

---

## 测试环境

```bash
cd cyber-databrew/Frontend
npm run dev:remote   # API 代理到远程 backend，端口 5176
```

回归 URL：`http://127.0.0.1:5176/pipeline?tab=pipelines`

---

# 第二部分：产品视角 Backlog（支撑 G1 / G2）

> **里程碑：** 主要服务 **M0→M1**（列表可运营、作者少踩坑）  
> **与第三部分关系：** P-21～P-26 覆盖 G3/G4 及 M3 产品面；B/C 系列是落地任务

## 核心判断

当前产品形态是 **四个 Tab 拼起来的工具集**（设计 / 流水线 / 执行记录 / 组件），而不是一条连贯的「设计 → 保存 → 发布 → 运行 → 排查」闭环。

**M1 之前**要优先补齐：**信任（G1）+ 列表可理解（G2）**。  
**M1** 要补齐：**批次可观测（G3）+ 可恢复（G4）**——见第三部分。  
**M2/M3** 不在本部分展开，见第四部分 C 系列。

---

## 用户旅程断点

```
资产页选数据 ──→ 选流水线 ──→ 运行 ──→ 看执行 ──→ 失败排查 ──→ 改 DAG ──→ 再发布
     ✅              ⚠️           ✅         ⚠️          ❌           ⚠️          ✅
```

| 断点 | 说明 |
|------|------|
| 资产 → 流水线 | 带了 `asset_ids` 仍要手动找模板，缺「推荐 / 最近使用 / 默认 prod」 |
| 运行 → 执行记录 | 跑完停留在列表或 toast，用户要自己切 Tab 找记录 |
| 执行记录 → 设计 | 知道 workflow 失败了，难一键回到「对应模板 + 版本 + 设计页」 |
| 设计 → 发布 | 有 confirm，但看不到将要发布什么变化（diff 预览） |

---

## 按用户角色：还缺什么

### 流水线作者（设计 + 保存 + 发版）

| 需求 | 现状 | 产品缺口 |
|------|------|----------|
| 从零开始建一条流水线 | 进「设计」Tab，默认名 `my-pipeline` | 无明确「新建」入口；空画布缺引导（示例、步骤说明） |
| 理解 dev / prod / 活跃版本 | 版本 Drawer 有说明 | 心智负担仍高；列表 tag 多，新人易混淆「保存」「发布」「活跃版本」 |
| 基于已有流水线改 | 只能编辑 dev | 缺「从 prod 复制到 dev」/ 克隆 |
| 保存后继续设计 | 保存后跳 URL | **保存会丢 `tab=design`**（编辑已修，保存未修） |
| 改了一半想换 Tab | 无拦截 | 无未保存变更提示，容易误丢工作 |
| 同名多版本管理 | 列表平铺 96 条 | 缺按名称分组/折叠；用户看到「版本碎片」而非「一条流水线」 |

### 运行者（选资产 → 跑流水线 → 看结果）

| 需求 | 现状 | 产品缺口 |
|------|------|----------|
| 从资产页批量跑 | 资产页有「运行 Pipeline」，带 `asset_ids` | 到流水线 Tab 还要再选模板，缺推荐/最近使用 |
| 确认跑的是哪条、哪版 | 卡片有 dev/prod tag | 运行 Modal 里 scope 不醒目；正式跑 prod 时缺强提示 |
| 跑完去哪看 | 单次 toast / 批量跳 batch | 单次运行缺「立即查看执行详情」明确 CTA |
| 重复跑同一套 | 每次手动选 | 无「最近运行」「收藏/固定模板」 |
| 定时/周期跑 | 无 | 产品能力空白（若业务需要，属大需求） |

### 运维 / 排查者（失败 → 定位 → 重试）

| 需求 | 现状 | 产品缺口 |
|------|------|----------|
| 执行失败快速定位 | 执行记录 Tab + Workflow 详情 | 列表 ↔ 模板 ↔ 设计 三处跳转割裂 |
| 看节点日志 | 有日志 Modal | 首轮反馈 loading 久；日志 Tab 体验仍是痛点 |
| 批量任务进度 | 有 BatchJob 页 | 从流水线列表进批量任务的入口弱；**缺批次级节点概览（B-1）** |
| 失败通知 | 无 | 无邮件/站内/webhook 通知，只能主动刷 |

### 管理员 / 协作（团队规模变大后）

| 需求 | 现状 | 产品缺口 |
|------|------|----------|
| 谁删了 / 谁发布了 | 无 | 无操作审计（创建人、发布人、删除时间） |
| 谁能发 prod | 无 | 无权限模型；prod 保护在前端，协作边界不清 |
| 清理废弃模板 | 有批量删 | 缺「长期未运行」「无 prod 的 dev」等治理视图 |

---

## 信息架构与文案

| 问题 | 建议 |
|------|------|
| 「部署」vs「运行」混用 | 设计页统一叫「运行」或「试跑」，列表页叫「运行」；「发布到正式版」单独用词 |
| 卡片显示 `createdAt`，排序用 `updated_at` | 列表应展示**最近更新**，与排序一致 |
| 「全选」 | 应叫「全选当前页」（产品信任问题，见工程任务 #2） |
| 四个 Tab 无主次 | 新用户不知道先点哪个；空状态应带下一步按钮（载入示例 / 新建 / 看文档） |
| dev/prod 筛选 | 缺「全部」的明确默认态；新人不知道默认看到的是混合列表 |

---

## 产品 Backlog（按 ROI 排序）

### 必须做 — 直接影响日常效率

| # | 任务 | 说明 |
|---|------|------|
| P-1 | 保存后留在设计 Tab | `handleSave` navigate 补 `tab=design`（编辑已修，保存未修） |
| P-2 | 未保存离开提示 | 切 Tab / 关页 / 换模板前拦截 |
| P-3 | 列表按「流水线名称」分组 | 同名版本折叠，默认展示最新 dev + prod 状态 |
| P-4 | 运行 Modal 强化 scope | prod 运行时绿色横幅「正在运行正式版」 |
| P-5 | 执行记录 → 打开对应模板/版本 | 闭环排查（对应工程任务 #7） |
| P-6 | 资产页带过来时高亮推荐模板 | 同名 / 最近用过 / 唯一 prod |
| P-21 | **批次节点概览（热力图）** | 10000 任务看每个节点成败分布 → **第三部分 B-1** |
| P-22 | **批次统一 rerun API** | 多 scope 重跑 + 换版 → **第三部分 B-2a/b** |
| P-23 | **勾选批量重试指定子任务** | 子任务列表 + 热力图联动 → **第三部分 B-2c** |
| P-24 | **Pilot 试跑闸门** | 大批量先跑 N 个再全量 → **第三部分 B-3** |
| P-25 | **十万级协调任务（产品面）** | 用户只提交 1 个 10 万任务 → **第四部分 C-6/C-8** |
| P-26 | **按筛选条件创建批（非 inline ID）** | 传 filter 不传 10 万 UUID → **第四部分 C-7** |

### 应该做 — 降低学习成本、减少误操作

| # | 任务 | 说明 |
|---|------|------|
| P-7 | 「新建流水线」主按钮 | 清空画布 + 命名向导 + 可选载入示例 |
| P-8 | 「从正式版复制到 dev」 | fork prod → 可编辑草稿 |
| P-9 | 发布前 diff 摘要 | 改了哪些节点/边（对应工程任务 #6） |
| P-10 | 最近运行 / 常用模板 | 列表顶部或侧栏 |
| P-11 | 新用户引导 | 首次进入：3 步卡片 — 拖组件 → 保存 → 运行 |

### 可以规划 — 团队化、规模化

| # | 任务 | 说明 |
|---|------|------|
| P-12 | 定时/触发式运行 | 若业务需要自动化 |
| P-13 | 运行失败通知 | 至少站内 + 可选邮件 |
| P-14 | 操作审计 | 发布、删除、批量删 |
| P-15 | 权限 | 谁可 publish prod、谁可删 |
| P-16 | 治理视图 | 90 天未运行、仅 dev 无 prod、重复命名 |

### 体验 polish — 有余力再做

| # | 任务 | 说明 |
|---|------|------|
| P-17 | 搜索增强 | 「最近运行过」「仅有 prod」等筛选 |
| P-18 | 卡片展示上次运行状态/时间 | 而不只是创建时间 |
| P-19 | 批量任务统一入口 | 从流水线列表直达 |
| P-20 | 术语与帮助文档 | dev / prod / 活跃版本一页纸 |

---

## 工程任务 ↔ 产品需求对照

| 工程任务 | 产品本质 |
|----------|----------|
| #1 只读禁配置节点 | 信任：「只读」要说做到只读 |
| #2 全选文案 | 预期管理：别让用户以为删了 91 条 |
| #3 Workflow 日志修复 | 运维闭环：失败排查的核心路径 |
| #4 搜索 debounce | 体验细节 |
| #6 发布 diff 预览 | 发版信心：知道改了什么 |
| #7 执行记录 → 模板 | 排查闭环 |
| #10 E2E | 质量保障 |
| #11 跨页全选删除 | 管理员治理需求 |
| B-1 批次节点概览 | 大批量可观测性：每个节点怎么样 |
| B-2a/b 统一 rerun + 锁版本 | 换版/按范围重跑，不必 10000 全重跑 |
| B-2c 勾选批量重试 | 指定子任务批量重试 |
| B-3 Pilot 试跑 | G4 防呆 | M1 |
| C-1～C-5 | 单批 1～3 万稳态 | M2 |
| C-6～C-9 | 逻辑 10 万任务 | M3 |

---

## 合并建议执行顺序

**工程 + 产品交叉优先级：**

```
Phase A（1–2 天，小改高收益）
  工程 #1 #2
  产品 P-1 P-4

Phase B（3–5 天，核心闭环）
  工程 #3 #7
  产品 P-2 P-3 P-5 P-6

Phase C（1–2 周，体验与质量）
  工程 #4 #5 #10
  产品 P-7 P-8 P-9 P-10

Phase D（按需）
  工程 #8 #9 #11 #12
  产品 P-11 ~ P-20
```

**若只能选 3 件产品向的事：**

1. 列表按名称分组 + 折叠版本 — 直接解决「96 条看不懂」
2. 资产 → 运行 → 执行详情 → 回到模板 闭环 — 最高频日常路径
3. 新建/引导 + 保存留 Tab + 未保存提示 — 作者流失和误操作

---

# 第三部分：大批量运行专项（M1：单批 10,000）

> **里程碑：** M1 可运营  
> **范围：** 在**现有 10,000/批硬上限内**，解决可观测、可恢复、可试跑  
> **不解决：** 突破 10,000 上限、Worker 化（见第四部分 C 系列）

> 更新时间：2026-06-14（细化：节点可观测性 + 指定重试 + 实现拆解）  
> 触发场景：流水线 10 个节点 × 10000 个资产批量跑

---

## 3.0 现状能力矩阵（FAQ）

### Q1：跑 10000 条，能否**快速**知道哪些节点有问题？

**答：不能（批次级）。** 只能知道「有多少条 workflow 失败」，不能一眼看出「10 个节点各自成败分布」。

| 观测层级 | 页面 / API | 能看到什么 | 能否快速定位问题节点 |
|----------|------------|------------|----------------------|
| L0 批次总览 | `/pipeline/batch/:id` | 成功/失败/总数、进度条 | ❌ 只有子任务级成败 |
| L1 子任务列表 | 同上，嵌入 `WorkflowExecutionList` | 每条资产 1 个 workflow 状态 | ⚠️ 知哪条资产挂，不知挂第几步 |
| L2 单条详情 | `/pipeline/executions/:name` | DAG 10 节点、日志、耗时、成本 | ✅ 需**逐条点开** |
| L3 节点矩阵（单条） | Workflow 详情「节点明细」 | `pipeline_run_asset_nodes` 按节点展开 | ✅ 仅 1 条 run |

**现有凑合排查路径（慢，不适合 10000 规模）：**

```
Batch 详情 → 子任务列表筛 Failed（如 177 条）
  → 人工点开 3～5 条 Workflow 详情
  → 看 DAG 哪个节点 Failed
  → 若均落在同一 pipelineNodeId → 可推断问题节点
```

**发现不了的场景：** 节点逻辑写错但状态 `Succeeded` → 失败率 0%，任何基于状态的聚合都无效。

---

### Q2：能否**批量重试指定的**任务？

**答：部分可以，不够灵活。**

| 操作 | API / 入口 | 作用范围 | 是否「指定」 | 现状 |
|------|------------|----------|--------------|------|
| 重试失败项 | `POST /backfill/:id/retry-failed` | 该 batch **全部** `failed` 子项 | ❌ 不可勾选 | ✅ 已有 |
| 暂停 | `POST /backfill/:id/pause` | 停止调度新 pending | — | ✅ 已有 |
| 继续 | `POST /backfill/:id/resume` | 恢复 pending 调度 | — | ✅ 已有 |
| 单条重试 | `POST /workflows/:name/retry` | 1 条 workflow | ✅ 单条 | ✅ 逐条操作菜单 |
| 勾选批量重试 | — | 用户选中的 N 条子任务 | ✅ 期望能力 | ❌ **无 API/UI** |
| 重跑已完成项 | — | `completed` 但结果不可信 | ✅ 期望能力 | ❌ **无** |
| 换新版重试 | — | 修 DAG 后用 vN 定义重跑 | ✅ 期望能力 | ❌ **无** |

**子任务列表 UI 现状（`BatchJobDetailPage` + `WorkflowExecutionList embedded`）：**

- 表格支持 `rowSelection` 勾选
- **无**「重试选中项」按钮
- `embedded=true` 时隐藏「批量删除」，保留「对比选中」（限 2～3 条）
- 单条「操作 → 重试」可用，10000 条不现实

**重试语义限制：**

1. `retry-failed` 只处理 `BackfillItem.status=failed`，**不碰 completed**
2. 重试使用 batch 创建时的 `templateId`，**未锁定 templateVersion**
3. Argo `retry` = 同一份 workflow 定义内重试失败步骤 ≠ 换 DAG 定义后重跑

---

### Q3：跑一半发现节点错了，一定要 10000 全重跑吗？

**答：视情况。**

| 情况 | 现在怎么做 | 浪费程度 |
|------|------------|----------|
| 还没跑完（如 5000 pending） | **立即暂停** → 修定义 → 继续/重试失败 | 仅已完成部分可能浪费 |
| 已完成但结果错（5000 completed） | 无批量入口；需新 batch 或手动再跑 | **高** |
| 仅 failed（177 条） | 「重试失败项」 | 低，但仍是旧定义（除非碰巧 active version 变了） |
| 修完定义要用新版本 | 无「换版重跑」 | **高** |

---

## 3.1 数据模型与现有 API（实现参考）

### 核心表 / 模型

```
backfill_jobs          id, template_id, total_count, completed_count, failed_count, status
backfill_items         id, job_id, asset_id, status, pipeline_run_id, workflow_name
pipeline_runs          id, batch_job_id, template_version, workflow_name, status
pipeline_run_asset_nodes   run_id, pipeline_node_id, display_name, status, message, ...
```

| 字段缺口 | 影响 | 建议在 B-2 补 |
|----------|------|---------------|
| `backfill_jobs.template_version` | 无法审计「这批跑的是 v几」 | 创建时写入 |
| `backfill_items.last_rerun_at` | 无法区分原 run 与重跑 run | 可选 |
| 无 `rerun` 接口 | 无法指定范围/版本重跑 | 新增 |

### 现有 Backfill API

| Method | Path | 说明 |
|--------|------|------|
| POST | `/api/v1/backfill` | 创建 batch，`{ name, templateId, assetIds[] }`，上限 10000 |
| GET | `/api/v1/backfill/:id` | 批次详情 |
| POST | `/api/v1/backfill/:id/pause` | 暂停 |
| POST | `/api/v1/backfill/:id/resume` | 继续（重调度 pending） |
| POST | `/api/v1/backfill/:id/retry-failed` | **全量**重试 failed 子项 |

### 现有单条 Workflow API（子任务级）

| Method | Path | 说明 |
|--------|------|------|
| POST | `/api/v1/workflows/:name/retry` | Argo retry，同定义 |
| POST | `/api/v1/workflows/:name/resubmit` | 整条重新提交 |
| GET | `/api/v1/pipeline-runs?view=summary&batchJobId=` | 子任务列表数据源 |

---

## B-1. 批次节点概览（热力图）— P0

### 用户故事

> 作为运维，我在 batch 跑了 10000 条后，希望在 **3 秒内** 看到 10 个节点各自成功/失败/运行中数量，并点击失败数下钻到具体资产，而不必打开 10000 条 Workflow 详情。

### 目标指标

| 指标 | 目标 |
|------|------|
| 首屏加载（10000 batch） | 节点概览 API P95 < 2s |
| 下钻列表分页 | 20 条/页，P95 < 500ms |
| 数据新鲜度 | 与 batch 刷新同步，延迟 ≤ run watcher 周期（文档注明） |

### UI 规格（`BatchJobDetailPage` 新增区块）

**位置：** 进度条 Card 下方、子任务列表上方。

**组件：** `BatchNodeSummaryPanel.tsx`

```
┌─ 节点概览 ─────────────────────────────────────────────────┐
│ 数据基于 4823 已完成 · 177 失败 · 23 运行中 · 4977 未开始      │
│ [刷新]  最后更新：12:34:56                                    │
├────────────┬──────┬──────┬──────┬──────┬────────┬────────────┤
│ 节点( DAG序)│ 成功 │ 失败 │运行中│ 未开始│ 失败率 │ 操作       │
├────────────┼──────┼──────┼──────┼──────┼────────┼────────────┤
│ 1 预处理    │ 4823 │ 0    │ 0    │ 5177 │ 0%     │ —          │
│ 2 抽特征 ⚠ │ 4800 │ 23   │ 0    │ 5177 │ 0.48%  │ 23失败 ▶   │
│ ...        │      │      │      │      │        │            │
└────────────┴──────┴──────┴──────┴──────┴────────┴────────────┘
```

**交互：**

| 操作 | 行为 |
|------|------|
| 点击失败数 | 打开 Drawer：该节点失败资产列表（分页） |
| 点击运行中 | Drawer：Running 资产列表 |
| 行失败率 > 阈值（默认 1%） | 行背景橙色 + ⚠ 图标 |
| 无 node 数据 | Alert「节点状态同步中，请稍后刷新」 |
| 从 Drawer 点资产 | 跳转 `/pipeline/executions/:workflowName` 并定位节点 |

**可选增强（P1）：**

- 堆叠条：每行 `成功|失败|运行中|未开始` 占比
- 与 DAG 设计页节点 `label` 一致（`displayName` fallback `pipelineNodeId`）
- 自动刷新：batch `status=running` 时每 30s poll `node-summary`

### 后端 API 1：节点聚合

```
GET /api/v1/backfill/:id/node-summary
```

**Query：** 无（或 `?refresh=1` 强制 reconcile）

**Response 200：**

```json
{
  "batchJobId": "uuid",
  "templateId": "tmpl-xxx",
  "templateVersion": 3,
  "subtasks": {
    "total": 10000,
    "completed": 4823,
    "failed": 177,
    "running": 23,
    "pending": 4977,
    "paused": false
  },
  "nodes": [
    {
      "pipelineNodeId": "step-extract",
      "displayName": "抽特征",
      "dagOrder": 2,
      "counts": {
        "Succeeded": 4800,
        "Failed": 23,
        "Running": 0,
        "Pending": 5177,
        "Skipped": 0,
        "Omitted": 0
      },
      "attempted": 4823,
      "failureRate": 0.0048,
      "topFailureReasons": [
        { "message": "exit code 1", "count": 18 },
        { "message": "OOMKilled", "count": 5 }
      ]
    }
  ],
  "dataCoverage": {
    "runsWithNodeRows": 5000,
    "runsTotal": 5000,
    "complete": true
  },
  "generatedAt": "2026-06-14T12:00:00Z"
}
```

**`counts` 状态枚举（与 Argo / 账本对齐）：**

`Pending | Running | Succeeded | Failed | Error | Skipped | Omitted`

**`未开始` 计算规则（实现二选一，文档固定一种）：**

| 方案 | 规则 | 优点 | 缺点 |
|------|------|------|------|
| **A 行级统计（推荐 v1）** | 仅统计 `pipeline_run_asset_nodes` 已有行；无行 = 该节点对该 run 未开始 | 实现简单、SQL 直接 | 进行中的 run 可能少计 |
| **B 推导补全** | 对 Running run：失败节点之后的 DAG 序标记 Pending | 更接近直觉 | 需 pipeline JSON 拓扑排序 |

**v1 采用方案 A**；`Pending` 列含义 = 「尚无该节点账本行的子任务数」需在 UI tooltip 说明。

**聚合 SQL（方案 A）：**

```sql
-- 按节点+状态计数
SELECT
  n.pipeline_node_id,
  COALESCE(NULLIF(n.display_name, ''), n.pipeline_node_id) AS display_name,
  n.status,
  COUNT(*) AS cnt
FROM pipeline_run_asset_nodes n
INNER JOIN pipeline_runs r ON r.id = n.run_id
WHERE r.batch_job_id = $1
GROUP BY 1, 2, 3;

-- 子任务总数（校验用）
SELECT status, COUNT(*) FROM backfill_items WHERE job_id = $1 GROUP BY 1;

-- Top 失败原因（可选）
SELECT n.pipeline_node_id, n.message, COUNT(*) AS cnt
FROM pipeline_run_asset_nodes n
INNER JOIN pipeline_runs r ON r.id = n.run_id
WHERE r.batch_job_id = $1 AND n.status IN ('Failed', 'Error')
GROUP BY 1, 2
ORDER BY cnt DESC
LIMIT 20;
```

**`dagOrder`：** 从 batch 关联 `templateId` 的 `pipeline_json.nodes` 拓扑排序写入响应（缓存 5min）。

### 后端 API 2：节点失败下钻

```
GET /api/v1/backfill/:id/node-failures
```

**Query：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `pipelineNodeId` | ✅ | 设计页节点 id |
| `status` | | 默认 `Failed,Error`；可传 `Running` |
| `page` | | 默认 1 |
| `pageSize` | | 默认 20，max 100 |
| `q` | | 资产 ID 模糊搜索 |

**Response 200：**

```json
{
  "items": [
    {
      "backfillItemId": "item-uuid",
      "assetId": "ast-001",
      "runId": "run-uuid",
      "workflowName": "my-pipeline-abc123",
      "pipelineNodeId": "step-extract",
      "displayName": "抽特征",
      "status": "Failed",
      "message": "main: exit code 1",
      "startedAt": "...",
      "finishedAt": "..."
    }
  ],
  "total": 23,
  "page": 1,
  "pageSize": 20
}
```

### 实现任务拆解（B-1）

| 序号 | 任务 | 层 | 估时 |
|------|------|-----|------|
| B-1.1 | 定义 `BatchNodeSummary` / `BatchNodeFailureItem` model | backend | 0.5d |
| B-1.2 | repo：`AggregateNodeStatusByBatchJobID` | backend | 1d |
| B-1.3 | usecase：`GetBatchNodeSummary` + dagOrder | backend | 1d |
| B-1.4 | handler + routes + openapi | backend | 0.5d |
| B-1.5 | handler/repo 单测（mock 10000 行聚合） | backend | 0.5d |
| B-1.6 | `batchJobApi.getNodeSummary` / `listNodeFailures` | frontend | 0.5d |
| B-1.7 | `BatchNodeSummaryPanel` + Drawer 下钻 | frontend | 1.5d |
| B-1.8 | 接入 `BatchJobDetailPage`，刷新联动 | frontend | 0.5d |
| B-1.9 | 组件测试 + e2e 冒烟（有 batch 数据环境） | qa | 1d |

**合计约 7 人天**

### 验收标准

- [ ] 10000 batch 打开详情，节点概览 <3s（不拉全量子任务明细）
- [ ] 10 节点行可见，`attempted` 之和 ≤ `subtasks.completed + failed + running`
- [ ] 点击失败数 → Drawer 分页 → 跳转 Workflow 详情
- [ ] `dataCoverage.complete=false` 时显示「同步中」非空表
- [ ] 失败率 >1% 行高亮
- [ ] 刷新按钮同时更新进度条 + 节点概览

### 局限（UI 必须注明）

- 逻辑错但 `Succeeded` → 热力图无法发现 → 配合 B-3 Pilot + 校验节点
- 依赖 watcher 写入 `pipeline_run_asset_nodes` 的及时性
- v1 `Pending` 为「无账本行」而非严格 DAG 推导

---

## B-2. 批次重跑与指定重试 — P0

拆为三个子能力：**B-2a 指定范围重试**、**B-2b 换版重跑**、**B-2c 勾选批量重试 UI**。

### B-2a. 统一重跑 API（替代仅 `retry-failed`）

```
POST /api/v1/backfill/:id/rerun
```

**Request body：**

```json
{
  "scope": "failed",
  "templateId": "tmpl-v4-id",
  "templateVersion": 4,
  "itemIds": [],
  "assetIds": [],
  "pipelineNodeId": "",
  "dryRun": false
}
```

**`scope` 枚举：**

| scope | 选中 backfill_items | 典型场景 |
|-------|---------------------|----------|
| `failed` | `status=failed` | 兼容现有「重试失败项」 |
| `pending` | `status=pending` | 暂停后手动触发剩余 |
| `incomplete` | `pending` + `failed` | 未完成全量继续（可换版） |
| `completed` | `status=completed` | **结果不可信，修定义后重跑** |
| `custom` | `itemIds[]` 或 `assetIds[]` 交集 | **用户指定任务** |
| `node_failed` | 通过 B-1 下钻：该节点 Failed 的 item | **从热力图一键重试** |

**`dryRun: true`：** 只返回将影响条数，不执行。

**Response 200：**

```json
{
  "status": "accepted",
  "dryRun": false,
  "matchedCount": 23,
  "templateId": "tmpl-v4-id",
  "templateVersion": 4,
  "skipped": [
    { "itemId": "...", "reason": "already_running" }
  ]
}
```

**执行语义（每条 matched item）：**

1. 将 item 置 `pending`，清空 error（保留 `pipeline_run_id` 历史可选：新建 run 推荐 **新 run_id**）
2. `UpsertBatchSubtaskRun` 新 ledger 行
3. `DeployByTemplateID(templateId, assetId, DeployOptions{ TemplateVersion, BatchJobID, PreallocatedRunID })`
4. 异步 `runItems` 与现逻辑一致

**与现有 `retry-failed`：** 保留路径，内部转调 `rerun{scope:failed}`，避免破坏兼容。

### B-2b. 创建时锁定模板版本

**`CreateBackfill` 改动：**

```go
// 请求体扩展
type CreateBackfillRequest struct {
    Name            string   `json:"name"`
    TemplateID      string   `json:"templateId"`
    AssetIDs        []string `json:"assetIds"`
    TemplateVersion *int     `json:"templateVersion,omitempty"` // 可选，默认解析当前
    PilotCount      int      `json:"pilotCount,omitempty"`      // B-3
}

// BackfillJob 扩展
TemplateVersion int `json:"templateVersion"`
PilotCount      int `json:"pilotCount,omitempty"`
PilotPhase      string `json:"pilotPhase,omitempty"` // none|running|review|done
```

创建时：

1. `FindByID(templateId)` 解析 version（与 `DeployByTemplateID` 规则一致）
2. 写入 `job.TemplateVersion`
3. 后续该 batch 默认重跑使用此 version，除非 `rerun` 显式指定新版

### B-2c. 前端：勾选批量重试

**位置：** `BatchJobDetailPage` 子任务列表工具栏（`embedded` 模式也显示）

```
[暂停] [继续] [重跑 ▼] [刷新]

子任务列表：
  ☑ 已选 12 条
  [重试选中项]  [重试全部失败]  [对比选中]
```

**「重跑 ▼」下拉：**

| 菜单项 | 调用 |
|--------|------|
| 重试全部失败 | `rerun { scope: "failed" }` |
| 重试未完成 | `rerun { scope: "incomplete" }` |
| 重试已完成（换版…） | Modal：选 template 版本 → `scope: "completed"` |
| 重试选中项 (N) | `scope: "custom", itemIds: [...]` |

**从 B-1 热力图联动：**

- 节点行「23 失败 ▶」Drawer 底栏：`[重试这 23 条]` → `scope: "node_failed", pipelineNodeId: "..."`

**确认 Modal 必含：**

- 影响条数 N
- 模板名称 + **版本 v4**（可改）
- 警告：「将重新提交 workflow，旧 run 记录保留」
- `dryRun` 预检条数

### 实现任务拆解（B-2）

| 序号 | 任务 | 层 | 估时 |
|------|------|-----|------|
| B-2.1 | DB migration：`backfill_jobs.template_version` 等 | backend | 0.5d |
| B-2.2 | `CreateBackfill` 锁定 version | backend | 0.5d |
| B-2.3 | `Rerun` usecase（scope 解析 + dryRun） | backend | 2d |
| B-2.4 | `scope=node_failed` 联查 asset_nodes | backend | 1d |
| B-2.5 | handler + openapi；`retry-failed` 转调 | backend | 0.5d |
| B-2.6 | usecase 单测（各 scope + 换版） | backend | 1d |
| B-2.7 | `batchJobApi.rerun` + 创建时传 version | frontend | 0.5d |
| B-2.8 | Batch 详情「重跑」下拉 + 确认 Modal | frontend | 1d |
| B-2.9 | 子任务列表「重试选中项」 | frontend | 0.5d |
| B-2.10 | B-1 Drawer「重试这 N 条」联动 | frontend | 0.5d |
| B-2.11 | e2e：勾选 3 条 → dryRun → 确认重跑 | qa | 1d |

**合计约 9 人天**（可与 B-1 并行后端）

### 验收标准

- [ ] `retry-failed` 行为与现网一致（回归）
- [ ] 勾选 12 条 →「重试选中项」→ 仅 12 条重新调度
- [ ] `scope=completed` + `templateVersion=4` → 只重跑已完成，pending 不动
- [ ] `dryRun` 返回 `matchedCount` 与 UI 一致
- [ ] 热力图节点失败 →「重试这 23 条」→ 23 条进入 running
- [ ] 旧 `pipeline_runs` 保留，新 run 新 ID
- [ ] 创建 batch 后 `job.templateVersion` 可查

---

## B-3. Pilot 试跑闸门 — P1

### 用户故事

> 作为运行者，提交 10000 资产 batch 时，希望**先自动跑 50 条**，我在节点概览 + 抽样输出确认无误后，再一键跑剩余 9950，避免跑一半才发现节点配置错误。

### 流程（状态机）

```
创建 batch(pilotCount=50, total=10000)
    │
    ▼
pilot_running ──仅调度 50 条──► pilot_review（50 跑完或部分失败）
    │                              │
    │                              ├─「继续全量」──► running（剩余 9950）
    │                              ├─「暂停」──────► paused
    │                              └─ B-2 换版重跑 pilot 子集
    ▼
running / paused / completed（与现网一致）
```

### 创建入口 UI

运行 Modal / 资产 batch 创建：

```
☑ 启用试跑闸门
  试跑数量：[ 50 ]（1 ~ min(500, total)）
  说明：试跑通过后再跑剩余 N 条
```

### API 扩展

**创建：**

```json
POST /api/v1/backfill
{
  "name": "batch-10k",
  "templateId": "...",
  "assetIds": ["..."],
  "pilotCount": 50
}
```

**继续全量：**

```
POST /api/v1/backfill/:id/continue-full
```

仅当 `pilotPhase=review` 且用户确认后可调用。

### 实现任务拆解（B-3）

| 序号 | 任务 | 估时 |
|------|------|------|
| B-3.1 | `pilotPhase` 状态机 + 仅调度 pilot 子集 | 1.5d |
| B-3.2 | `continue-full` API | 0.5d |
| B-3.3 | 创建 UI 试跑开关 | 0.5d |
| B-3.4 | Batch 详情 Pilot 阶段 Banner + 继续按钮 | 1d |
| B-3.5 | 单测 + e2e | 1d |

**合计约 4.5 人天**

### 验收标准

- [ ] `pilotCount=50, total=10000`：首屏仅 50 条 running/completed
- [ ] Pilot 阶段 B-1 节点概览仅统计已跑 subset
- [ ] 「继续全量」后剩余 9950 进入调度
- [ ] Pilot 未确认前不可误触全量（按钮 disabled + 说明）

---

## B-4. 步骤级输出缓存（长期）— P3

### 用户故事

> DAG 为 A→B→C，仅 B 节点定义变更。重跑时 A 的输出可复用，只重算 B、C，节省算力。

### 技术方向

| 项 | 说明 |
|----|------|
| 缓存键 | `hash(assetId + pipelineNodeId + nodeDefHash + inputArtifactHashes)` |
| 存储 | R2 / GCS artifact + DB 索引表 `pipeline_node_output_cache` |
| 执行 | Argo memoization 或自定义 skip 节点注入 |
| 与 B-2 联动 | `rerun` 响应增加 `skippedNodesDueToCache` |

### 实现任务（远期，仅列方向）

- B-4.1 节点定义 hash 计算（pipeline compile 时）
- B-4.2 产物登记与命中查询
- B-4.3 重跑时 partial DAG 编译
- B-4.4 UI 显示「缓存命中 N 步，节省预估 $X」

### 验收标准（远期）

- [ ] 仅改 B 后重跑，A 节点 0 次 Pod 创建（可观测）
- [ ] 缓存未命中时回退全量重算

---

## 3.2 运维场景 Runbook（实现后预期操作）

### 场景 A：跑 10000，中途发现节点 2 脚本错误

```
1. Batch 详情 → 立即「暂停」
2. 查看「节点概览」→ 确认节点 2 失败率 / 是否已有 completed 带错结果
3. 设计页修 DAG → 保存 v4
4. 「重跑 ▼」→「重试已完成（换版 v4）」或「重试选中项」
5. 节点概览确认节点 2 失败率归零后 →「继续」跑剩余 pending
```

### 场景 B：仅 23 条资产在节点 2 失败，其余正常

```
1. 节点概览 → 节点 2「23 失败」→ Drawer
2. Drawer 底栏「重试这 23 条」（scope=node_failed）
3. 无需动其他 9977 条
```

### 场景 C：新建 10000 batch，想避免翻车

```
1. 创建时勾选 Pilot 50
2. 50 条跑完 → 节点概览 + 抽 3 条看输出
3. 「继续全量」
```

### 场景 D：当前系统（未实现 B-1/B-2）临时做法

```
1. 暂停 batch
2. 子任务列表筛 Failed → 人工点开 3～5 条看 DAG
3. 修定义后只能 retry-failed（failed）或新建 batch（completed 全重跑）
```

---

## 第三部分任务编号速查（细化版）

| 编号 | 任务 | 优先级 | 依赖 | 估时 |
|------|------|--------|------|------|
| B-1 | 批次节点概览 API + UI | P0 | 无 | ~7d |
| B-2a | 统一 `rerun` API（多 scope） | P0 | 无 | ~5d |
| B-2b | 创建锁定 templateVersion | P0 | 无 | ~1d |
| B-2c | 勾选批量重试 + 热力图联动 | P0 | B-1,B-2a | ~3d |
| B-3 | Pilot 试跑闸门 | P1 | B-2b | ~4.5d |
| B-4 | 步骤级输出缓存 | P3 | B-2 | TBD |

**B-1 + B-2 可并行开发，合计约 2 周（1 后端 + 1 前端）。**

---

## 测试计划（大批量专项）

| 用例 | 步骤 | 期望 |
|------|------|------|
| T-B1-1 | 打开 1000+ batch 详情 | 节点概览 <3s 展示 10 行 |
| T-B1-2 | 点击某节点失败数 | Drawer 分页正确 |
| T-B2-1 | `dryRun scope=failed` | matchedCount = failedCount |
| T-B2-2 | 勾选 5 条 → 重试选中 | 仅 5 条变 pending/running |
| T-B2-3 | `scope=completed` + v4 | completed 重跑，pending 不变 |
| T-B2-4 | 热力图 23 失败 → 重试 | 23 条重调度 |
| T-B2-5 | 回归 `retry-failed` | 与现网行为一致 |
| T-B3-1 | pilotCount=10 | 仅 10 条先跑 |
| T-B3-2 | continue-full | 剩余开跑 |

**E2E 文件建议：** `Frontend/e2e/pipeline-batch-node-summary.spec.ts`

---

## 更新后的合并执行顺序（与里程碑对齐）

```
┌─────────────────────────────────────────────────────────────────┐
│ M0→M1  流水线列表 + 单批 1 万可运营                                │
├─────────────────────────────────────────────────────────────────┤
│ Phase A（1–2 天）目标：G1 信任补齐                                  │
│   工程 #1 #2  产品 P-1 P-4                                       │
│   出口：只读真只读；保存留 design Tab                                │
├─────────────────────────────────────────────────────────────────┤
│ Phase B（3–5 天）目标：G2 日常闭环                                  │
│   工程 #3 #7  产品 P-2 P-3 P-5 P-6                               │
│   出口：排查链路通；模板列表可理解                                   │
├─────────────────────────────────────────────────────────────────┤
│ Phase B+（约 2 周）目标：G3 + G4（M1 核心）                         │
│   B-1 节点概览 → B-2a/b rerun → B-2c 勾选重试                      │
│   可选：B-3 Pilot                                                │
│   出口：1 万批能看清节点、能指定重跑、能试跑                          │
├─────────────────────────────────────────────────────────────────┤
│ M1→M2  单批 1～3 万稳态                                            │
├─────────────────────────────────────────────────────────────────┤
│ Phase C+（3–6 周）目标：架构扛住更大批量                              │
│   C-1 Worker  C-2 懒物化  C-3 并发可配  C-4 items 分页  C-5 限流 │
│   出口：API 重启不丢批；单批 3 万可创建可跑完（集群允许前提下）        │
├─────────────────────────────────────────────────────────────────┤
│ M2→M3  逻辑 10 万任务                                              │
├─────────────────────────────────────────────────────────────────┤
│ Phase D（1–2 月）目标：用户视角一个 10 万任务                         │
│   C-6 协调器  C-7 filter_json 流式  C-8 统一进度  C-9 观测增强     │
│   B-4 步骤缓存（可选）                                             │
│   出口：提交 100k 筛选条件 → 自动拆批 → 统一仪表盘                   │
├─────────────────────────────────────────────────────────────────┤
│ 持续 polish                                                      │
│   工程 #4 #5 #8 #9 #10 #11 #12  产品 P-7～P-20                   │
└─────────────────────────────────────────────────────────────────┘
```

**按里程碑选型（给决策用）：**

| 你若需要… | 做到哪一档 | 最小任务集 |
|-----------|------------|------------|
| 列表好用、prod 安全 | M0+ | Phase A |
| 1 万批能排查、能局部重跑 | **M1** | Phase A + B + **B+（B-1,B-2）** |
| 1 万批尽量不翻车 | M1+ | 上列 + **B-3 Pilot** |
| 单批突破 1 万或 API 可靠跑批 | M2 | M1 + **C-1～C-5** |
| 一次提交 10 万资产 | M3 | M2 + **C-6～C-9** |

**若只能选 3 件「大批量」相关的事（M1）：**

1. **B-1** — G3：每个节点大概什么情况  
2. **B-2a + B-2c** — G4：批量重试指定任务 + 换版  
3. **B-3** — G4：Pilot 试跑防呆  

**当前系统（M0）临时能力：**

- 快速看节点问题：❌ 只能逐条进 Workflow 详情  
- 批量重试指定任务：❌ 仅 `retry-failed` 全量或单条操作菜单  
- 止血：✅ 暂停 batch  
- 一次提交 100k：❌ 硬限 10000  

---

## 扩展关键文件索引

| 文件 | 职责 |
|------|------|
| `Frontend/src/pages/BatchJobDetailPage.tsx` | Batch 详情、暂停/重试、节点概览入口、重跑下拉 |
| `Frontend/src/api/batchJobApi.ts` | Batch API；**扩展** `getNodeSummary` / `rerun` |
| `Frontend/src/components/pipeline/BatchNodeSummaryPanel.tsx` | **待建** 节点热力图 + 失败下钻 Drawer |
| `Frontend/src/components/pipeline/BatchRerunModal.tsx` | **待建** 重跑确认（scope/版本/dryRun） |
| `Frontend/src/pages/WorkflowExecutionList.tsx` | 子任务列表；**扩展** embedded 模式「重试选中项」 |
| `backend/internal/usecase/backfill/usecase.go` | pause/resume/retry；**扩展** `GetNodeSummary` / `Rerun` |
| `backend/internal/handlers/backfill/handler.go` | **扩展** `GET node-summary`、`POST rerun` |
| `backend/internal/models/backfill.go` | BackfillJob / BackfillItem；**扩展** templateVersion |
| `backend/internal/models/pipeline.go` | `PipelineRunAssetNode`、run 账本 |
| `backend/internal/postgres/pipeline_repo.go` | `pipeline_run_asset_nodes` 聚合查询 |
| `backend/internal/postgres/backfill_repo.go` | backfill_items scope 查询 |
| `backend/routes/routes.go` | 注册新路由 |
| `api/openapi.yaml` | 补充 backfill 端点文档 |
| `Frontend/e2e/pipeline-batch-node-summary.spec.ts` | **待建** 大批量专项 e2e |
| `Frontend/src/pages/WorkflowDetailPage.tsx` | 单条节点明细（下钻终点） |
| `Frontend/src/components/pipeline/WorkflowNodeSummaryTable.tsx` | 可复用节点表列定义 |

---

# 第四部分：架构容量与规模化（M2 / M3）

> **里程碑：** M2（单批 1～3 万稳态）→ M3（逻辑 10 万任务）  
> **前提：** M1（B 系列）完成后再做；否则只会在脆弱调度上叠功能

---

## 4.0 容量评估结论（FAQ）

### 现在能跑多少？

| 问题 | 答案 |
|------|------|
| 单批最多多少？ | **10,000**（前后端硬编码 `MaxBackfillAssetCount`） |
| 一次 POST 100,000 个 assetId？ | **拒绝**（400 `too many assets`） |
| 10,000 能跑完吗？ | **能**，但耗时主要取决于 **K8s 算力**，不是 PG |
| 架构能撑 100,000 吗？ | **不能**（调度、物化、观测、执行平面均未按 10 万设计） |

### 当前架构瓶颈（代码事实）

| 组件 | 现值 | 10k 影响 | 100k 影响 |
|------|------|----------|-----------|
| `MaxBackfillAssetCount` | 10,000 | 顶满上限 | 直接拒绝 |
| `maxConcurrentBatchItems` | **5** | 吞吐低；10k×15min/5 ≈ **~21 天**（粗算） | 不可接受 |
| 调度载体 | API 内 `go materializeAndRunBatch()` | 实例重启可能卡批 | 完全不适合 |
| 创建时物化 | 每条 `UpsertBatchSubtaskRun` 预建 `pipeline_run` | 1 万次 DB 写 | 10 万次不可行 |
| `FindItemsByJobID` | 全量加载 items | 1 万行尚可 | 内存/延迟风险 |
| Run watcher | `activeScanLimit` 默认 **100**/轮 | 状态同步滞后 | 严重滞后 |
| 子任务 UI 列表 | `listPipelineRuns` 分页 ✅ | 可用 | 需协调器汇总 |
| `filter_json` 字段 | schema 有，**创建 API 未用** | 只能 inline assetIds | 无法流式喂资产 |

### 存储膨胀估算（10 节点 DAG）

| 规模 | backfill_items | pipeline_runs | pipeline_run_asset_nodes（约） |
|------|----------------|---------------|------------------------------|
| 10,000 | 1 万 | 1 万（预建） | **10 万** |
| 100,000 | 10 万 | 10 万 | **100 万** |

PG 在 M2 前可扛 10k 级；**瓶颈是调度模型与 K8s，不是列表分页。**

### 四层能力模型（100k 差距）

```
L1 产品/API     ❌ 硬限 1 万；无协调任务 API
L2 调度执行     ❌ 5 并发；无队列 Worker
L3 数据物化     ❌ 预建全量 ledger；全量 FindItems
L4 执行平面     ❓ 取决于集群；需 Argo 提交限流
```

---

## 4.1 M2 目标定义（单批 1～3 万稳态）

**完成定义：**

- [ ] 单批上限可配置至 **30,000**（或保持 10k 但允许**多批编排**，二选一在 C-6 定案）
- [ ] Backfill 调度在 **独立 Worker** 中执行，Cloud Run API 重启**不丢失**进行中的批
- [ ] 创建批任务时**不预建**全部 `pipeline_run`（懒物化，执行时再建）
- [ ] `maxConcurrentBatchItems` 来自配置（如 20～50），且有 **Argo 侧限流**
- [ ] `GET /backfill/:id/items` 分页；禁止运维路径全量 load 1 万+ items 进内存
- [ ] 10k 批创建物化 P95 **< 60s**（不含执行）

---

## C 系列任务拆解

### C-1. Backfill Worker 与 API 解耦 — P0（M2 入口）

| 项 | 内容 |
|----|------|
| 问题 | `materializeAndRunBatch` / `runItems` 在 API 进程 goroutine 内，实例重启丢批 |
| 方案 | 引入队列（Pub/Sub / Cloud Tasks）+ Worker（Cloud Run Job 或 K8s Deployment） |
| API 职责 | 只写 `backfill_jobs` + 发消息 + 返回 202 |
| Worker 职责 | 物化、调度、`executeItem`、进度同步 |
| 涉及 | 新 `cmd/backfill-worker` 或扩展现有 server 模式；`usecase/backfill` 拆分 |
| 验收 | 杀掉 API 实例后 Worker 继续跑；job 状态正确推进 |

**估时：** 5～8 人天

---

### C-2. 懒物化 pipeline_run — P0（M2）

| 项 | 内容 |
|----|------|
| 问题 | 创建 1 万条时同步 `UpsertBatchSubtaskRun` × 10000，耗时长、占连接 |
| 方案 | 创建时只写 `backfill_items`；在 `executeItem` 首次调度时再 `UpsertBatchSubtaskRun` |
| 兼容 | 已有 pre-allocated run_id 逻辑可保留为可选优化 |
| 验收 | 创建 10k 批 API 返回后 **<10s** 完成 items 入库；pipeline_runs 行数随执行逐步增长 |

**估时：** 3～5 人天（依赖 C-1 更易落地）

---

### C-3. 可配置并发 + Argo 限流 — P0（M2）

| 项 | 内容 |
|----|------|
| 问题 | `maxConcurrentBatchItems = 5` 写死，无法按集群容量调节 |
| 方案 | 配置项 `BACKFILL_MAX_CONCURRENT`（默认 5，上限如 100）；全局 token bucket 限制对 Argo 的提交 QPS |
| 观测 | Worker metrics：`backfill_items_in_flight`、`argo_submit_errors` |
| 验收 | 调到 20 后吞吐上升且 Argo 5xx 不超阈值；调到 5 行为与现网一致 |

**估时：** 3～4 人天

---

### C-4. backfill_items 分页 API — P1（M2）

```
GET /api/v1/backfill/:id/items?page=1&pageSize=50&status=failed
```

| 项 | 内容 |
|----|------|
| 问题 | `FindItemsByJobID` 全量；resume/retry 加载全部 items |
| 方案 | 分页查询 + retry/rerun 按 scope 在 DB 侧过滤，流式处理 |
| 验收 | 3 万 items job 的 resume 不出现 OOM；API P95 < 500ms |

**估时：** 2～3 人天

---

### C-5. 提升单批上限至 30,000（可选）— P2（M2）

| 项 | 内容 |
|----|------|
| 前提 | C-1～C-4 完成 |
| 改动 | `MaxBackfillAssetCount`、前端 `MAX_BATCH_ASSET_COUNT`、e2e |
| 风险 | 需压测物化 + PG 索引；需用户确认 K8s 容量 |
| 验收 | 30k 批创建成功；Worker 稳定运行 24h |

**估时：** 1～2 人天 + 压测

---

### C-6. 十万级协调任务（Coordinator）— P0（M3 入口）

| 项 | 内容 |
|----|------|
| 用户视角 | 提交 **1 个**「10 万资产」任务，而不是手动拆 10 个 batch |
| 模型 | 新表 `backfill_coordinator_jobs`（id, name, template_id, total_count, child_job_ids[]） |
| 行为 | Coordinator 按 chunk（如 5k～10k）创建多个 child `backfill_jobs`；统一进度聚合 |
| API | `POST /backfill/coordinated` body: `{ filterJson \| assetIds, chunkSize }` |
| 验收 | 提交 100k 筛选条件 → 自动 10～20 子批 → 仪表盘总进度正确 |

**估时：** 8～12 人天

---

### C-7. `filter_json` 流式资产源 — P0（M3）

| 项 | 内容 |
|----|------|
| 问题 | 100k assetIds 不能 inline POST |
| 方案 | 创建任务时只传 `filterJson`（如 import_batch、ES 查询 ID）；Worker 游标拉资产 |
| 已有 | `backfill_jobs.filter_json` 列已存在，创建 API 未接 |
| 验收 | 100k 逻辑任务请求体 **< 100KB**；Worker 稳定枚举资产 |

**估时：** 5～8 人天（依赖资产查询 API 能力）

---

### C-8. 协调任务统一仪表盘 — P1（M3）

| 项 | 内容 |
|----|------|
| UI | `/pipeline/batch/coordinator/:id`：总进度、子批列表、**聚合节点热力图**（跨子批 SUM） |
| API | `GET /backfill/coordinator/:id/summary` |
| 验收 | 10 个子批的节点失败率在总览可合并查看 |

**估时：** 5～7 人天（复用 B-1 聚合逻辑）

---

### C-9. Watcher 与同步规模化 — P1（M3）

| 项 | 内容 |
|----|------|
| 问题 | `activeScanLimit=100`，10 万 run 状态严重滞后 |
| 方案 | 按 `batch_job_id` 分片扫描；优先活跃批；可配置 limit |
| 验收 | 10 万 run 存在时，单批详情 status 延迟 **< 2min**（SLA 文档化） |

**估时：** 4～6 人天

---

### C-10. 步骤级输出缓存 — P3（M3 省算力）

与 **B-4** 相同，提升到 M3 必选项之一（10 万重跑成本）。

**估时：** TBD（2～4 周专项）

---

## C 系列速查

| 编号 | 任务 | 里程碑 | 优先级 | 估时 | 依赖 |
|------|------|--------|--------|------|------|
| C-1 | Backfill Worker 解耦 | M2 | P0 | 5～8d | — |
| C-2 | 懒物化 pipeline_run | M2 | P0 | 3～5d | C-1 推荐 |
| C-3 | 可配置并发 + 限流 | M2 | P0 | 3～4d | C-1 |
| C-4 | items 分页 API | M2 | P1 | 2～3d | — |
| C-5 | 上限 30k（可选） | M2 | P2 | 1～2d | C-1～4 |
| C-6 | 十万协调器 | M3 | P0 | 8～12d | M2 |
| C-7 | filter_json 流式 | M3 | P0 | 5～8d | C-6 |
| C-8 | 协调器仪表盘 | M3 | P1 | 5～7d | C-6,B-1 |
| C-9 | Watcher 分片 | M3 | P1 | 4～6d | — |
| C-10 | 步骤缓存 | M3 | P3 | TBD | B-4 |

**M2 最小集：** C-1 + C-2 + C-3（约 2～3 周）  
**M3 最小集：** M2 + C-6 + C-7 + C-8（约再加 3～4 周）

---

## 4.2 吞吐粗算（给容量规划用）

假设：10 节点 DAG，单资产 wall time **15 分钟**，批内并发 **5**：

```
T ≈ (资产数 / 并发) × 单条耗时
10,000  → 10,000/5 × 15min = 30,000min ≈ 21 天
100,000 → 约 210 天（同样并发；仅作数量级参考）
```

并发提到 **50** 且集群撑得住 → 时间约 ÷10。  
**结论：** 十万级必须 **拆批 + 提高并发 + 缓存重跑**，不能靠单批硬扛。

---

## 4.3 压测与验收清单（M2/M3 上线前）

| 编号 | 场景 | 通过标准 |
|------|------|----------|
| ST-C1 | 创建 10k 批 | 物化 P95 < 60s；无 API OOM |
| ST-C2 | Worker 杀进程重启 | 批继续推进，无 duplicate workflow |
| ST-C3 | 并发 20 跑 1k 批 | Argo 错误率 < 1%；K8s pending 可接受 |
| ST-C4 | Coordinator 100k | 子批全部创建；总进度误差 < 0.1% |
| ST-C5 | 100k 逻辑任务请求体 | < 100KB；无 inline 100k UUID |

---

## 工程 ↔ 产品 ↔ 架构 对照（完整）

| 任务 | G 支柱 | 里程碑 |
|------|--------|--------|
| #1 #2 P-1 | G1 | M0+ |
| B-1 | G3 | M1 |
| B-2 B-3 | G4 | M1 |
| C-1～C-5 | G3 G4 + 容量 | M2 |
| C-6～C-9 | G3 G4 + 容量 | M3 |
| C-10 / B-4 | G4 省算力 | M3 |
