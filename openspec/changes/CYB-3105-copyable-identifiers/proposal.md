# CYB-3105 — One-click copy for identifier fields (batch detail / executions)

## Problem

On the batch job detail page (and the shared executions list), identifier fields
like the run/subtask 名称, 资产 asset_id, 批次 ID, and 模板 can only be copied by
manually selecting the text and pressing Ctrl+C. The run ID and 所属用户 already
have one-click copy; the rest do not, which is inconsistent and slow.

## What

Add antd `Typography.Text copyable` one-click copy icons to the identifier fields,
matching the existing copy affordance on the run ID / owner:

- Shared `WorkflowExecutionList` (batch subtask table + main executions list):
  **名称**, **资产 / asset_id** (both the canonical `AssetIdLink` branch and the
  non-canonical code-text branch).
- `BatchJobDetailPage` header: **批次 ID**, **模板**.

## Scope

- Frontend only. Purely additive `copyable` props — no behavior change to existing
  data flow, no new dependency (antd built-in).
- The subtask table is a shared component, so the 名称 copy also appears on the main
  「执行记录」page (intended, consistent).

## Out of scope (可后续)

- The 批次运行 navigation chips (they are jump buttons; their asset_id is copyable in
  the subtask table below).
- The 命名空间 tag and the 标签-column asset_id tag (the 资产 column already exposes
  the same asset_id with copy).
