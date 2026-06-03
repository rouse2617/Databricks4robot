# CYB-1623 Clean Pipeline Console Warnings

## Problem

Pipeline pages currently emit framework warnings during normal use, including AntD compatibility/usage warnings and React Flow node type stability warnings. These warnings make browser verification noisy and can hide real regressions.

## Scope

- Remove or prevent recurring AntD React 19 compatibility warning where possible in the app entry.
- Replace AntD static message usage in affected pipeline flows with context-aware usage.
- Stabilize React Flow `nodeTypes` / `edgeTypes` references.
- Remove invalid AntD Spin `tip` usage in nested/non-fullscreen cases.

## Out of Scope

- Backend API changes.
- UI redesign.
- Dependency upgrades unless required for warning cleanup.
