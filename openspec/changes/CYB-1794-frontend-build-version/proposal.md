# Proposal — CYB-1794

## Why

QA and developers need an always-visible frontend build marker so screenshots, bug reports, and dev regression runs can be tied to the exact deployed UI version.

## What Changes

### New Capabilities
- The app shell shows a low-noise frontend version marker in the lower corner of the UI.
- The marker exposes short commit/build reference, package version, and environment label when available.
- The marker provides full build details for debugging without requiring a backend API call.

### Modified Capabilities
- Frontend build metadata becomes globally visible instead of only being shown on selected pages.

## Impact

- **Affected code**: `Frontend/src/components/AppLayout.tsx`, `Frontend/src/lib/appVersion.ts`, frontend styles, Vite build metadata wiring if needed.
- **New APIs**: None.
- **Dependencies**: None.

## Scope

- **In scope**: Global version marker, build metadata formatting, desktop/mobile non-overlap behavior, dev deploy and Chrome DevTools MCP verification.
- **Out of scope**: Backend version endpoint, release management changes, changing package versioning policy.

## Success Criteria

- [ ] The frontend shows a lower-corner build/version marker on normal app pages.
- [ ] The marker includes package version and a short commit/build reference when available.
- [ ] The marker offers full details on hover/click for debugging.
- [ ] The marker stays visually secondary and does not block primary app controls.
- [ ] The deployed dev Worker is verified with Chrome DevTools MCP.

## Goals (SLO)

- **Latency**: No network request is added for version display.
- **Quality**: Existing app shell rendering and navigation continue to work.
