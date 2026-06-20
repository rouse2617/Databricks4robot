# Decisions — CYB-3002

## 2026-06-20 — Placeholder Linear ID
- **Context**: The agent environment still has no Linear credential available, so a real Linear issue cannot be created or linked from this session.
- **Decision**: Use `CYB-3002` as a temporary Runtime OS phase-two placeholder ID and keep the OpenSpec change traceable in-repo.
- **Alternatives**: Stop and request Linear credential setup before writing the OpenSpec.
- **Rationale**: The active Runtime OS objective explicitly asks for autonomous iteration. The missing credential does not block in-repo planning.

## 2026-06-20 — OpenSpec checkpoint retained
- **Context**: The active objective asks the agent to continue autonomously, while repository rules require an OpenSpec checkpoint before editing runtime code for a new feature phase.
- **Decision**: Create the OpenSpec artifacts for the next runtime phase and stop before application-code edits until the user confirms the checkpoint.
- **Alternatives**: Treat the earlier phase-one autonomy approval as sufficient for all future phases.
- **Rationale**: This is a new feature slice after phase one merged to `dev`; preserving the checkpoint reduces scope drift before changing backend/frontend runtime paths again.

## 2026-06-20 — OpenSpec checkpoint approved
- **Context**: After the CYB-3002 OpenSpec artifacts were created, the user replied "ok".
- **Decision**: Treat that reply as approval to proceed from the OpenSpec checkpoint into runtime implementation for the scoped Run Tree / RunStateMachine phase-two slice.
- **Alternatives**: Ask for a second explicit "OpenSpec OK，继续" confirmation before editing code.
- **Rationale**: The repository rule requires user confirmation before runtime edits, and the latest user response provides that confirmation in the active task context.

## 2026-06-20 — RunStateMachine package placement
- **Context**: The original design mentioned adding the pure status helper under `runtimeos/run`, but that package already imports the pipeline usecase adapter.
- **Decision**: Place the pure status helper under `runtimeos/state` so `pipeline.Usecase` can import it without creating a `runtimeos/run` ↔ `pipeline` import cycle.
- **Alternatives**: Inline aggregation in `pipeline.Usecase` or refactor the existing `runtimeos/run` service package.
- **Rationale**: A small pure package preserves the Runtime OS boundary while keeping the implementation low-risk for this phase.

## 2026-06-20 — UI label tweak verified locally
- **Context**: After deployed verification, the user pointed out that the visible label `运行树` was too abstract and explicitly allowed frontend validation on a local deployment.
- **Decision**: Change the user-facing Batch Detail label to `批次运行` / `子运行` while keeping the internal API/OpenSpec term `Run Tree`, then verify the wording through local Vite against the dev backend instead of redeploying the frontend Cloud Run revision again.
- **Alternatives**: Redeploy frontend Cloud Run and Cloudflare Worker/Assets for the wording-only change before commit.
- **Rationale**: The backend and public dev integration had already been verified; the final adjustment is display copy only, and the user approved local frontend validation for this step.
