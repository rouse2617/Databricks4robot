# CYB-1125 decisions

## 2026-05-23 — Runtime edit approval

- **Context**: Project workflow requires an OpenSpec checkpoint before editing runtime frontend code.
- **Decision**: User approved continuing with runtime edits in chat with “可以,用 worktree 弄哈,不要影响现在的”. Work will run in isolated worktree `/Users/hrp/cyber/cyber-databrew/.claude/worktrees/CYB-1125-ui-qa-blockers`.
- **Alternatives**: Stop after OpenSpec and wait for another explicit confirmation.
- **Rationale**: The user explicitly approved continuing and requested worktree isolation to avoid affecting the existing CYB-1097 checkout.

## 2026-05-23 — Chrome DevTools MCP unavailable

- **Context**: Diff touches `Frontend/`, so post-deploy UI verification should use Chrome DevTools MCP if available.
- **Decision**: Current Claude Code tool set does not expose Chrome DevTools MCP tools (`navigate_page`, `take_snapshot`, `list_console_messages`, etc.), so deploy verification is blocked at the MCP browser step. Automated verification completed: targeted frontend tests passed, changed-file Biome passed, frontend build passed, backend `go test ./...` passed, backend search sync smoke returned 200 with `admin_search_enabled:false`, and both dev services were deployed.
- **Alternatives**: Claim UI verification complete without browser evidence, or ask the user to manually browse without structured steps.
- **Rationale**: Project MCP fallback policy requires recording the blocker and requesting structured UI retest rather than claiming deployed frontend verification is complete.

## 2026-05-24 — Dev deploy blocked by expired gcloud auth

- **Context**: Post-deploy QA fixes touch both backend and Frontend runtime paths, so deploy-before-commit requires backend/frontend dev deploy and Chrome DevTools MCP verification.
- **Decision**: Local backend image build completed, but pushing to Artifact Registry failed because the active gcloud account token cannot refresh in this non-interactive session. `gcloud auth print-access-token`, ADC token refresh, and Docker credential helper all returned reauthentication-required errors.
- **Alternatives**: Claim deployed verification against the previous dev revision, or bypass the required dev deploy.
- **Rationale**: The fixed code is not yet running on Cloud Run dev, so browser verification against dev would not prove these fixes. Deployment must resume after `gcloud auth login` / `gcloud auth application-default login` is refreshed by the user.

## 2026-05-24 — User approved skipping GCP deploy

- **Context**: Deploy-before-commit normally requires Cloud Run dev deployment and MCP verification after runtime Frontend/backend changes.
- **Decision**: User initially approved skipping GCP deployment in chat: “gcp可以不部署了”. This was superseded later in chat by “你一定要部署哈，然后用dev 的网址来测试”.
- **Alternatives**: Stop until gcloud auth is refreshed and Cloud Run dev deploy can complete.
- **Rationale**: Latest user instruction requires Cloud Run dev deployment and dev URL verification; local-only verification is no longer sufficient.

## 2026-05-24 — GCP deploy required after user reversal

- **Context**: User explicitly reversed the skip-deploy instruction and requested deployment plus dev URL testing.
- **Decision**: Resume Cloud Run dev deployment. Current blocker remains expired gcloud user credentials; no alternate local ADC/service-account credential is available in the session.
- **Alternatives**: Continue local-only verification.
- **Rationale**: The newest user instruction has precedence, and frontend fixes must be verified on the dev URL after deployment.

## 2026-05-24 — Dev deploy and MCP verification completed

- **Context**: User refreshed GCP auth and required deployment plus dev URL testing for the CYB-1125 UI QA fixes.
- **Decision**: Deployed backend `cyber-databrew-backend-dev-00171-28z` and frontend `cyber-databrew-frontend-dev-00191-rf7`, then verified fixed flows on `https://cyber-databrew-frontend-dev-wtttm6suaq-uc.a.run.app` with Chrome DevTools MCP.
- **Alternatives**: Treat the earlier local MCP pass as sufficient, or keep the earlier deploy blocker open.
- **Rationale**: The fixed code is now running on Cloud Run dev; MCP checks covered `/settings`, `/assets`, `/deliveries`, and `/assets/1nSlczDA`, including the expected failure path and console review.
