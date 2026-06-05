# CYB-1686 — Pipeline UI follow-up

## Problem

Pipeline authoring and execution-list views expose two UI issues:

- Clicking components into the canvas uses a small diagonal offset, so multiple nodes overlap and look collapsed.
- Execution rows mix workflow/run identity and pipeline template identity in the name column, making it hard to read `pipeline id : version`.

## Scope

- Space auto-added pipeline nodes on a stable grid while preserving explicit drag/drop positions.
- Give pipeline nodes stable dimensions so port chips do not resize the card unpredictably.
- Add a dedicated execution-list `流水线` column that displays `templateId : version` and keeps workflow/run identity in `名称`.

## Out of scope

- Backend schema/API changes.
- Pipeline run data model changes.
- Mobile layout redesign.
