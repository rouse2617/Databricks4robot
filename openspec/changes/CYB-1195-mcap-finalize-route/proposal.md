# CYB-1195: Fix mcap finalize route mismatch

## Summary

Backend registers `POST /api/v1/mcap/upload/finalize` but SDK and frontend call `POST /api/v1/mcap-files/{id}/finalize`. The route mismatch means finalize calls always 404.

## Root cause

The route was registered at a non-RESTful path (`/mcap/upload/finalize`) while the SDK design docs and frontend API module expected a RESTful resource path (`/mcap-files/:id/finalize`).

## Fix

1. Add `POST /api/v1/mcap-files/:id/finalize` route in routes.go
2. Update `FinalizeUpload` handler to read `mcap_file_id` from URL param `c.Param("id")`, falling back to body for backward compat
3. Add OpenAPI spec entry for the new route
4. Keep the old route for backward compatibility

## Verification

1. `curl -X POST https://dev/api/v1/mcap-files/<valid-id>/finalize` → 200
2. `curl -X POST https://dev/api/v1/mcap/upload/finalize -d '{"mcap_file_id":"..."}'` → 200 (backward compat)
3. Invalid ID → 400
