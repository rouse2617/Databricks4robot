# Proposal — CYB-1542 Pipeline Template Versioning

## Problem

Pipeline templates already persist a new row with an incrementing `version` on every save, but the product still behaves as if templates are flat records:

- Template management can show duplicate rows for the same template name.
- Users cannot clearly see which template version they are editing.
- Users cannot choose a historical version to view, base edits on, or run.
- Run records can store `template_version`, but the UI and deploy request do not let users bind a run to an explicit version.

This makes pipeline changes hard to reason about. A user can save a new template snapshot, but cannot confidently answer "which version did this run use?"

## Goals

- Treat a saved pipeline as a **pipeline template** with immutable version snapshots.
- Show one row per template in the template list, representing the latest version.
- Make historical versions discoverable from the template list and edit page.
- Let users load any version into the editor and save a new version without overwriting history.
- Let users choose the version to run; default to latest.
- Persist and display the chosen template version on run/deployment records.

## Non-Goals

- No new database table; use existing `pipeline_templates.version` and `pipeline_runs.template_version`.
- No version diff viewer in this iteration.
- No automatic old-version retention policy.
- No parameter-level or component-level independent versioning.
- No destructive overwrite of historical versions.

## User Experience

The user model should be:

> I have one pipeline template. Every save creates a snapshot version. I can view old snapshots, run any snapshot, and always know which version a run used.

Primary surfaces:

- **Pipeline template list**: one row per template, with `vN latest` and total version count.
- **Pipeline editor**: version selector near the name field; switching versions restores that snapshot.
- **Run dialog**: version selector defaults to latest and labels the run button with the selected version.
- **Execution list/detail**: show `templateName · vN` where known.

## API Impact

- `GET /api/v1/pipelines` returns only latest version per template name.
- `GET /api/v1/pipelines/{id}/versions` returns all versions for the template resolved by `{id}`.
- Deploy-by-template request accepts optional `version`.
- Pipeline run/deployment responses expose `templateVersion` where available.
- OpenAPI and api-guide are updated in the same PR.

## Acceptance Criteria

- Saving the same template name multiple times creates `v1`, `v2`, `v3`.
- Template list shows one row for that template, not three duplicate rows.
- Template row shows latest version and historical version count.
- Version selector can load an older snapshot into the canvas.
- Saving while viewing any version creates the next version.
- Running from list or editor can choose a version; default is latest.
- `pipeline_runs.template_version` is set to the selected version.
- Execution UI shows the template version used for the run where backend data is available.
