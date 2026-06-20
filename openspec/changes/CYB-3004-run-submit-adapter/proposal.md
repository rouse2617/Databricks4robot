# Proposal - CYB-3004

## Why
Run lifecycle controls now prefer RuntimeAdapter, but Run submission still calls Argo workflow-client methods directly. Runtime OS needs submission to use the same runtime boundary so Run Kernel owns product execution while Argo remains replaceable.

## What Changes

### New Capabilities
- Adapter-backed Run submission for Argo workflow creation.
- Adapter-neutral runtime job reference capture during submit.
- Submit-path tests that prove adapter submission preserves manifest, namespace, status, uid, and failure semantics.

### Modified Capabilities
- `CreateRun` / `CreateRunByTemplateID` submit runtime work through the configured RuntimeAdapter when available.
- Legacy workflow-client submit remains as fallback for direct unit-test construction and transitional compatibility.
- Runtime config projection cleanup and owner lookup continue to work after adapter-backed submission.

## Impact
- **Affected code**:
  - `backend/internal/usecase/pipeline/usecase.go`
  - `backend/internal/usecase/pipeline/usecase_test.go`
  - `backend/internal/runtimeos/adapter/interface.go`
  - `backend/internal/runtimeos/adapter/argo/adapter.go`
  - `backend/internal/runtimeos/adapter/argo/adapter_test.go`
  - `backend/cmd/server/core.go`
  - `docs/review/api-guide.md`
  - `openspec/changes/CYB-3004-run-submit-adapter/specs/runtime-os/spec.md`
- **New APIs**: none.
- **Changed APIs**: none.
- **Dependencies**: none.

## Scope
- **In scope**:
  - Route non-dry-run Run submission through RuntimeAdapter when configured.
  - Preserve existing manifest generation, resource guard, runtime mount, runtime config projection, and RunEvent behavior.
  - Preserve workflow-client fallback for tests and old construction paths.
  - Keep dry-run preview behavior unchanged.
  - Add focused tests for adapter submit success and failure.
- **Out of scope**:
  - Moving transpiler/manifest construction out of pipeline usecase.
  - Adding new Run schema or relation tables.
  - Changing public Run API request/response shapes.
  - Removing Argo workflow-client from the codebase.
  - Moving logs, pod diagnostics, or metrics to adapter-only paths.

## Success Criteria
- [ ] A configured RuntimeAdapter receives submit calls for non-dry-run Run creation.
- [ ] Adapter submit returns runtime ref data that is persisted on the Run where available.
- [ ] Adapter submit failures return the same product error semantics as existing Argo submit failures.
- [ ] Runtime config projection owner lookup and cleanup remain correct.
- [ ] Existing dry-run and legacy workflow-client fallback tests continue to pass.

## Goals (SLO)
- **Latency**: No additional runtime round trip beyond the current workflow creation plus existing status/detail lookup.
- **Concurrency**: One adapter submit call per created Run.
- **Quality**: Tests cover adapter success, adapter failure, fallback success, and dry-run no-submit behavior.
