# Decisions — CYB-3906

## 2026-07-23 — Proceed with a temporary Linear placeholder

- **Context**: Repository policy normally requires a Linear issue before a
  runtime change. The user explicitly instructed the agent to skip Linear for
  now and said they will supply it later.
- **Decision**: Use an explicit temporary placeholder in the worktree branch
  and OpenSpec change, then replace it with `CYB-3906` before commit and PR.
- **Alternatives**: Block all work until Linear is available.
- **Rationale**: The user's current explicit instruction takes precedence, and
  the temporary marker keeps the missing traceability visible rather than
  silently attaching the fix to an unrelated issue.

## 2026-07-23 — Canonicalize on UUID instead of the business placeholder name

- **Context**: CYB-3677's INV-2 decision assumed
  `batchSubtaskWorkflowName` was the name submitted to Argo. CYB-3076 and the
  current Deploy implementation instead force `workflow_name = runID`.
- **Decision**: Preserve the UUID runtime convention and make the placeholder,
  automatic retry, AlreadyExists lookup, and watcher identity use that same
  UUID.
- **Alternatives**: Change Deploy back to the `<pipeline>-batch-...` business
  name.
- **Rationale**: UUID workflow names are already deployed and support direct
  pod-to-run routing. Aligning the placeholder to runtime truth removes the
  split identity without regressing that behavior.

## 2026-07-23 — Correctness does not depend on the cycle advisory lock

- **Context**: The per-cluster cycle lock reduces duplicate work but
  intentionally fails open on lock errors, and recovery/materialization paths
  can still overlap.
- **Decision**: Derive the initial attempt UUID deterministically from the full
  job and asset IDs. The lock remains an efficiency mechanism; deterministic
  identity plus Argo AlreadyExists remains the correctness backstop.
- **Alternatives**: Use random initial UUIDs and assume the advisory lock always
  serializes submission.
- **Rationale**: A P0 exactly-once invariant must survive the documented
  fail-open path.

## 2026-07-23 — OpenSpec approved

- **Context**: The runtime fix requires an explicit OpenSpec checkpoint before
  application code changes.
- **Decision**: The user approved the proposed design and authorized
  implementation, dev verification, and a pull request targeting `dev`.
- **Alternatives**: Keep the change at specification-only status.
- **Rationale**: The approval satisfies the implementation checkpoint; the
  temporary traceability marker was subsequently replaced with `CYB-3906`.

## 2026-07-23 — Concurrency verification isolates the changed invariant

- **Context**: `go test -race ./internal/usecase/backfill` reports a pre-existing
  race in the shared `pausedSyncRepo` test fake during
  `TestRunSubmitterCycle_ShardsByCluster`; the reported reads and writes are in
  `sync_progress_test.go`, outside this change.
- **Decision**: Keep the package's normal test suite as the acceptance gate and
  run the new deterministic-identity concurrency regression under `-race`
  separately.
- **Alternatives**: Expand this P0 patch to repair the unrelated shared fake.
- **Rationale**: The targeted race test validates the new concurrent identity
  invariant without mixing an unrelated test-harness repair into the runtime
  fix.

## 2026-07-23 — Dev deploy requires explicit secret-access approval

- **Context**: The pre-commit backend image built successfully, but revision
  `cyber-databrew-backend-dev-01517-97v` failed startup because the shared dev
  service still supplies default `DATABREW_TOKEN` and `JWT_SECRET` values that
  current production-mode validation rejects. Binding the existing
  `cyber-databrew-dev-databrew-token` secret requires granting the dev Cloud Run
  service account access to that secret.
- **Decision**: Leave traffic on ready revision
  `cyber-databrew-backend-dev-01516-rgf` and wait for the user's explicit
  authorization before changing IAM.
- **Alternatives**: Put secret values into plain Cloud Run environment variables
  or bypass startup validation.
- **Rationale**: A narrowly scoped Secret Manager binding is safer than plain
  credentials or validation bypass, but it is still a shared-environment
  security-boundary change and needs explicit approval.

## 2026-07-23 — User waived dev deployment

- **Context**: The pre-commit image built, but the new Cloud Run revision could
  not start because of the pre-existing shared dev token/JWT configuration.
  Traffic remained 100% on ready revision
  `cyber-databrew-backend-dev-01516-rgf`.
- **Decision**: Per the user's explicit instruction, stop dev deployment and
  proceed to a pull request targeting `dev` using local Tier L and regression
  evidence.
- **Alternatives**: Change shared IAM/configuration to complete the deploy.
- **Rationale**: The current user instruction explicitly waives the deploy gate,
  and avoiding an unrelated shared-environment security change keeps this P0
  patch scoped.

## 2026-07-23 — Pre-commit infrastructure-only failures

- **Context**: `pre-commit run --all-files` passed formatting, merge-conflict,
  EOF, whitespace, YAML, JSON, and schema-path checks. It failed only on
  Terraform provider DNS, a local `tflint` plugin handshake, Docker socket
  permission for fresh-PG migration replay, and an existing detect-secrets
  finding in `docs/review/subscription-task-integration.md`. The hook's automatic
  `.secrets.baseline` timestamp/line update was reverted.
- **Decision**: Do not modify unrelated Terraform, migration, secret-baseline, or
  documentation files in this P0 patch.
- **Alternatives**: Broaden the change to repair local infrastructure and the
  pre-existing secret finding.
- **Rationale**: Repository policy explicitly allows excluding infra-only hook
  failures; targeted and full backend verification already pass.
