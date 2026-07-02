# CYB-1517: Register missing search/assets and customers routes

## Problem

Two handlers are fully implemented but never wired in `routes.go`:

1. `GET /api/v1/search/assets` — handler at `internal/handlers/search/handler.go:38` (`SearchAssets`), documented in `api/openapi.yaml`, but route block only registers `sync-status` and `sync-progress`.

2. Customer CRUD — handler at `internal/handlers/customer/handler.go` (Create, Get, Update, List), documented in `api/openapi.yaml`, but `customerHandler` is passed as unused `_` in routes.go.

Both return 404 in production.

## Fix

Add the missing route registrations in `backend/routes/routes.go`.

## Impact

- No new handlers, no schema changes, no SDK changes
- Existing handlers get wired to their documented routes
- `api/openapi.yaml` already documents these endpoints — no OpenAPI change needed
