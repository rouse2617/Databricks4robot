# CYB-1684 Disable Deploy Data Bindings

## Summary

Fix pipeline deployment so `SkipOutputArtifacts` disables output parameter data bindings end to end. This prevents old or imported pipeline JSON containing `args[].from` / `env[].from` from generating Argo DAG arguments that reference skipped upstream outputs.

## Problem

After ordinary canvas edges were changed to dependency-only semantics, dev can still fail with:

```text
templates.dag.tasks.<step> failed to resolve {{tasks.<upstream>.outputs.parameters.output}}
```

The deploy path sets `SkipOutputArtifacts: true`, which removes upstream output parameter declarations. However, explicit `arg.From` / `env.From` references still generate downstream task arguments. That creates an invalid Argo workflow when saved/imported pipeline JSON still contains old data-binding fields.

## Goals

- In deploy/output-skipping mode, do not create Argo input parameters or DAG task arguments from `arg.From` / `env.From`.
- Keep dependency ordering from canvas edges unchanged.
- Preserve explicit data-binding behavior for non-output-skipping transpiler use.

## Non-Goals

- Add a new UI data-binding feature.
- Change Argo workflow submission APIs.
- Migrate existing saved pipeline JSON.
