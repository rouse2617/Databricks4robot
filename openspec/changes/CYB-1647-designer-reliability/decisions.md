# Decisions — CYB-1647

## 2026-06-07 — Reuse existing CYB-1647 issue
- **Context**: The deep test report found pipeline designer reliability issues. Linear already has CYB-1647 for Pipeline page UI/UX and run-flow wording.
- **Decision**: Reuse CYB-1647 instead of creating a duplicate issue. Scope this PR to generated-name replacement and visible component add action.
- **Alternatives**: Create a new narrow Linear issue for the report findings.
- **Rationale**: CYB-1647 already covers Pipeline designer UI/UX; this slice advances that issue without mixing in unrelated backend or diagnostics work.

## 2026-06-07 — Do not rewrite drag/drop in this slice
- **Context**: The report shows Chrome DevTools MCP drag/drop can add the wrong component or fail. The report also notes MCP HTML5 drag/drop can be imprecise with React Flow.
- **Decision**: Add a deterministic visible click-add path now and leave drag/drop hardening for a follow-up with a real Playwright mouse-drag feedback loop.
- **Alternatives**: Rewrite drag/drop internals immediately.
- **Rationale**: Click-add addresses the user-facing reliability and regression-test gap with lower risk; drag/drop needs a separate reproducible browser harness.

## 2026-06-07 — Compact recovery scratchpad missing
- **Context**: Project rules require re-reading `.agent/context/current-work.md`, but the file is absent in this checkout.
- **Decision**: Continue using repository rules, Linear CYB-1647, the deep test report, and branch state as the active context.
- **Alternatives**: Stop and ask the user to recreate the scratchpad.
- **Rationale**: The missing scratchpad is not required to define this scoped frontend UX fix.

## 2026-06-07 — OpenSpec checkpoint approved
- **Context**: Runtime changes under `Frontend/` require an OpenSpec checkpoint before application code edits.
- **Decision**: User replied "ok，改代码吧" after reviewing the CYB-1647 OpenSpec checkpoint, so implementation may proceed.
- **Alternatives**: Keep waiting for a more formal approval phrase.
- **Rationale**: The reply directly approved moving from OpenSpec to code changes.
