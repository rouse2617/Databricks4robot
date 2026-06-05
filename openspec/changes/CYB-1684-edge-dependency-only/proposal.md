# Proposal — CYB-1684

## Why

Pipeline canvas edges currently act as data bindings by default. A normal line between two steps generates an Argo expression such as `{{tasks.step-a.outputs.parameters.output}}`, so deployment fails when the upstream step does not produce that output file. Users expect a simple line to express execution order unless they explicitly configure data passing.

## What Changes

- Treat ordinary canvas edges as DAG execution dependencies.
- Stop generating task input parameter references from every edge by default.
- Keep component input/output declarations available for future explicit data-binding behavior.

## Impact

- **Affected code**: `backend/internal/transpiler/transpiler.go`, focused transpiler tests.
- **New APIs**: none.
- **Dependencies**: none.

## Scope

- **In scope**: Argo workflow generation from pipeline edges.
- **Out of scope**: new UI for explicit data mapping, new edge types, API contract changes.
