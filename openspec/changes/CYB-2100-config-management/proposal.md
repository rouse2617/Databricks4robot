# Proposal — CYB-2100

## Why
The current `/registry` page is a flat reference dashboard for algo/tag/metric/lifecycle dictionaries. It does not give user-owned configuration files a first-class home, and it blurs the line between immutable reference data, editable operational state, and deploy-time runtime wiring.

## What Changes

### New Capabilities
- `/registry` becomes a governance hub with a clearly separated configuration management area.
- Configuration records become standalone user-owned resources with independent identity, search, lifecycle handling, and version history.
- Users can register their own config files and select only their own configs during deploy.
- Pipeline nodes can reference a configuration by ID without coupling that configuration to a component or release.
- Components can declare where a chosen config file should be mounted inside the container, without owning that config.

### Modified Capabilities
- Existing registry views remain authoritative read-only dictionaries, but they stop being treated as the place to edit operational configuration.
- Component pages stay focused on components and releases, not on low-level config authoring.
- Pipeline authoring can consume config references without inventing a component-to-config binding.
- Deploy-time UX becomes a config picker plus a component mount contract, not a free-form metadata editor.

## Impact
- **Affected code**: `Frontend/src/pages/RegistryCenterPage.tsx`, `Frontend/src/api/registry.ts`, `Frontend/src/App.tsx`, `Frontend/src/components/AppLayout.tsx`, `Frontend/src/pages/PipelinePage.tsx`, `Frontend/src/components/pipeline/*`, `backend/internal/handlers/registry`, `backend/routes/routes.go`, `backend/internal/models`, `backend/internal/postgres`, `api/openapi.yaml`, `docs/review/api-guide.md`
- **New APIs**: configuration list/detail/create/update/deprecate endpoints under the public API surface, plus deploy-time config lookup filtered by ownership
- **Dependencies**: component-release work remains separate; any config snapshot logic must not depend on component identity

## Scope
- **In scope**: config resource model, `/registry` page structure, config search and inspection, config version history, config lifecycle rules, deploy-time ownership filtering, pipeline-node config references, snapshotting of selected config at save time, component mount-path contract
- **Out of scope**: binding configs to components, requiring config authoring inside component manager, replacing existing algo/tag/metric/lifecycle registries, user-editable low-level build metadata, making the component editor own config content

## Success Criteria
- [ ] Users can open `/registry` and clearly distinguish reference registries from configuration management.
- [ ] Configs can be found by name, description, tag, or owner without going through component pages.
- [ ] A config can expose multiple immutable versions and identify its current selectable version.
- [ ] A user can register a config file and later pick only their own configs during deploy.
- [ ] A pipeline node can point at `configId`, and the saved node keeps an immutable config snapshot.
- [ ] A component can declare a mount path for the selected config file, and the runtime uses that path to project the file into the container.
- [ ] Existing registry dictionaries remain read-only and do not become edit forms.
- [ ] Component and config lifecycles remain decoupled.

## Goals (SLO)
- **Latency**: config list/search p95 < 500 ms for the first 500 records
- **Concurrency**: repeated config refresh or sync actions stay idempotent
- **Quality**: config references never resolve to mutable live state after a run is saved
