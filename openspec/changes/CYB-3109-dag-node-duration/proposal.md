# CYB-3109 — DAG node: per-step duration + absolute start/finish

## Problem

In the run-detail DAG, each node showed only "成功 · X 小时前" (which is the
finish time relative to now), and the hover tooltip showed only the absolute start
time. There was no at-a-glance per-step duration.

## What

- `WorkflowDagNode`: show the per-step run duration on the card next to the
  relative time (e.g. `5m 12s · 13 小时前`), reusing `formatDuration(startedAt,
  finishedAt)` from `lib/workflow-utils` (returns "-" when there's no valid span,
  so nodes that never ran show just the relative time / nothing).
- Enrich the hover tooltip from start-time-only to three lines: 开始 / 完成 / 耗时
  (absolute timestamps + duration).

## Scope

- Frontend only (`WorkflowDagNode.tsx`), additive display. Reuses the existing
  shared `formatDuration`; no new dependency, no data change.

## Notes

- Duration is the node's own run time (`finished − started`); while running it
  shows time-so-far. This is the step-level counterpart of CYB-3108's run-level
  "real run duration".
