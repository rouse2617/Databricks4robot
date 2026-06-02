# Design

## Approach

Make small, targeted interaction fixes rather than broad component rewrites. Preserve existing data fetching, routing, and API contracts.

## Dashboard

When lakehouse-backed widgets fail because the backend is not configured, the UI should communicate that those metrics are temporarily unavailable while preserving any available asset/event data. Raw backend error strings can appear in expandable details or tooltip text, but not as the primary message.

## Assets

Improve the search/filter area by making the next step obvious:

- Search help should include directly usable examples.
- Add-filter shortcuts should look actionable and should select/populate the intended field where feasible.
- Disabled batch actions should explain why they are disabled.
- High-impact bulk actions such as selecting all filtered results should communicate scope clearly.

Avoid rewriting the whole asset list layout in this change; any nested button/checkbox structure that requires a deeper refactor can be recorded as follow-up if it cannot be safely fixed locally.

## Pipeline Designer and Deploy

Component cards support both click and drag. The label should reflect this: clicking adds the component, dragging remains available.

Save should provide an explicit success state without disrupting the current selection more than necessary.

Deploy preview should default to a human-readable summary:

- workflow name
- target/environment
- asset mode
- step count and component list
- images/commands at a glance

Raw YAML should remain available behind an advanced/details control.

## Executions

Argo labels should be displayed with human-friendly aliases when known, while retaining full keys in tooltip/copyable detail. The UI should avoid presenting `events.argoproj.io/...` as the main label text.

## Components

View mode should show a read-only detail layout with actual values. It should not show an empty disabled edit form. Edit/create mode can keep using the form.

## Verification

Use local frontend dev server with Chrome DevTools MCP to walk:

- `/dashboard`
- `/assets`
- `/pipeline`
- `/pipeline?tab=executions`
- `/pipeline?tab=components`

Run frontend lint, related tests where available, and build before PR.
