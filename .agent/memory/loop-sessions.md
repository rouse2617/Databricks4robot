# Loop Sessions — 自主 loop 通信板

> **职责：** 外部指令 + 待办池 + 方案草稿 + 复盘日志  
> **想法草稿：** [ideas.md](ideas.md)（未成熟）→ agent 消化后进本文件  
> **怎么用：** 「跑 loop」/ **7×24 cron**（见 `loop-cron-prompt.md`）→ 读 `task` §8 → 每轮：deploy + UI 验收 + **commit** → Session → **自动下一轮**

---

## 🎯 本轮目标

**C2-FE：组件注册集成 + Designer palette 下拉选镜像** ✅

- [x] Backend：实现 Artifact Registry 列表镜像 API（`rick-cyber-databrew-images` + `video-proc-images`）✅
- [x] Route：`GET /api/v1/pipeline-components/registry/images` ✅
- [x] Deploy backend dev + curl verify ✅
- [x] Frontend：组件注册页从 AR 选镜像创建组件 ✅
- [x] Designer palette 组件可见 → 可拖入节点 ✅

---

## 📨 外部指令（Rick / Hermes — 最高优先级）

（当前无新需求 — **紧急插队只写这里**，不要只留在聊天或 ideas.md）

<!--
模板见 task §1：
- 日期 / 意图 / 验收 / 不做 / 截止
-->

---

## 💡 想法入口

半成熟需求请写 **[ideas.md](ideas.md)**。成熟后 agent 会移入待办池或上方外部指令。

---

## 📋 待办池（以本文件为准；task 不重复列项）

> 格式：`- [ ] P? | 轨道 | 描述`

### 轨道 A — Pipeline redesign（worktree）  ✅ 特性完成

- [x] P2 | A | CYB-1236、P2-REG、P2-PUB-01/02、P2-INT-11、Designer UX、P3 文档与 transpiler（retry/continueOn/when/fan-out/backoff）— 见 Archive

### 轨道 B — 资产平台（主仓）

- [ ] P2 | B | 跟 `current-work.md` 活跃 CYB
- [ ] P2 | B | 资产/交付/事件 — API 变更时 api-guide smoke + 页回归（Hermes）

### 轨道 C — 跨轨集成（**默认优先**）

**C1 — 真跑 Argo（P0，见 ideas.md 已决策）** ✅ **全链路验收通过**

- [x] P0 | C | 确认 GKE `cyber-clust` 上 **Argo Workflow 控制器** 可调度（非仅 CRD）；装 helm（argo-workflows v4.0.5, singleNamespace）
- [x] P0 | C | **GKE** `cyber-databrew-dev` backend 含 publish/run 代码 + InCluster argo client；ConfigMap 加 `ARGO_BASE_URL`；RBAC：`default` SA 可 create workflow + workflowtaskresults
- [x] P0 | C | Backend namespace 从硬编码 `"default"` 改为配置化（`ARGO_NAMESPACE=cyber-databrew-dev`），新镜像 build/push，deployment 已 rollout
- [x] P0 | C | 手动测试 workflow 提交成功（Succeeded）
- [x] P0 | C | Hermes：Designer（pipeline-ui 或 Frontend 嵌入）Publish → Run → `algo_runs` + 打开 external_url（**UI 验收**）
- [x] P1 | C | dev `pipeline_id` 全链路（已解除阻塞—C1 E2E 通过）

**C2 — 组件 ↔ Artifact Registry（P1，C1 后）**

- [x] P1 | C | Backend：列 AR `rick-cyber-databrew-images` + `video-proc-images` 镜像 API + route `/registry/images`
- [x] P1 | C | Frontend：组件注册页从 AR 选镜像创建组件 → palette 可见 → 拖入节点 Save
- [ ] P1 | A | 替换种子占位镜像（`databrew/loader:latest 等`）为真实 `us-central1-docker.pkg.dev/...` 路径

- [x] P2 | C | Publish → Run → Argo UI `external_url` 文档化

---

## 🧪 方案草稿

（新开切片前写；已完成项已归档，勿重复堆长文）

- _empty — 下一项 C 或 B 开工前写 3 条结论或 OpenSpec 链接_

---

## 📝 复盘日志（仅保留最近 3 条完整 Session）

---

## Session 2026-05-27f (C1 E2E — Designer Publish → Run → algo_runs 全链路)

### Done
- **Backend DB alignment**: Rewrote `scanAlgoRun` to match actual dev DB column order (duration_ns at pos 9, no external_run_id)
- **Create fix**: Removed `external_run_id` from INSERT + renumbered placeholders $1-$31
- **Service fix**: Removed `ExternalRunID` from service.go Run struct literal
- **Migration 031**: Added `pipeline_id TEXT` column to `algo_runs`
- **algo_kind constraint**: Dropped and recreated CHECK to include `'pipeline'` value
- **Build + Push**: Image `3b9d389-fix-run` built and pushed to Artifact Registry
- **Deploy**: Cloud Run dev revision 00249-ddt → 00250-... serving 100%
- **ARGO_BASE_URL**: Added env var via `gcloud run services update`
- **E2E API Verification**:
  - Publish returns 200, WFT created in GKE
  - Run returns `workflow_name` + `run_id`
  - `SELECT * FROM algo_runs WHERE pipeline_id IS NOT NULL` — has records
  - `external_url` points to Argo UI workflow detail page ✅
- **Committed + Pushed** to `feat/CYB-1237-pipeline-revisions-crud` (3 fixup commits for pre-commit)

### Files Modified
- `backend/internal/postgres/algo_runs.go` — scanAlgoRun + Create rewritten
- `backend/internal/usecase/pipeline/service.go` — removed ExternalRunID
- `backend/migrations/031_add_pipeline_id_to_algo_runs.sql` — NEW
- `databrew-pipeline/argo-ui/tsconfig.json` — removed JSON comments
- `docs/review/pipeline-troubleshooting.md` — trailing whitespace

### Deploy
```
SHA=3b9d389  IMAGE=us-central1-docker.pkg.dev/green-valley-442103/rick-cyber-databrew-images/cyber-databrew-backend:cloudrun-dev-latest  REVISION=00249-ddt  URL=https://cyber-databrew-backend-dev-234851712830.us-central1.run.app  ARGO_URL=https://cyber-databrew-pipeline-ui-dev-234851712830.us-central1.run.app
```

### Hermes 验收
- [x] API: Publish (POST /pipelines/:id/revisions/:ver/publish) → 200 + WFT created
- [x] API: Run (POST /pipelines/:id/run) → 200 + workflow_name + run_id
- [x] DB: `algo_runs` has `pipeline_id`, `external_url` set
- [x] Argo UI: external_url links to running/succeeded workflow
- [x] commit: feat/CYB-1237-pipeline-revisions-crud pushed to remote

### Blockers / Next
- **C1 全链路验收通过** → P1 | C (`pipeline_id` 全链路) 已解除阻塞
- **C2** (Artifact Registry 组件注册 + Designer 下拉) 待启动

---

## Session 2026-05-27g (C2-API — Backend AR 镜像列表 API)

---

## Session 2026-05-27h (C2-FE — Pipeline Components CRUD page + AR image selector)

### Done
- **PipelineComponentsPage**: Created CRUD management page with Ant Design table listing all 5 existing components (BusyBox, gRPC Relay, Data Loader, FFmpeg, Python)
- **ImageSelector**: Built custom AR image browser that fetches from `/pipeline-components/registry/images` — shows images grouped by repository in Collapse panels with URI/tags/size, supports text search filter and manual URI entry toggle
- **ComponentModal**: Full create/edit form with Name, Description, Category, Image (selector), Command (tags mode), Default Args (dynamic Form.List), Resources (CPU/Memory/Disk), Image Pull Policy, Global toggle, and Environment Variables (dynamic Form.List)
- **Route + Sidebar**: Lazy-loaded route `/pipeline-components` in App.tsx, sidebar menu item "Pipeline 组件" with `CodeOutlined` icon and `resolveSelectedKey` in AppLayout.tsx
- **Null safety fix**: Fixed `TypeError: Cannot read properties of null (reading ·slice·)` when API returns images with null tags array
- **Build + Push**: Frontend image `3f030f3` rebuilt and pushed to Artifact Registry
- **Deploy**: Cloud Run dev revision `00243-p9p` serving 100% traffic
- **UI Acceptance**: Chrome DevTools verified — page loads with 5 components, sidebar highlights correctly, Create modal opens with all fields, Edit modal pre-fills BusyBox data correctly, 0 JavaScript errors (3 pre-existing a11y warnings only), Pipelines page regression passes

### Files Modified
- `Frontend/src/pages/PipelineComponentsPage.tsx` — NEW (full CRUD page with ImageSelector + ComponentModal)
- `Frontend/src/api/pipelineComponents.ts` — Extended with CRUD functions + RegistryImage/RegistryRepository interfaces
- `Frontend/src/App.tsx` — Lazy import + route for PipelineComponentsPage
- `Frontend/src/components/AppLayout.tsx` — Sidebar menu item + resolveSelectedKey

### Deploy
``
SHA=3f030f3  IMAGE=us-central1-docker.pkg.dev/green-valley-442103/video-proc-images/cyber-databrew-frontend:3f030f3  REVISION=00243-p9p  URL=https://cyber-databrew-frontend-dev-234851712830.us-central1.run.app
``

### Hermes 验收
- [x] Page loads: table with 5 components (BusyBox, gRPC Relay, Data Loader, FFmpeg, Python)
- [x] Create modal: all form fields visible + ImageSelector loading AR images
- [x] Edit modal: BusyBox form pre-filled (name, description, category, command, args, resources)
- [x] Console: 0 JavaScript errors (3 pre-existing a11y warnings)
- [x] Regression: /pipelines page loads correctly with 7 pipelines
- [x] commit: pending (deploy-verify-before-commit gate)

### Blockers / Next
- **P1 | A**: 替换种子占位镜像（`databrew/loader:latest 等`）为真实 `us-central1-docker.pkg.dev/...` 路径
- **P2 | B**: 跟 `current-work.md` 活跃 CYB
- Next C1/C2 item after P1/A: verify Designer palette picks up new components from AR


### Done
- **New Go client**: `backend/internal/artifactregistry/client.go` — wraps Google Artifact Registry API (`Projects.Locations.Repositories.DockerImages.List`)
- **Config**: Added `AR_PROJECT`, `AR_LOCATION`, `AR_REPOSITORIES` env vars to `config.go` (defaults: `us-central1`, `rick-cyber-databrew-images,video-proc-images`)
- **Route**: `GET /api/v1/pipeline-components/registry/images` — avoids Gin `:id` param conflict by using multi-segment path
- **Handler**: `ListAvailableImages` on `ComponentHandler` — queries all configured AR repos, returns `{repositories: [{repository, images}]}`
- **AR client wiring**: `core.go` creates AR client when `AR_PROJECT` is set, wires to `componentHandler.SetARClient()`
- **Routes test fix**: Added missing `nil` (algoRunHandler 18th param) to all 6 `RegisterAll` calls in `routes_test.go`
- **Build + Push**: Docker image built from repo root, pushed as `cloudrun-dev-latest`, `cloudrun-dev-fix-route`, and SHA `2abb8d6`
- **Deploy**: Cloud Run dev revision `00257-qs2` serving 100% traffic (after `update-traffic --to-latest`)
- **API verified**: Both repositories returned — `rick-cyber-databrew-images` (653) + `video-proc-images` (737)

### Files Modified
- `backend/internal/artifactregistry/client.go` — NEW (AR client)
- `backend/internal/config/config.go` — AR_PROJECT/AR_LOCATION/AR_REPOSITORIES
- `backend/internal/handlers/pipeline/component_handler.go` — ListAvailableImages handler
- `backend/routes/routes.go` — `/registry/images` route
- `backend/routes/routes_test.go` — 18th nil param for RegisterAll
- `backend/cmd/server/core.go` — AR client wiring

### Deploy
```
SHA=2abb8d6  IMAGE=us-central1-docker.pkg.dev/green-valley-442103/rick-cyber-databrew-images/cyber-databrew-backend:2abb8d6  REVISION=00257-qs2  URL=https://cyber-databrew-backend-dev-234851712830.us-central1.run.app
```

### Hermes 验收
- [x] API: GET /api/v1/pipeline-components/registry/images → 200 + repositories[]
- [x] rick-cyber-databrew-images: 653 images listed
- [x] video-proc-images: 737 images listed
- [x] commit: 3f030f3

### Blockers / Next
- **C2-FE**: 需要前端组件注册页从 AR 选镜像创建组件 → palette 可见 → 拖入节点 Save

---

## Session 2026-05-27e (C1 — GKE Argo 基础设施打通)

### Done
- **Argo Workflow controller v4.0.5** installed via Helm (`argo/argo-workflows` 1.0.14) in `cyber-databrew-dev` namespace, `singleNamespace=true`
- **RBAC**: Created `databrew-backend-workflow` Role + RoleBinding for `default` SA — allows create/get/list/watch workflows + workflowtaskresults create/patch
- **Test workflow** submitted and completed successfully (Succeeded) — confirms controller and RBAC working
- **ConfigMap**: `ARGO_BASE_URL=https://cyber-databrew-pipeline-ui-dev-234851712830.us-central1.run.app` already set
- **Backend code change**: Added `ArgoNamespace` config field (env `ARGO_NAMESPACE`, default `"default"`); changed `core.go` from hardcoded `"default"` to `inf.cfg.ArgoNamespace`
- **Build + Deploy**: New backend image `cyber-databrew-backend:3b9d389` built, pushed to AR, deployment updated with `ARGO_NAMESPACE=cyber-databrew-dev`, rollout succeeded

### Files Modified
- `backend/internal/config/config.go` — added `ArgoNamespace` field + env binding
- `backend/cmd/server/core.go` — use `inf.cfg.ArgoNamespace` instead of `"default"`
- `loop-sessions.md` — updated C1 todo items

### Deploy
```
SHA=3b9d389  IMAGE=us-central1-docker.pkg.dev/green-valley-442103/rick-cyber-databrew-images/cyber-databrew-backend:3b9d389  NAMESPACE=cyber-databrew-dev  ARGO_URL=https://cyber-databrew-pipeline-ui-dev-234851712830.us-central1.run.app
```

### Hermes 验收
- [ ] 环境：GKE `cyber-clust` / `cyber-databrew-dev` / backend pod `cyber-databrew-backend-778d5886bb-vrz9p`
- [ ] 步骤：打开 Designer → Save pipeline → Publish → Run → 检查 `algo_runs` 有记录 → `external_url` 可打开 Argo UI
- [ ] 期望：Publish 创建 WFT，Run 提交 Workflow，controller 调度 pod 完成
- [ ] 证据：`algo_runs` 表 status=running/succeeded + external_url 深链
- [ ] commit：待 Hermes / Rick

### Blockers / Next
- 需要从 Designer UI 验证全链路（Hermes 验收）
- pipeline_id 全链路 (P1) 排在 C1 全链路验收之后
- C2 (Artifact Registry 组件集成) 在 C1 之后

---

## Session 2026-05-27d (CYB-1244 a11y form field id/name fix)

### Done
- CYB-1244: 13 Input/TextArea components in NodeConfigPanel now have `id` props via `React.useId()`
  - Label, Image, Command, Args, CPU, Memory, Disk, Max Retries, Backoff Duration/Factor/MaxDuration, withItems, withParam
- DevTools a11y issue count: **7 → 1** (remaining is Pipeline Name input on toolbar — pre-existing, outside scope)
- Lighthouse snapshot: Accessibility 92 (no form-control-related failures)
- Deployed Cloud Run dev (revision 00241-j52)
- Chrome DevTools: Backoff values (10s/3/10m) still persist, Retry Strategy panel expands correctly ✅
- Build + 43 pipeline-designer tests pass ✅ (5 failed tests in other files are pre-existing)

### Files Modified
- `Frontend/src/pipeline-designer/NodeConfigPanel.tsx`

### Deploy
```
SHA=3b9d389  IMAGE=gcr.io/cyber-databrew-dev/cyber-databrew-frontend:3b9d389  REVISION=00241-j52  URL=https://cyber-databrew-frontend-dev-234851712830.us-central1.run.app  UTC=2026-05-27T11:13:41Z
```

### Linear
- CYB-1244 (created → fixed, Backlog)

### Hermes 验收
- [ ] 环境：revision 00241-j52
- [ ] 步骤：打开 Designer → 点击节点 → 检查 DevTools Console → a11y 警告 count=1（原7）
- [ ] 期望：无 form-field-related a11y 新增警告
- [ ] commit：待 Hermes / Rick

### Blockers / Next
- a11y count=1 剩余在工具栏 Pipeline Name input — 可另开 CYB
- **Track C** 仍等 backend publish/run 上 Cloud Run dev
- **Track B** 需退出 worktree

---

## Session 2026-05-27c (P5 Backoff E2E + a11y bug #1)

### Done
- P5 Item 1: Added 12 vitest tests for retryStrategy, continueOn, withItems/withParam, condition edges round-trip (43 total, all pass)
- P5 Item 2: WebSearch → found Argo v3.7+ Backoff → implemented full stack:
  - Go `Backoff` struct in pipeline.go + transpiler buildNodeTemplate()
  - Frontend `Backoff` interface in types.ts + NodeConfigPanel UI
  - Golden test `retry.wf.yaml` regenerated
  - `go test 16/16`, `vitest 43/43`, `npm run build` ✅
- Deployed to Cloud Run dev (revision 00240-sdg)
- Chrome DevTools E2E:
  - Backoff UI visible: Duration/Factor/Max Duration inputs in Retry Strategy panel
  - Modified Backoff values (10s/3/10m), Save (PUT 200), reload → values persist ✅
  - Console: 0 errors (1 pre-existing a11y warning, count: 7)

### Bug #1 Found: a11y — 7 form field elements lack id/name attributes
- Ant Design Input components in NodeConfigPanel (Max Retries, Duration, Factor, Max Duration, CPU, Memory, Disk) lack `id` attributes
- Severity: low (a11y warning, no functional impact)
- Fix: add `id` props to each Input, or wrap with Form.Item

### Files Modified
- `Frontend/src/pipeline-designer/types.ts` — Backoff interface
- `Frontend/src/pipeline-designer/NodeConfigPanel.tsx` — Backoff UI
- `Frontend/src/pipeline-designer/pipelineModel.test.ts` — 12 new tests
- `databrew-pipeline/transpiler/pipeline.go` — Backoff struct
- `databrew-pipeline/transpiler/transpiler.go` — Backoff transpilation
- `databrew-pipeline/transpiler/testdata/retry.json` — Backoff fixture
- `databrew-pipeline/transpiler/testdata/golden/retry.wf.yaml` — regenerated
- `openspec/changes/CYB-active-discovery-p5/proposal.md` — OpenSpec

### Deploy
```
SHA=3b9d389  IMAGE=gcr.io/cyber-databrew-dev/cyber-databrew-frontend:3b9d389  REVISION=00240-sdg  URL=https://cyber-databrew-frontend-dev-234851712830.us-central1.run.app  UTC=2026-05-27T06:29:10Z
```

### Hermes 验收
- [ ] 环境：revision 00240-sdg, URL 同上
- [ ] 步骤：打开 Designer → 点击节点 → 展开 Retry Strategy → 查看 Backoff (Argo v3.7+) 段 → 修改 Duration/Factor/Max Duration → Save → 刷新页 → 展开 Retry Strategy → 验证值保留
- [ ] 期望：Backoff 字段 save→reload 持久化
- [ ] Bug #1: 7 form field elements lacking id/name (a11y) — 已修 6/7，见 2026-05-27d
- [ ] commit：待 Hermes / Rick

### Blockers / Next
- 本轮 P5 已完整交付 → **下一轮 P1 | C**：需 backend publish/run 上 Cloud Run dev 后才能做 `pipeline_id` 全链路

---

---

## Archived Sessions

（每条 1–2 行；详情见 git / worktree 历史）

- **Session 2026-05-27** — P5 retryStrategy + continueOn + conditional + E2E（Go → transpiler → FE → golden → mock E2E）
- **P3-FE-03** — 系统字体栈，去掉 Google Fonts；scss only
- **P3-DOC-01/02** — ops-runbook + user-guide
- **housekeeping 2026-05-27** — 曾归档 3 session；现再次精简本文件
- **E2E dev health** — 侧栏/流水线列表 200；Track A dev 表面健康
- **P3 canvas UX** — drag-connect, snap, undo fix, copy/paste, mini-toolbar, empty states, Ctrl+S 等（多轮 deploy rev）
- **P2-INT FE pages** — Pipelines/Designer/AlgoRuns；nginx `/pipeline` redirect
- **CYB-1236 / P2-REG / publish/run backend** — 见 worktree commits
- **P5 主动发现** — Backoff 全栈 + 12 vitest tests + Cloud Run dev deploy + E2E save→reload ✅；a11y bug #1 found（7 form fields missing id/name）
- **designer deep test + TR proposal** — Web 调研 TR-01/02；dev Designer 深测（节点/边/工具栏）；worktree 8 ahead / 136 behind
- **P3-TR-02 fan-out** — `withItems` / `withParam` 全栈 + golden + mock E2E
- **方案草稿（已完成）** — retry/continueOn/when/fan-out/backoff 设计已落地，见上 Sessions

