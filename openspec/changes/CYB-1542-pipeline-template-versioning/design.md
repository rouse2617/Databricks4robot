# Design — CYB-1542 Pipeline Template Versioning

## Concepts

- **Pipeline template**: logical user-managed pipeline identified by `name`.
- **Template version**: immutable snapshot row in `pipeline_templates`, identified by UUID and `version`.
- **Latest template row**: highest `version` for a given `name`.
- **Run binding**: a run records both `template_id` and `template_version` for traceability.

## Backend Design

### Repository

`PipelineTemplateRepository.FindAll` should match its existing interface comment: latest version of each template ordered by recency. PostgreSQL can implement this with `DISTINCT ON (name)` or a window function.

Add a repository method:

```go
FindByNameAndVersion(ctx context.Context, name string, version int) (*models.PipelineTemplate, error)
```

This avoids forcing handlers to know historical version UUIDs when the user selects `v2` from a template's version list.

### Version Listing

Keep the existing route:

```text
GET /api/v1/pipelines/{id}/versions
```

But resolve `{id}` as a template UUID first:

1. `FindByID(id)` gets the selected/latest row.
2. Use its `Name` to call `FindVersionsByName(name)`.
3. Return all versions ordered by `version DESC`.

This preserves the existing route while making the URL semantics match the rest of the template API.

### Deploy Version Selection

Both deploy-by-template endpoints should accept:

```json
{
  "asset_ids": [],
  "target_id": "default",
  "version": 2
}
```

If `version` is absent or `0`, use the template UUID as-is. This keeps existing behavior: running from a latest-list row runs latest.

If `version` is positive:

1. Load the template by UUID.
2. Resolve its logical `Name`.
3. Load `FindByNameAndVersion(name, version)`.
4. Deploy the resolved snapshot.
5. Write `TemplateID` and `TemplateVersion` from the resolved snapshot.

### Response Shape

`PipelineTemplate` already exposes `version`. Add metadata only if the frontend cannot derive it efficiently. For P0 the frontend can derive count from `GET /pipelines/{id}/versions`.

`PipelineDeployment` should expose optional `templateVersion` for compatibility endpoints because the execution list and run dialog still use deployment-shaped responses.

`PipelineRun` already exposes `templateVersion`.

## Frontend Design

### Template List

Each row/card should read as a template, not a version row:

```text
my-pipeline
ID: ea7a26e8
v4 latest · 4 versions · 3 nodes · saved today 15:30
[Run] [Edit] [Versions] [Delete]
```

The `Versions` control can be a compact dropdown/popover. It should list:

```text
v4 latest · today 15:30 · 3 nodes  [Edit] [Run]
v3        · today 14:20 · 2 nodes  [View] [Run]
v2        · yesterday   · 2 nodes  [View] [Run]
```

### Editor

Add a version selector next to the template name when editing a saved template:

```text
Name: my-pipeline    Version: v4 latest ▼    [Save new version]
```

When viewing an older version, show a small warning:

```text
Viewing v2. Saving creates v5 and does not overwrite v2.
```

The save button should be worded as `Save new version` / `基于此版本保存新版本` rather than implying overwrite.

### Run Dialog

Run dialogs should show a version selector:

```text
Pipeline template: my-pipeline
Version: v4 latest ▼
Assets: ...
[Run v4]
```

Default selected version is the current/latest template version. Selecting `v2` sends `version: 2`.

### Execution UI

List/detail should show template version where available:

```text
Pipeline: my-pipeline · v3
Run ID: 6724f609
```

When a legacy record has no `templateVersion`, omit the version rather than showing misleading data.

## Compatibility

- Existing saved templates remain valid.
- Existing `/pipelines/{id}` continues to load a specific snapshot by UUID.
- Existing deploy callers that omit `version` keep current behavior.
- Legacy Argo-only workflow rows may not have template version metadata; UI must handle absence.

## Risks

- Changing `FindAll` changes API behavior from all rows to latest rows. This is intended but must be documented.
- Deleting a latest version row could reveal a previous version as the latest list row. P0 keeps current single-row delete semantics; whole-template delete can be a follow-up if needed.
- Frontend must avoid issuing N+1 version-list requests for very large template lists without caching. P0 can fetch versions lazily when a row's version menu opens.
