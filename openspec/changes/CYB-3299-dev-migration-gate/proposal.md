# CYB-3299 — dev 迁移门禁

## Why
`db-migrate-lint.yml` 只在 `pull_request → main` 跑，但 dev 库在每次 push→dev 就被 `deploy-dev.yml` 的 migrate Job 应用。→ 坏/重复前缀/checksum 不符的迁移先落 dev、无强制拦截（本地 hook 常被 `--no-verify` 跳过）。审计 P0-2/P0-3。

## What Changes
仅改 `.github/workflows/db-migrate-lint.yml`：
1. `pull_request.branches: [main]` → `[main, dev]`，让同一套 lint 也在 PR→dev 跑。
2. 顺序 guard（step 3）比较基准由硬编码 `origin/main` 改为 `origin/${{ github.base_ref }}`，对 main / dev 两种 base 都正确。

## Impact
- Affected code: `.github/workflows/db-migrate-lint.yml`（仅）
- 不改迁移文件，不碰 migrations 内容。
- 「强制阻塞合并」还需仓库 branch protection 把该 check 设为 dev 必需（repo admin 动作，本 PR 说明为后续）。

## 验证
- 本 PR 修改的 workflow 在 paths 内 → PR→dev 上应触发该 workflow 并通过（无新迁移：fresh-apply 跑存量、顺序 guard 无新文件）。
