# CYB-1684 Decisions

## 2026-06-05 - PR before dev deploy

Context:
- The deploy-before-commit workflow normally requires Cloud Run dev deploy verification before committing runtime changes.
- This change is backend runtime behavior: canvas edges should compile to DAG dependencies only, not implicit output parameter bindings.
- The user explicitly requested opening the PR first: "提交pr".

Decision:
- Submit the PR with targeted backend tests, full backend tests, fmt, vet, pre-commit, and local CI evidence.
- Leave Cloud Run dev deploy verification pending for the deploy owner to run after the PR is available.

Rationale:
- The requested fix is covered by transpiler and pipeline usecase tests.
- No database migration or external API contract change is included.
- The user is coordinating deployment separately and asked for the PR now.
