## ADDED Requirements

### Requirement: Admin previews a fixed dev-to-prod promotion

- **Priority**: P0
- **Rationale**: An admin must see exact dependencies and target blockers before any production write.

#### Scenario: Ready promotion plan

- **Given** an authenticated admin selects a saved dev pipeline version whose dependencies can be resolved
- **When** the admin requests a promotion plan for the fixed prod target
- **Then** the system returns a plan digest, exact component/config dependencies, required mappings, warnings, blockers, and `ready=true` without writing prod data

#### Scenario: Missing target mapping

- **Given** a dev pipeline references a Secret, storage, or environment-bound config with no valid prod mapping
- **When** the admin requests a promotion plan
- **Then** the response identifies the unresolved logical requirement, returns `ready=false`, and exposes no resource value or credential

#### Scenario: Non-admin requests a plan

- **Given** an authenticated non-admin user
- **When** the user requests a promotion plan
- **Then** the system returns `403` and does not contact the prod importer

### Requirement: Promotion bundle preserves exact portable dependencies

- **Priority**: P0
- **Rationale**: The promoted template must execute the same component artifacts and safe runtime configuration that were reviewed in dev.

#### Scenario: Release-backed component is bundled

- **Given** a pipeline node references an immutable component release
- **When** the dev backend builds the promotion bundle
- **Then** it includes the component/release identity, release label, digest-pinned image, command, args, ports, resources, and non-sensitive environment snapshot

#### Scenario: Legacy component has one digest match

- **Given** a legacy pipeline node has no release ID but has a digest-pinned image matching exactly one release
- **When** the bundle is built
- **Then** the system resolves that release explicitly and reports the fallback as a warning

#### Scenario: Legacy component is ambiguous

- **Given** a legacy node has no release ID and its digest has zero or multiple release matches
- **When** the bundle is built
- **Then** the plan is blocked and the system does not invent a release identity

#### Scenario: Sensitive plaintext is detected

- **Given** a component environment entry or referenced config appears to contain a token, password, Secret, key, or credential
- **When** the bundle is built
- **Then** the sensitive value is excluded and promotion is blocked until it is represented by an approved prod resource mapping

#### Scenario: Target release label conflicts by digest

- **Given** prod already has the same component and release label with a different digest
- **When** prod validates or imports the bundle
- **Then** prod returns `409` and does not overwrite the existing release

### Requirement: Prod import is idempotent, atomic, and auditable

- **Priority**: P0
- **Rationale**: Retries and failures must never create duplicate or partial production versions.

#### Scenario: First successful import

- **Given** a ready plan, valid mappings, a fresh idempotency key, and an unchanged bundle
- **When** the admin executes promotion
- **Then** prod transactionally imports safe dependencies, creates one new prod pipeline version, records a completed audit row, and returns its promotion ID and prod version

#### Scenario: Identical retry

- **Given** a completed promotion for an idempotency key and bundle digest
- **When** the same request is retried with the same key and digest
- **Then** prod returns the original result without creating another pipeline version

#### Scenario: Idempotency key is reused for different content

- **Given** an idempotency key already belongs to one bundle digest
- **When** a request uses that key with a different digest
- **Then** prod returns `409` and writes no dependencies or pipeline version

#### Scenario: Import fails inside the transaction

- **Given** a valid request whose dependency or template persistence fails
- **When** prod imports the bundle
- **Then** the transaction rolls back all dependency/template writes and records or returns only a sanitized failure

#### Scenario: Accepted plan became stale

- **Given** prod catalog state or source bundle content changed after preview
- **When** execution supplies the old plan digest
- **Then** prod returns `409`, creates no version, and requires a fresh preview

### Requirement: Environment-bound resources are mapped and validated

- **Priority**: P0
- **Rationale**: Dev resource identifiers are not portable and must never silently address prod infrastructure.

#### Scenario: Logical resource has a valid prod mapping

- **Given** the plan contains a Secret, storage, or config requirement with an explicit valid prod resource
- **When** prod validates the mapping
- **Then** it rewrites the promoted pipeline reference to the prod resource identity without copying the underlying value

#### Scenario: Target resource is unavailable

- **Given** a required prod catalog resource is missing, not ready, or unauthorized
- **When** prod validates the plan
- **Then** promotion is blocked with a sanitized actionable reason and no prod write

#### Scenario: Image digest is not pullable in prod

- **Given** a bundled component image is digest-pinned but prod runtime identity cannot pull it
- **When** prod validates readiness
- **Then** promotion is blocked before a pipeline version is created

### Requirement: Admin prod edits create new versions

- **Priority**: P0
- **Rationale**: Admins need operational flexibility without destroying historical reproducibility.

#### Scenario: Admin saves a prod edit

- **Given** an admin edits an existing prod pipeline version with the current base version
- **When** the admin saves
- **Then** prod creates the next version, leaves the previous version unchanged, and returns the new complete template

#### Scenario: Non-admin edits prod

- **Given** a non-admin user can view or run a prod pipeline
- **When** the user attempts to edit, delete, or activate a prod version
- **Then** the system returns `403` and changes no prod state

#### Scenario: Admin edit is stale

- **Given** another prod version was created after the admin opened the editor
- **When** the admin saves using the older base version
- **Then** the system returns `409` and does not create a version

#### Scenario: Historical run is inspected

- **Given** a run references an older prod pipeline version
- **When** newer promoted or edited versions are created
- **Then** the run and older version continue to resolve their original normalized pipeline and component digests

#### Scenario: Admin activates a prod version

- **Given** an admin selects an existing prod version
- **When** the admin activates it
- **Then** the active pointer changes without modifying any version content and the audit actor/time are retained

### Requirement: Promotion transport is fixed and strongly authenticated

- **Priority**: P0
- **Rationale**: Cross-environment writes require stronger trust than a client-selected URL or ordinary browser token.

#### Scenario: Known dev service calls prod

- **Given** an admin-authorized request reaches dev and dev presents the configured service identity to the fixed prod audience
- **When** prod validates or imports a bundle
- **Then** prod accepts the service call and separately applies all bundle and mapping validation

#### Scenario: Caller supplies a target URL

- **Given** a promotion request contains an arbitrary host, URL, redirect, or unsupported target
- **When** dev processes the request
- **Then** dev rejects it and never sends credentials or bundle data to that destination

#### Scenario: Internal caller authentication fails

- **Given** a request to an internal promotion endpoint has a missing, invalid, wrong-audience, or unexpected service identity
- **When** prod authenticates the request
- **Then** prod returns `401` or `403`, writes no audit/dependency/template data, and logs no credential

## MODIFIED Requirements

### Requirement: Existing promote action releases across environments

- **Priority**: P0
- **Rationale**: Users should keep one recognizable pipeline release action while its behavior becomes a real fixed dev-to-prod operation.
- **Before**: `POST /api/v1/pipelines/{id}/promote` creates a prod-scoped copy in the same database.
- **After**: The endpoint requires an accepted plan digest and idempotency key, coordinates the authenticated prod import, and returns the target promotion/version result.

#### Scenario: Existing promote action is used

- **Given** an admin has accepted a ready promotion plan
- **When** the UI invokes the existing promote action with mappings and an idempotency key
- **Then** the action executes the fixed dev-to-prod import rather than creating a same-database scope copy

#### Scenario: Promote is called without accepted plan

- **Given** an admin has not supplied a current accepted plan digest or idempotency key
- **When** the promote endpoint is called
- **Then** it returns `400` or `409` and creates neither a local prod copy nor a remote prod version
