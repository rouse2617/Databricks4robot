# Change: Promote pipeline versions from dev to prod

## Why

Pipeline authors currently recreate production templates and component settings by hand after validating them in dev. That is slow and error-prone, and the existing `promote` endpoint only copies a template inside one environment rather than producing a reproducible cross-environment release.

## New Capabilities

- An admin can preview and execute a promotion from the fixed dev environment to the fixed prod environment.
- A promotion bundle contains the selected pipeline version, exact component releases and immutable runtime snapshots, and referenced non-sensitive configuration versions.
- The preview identifies environment-bound Secret, storage, configuration, and registry requirements and requires valid prod mappings before release.
- Prod records every promotion idempotently and atomically, with an audit record and a target-assigned pipeline version.
- An admin can edit a prod pipeline; every save creates a new prod version rather than mutating history.

## Modified Capabilities

- `POST /api/v1/pipelines/{id}/promote` changes from a same-database scope copy into a real dev-to-prod release operation.
- Prod pipeline write protection changes from a blanket lock to role-based versioned writes: admins may create/activate versions, while non-admin users remain read-only.
- Pipeline DSL serialization preserves component release identity (`componentId`, `releaseId`, and `componentVersionLabel`) instead of dropping it during backend normalization.

## Impact

- Backend: pipeline handlers/use cases/repositories, transpiler JSON model, component release and pipeline config resolution, service-to-service authentication, routing, config, and a new promotion audit/idempotency store.
- Frontend: deployment panel, promotion plan/mapping flow, prod admin edit controls, version history, and typed API client.
- Contract: OpenAPI, API guide, Python SDK, unit tests, and dev smoke coverage.
- Deployment: fixed dev/prod backend targets and dedicated promotion caller authentication.
- Schema: a hand-written Atlas migration under `backend/migrations/` is proposed and requires explicit off-limits approval before implementation.

## Scope

### In scope

- Fixed one-way dev-to-prod promotion initiated by an admin.
- Exact pipeline version selection and target-owned prod version allocation.
- Component release metadata and digest-pinned runtime snapshots.
- Referenced ready, non-sensitive pipeline configuration versions.
- Explicit mapping and target validation for Secret, storage, and environment-specific configuration resources.
- Idempotent, atomic import with audit status.
- Admin-only prod edits and active-version changes, with version history retained.

### Out of scope

- Prod-to-dev promotion, arbitrary target URLs, or multi-target release orchestration.
- Copying Secret values, Kubernetes Secrets/ConfigMaps, PVC names, service accounts, namespaces, or cluster credentials.
- Provisioning missing prod infrastructure.
- Copying or retagging container images; the first version validates that the digest is pullable from prod.
- A multi-party approval workflow.
- Non-admin writes to prod pipelines.

## Success Criteria

- An admin can preview all dependencies and blockers before making a prod write.
- A ready dev pipeline can be promoted once and appears in prod with the same normalized behavior and exact component image digests.
- Repeating a request with the same idempotency key and bundle digest does not create another prod version.
- A missing or ambiguous dependency, digest conflict, invalid mapping, or authentication failure creates no partial prod data.
- Admin edits create a new prod version; prior versions and runs remain reproducible.
- Promotion payloads, audit logs, and UI responses contain no Secret values.

## SLO

- For a warm service and a pipeline of up to 100 nodes, promotion preview p95 is at most 2 seconds, excluding external registry latency.
- Accepted promotion metadata/import p95 is at most 5 seconds, excluding Cloud Run cold start.
- A single idempotency key and bundle digest produces at most one prod pipeline version.
- Import is transactional: any validation or persistence failure leaves zero partial component/config/template writes.
