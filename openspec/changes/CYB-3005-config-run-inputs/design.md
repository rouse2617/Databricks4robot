# Design - CYB-3005 Config Run Inputs

## Approach

Use the existing `pipeline_runs.pipeline_json` compatibility payload as a lightweight materialized projection source.

At Run creation time, after deploy-level and node-level configs are resolved and after projection keys are assigned, the usecase writes a sanitized `_run_config_inputs` array into `PipelineJSON`. Each entry contains metadata only:

- scope: `global` or `node`
- nodeId for node-level bindings
- mode/source
- configId and version for saved configs
- fileName, mountPath, targetFilename
- projectionKey when available
- contentHash as `sha256:<hex>`

`ListRunInputs` then prefers `_run_config_inputs` for config rows. If that array is absent, it keeps the existing fallback that scans node `runtimeConfig` fields so old Runs continue to show node config inputs.

## API Shape

`RunInput` gains optional `contentHash` and `projectionKey` fields. Existing fields and response envelopes remain unchanged.

## Safety

Raw config content is never stored in the RunInput snapshot. Inline/upload content may still exist in historical request payloads, but the new materialized input projection excludes it.

## Rollback

The materialized `_run_config_inputs` field is additive in `PipelineJSON`. Rolling back code ignores the extra field.
