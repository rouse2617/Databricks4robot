# Tasks — CYB-1542 Pipeline Template Versioning

## Checkpoint

- [x] Create/link Linear issue `CYB-1542`.
- [x] Create branch/worktree from latest `origin/dev`.
- [x] Write OpenSpec proposal, design, tasks, and spec delta.
- [x] Stop for user confirmation before runtime edits.

## Backend

- [x] Update `PipelineTemplateRepository` with `FindByNameAndVersion`.
- [x] Implement Postgres `FindAll` as latest-version-per-template.
- [x] Implement Postgres `FindByNameAndVersion`.
- [x] Fix `ListVersions` to resolve route `{id}` as UUID, then list by template name.
- [x] Add optional `version` field to deploy-by-template and create-run-by-template request bodies.
- [x] Update usecase deploy-by-template logic to resolve and run the selected version.
- [x] Ensure deployment/run records receive selected `templateVersion`.
- [x] Add backend unit tests for latest-list, version listing, and versioned deploy.

## API Contract

- [x] Update `api/openapi.yaml` for latest-list behavior, version listing, deploy `version`, and `templateVersion` response fields.
- [x] Update `docs/review/api-guide.md` with template version examples.
- [x] Add or update a targeted smoke script for save versions, list latest, list versions, and deploy selected version.
- [x] Update handler swagger comments if present. No pipeline handler swagger blocks are present in this package; OpenAPI remains the contract source.

## Frontend

- [x] Extend `Frontend/src/api/pipelineApi.ts` types with `version`, `templateVersion`, and version APIs.
- [x] Template list shows one row per logical template with latest `vN` and historical version count.
- [x] Add lazy versions dropdown/popover for each template row.
- [x] Editor top bar shows selected version when editing saved templates.
- [x] Switching versions loads that snapshot into the canvas.
- [x] Saving any loaded version creates a new latest version and refreshes version options.
- [x] Run dialog includes a version selector and sends selected version.
- [x] Execution list displays template version where available.
- [ ] Execution detail displays template version where available.

## Verification

- [x] Backend targeted tests: pipeline repo/usecase/handler packages.
- [x] Frontend focused tests for pipeline API and affected UI behavior.
- [x] Frontend build.
- [x] API smoke against local backend.
- [x] Chrome DevTools MCP flow:
  - [x] Save same template multiple times.
  - [x] Confirm list has one template row with latest version and count.
  - [x] Switch editor to an older version and confirm canvas changes.
  - [x] Run selected older version.
  - [x] Confirm execution record list shows selected version.
  - [ ] Confirm execution detail shows selected version.

## Follow-Ups

- [ ] Whole-template delete semantics (delete all versions) if product wants that behavior.
- [ ] Version diff UI.
- [ ] Version retention policy.
