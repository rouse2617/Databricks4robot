# CYB-1562 Frontend Interaction Polish

## Problem

Chrome DevTools MCP review of the local DataBrew frontend found several usable but rough interaction details across high-traffic pages:

- Dashboard surfaces raw backend/configuration errors such as `lakehouse backend not configured`.
- Asset search/filter controls expose technical syntax without enough direct examples or next-step guidance.
- Some disabled actions do not explain why they are unavailable.
- Pipeline designer wording says "drag in component" even though clicking also adds components.
- Pipeline deploy preview shows a full Argo workflow YAML first, which overwhelms normal users.
- Execution label filters expose raw Argo label keys.
- Component view mode opens a disabled form with empty values instead of readable component details.
- Some touched forms emit browser accessibility issues for missing stable `id` / `name`.

These are not blocker bugs, but they slow down normal user workflows and make the app feel less stable.

## Goals

- Improve user-facing copy and affordances on the reviewed pages.
- Prefer concise summaries and progressive disclosure over raw technical dumps.
- Keep advanced technical details available when useful.
- Fix clear UI bugs and console/deprecation warnings in touched surfaces.
- Keep changes frontend-only and scoped.

## Non-Goals

- No backend API design or implementation.
- No database migrations.
- No full redesign of Dashboard, Assets, or Pipeline.
- No mobile-specific work.
- No changes to auth, middleware, or deployment scripts.

## Scope

Frontend pages and components involved in:

- Dashboard lakehouse unavailable messaging.
- Asset search/filter and selected-action affordances.
- Pipeline designer component add wording, save feedback, and deploy preview summary.
- Workflow execution labels and related form accessibility.
- Pipeline component view/edit modal display.
