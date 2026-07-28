# Decisions - CYB-3007

## 2026-06-20 - Placeholder Linear ID

- **Context**: `LINEAR_API_KEY` is not present in the environment, so the agent cannot create or verify a real Linear issue in this session.
- **Decision**: Use `CYB-3007` as a temporary Runtime OS trace id for the local OpenSpec change.
- **Alternatives**: Stop implementation until Linear credentials are available.
- **Rationale**: The active Runtime OS goal asks for continued engineering progress, and missing Linear credentials do not block in-repo planning or tests.

## 2026-06-20 - Continue without another checkpoint prompt

- **Context**: The user explicitly said: "ok，这个以后不要问我了，你直接执行".
- **Decision**: Create the OpenSpec artifacts, record scope, and continue implementation without stopping for another confirmation.
- **Alternatives**: Stop after the OpenSpec checkpoint and ask the user to approve CYB-3007.
- **Rationale**: The current explicit user instruction has higher precedence than the default checkpoint pause while preserving traceability.

## 2026-06-20 - Keep runtime debug references in detail views

- **Context**: Logs, Pod diagnostics, terminal, and resource metrics still use runtime workflow APIs.
- **Decision**: Remove workflow listing only from the product execution list path, while retaining workflow debug usage in Run Inspector paths.
- **Alternatives**: Remove all frontend workflow API usage immediately.
- **Rationale**: Runtime OS requires the product list to be Run-centric, but removing debug APIs before replacement Run debug endpoints would regress existing operations.

## 2026-06-20 - Repo-wide frontend lint baseline remains dirty

- **Context**: `cd Frontend && npm run lint -- --max-diagnostics=40` still fails on unrelated pre-existing Biome diagnostics in files outside this Runtime OS change, including CSS `!important`, import ordering, and formatting in asset/pipeline-designer files not otherwise touched by this slice.
- **Decision**: Keep the Runtime OS diff scoped. Verify all touched frontend files with targeted `npx biome check`, and rely on full frontend `npm run test -- --run` plus `npm run build` for this goal instead of formatting unrelated files into the Runtime OS commit.
- **Alternatives**: Run repo-wide Biome write/fix and include a broad unrelated formatting cleanup.
- **Rationale**: Broad formatting would inflate the runtime PR and raise merge risk. Touched files are clean, and product behavior is covered by tests/build plus dev deploy verification.

## 2026-06-20 - Public dev deployment entry

- **Context**: `https://cyber-databrew-dev.cyberorigin.ai/` is served by the Cloudflare Worker static asset entry, while `cyber-databrew-frontend-dev` is a separate Cloud Run frontend service.
- **Decision**: Deploy both frontend Cloud Run dev and the Cloudflare Worker `site/` assets, then verify the public dev domain with Chrome DevTools MCP.
- **Alternatives**: Deploy only Cloud Run frontend and leave the public dev domain on the previous static asset bundle.
- **Rationale**: The user validates through the public dev domain, so Cloud Run-only deployment would leave the visible environment stale.

## 2026-06-20 - Terminal behavior on completed Pods

- **Context**: The deployed smoke run used for log, Pod, and monitoring verification had already completed.
- **Decision**: Treat disabled terminal controls with the explicit message "Pod 已完成，终端仅支持运行中的 Pod" as expected behavior, while still verifying Pod diagnostics and resource monitoring APIs return 200.
- **Alternatives**: Create another long-running dev workflow only to verify interactive exec in this pass.
- **Rationale**: This deploy verification focused on the run list/detail/log/Pod/monitoring regression surface; completed Pods should not expose exec, and the current UI states that clearly.

## 2026-06-23 - Watcher owns bounded anomaly repair

- **Context**: Default `/runs` summary reads must stay DB-backed for large Run counts, but terminal rows misclassified as workflow-unavailable can remain stale unless a user opens the detail page.
- **Decision**: Add a bounded anomaly-repair pass to the background watcher for recent or stale-message terminal rows, while keeping default list reads passive.
- **Alternatives**: Re-enable Argo reconciliation from default list reads, or require users to open detail pages to repair each row.
- **Rationale**: Background repair preserves Runtime OS list scalability and lets stale summaries converge automatically.

## 2026-06-23 - Batch child repairs sync item status

- **Context**: Batch-scoped summaries derive status from `backfill_items.status` before `pipeline_runs.status`, so repairing only the Run ledger can leave batch pages showing stale failures.
- **Decision**: When a refreshed batch child Run is persisted, sync the linked backfill item status from the repaired Run.
- **Alternatives**: Leave item repair to batch detail refresh only.
- **Rationale**: Watcher-driven convergence should apply to both global Run summaries and batch child summaries.

## 2026-06-23 - Pre-commit all-files false positive

- **Context**: `pre-commit run --all-files` failed in `detect-secrets` on tracked `.agent/reports/databrew-user-guide.html:300`, an unrelated embedded base64 screenshot line.
- **Decision**: Keep this runtime fix scoped and run pre-commit on changed files, which passed.
- **Alternatives**: Modify the unrelated report file to add an allowlist marker.
- **Rationale**: The failing file is outside this change and is not staged; changing it would mix unrelated report maintenance into the backend watcher fix.

## 2026-07-26 - Use PR CI/CD instead of manual dev deployment

- **Context**: The user explicitly instructed: “不用，推送了pr，自动会cicd的”. The local pre-push all-files gate is also blocked by unrelated environment/baseline issues (Terraform/TFLint unavailable, Docker unavailable, and a pre-existing detect-secrets finding outside this diff).
- **Decision**: Skip the manual frontend dev deploy and Chrome DevTools MCP verification for this PR, and use `SKIP_PREPUSH=1` for this push only; rely on GitHub PR CI/CD for remote verification.
- **Alternatives**: Complete a manual Wrangler dev deployment and browser verification, and repair all unrelated local pre-push prerequisites before pushing.
- **Rationale**: The current explicit user instruction takes precedence, while this append-only record preserves the verification exception and the skipped local-gate rationale for reviewers.
