# Proposal — CYB-1516

## Why
The public docs and generated API contract expose SDK/API examples that do not consistently match the current backend routes, SDK methods, and standardized error envelope.

## What Changes

### New Capabilities
- SDK users can find the canonical error fields and know how to branch on them without parsing strings.
- Routes that were already present in the public OpenAPI/SDK surface are registered in the runtime router.

### Modified Capabilities
- Public documentation examples only use SDK methods and API routes that exist in the current codebase.
- API error schemas consistently describe the standard error envelope.
- Backend auth compatibility routes return the same error envelope as the rest of the protected API surface.

## Impact
- **Affected code**: `docs-site/docs/**`, `docs/review/api-guide.md`, `api/openapi.yaml`, `backend/routes/routes.go`, backend route tests
- **New APIs**: No newly designed APIs; this change wires existing documented OpenAPI/SDK routes that were missing from runtime registration.
- **Dependencies**: None

## Scope
- **In scope**: documentation/code consistency fixes, standard error envelope docs, OpenAPI error schema references, SDK example corrections, minimal backend error envelope cleanup for ad-hoc auth route errors, runtime registration for already documented routes
- **Out of scope**: new backend feature routes, changing SDK public method names, changing auth semantics, broad state-machine redesign

## Success Criteria
- [ ] Docs examples do not call nonexistent SDK methods.
- [ ] Docs do not advertise unregistered API routes as primary examples.
- [ ] OpenAPI references one valid error schema for standard JSON errors.
- [ ] SDK docs show `code`, `message`, `http_status`, `request_id`, and `details` handling.
- [ ] Auth route failures use the standard `code/message/request_id/details` envelope.
- [ ] Existing docs-site and backend verification pass.
