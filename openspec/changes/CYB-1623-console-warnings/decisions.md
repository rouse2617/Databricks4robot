# Decisions

## 2026-06-03 — Filter React Flow type warning
- **Context**: Pipeline design uses `@ant-design/pro-flow`, which emits React Flow error `002` during initialization even after `nodeTypes` and `edgeTypes` are module-level stable references.
- **Decision**: Install a narrow console warning filter for the exact React Flow `nodeTypes or edgeTypes` warning string. Other console warnings and errors continue to pass through.
- **Alternatives**: Replace pro-flow with raw React Flow in this PR, or leave the warning. Raw React Flow would be a larger behavior change; leaving the warning keeps browser verification noisy.
- **Rationale**: The app code now uses stable type objects; the remaining warning is a library initialization false positive in this integration path.

## 2026-06-03 — Skip Cloud Run dev deploy before PR
- **Context**: Frontend runtime changes normally require Cloud Run dev deploy verification before commit/push.
- **Decision**: Skip Cloud Run dev deploy for this PR because the user explicitly requested: "直接提pr 到dev".
- **Alternatives**: Deploy frontend dev before commit. That would follow the default deploy gate but conflicts with the current user instruction.
- **Rationale**: User instruction in the current message has highest precedence. Local build and Chrome DevTools MCP verification were completed before PR.
