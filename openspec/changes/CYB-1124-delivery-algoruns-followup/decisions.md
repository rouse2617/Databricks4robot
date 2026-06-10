# Decisions — CYB-1124

## 2026-05-23 — Review findings split from CYB-1123
- **Context**: CYB-1123 was already marked Done and merged to `dev`, but follow-up review found additional runtime bugs in the same feature area.
- **Decision**: Create CYB-1124 as a separate bugfix issue and OpenSpec change instead of reopening CYB-1123.
- **Alternatives**: Reopen CYB-1123 and amend the existing change directory.
- **Rationale**: A new issue preserves auditability for the merged commit while keeping the follow-up fix scoped and reviewable.

## 2026-05-23 — SDK out of scope unless contract surface changes
- **Context**: The planned fix changes behavior of existing endpoints but does not add SDK methods or request fields.
- **Decision**: Do not update SDK unless implementation changes public SDK-callable shapes.
- **Alternatives**: Add SDK coverage for delivery C2/idempotency in this PR.
- **Rationale**: SDK parity is already tracked separately; this fix should remain focused on runtime correctness and contract docs/smoke.

## 2026-05-23 — OpenSpec checkpoint approved
- **Context**: Proposal, tasks, context files, decisions, and spec deltas were created before runtime edits.
- **Decision**: Proceed with backend implementation after the user approved in chat with "开始".
- **Alternatives**: Wait for a literal "OpenSpec OK" phrase.
- **Rationale**: The user explicitly instructed to start after seeing the checkpoint summary.
