# Agent Environment Requirements

**Audience: AI agents only.** End users should never need to read or run this file.

## When to run

- At the **start of the first task** in a session (or when a check fails mid-task)
- **Silently** — do not narrate "I am now running setup" unless something blocks progress
- **Do not** require the user to say "prepare environment" or "follow SETUP"

## Target states

### MCP: Linear

- [ ] Linear MCP works (e.g. `list_issues` succeeds)
- Credential: `LINEAR_API_KEY` — if missing, ask user once to set it; then continue
- Never commit credentials

### MCP: Chrome DevTools（仅当会改 `Frontend/` 代码）

- [ ] **If** the planned change includes `Frontend/` paths: verify Chrome DevTools MCP is available (`navigate_page`, `take_snapshot`, … under `mcps/user-chrome-devtools/tools/`)
- [ ] After frontend dev deploy, Agent **runs** UI verification via MCP — see [`deploy-verification.md`](deploy-verification.md) §1.3
- [ ] **If** the change is backend/sdk-only: **skip** this block — no browser MCP
- **If MCP required but unavailable:** follow [`MCP-POLICY.md`](MCP-POLICY.md) → **Chrome DevTools MCP — fallback** (record blocker + automated checks; then ask user for structured UI retest)

### Skills

- [ ] `.agents/skills/` exists (`tdd`, `diagnose`, …). Optional local lock: `./skills-lock.json` (gitignored; see [`SKILLS.md`](SKILLS.md))
- If missing: `npx skills@latest add mattpocock/skills --agent <cursor|codex|...> -y --copy` (match the user's IDE)

### OpenSpec（仓库内文档，无需安装 CLI）

- [ ] `openspec/specs/` 与 `openspec/config.yaml` 存在于仓库中（clone 即有）
- [ ] 运行时改动前在 `openspec/changes/CYB-xxx-.../` 创建 Markdown（由 Agent 编写）
- **不要**要求用户或 Agent 安装 `@fission-ai/openspec`；合入主 spec 见 [`spec-driven-workflow.md`](spec-driven-workflow.md)「合并后归档」

### Backend (Go)

- [ ] `cd backend && go build ./...` succeeds
- If not: `go mod download`

### Frontend (Node)

- [ ] `cd Frontend && npm run build` succeeds
- If not: `npm install` in `Frontend/`

### SDK (Python)

- [ ] `cd sdk && uv run ruff check src/` succeeds (Tier S — same as [`AI-RULES.md`](AI-RULES.md) / `openspec/config.yaml` `verification.sdk`)
- [ ] Optional spot-check: `cd sdk && uv run pytest tests/unit/ -q` when SDK logic changed
- If not: `uv sync --dev` in `sdk/`

### Git commit-msg hook（本仓库，Agent 自动配置）

本地 `git commit` 时校验 Conventional Commits（与 CI `commitlint` 规则一致）。**用户不必手动执行**；Agent 在首次任务时静默配置即可。

- [ ] 在仓库根目录检查：`git config --get core.hooksPath` 是否为 `.githooks`
- 若未设置，在仓库根目录执行（**仅本仓库**，不要用 `--global`）：

  ```bash
  git config core.hooksPath .githooks
  chmod +x .githooks/commit-msg
  ```

- 验证：故意错误的 message 应被 hook 拒绝（可选，不必每次做）

**说明：**

- 合并不合规的 commit 仍由 **CI `commitlint`** 拦截（PR 上硬兜底）；本地 hook 只是提前反馈。
- Agent 只改当前 clone 的 `core.hooksPath`，不修改用户全局 git 配置。

## Credentials (interrupt user only when needed)

| Credential | When |
|------------|------|
| `LINEAR_API_KEY` | Before creating/querying Linear issues |
| `DATABREW_TOKEN` | Backend local API debug only |
| GCP | Deploy verification only |
