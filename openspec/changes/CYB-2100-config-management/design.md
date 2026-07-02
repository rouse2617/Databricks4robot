# Design — CYB-2100

## Architecture Context
- **Constraints**: React 19 + TypeScript + Ant Design 5 frontend; the current config CRUD backend only supports saved config records, not deploy-time ad hoc file uploads or inline draft persistence.
- **Goals**: let users express deploy-time file intent, serialize that intent into a concrete deploy payload, and project the selected file into workflow Pods without requiring new database schema work.
- **Non-Goals**: persistent library storage for ad hoc upload/inline drafts, multi-file runtime mounts per run, or generic parsing of arbitrary config contents into many environment variables.

## Affected Modules
- `Frontend/src/components/pipeline/DeployPanel.tsx` — add config source selector, config summary, mount-path editor, and upload/inline draft controls
- `Frontend/src/components/pipeline/DeployPanel.test.tsx` — cover mode switching, validation, and summary rendering
- `Frontend/src/api/pipelineConfigs.ts` — reuse saved-config list/detail types for the picker
- `Frontend/src/api/pipelineApi.ts` / `Frontend/src/api/deployPipelineRun.ts` / `Frontend/src/api/batchJobApi.ts` — send config selection in single-run and batch-run payloads
- `backend/internal/handlers/pipeline/handler.go` / `backend/internal/handlers/backfill/handler.go` — accept config selection contract
- `backend/internal/usecase/pipeline/usecase.go` / `backend/internal/usecase/backfill/usecase.go` — resolve config content snapshots and carry them into workflow creation
- `backend/internal/transpiler/*` — emit runtime mount and env projection
- `api/openapi.yaml` / `docs/review/api-guide.md` — document the new deploy contract

## Architecture Decisions

### Decision 1: Model deploy-time config selection as a three-mode source switch
- **Approach**: the deploy panel exposes one segmented control / radio group with `saved`, `upload`, and `inline` modes, and renders exactly one source editor at a time.
- **Alternative**: show all three input styles on the page simultaneously.
- **Rationale**: these inputs are mutually exclusive runtime intents; a mode switch reduces confusion and keeps the review surface compact.
- **Trade-off**: switching modes needs explicit draft reset or draft preservation rules.

### Decision 2: Normalize all three source modes into one deploy-time config snapshot contract
- **Approach**: the frontend resolves the selected mode into one payload shape containing source metadata, filename, content or saved-config reference, mount path, and target filename. The backend resolves saved-config references to immutable version content and trusts upload/inline draft content as request-scoped runtime input.
- **Alternative**: only support saved configs end-to-end and keep upload/inline as preview-only.
- **Rationale**: the active user goal explicitly requires upload, inline edit, mounted-file verification, and environment-variable verification.
- **Trade-off**: deploy payloads become larger for ad hoc content, but this avoids new persistence schema and keeps runtime behavior explicit.

### Decision 3: Mount target is edited beside the source selector
- **Approach**: place mount-path and target filename inputs in the same config section so users define source and runtime destination together.
- **Alternative**: hide mount-path editing in component manager or a separate advanced drawer.
- **Rationale**: the runtime question is "what file goes where" and the UI should present both halves in one place.
- **Trade-off**: deploy panel becomes denser, so labels and helper text need to be explicit.

### Decision 4: Project runtime config as a workflow-mounted config volume plus companion env vars
- **Approach**: the backend creates a single runtime config projection per deploy request. The workflow receives a dedicated mounted file at `<mountPath>/<targetFilename>` and a small set of env vars such as `PIPELINE_CONFIG_PATH`, `PIPELINE_CONFIG_FILENAME`, and `PIPELINE_CONFIG_SOURCE`.
- **Alternative**: inject raw file content only through env vars, or defer to a later init-container secret store design.
- **Rationale**: the user explicitly asked for a mounted-file form and environment-variable verification. A mounted file matches common app expectations, while companion env vars make the projection discoverable and testable.
- **Trade-off**: projected content is request-scoped and duplicated per run instead of being shared as a long-lived cluster object.

## Data Flow

```text
User opens DeployPanel
        ↓
User chooses config source mode
        ↓
saved mode  -> fetch/select existing config metadata
upload mode -> read local file metadata/content into local draft state
inline mode -> capture editor content into local draft state
        ↓
User sets mount path / target filename
        ↓
Deploy panel summary shows source type + selected file metadata + mount target
        ↓
Deploy request sends normalized config selection payload
        ↓
Backend resolves saved-config references to immutable version content
        ↓
Backend injects runtime config projection into workflow generation
        ↓
Workflow Pod gets mounted config file and companion env vars
```

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| Users assume upload/inline persists to the platform library | Confusion and false confidence | Keep helper copy explicit that these two modes are request-scoped runtime inputs |
| Mode switching destroys draft unexpectedly | Frustrating deploy authoring UX | Decide and test whether each mode preserves its own last draft during the session |
| Mount-path semantics are unclear | Wrong runtime expectation | Use concrete labels and examples such as `/app/configs/model.yaml` |
| Large ad hoc config content inflates request size | Deploy failure or slow request handling | Reuse the existing config-content size discipline, validate content early, and keep the first slice single-file only |
