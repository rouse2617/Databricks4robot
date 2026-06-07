# Design — CYB-1794

## Architecture Context

- **Constraints**: The frontend is built with Vite and React. `vite.config.ts` already defines `__APP_VERSION__` and `__APP_BUILD_REF__`, and `Frontend/src/lib/appVersion.ts` already exposes basic version formatting.
- **Goals**: Make build identity visible globally, keep the UI unobtrusive, and avoid backend coupling.
- **Non-Goals**: Introduce a release service, add an HTTP endpoint, or change deployment topology.

## Affected Modules

- `Frontend/src/lib/appVersion.ts` — normalize build metadata and short/full label formatting.
- `Frontend/src/components/AppLayout.tsx` — render the global marker inside the authenticated app shell.
- `Frontend/src/styles/design-tokens.css` or component CSS — position and style the marker for desktop and mobile.
- `Frontend/vite.config.ts` / deploy command — use existing build metadata definitions; only adjust if the deployed build does not receive a useful reference.

## Architecture Decisions

### Decision 1: Use Vite compile-time metadata
- **Approach**: Use `__APP_VERSION__` and `__APP_BUILD_REF__` injected at build time, with `import.meta.env.MODE` as the environment label.
- **Alternative**: Fetch version from a backend endpoint.
- **Rationale**: The frontend artifact identity should describe the loaded static bundle. A backend API would identify the server, not necessarily the currently cached Worker asset bundle.
- **Trade-off**: Build commands must provide a useful `VITE_BUILD_REF`; otherwise the marker falls back to `local`.

### Decision 2: Render a compact fixed marker in the app shell
- **Approach**: Add a small fixed lower-corner marker with a compact text label and tooltip/click details.
- **Alternative**: Put the version only in Settings or Dashboard.
- **Rationale**: QA screenshots often capture arbitrary pages. A global marker removes ambiguity without requiring navigation.
- **Risk**: Fixed UI can overlap page controls on small screens.
- **Rollback**: Remove the marker component from `AppLayout`.

## Data Flow

```text
VITE_APP_VERSION / package.json version
VITE_BUILD_REF / fallback local
Vite define constants
Frontend appVersion helper
AppLayout build marker
```

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| Marker overlaps floating controls | Users may lose access to page actions | Keep marker compact, low z-index relative to modals, and position in a quiet corner with responsive offsets |
| Build ref is missing in local/dev scripts | Marker shows `local` instead of commit | Use fallback text and verify deploy command can provide `VITE_BUILD_REF` |
| Too much visual noise | Operational pages feel cluttered | Use muted colors and reveal full details only on hover/click |
