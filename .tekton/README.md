# Tekton Pipelines-as-Code (Cloud Run)

PAC (`pipelines-as-code` namespace) reads this directory and starts PipelineRuns on matching git events.

**GKE deploy pipelines removed** (`promote-backend`, `push-dev`, `deploy-dev`). Runtime deploy is **Cloud Run only**.

## Pipeline matrix

| File | Trigger | Purpose |
|------|---------|---------|
| `build-backend.yaml` | PR → `main`, `backend/**` | BuildKit → image `:<sha>` + `:pr-N` (no deploy) |
| `deploy-cloudrun-dev.yaml` | PR comment **`/deploy-cloudrun-dev`** (PR → `main`) | Build + deploy `cyber-databrew-backend-dev` |
| `push-backend-cloudrun-dev.yaml` | **push** non-`main` + `backend/**` etc. | Build + deploy backend **dev** |
| `push-frontend-cloudrun-dev.yaml` | **push** non-`main` + `Frontend/**` etc. | Build + deploy frontend **dev** |
| `push-backend-cloudrun-prod.yaml` | PR comment **`/deploy-cloudrun-prod-backend`** (PR → `main`) | Build + deploy `cyber-databrew-backend-prod` (manual; **not** on merge) |
| `push-frontend-cloudrun-prod.yaml` | PR comment **`/deploy-cloudrun-prod-frontend`** (PR → `main`) | Build + deploy `cyber-databrew-frontend-prod` (manual; **not** on merge) |

### Typical flow

1. Work on `fix/CYB-xxx-*` → push triggers **dev** Cloud Run deploy (if paths match).
2. Open PR to `main` → GitHub Actions CI; optional **`/deploy-cloudrun-dev`** on the PR for dev.
3. Merge PR → **no** automatic prod build/deploy.
4. When ready for prod → comment on **that PR** (open or **closed/merged**):

   ```
   /deploy-cloudrun-prod-backend
   /deploy-cloudrun-prod-frontend
   ```

   Merge 后 GitHub bot 会在 PR 里留言提醒；飞书 A 群也会收到带上述命令的合并通知。

## Prerequisites

- PAC controller + GitHub App webhook on this repo
- Cluster tasks: `git-clone`, `buildkit` in `tekton-pipelines`
- `tekton-builder` SA: Artifact Registry push + Cloud Run deploy roles

## Feishu notifications

### GitHub Actions (repo Secrets)

Shared script: `.github/scripts/feishu-notify.sh` (`FEISHU_ROUTE_BRANCH` + `FEISHU_MESSAGE`).

| Workflow | When | Webhook route |
|----------|------|----------------|
| `notify-feishu.yml` | Every **push** | branch name → main / other |
| `notify-feishu-ci.yml` | **CI failure** → **A 群**；**CI success** 仅 `main` → A 群 | 含 PR 关联与日志链接 |
| `notify-feishu-pr.yml` | PR **opened/reopened/synchronize/merged** (target `main`) → **A 群** | see workflow |

Legacy `FEISHU_BOT_WEBHOOK` is fallback if branch-specific secret is unset.

### Tekton PAC (`feishu-open-notify` in `tekton-pipelines`)

Pipelines with `finally: notify-feishu-open` post deploy result to **one** cluster Secret (`webhook_url` or app IM keys). Point this at your **prod / deploy** group (e.g. main 群) if you only want deploy outcomes there.

Pipelines with Feishu notify: `deploy-cloudrun-dev.yaml`, `push-backend-cloudrun-dev.yaml`, `push-frontend-cloudrun-dev.yaml`, `push-backend-cloudrun-prod.yaml`, `push-frontend-cloudrun-prod.yaml`.

### Optional follow-ups

| Event | Status |
|-------|--------|
| Tekton PAC **started** (queued) | Not implemented — needs PAC/check integration |
| PR closed without merge | Skipped intentionally (reduce noise) |

## Image tags

| Tag | Set by |
|-----|--------|
| `:<sha>` / `{{revision}}` | Every build |
| `:pr-{n}` | PR build (`build-backend.yaml`) |
| `:buildcache` | BuildKit cache |

Prod deploy uses `:<revision>` from the PR commit you commented on—not `:latest` from merge.
