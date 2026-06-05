# CYB-1693 Execution List Layout

## Why

The Pipeline execution records tab can clip content at the left edge and lose the right-side operation column on desktop-width screens. The wide table and filter toolbar need to stay inside a stable content container and scroll horizontally within the table area when needed.

## What Changes

- Keep Pipeline page content, tabs, toolbar, filters, and table inside a padded full-width content container.
- Give the execution table an internal horizontal scroll surface with stable min width instead of letting it push the page off-screen.
- Keep important table columns usable, especially name and actions.
- Make failed-row diagnosis text a clearer detail entry.
- Add tooltip context for disabled bulk delete.

## Impact

Frontend layout and interaction polish only. No backend, API, SDK, pipeline contract, or transpiler changes.
