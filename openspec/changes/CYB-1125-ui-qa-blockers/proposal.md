# CYB-1125 — UI QA blockers

## Summary

Fix frontend QA blockers found during Chrome MCP full-site verification on the deployed dev frontend.

## Problem

The dev UI is broadly reachable, but targeted pages still expose user-visible blockers:

- `/settings` attempts admin search reindex/audit calls that can return 404 on the deployed environment and surface console/resource errors.
- The settings reindex confirmation cancel path leaves the modal open.
- Asset detail delivery history can stay loading while repeatedly paginating through global deliveries.
- A few request-burst follow-ups were observed during full-site QA.

## Scope

- Frontend-first fixes for the settings admin widgets and asset detail delivery history tab.
- Low-risk local request burst fixes where the existing component seam is clear.
- No backend route or persistence change unless code investigation proves the frontend cannot safely gate around deployed capabilities.

## Out of scope

- Broad redesign of settings/admin search operations.
- Metrics virtualization or pagination redesign beyond a safe default/limit if trivial.
- Backend admin API implementation changes unless needed to satisfy an already documented contract.

## Verification

- Frontend unit tests for the changed components/hooks where the codebase has a seam.
- `cd Frontend && npm run lint && npm run test -- --run && npm run build` before deploy verification.
- After frontend dev deploy, Chrome DevTools MCP targeted rerun for `/settings` and `/assets/:id` delivery history, with console/network checks.
