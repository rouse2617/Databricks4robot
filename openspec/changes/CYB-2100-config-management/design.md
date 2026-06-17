# Design — CYB-2100

## Architecture Context
- **Constraints**: React 19 + Ant Design front end; Go/Gin backend; existing `/registry` page is already a read-only dictionary hub; component-release work is already separated in `CYB-1824` and must not be re-coupled here.
- **Goals**: make configuration management first-class, keep reference registries readable, keep the `/registry` navigation stable, support config version history, preserve immutable runtime snapshots for pipeline execution, and support deploy-time config file mounting into container directories.
- **Non-Goals**: component-owned config subresources, editable low-level image/build metadata in the config center, replacing the current registry dictionaries, or making the component editor own config content.

## Current Problems With `/registry`
- The page presents four unrelated dictionaries side by side, so users cannot tell what is reference data and what is operational configuration.
- Lifecycle states are shown under the same umbrella even though they are validation vocabulary, not a user-managed registry.
- There is no first-class place for configs, so any future config UX would have to be hidden inside component or pipeline pages.
- The current mental model makes it easy to search in the wrong place, especially once component releases and configs are both in play.
- The current page has no version-management surface for config files, so users cannot inspect current versus historical config versions.

## Affected Modules
- `Frontend/src/pages/RegistryCenterPage.tsx` — split the current hub into reference data and config management sections, including config version history
- `Frontend/src/api/registry.ts` — keep existing read-only registry clients and add config client entrypoints
- `Frontend/src/App.tsx` / `Frontend/src/components/AppLayout.tsx` — preserve the `/registry` route and menu entry, but clarify naming and selected-state behavior
- `Frontend/src/pages/PipelinePage.tsx` / `Frontend/src/components/pipeline/*` — let node configuration pick a config reference and snapshot
- `Frontend/src/pages/PipelinePage.tsx` / `Frontend/src/components/pipeline/*` — let deploy-time node configuration pick a user-owned config reference, snapshot it, and expose the mount path contract
- `backend/internal/handlers/registry` — keep existing registry endpoints read-only and add config endpoints
- `backend/internal/models` / `backend/internal/postgres` — store independent config records and snapshots
- `backend/routes/routes.go` — route wiring for registry and config resources
- `api/openapi.yaml` / `docs/review/api-guide.md` — public contract sync for any new config endpoints

## Architecture Decisions

### Decision 1: Keep `/registry` as the entry point, but split the content model
- **Approach**: retain the existing route and nav entry, then render two distinct areas: read-only reference registries and configuration management.
- **Alternative**: introduce a brand-new `/configs` entry and leave `/registry` as-is.
- **Rationale**: the current route already owns the governance mental model; keeping it stable avoids duplicate navigation while still allowing a clear product split.
- **Trade-off**: the page becomes broader, so the layout must be explicit about the two different resource classes.

### Decision 2: Configuration is a standalone resource library
- **Approach**: model configs as first-class records with their own identity, ownership, tags, description, file payload, and content history.
- **Alternative**: make configs children of components or releases.
- **Rationale**: the user requirement is explicit that configs must not be bound to components; keeping them independent avoids shared lifecycle and permission coupling.
- **Trade-off**: the UI and backend need an extra resource type, plus a separate search surface.

### Decision 2a: Config records expose version history in the config center
- **Approach**: show each config with a current version and expandable immutable version history.
- **Alternative**: only show one mutable config row and hide history in an audit page.
- **Rationale**: users need to reason about which config version is selectable or rollbackable without entering deploy or component pages.
- **Trade-off**: the config list has more density, so version details should stay collapsed by default.

### Decision 3: Deploy-time config selection is ownership-scoped
- **Approach**: the deploy picker filters to configs owned by the current user or otherwise shared to them, and a click selects exactly one config to attach to the deploy.
- **Alternative**: expose the full config library to every deploy context and rely on manual discipline.
- **Rationale**: the user wants to attach only their own configs, which keeps the deploy UI simple and prevents cross-user leakage.
- **Trade-off**: ownership and sharing rules must be explicit in the backend query contract.

### Decision 4: Pipeline nodes store a config reference plus a snapshot
- **Approach**: persist `configId` on the node and copy an immutable config snapshot into the saved pipeline state.
- **Alternative**: only store the config ID and resolve live config at execution time.
- **Rationale**: execution history must not drift when a config changes later.
- **Trade-off**: snapshot storage adds duplication, but it guarantees reproducibility and auditability.

### Decision 5: Existing registry data stays read-only
- **Approach**: algo/tag/metric/lifecycle registries remain lookup dictionaries with no editing affordances in this slice.
- **Alternative**: reuse those tabs as the editing surface for configs.
- **Rationale**: the current registries already serve validation and discovery; mixing them with config CRUD would recreate the same ambiguity we are trying to remove.
- **Trade-off**: the new config center has to earn its own space instead of piggybacking on an existing table.

### Decision 6: Component editing defines a mount contract, not config ownership
- **Approach**: component editing can expose a mount path / directory mapping for a config file, so deploy-time selection knows where to project the chosen file in the container.
- **Alternative**: allow components to directly point at a specific config record.
- **Rationale**: the user wants to map a config file into a container directory without making the component own that config.
- **Trade-off**: component runtime metadata needs a clear distinction between “consumes config here” and “owns config”.

## Data Flow

```text
User opens /registry
        ↓
Reference registries load read-only dictionaries
        ↓
Config management loads standalone user-owned config records
        ↓
User creates/edits config file version
        ↓
Backend stores config identity + immutable versions
        ↓
Deploy-time picker shows only own/shared configs
        ↓
User clicks one config to attach
        ↓
Component mount contract provides container directory path
        ↓
Pipeline authoring selects configId
        ↓
Saved pipeline node stores configId + immutable config snapshot
```

## Data Model Changes
- **Table**: `pipeline_configs`
- **Change**: standalone config identity, name, description, tags, owner, file payload metadata, current status, timestamps
- **Table**: `pipeline_config_versions`
- **Change**: immutable content snapshots, version number, version status, change author, creation time, digest/summary metadata
- **Migration**: a new approved migration under `backend/migrations/` when implementation starts
- **Runtime field**: component config mount path / container directory mapping stored as part of component runtime metadata or template snapshot, not as config ownership

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| Users confuse read-only registries with editable configs | Mis-edit or missearch | Separate copy, layout, and empty states in the UI |
| Config snapshots drift from live config if omitted | Historical runs become non-reproducible | Persist snapshots at save time and verify in tests |
| Config lifecycle becomes too coupled to components | Reintroduces the old problem | Keep config APIs and component APIs independent |
| `/registry` grows too large | Navigation friction | Use tabs/sections with clear labels and count badges |
| Ownership filtering misses shared configs | Users cannot deploy expected configs | Make deploy list query contract explicit and test owner/shared visibility |
| Mount path mapping is ambiguous | Config file lands in wrong container location | Normalize path semantics and validate against the component runtime contract |
