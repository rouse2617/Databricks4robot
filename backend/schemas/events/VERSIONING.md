# Event Schema Versioning Guide

This document explains how to perform minor and major version bumps for event schemas in `schemas/events/`.

## Version Convention

Each schema file is named `<event_type>.v<N>.json` (e.g. `asset_created.v1.json`).
The `registry.json` file tracks the current version for each event type.

---

## Minor Bump (backward-compatible)

A minor bump adds **optional** fields to an existing schema version. The version number does NOT change — the same `v1` file is updated in place.

### Rules

- Only add new **optional** properties (not in `required`).
- Do not change the type of existing fields.
- Do not remove existing fields.
- Keep `additionalProperties: true` so consumers ignore unknown fields.

### Example: Add `reason` to `algo_started.v1`

1. Edit `algo_started.v1.json`:

```json
{
  "properties": {
    ...existing fields...
    "reason": {
      "type": ["string", "null"],
      "description": "Optional reason for starting the algorithm."
    }
  }
}
```

2. No change to `registry.json` (version stays `v1`).

3. CI will verify no new required fields were added.

---

## Major Bump (breaking change)

A major bump creates a **new schema file** with an incremented version number. The old file is kept for consumers still on the previous version.

### When to use

- Adding a new **required** field.
- Changing the type of an existing field.
- Removing a field.
- Renaming a field.

### Example: Add required `tenant_id` to `asset_created`

1. Copy the current schema to a new version:

```bash
cp schemas/events/asset_created.v1.json schemas/events/asset_created.v2.json
```

2. Edit `asset_created.v2.json`:

```json
{
  "id": "https://data-platform/schemas/events/asset_created.v2.json",
  "title": "asset_created.v2",
  "required": ["asset_id", "mcap_file_id", "asset_type", "lifecycle_state", "tenant_id"],
  "properties": {
    ...existing fields...
    "tenant_id": {
      "type": "string",
      "description": "Tenant identifier (required in v2)."
    }
  }
}
```

3. Update `registry.json`:

```json
{
  "events": {
    "asset_created": {
      "current_version": "v2",
      "schema_file": "asset_created.v2.json",
      ...
    }
  }
}
```

4. Update the producer code to set `payload_schema_version = "v2"` when emitting `asset_created` events.

5. Keep `asset_created.v1.json` in the repo — consumers may still need to read historical v1 events.

---

## Checklist for any schema change

- [ ] Edit or create the schema file in `schemas/events/`
- [ ] Update `registry.json` if the current version changed
- [ ] Run `go build ./...` and `go vet ./...` (no Go changes needed for schema-only edits)
- [ ] CI `schema-events` workflow will validate syntax and backward compatibility
- [ ] For major bumps: update producer code to emit the new `payload_schema_version`
