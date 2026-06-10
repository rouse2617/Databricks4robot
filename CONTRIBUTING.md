# 贡献指南 Contributing

感谢参与 **cyber-databrew**。本文是面向人类贡献者的速查；权威规则与 AI 协作规范以
[`docs/agents/`](docs/agents/) 为准（`AI-RULES.md` 为唯一真相源），仓库导航与本地启动见
[`README.md`](README.md)，架构基线见 [`docs/review/README.md`](docs/review/README.md)。

> TL;DR：从 `main` 切分支 → 写代码并补测试 → 同步受影响的文档 → 本地 `make ci-local` 通过
> → 用 Conventional Commits 提交 → 开 PR 到 `main`，等 CI 绿、评审通过后合并。

---

## 1. 准备开发环境

```bash
git clone https://github.com/CyberOrigin2077/cyber-databrew.git
cd cyber-databrew

# 启用本地 git 钩子（一次性）：push 前自动跑 ci-local（提交信息 + 单测）
git config core.hooksPath .githooks

# 本地全栈（仅本地调试 / 部署验证时需要）：Postgres + 后端 + 前端
make dev-up          # 起依赖；make dev-down 关闭
make local-migrate   # 应用迁移
```

各子系统的依赖与命令见对应目录 README：[`backend/`](backend/README.md)、
[`Frontend/`](Frontend/)、[`sdk/`](sdk/README.md)、[`deploy/local/`](deploy/local/README.md)。

## 2. 分支

- 从最新的 `main` 切分支，**不要直接在 `main` 上提交**。
- 分支名用 `<type>/<简述>`，type 与提交类型一致：
  `feat/pipeline-designer`、`fix/sse-flush`、`docs/contributing`、`chore/deps`…

## 3. 提交信息（Conventional Commits，CI 强校验）

由 [`.commitlintrc.json`](.commitlintrc.json) 强制，规则：

| 规则 | 要求 |
|------|------|
| `type` | 必须是 `feat` / `fix` / `docs` / `refactor` / `test` / `chore` / `ci` / `perf` 之一 |
| `scope`（可选） | `backend` / `frontend` / `sdk` / `docs` / `dagster` / `deploy` / `api` |
| subject 大小写 | **全小写** |
| header 长度 | ≤ 100 字符 |

示例：`feat(backend): add workflow log sse endpoint`、`fix(sdk): preserve base_url path prefix`。

## 4. 测试

行为变更必须带测试（见 `tdd` 工作流）。常用命令：

```bash
make test          # 后端 + SDK 单测（最常用）
make test-full     # 后端 + 前端（+ SDK，若装了 uv）
make backend-test  # 仅 Go
make frontend-test # 仅前端
make sdk-test      # 仅 Python SDK
make smoke-local   # 需本地 Postgres + 后端在 :8080
```

## 5. 同步文档（与代码同一个 PR）

- **改了代码就更新对应文档**：每个改动文件若被某个 Wiki 页“拥有”，需在同一 PR 内更新该页，
  否则会被 [`repo-wiki-divergence`](.github/workflows/repo-wiki-divergence.yml) 提示（非阻塞）。
  查归属：`python3 scripts/repo-wiki/wiki_sync.py owners <changed-file>...`；
  纯重构 / 测试 / 生成代码可给 PR 打 `wiki-exempt`（或 `docs-only` / `infra-ci-only`）标签。
- **规格变更**走 OpenSpec：在 [`openspec/`](openspec/) 提 proposal/design/tasks（全是仓库内 Markdown，无需装 CLI）。
- **HTTP 契约**以 [`api/openapi.yaml`](api/openapi.yaml) 为准，须与后端实现一致。

## 6. 提交前自检

```bash
make ci-local        # 提交信息 + 单测（pre-push 钩子也跑这个）
make ci-local-full   # 更完整的本地校验
```

`pre-push` 钩子会自动运行 `ci-local`；确需跳过用 `SKIP_PREPUSH=1 git push`（不推荐）。

## 7. 开 Pull Request

1. 目标分支为 `main`。
2. PR 描述写清**做了什么 / 为什么 / 如何验证**；关联 Linear / Issue。
3. 确认 CI 全绿（含 commitlint、单测、wiki 同步检查）。
4. 至少一名 reviewer 通过；评审意见以行内 comment 跟进。
5. 保持 PR 聚焦单一主题；大改动请拆分。

## 8. AI 协作

本仓库是 AI-辅助开发的（Cursor / Codex / Claude 等）。Agent 与人都遵循同一套规则，
真相源在 [`docs/agents/AI-RULES.md`](docs/agents/AI-RULES.md)；可复用工作流（调试、TDD、
repo-wiki、验证等）见 [`docs/agents/SKILLS.md`](docs/agents/SKILLS.md)。

---

有疑问先查 [`docs/agents/`](docs/agents/) 与 [`docs/review/`](docs/review/)，仍不清楚就在 PR / Issue 里提出。
