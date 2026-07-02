# Decisions — CYB-1824

## 2026-06-15 — 复用现有 issue 并从 dev 开分支
- **Context**: 当前工作和已有 Linear issue CYB-1824 高度重合，都是 PipelineComponent 镜像/版本管理。
- **Decision**: 复用 CYB-1824，并从最新 `origin/dev` 创建 `feat/CYB-1824-component-releases`。
- **Alternatives**: 新建一个 ComponentRelease issue。
- **Rationale**: CYB-1824 已经归属 DataBrew 且覆盖这个产品方向；直接更新方向可以避免重复跟踪。

## 2026-06-15 — OpenSpec approval
- **Context**: 用户确认「ok，现在开发吧」，随后澄清 Phase 1 允许 DataBrew 前端/后端大改，但不要求算法 repo 里的现有 task 改配置。
- **Decision**: 开始实现 Phase 1，保持算法 task authoring 不变，DataBrew 侧新增 ComponentRelease 台账、校验和展示能力。
- **Alternatives**: 先停留在方案讨论或要求算法 repo 新增 `component.yaml`。
- **Rationale**: 当前目标是先接住现有 task 和构建产物，避免把平台元数据维护成本转嫁给算法同学。

## 2026-06-15 — Phase 1 permission boundary
- **Context**: 用户询问是否需要申请 Cloud Build / Artifact Registry / GitHub 权限。
- **Decision**: Phase 1 不直接从 DataBrew 扫描 GCP/GitHub；先提供 `/pipeline-component-releases/sync` 接收平台或 CI 生成的 release manifest，并在 DataBrew 内做台账、校验和可选状态。自动扫描后续通过独立 provider 接口接入。
- **Alternatives**: 让 DataBrew 后端直接持有 Cloud Build Viewer、Artifact Registry Reader、GitHub contents read 权限并实时扫描。
- **Rationale**: 先解耦权限和产品模型，避免把权限申请、云厂商 API 细节和 UI 选择器强绑定。

## 2026-06-15 — Frontend full test suite existing failures
- **Context**: 本次改动触及 `Frontend/`，已运行相关组件页测试、lint 和 build；额外运行全量 `npm run test -- --run` 时，Settings、Workflow、Pipeline 旧用例失败。
- **Decision**: 记录 full suite 非本次触碰失败；本次相关 `src/pages/ComponentManager.test.tsx` targeted test 已通过，`npm run lint` 和 `npm run build` 已通过。
- **Alternatives**: 在本 PR 顺手修复 Settings/Workflow/Pipeline 全量测试问题。
- **Rationale**: 失败集中在未触碰模块，包含 localStorage test env、Workflow terminal/pipeline jsdom 超时等既有问题；顺手修复会扩大本 PR 范围。

## 2026-06-15 — Pod preview verification
- **Context**: 用户要求使用 pod 部署方式，而不是 Cloud Run dev。仓库的 `deploy/preview/fast-preview.sh` 是 commit-based，会从 GitHub tarball 拉取 commit；当前改动尚未提交，直接运行会部署旧 HEAD。
- **Decision**: 手动复用 preview pod 的 Deployment/Service 结构，使用本地工作树 build/push 的 backend image `manual-cyb1824-4db0fdf-3ce73f3397a9`，创建 hex preview alias `3ce73f3397a9`，本地 frontend dev server 对接该 preview backend。
- **Alternatives**: 先做本地 WIP commit 再跑 `fast-preview.sh HEAD`，或继续使用 Cloud Run dev。
- **Rationale**: 这样能验证当前未提交代码，同时满足用户想用 pod 方式验证的要求。

## 2026-06-15 — 融合 legacy component 与 generated release 展示
- **Context**: 用户反馈 `PipelineComponent` 旧组件和 `ComponentRelease` 新版本库应该融合，而不是在组件页里拆成两个割裂的区域。
- **Decision**: 前端先做 presentation-layer 融合：组件页改为单一“组件库”表格，一行代表一个任务/组件，同行展示 legacy component、generated releases、状态、镜像和来源；legacy CRUD 入口保留，release detail 入口保留。
- **Alternatives**: 后端立即合并两套 storage/model，或继续用两个独立区块展示。
- **Rationale**: 当前 Phase 1 仍需要保留旧组件 CRUD 和新 release 台账的不同生命周期；先在 UI 聚合，能降低用户理解成本，同时避免过早破坏既有 pipeline component 行为。

## 2026-06-15 — CI manifest 为主路径，不做全项目扫描
- **Context**: 用户希望一步到位打通“task 出包后 DataBrew 感知”，但不希望 DataBrew 全项目扫描 Cloud Build/Artifact Registry，也不希望普通用户填写 image/repo/digest。
- **Decision**: ComponentRelease 主路径定为 CI push manifest：CI 构建成功后向 `/api/v1/pipeline-component-releases/sync` 提交 batch `source + items`；DataBrew 只接收、校验、入库、展示和搜索，不负责猜测 task/image 关系。
- **Alternatives**: DataBrew 主动扫描全项目 build/image，或 UI 让用户按 commit/tag 懒加载后导入。
- **Rationale**: CI 在出包时拥有最准确的 task、commit、image digest 和 build metadata；DataBrew 只做 ingest contract 能避免误扫其它团队镜像，也降低平台耦合。

## 2026-06-16 — User-approved PR before deploy verification
- **Context**: Runtime diff touches Frontend/backend/sdk. The default deploy-before-commit workflow requires dev deploy verification before commit/push, but the user explicitly interrupted the build and said they would verify themselves, asking to submit the PR immediately.
- **Decision**: Commit and open the PR without completing Cloud Run dev deploy verification. Include completed local checks and the skipped build/deploy status in the PR.
- **Alternatives**: Continue running build, deploy frontend/backend dev, verify via Chrome DevTools, then ask for commit approval.
- **Rationale**: The current user instruction explicitly prioritizes getting the PR opened now; remaining verification is handed off to the user.
