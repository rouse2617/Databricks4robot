# Decisions — CYB-1646

## 2026-06-07 — Scope CYB-1646 to frontend operations copy
- **Context**: CYB-1646 contains P0/P1 workflow operations UX issues plus broader P2 component and asset navigation polish. Dev browser regression reproduced the missing-cost empty state on a completed complex DAG and confirmed the workflow detail UI is the highest-signal path.
- **Decision**: Scope this PR to workflow execution detail, node troubleshooting copy, and pipeline run dialog/action wording. Do not change backend workflow execution, API contracts, Kubernetes/Argo scheduling behavior, or larger IA areas in this slice.
- **Alternatives**: Bundle component marketplace metadata, asset batch-run IA, and backend scheduling event aggregation into the same PR.
- **Rationale**: The selected slice directly addresses the reproduced P0/P1 symptoms with low backend risk and can be verified with focused tests plus Chrome DevTools MCP on dev.

## 2026-06-07 — Compact recovery scratchpad missing
- **Context**: Project rules require re-reading `.agent/context/current-work.md` after compaction, but the file is not present in this checkout.
- **Decision**: Continue from the repository rules, Linear issue context, branch state, and OpenSpec artifacts.
- **Alternatives**: Stop and ask the user to recreate the scratchpad.
- **Rationale**: The missing scratchpad is not required to define or verify CYB-1646, and the branch plus Linear issue provide the active work context.

## 2026-06-07 — OpenSpec checkpoint approved
- **Context**: Runtime changes under `Frontend/` require the OpenSpec checkpoint before editing application code.
- **Decision**: User replied "ok" after reviewing the CYB-1646 OpenSpec checkpoint, so implementation may proceed.
- **Alternatives**: Keep waiting for a more formal approval phrase.
- **Rationale**: The reply directly followed the checkpoint request and confirms the proposed scope.

## 2026-06-07 — Pre-commit terraform hooks unavailable locally
- **Context**: `uvx pre-commit run --all-files --show-diff-on-failure` failed only on `terraform_fmt`, `terraform_validate`, and `terraform_tflint` because this machine does not have Terraform/OpenTofu or `tflint` installed.
- **Decision**: Re-run pre-commit with `SKIP=terraform_fmt,terraform_validate,terraform_tflint`; all non-Terraform hooks passed. Keep the full failure noted here and rely on CI for Terraform hooks because CYB-1646 does not touch Terraform or infra files.
- **Alternatives**: Install global Terraform/OpenTofu and `tflint` locally before committing.
- **Rationale**: The PR scope is frontend runtime UX plus OpenSpec text; installing unrelated global infra tooling would not increase confidence in the changed files.
