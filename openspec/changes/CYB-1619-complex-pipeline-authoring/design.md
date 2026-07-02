# Design — CYB-1619 Complex Pipeline Authoring

## Product Model

### Component Ports

Components should expose `inputPorts` and `outputPorts` as editable product fields, not hidden implementation details.

For each port:

- `name`: stable port identifier used in edges and Argo parameter names.
- `type`: simple semantic type for now, such as `asset`, `string`, `json`, or `artifact`.
- `description`: optional user-facing explanation.

P0 can persist these through the existing component payload and pipeline node definitions if current API shapes already support them. If persistence gaps are found, API contract sync must be done in the same PR.

### Output Contract

For shell/container components, every output port maps to `/tmp/outputs/<portName>` when consumed.

The UI should surface this near the output port list and script/args editor. It should not force users to write outputs for unconsumed ports, but examples and warnings should make the contract clear.

### Fan-In Authoring

The designer should treat a target input as a single-bind destination:

- When connecting to a node, the target input port should be visible/selectable.
- If a target input already has an upstream edge, the UI should warn or prevent connecting another edge to that same input.
- For join nodes, users should be able to define multiple input ports (`left`, `right`, `metadata`, etc.) and connect each upstream to a different port.

P0 does not need a full custom edge creation wizard if a simpler interaction can meet the acceptance criteria.

### Standard Examples

Examples should be available from the pipeline design page or pipeline management surface as reusable starting points. P0 can implement them as frontend-provided templates if backend persistence of built-in templates is not yet ready.

Suggested examples:

1. Sequential chain
   - `prepare` -> `transform` -> `finalize`
   - Each step sleeps briefly and emits/consumes output.

2. Fan-out/fan-in join
   - `source` -> `left-branch` and `right-branch` -> `join`
   - Join has distinct inputs.

3. Observation workflow
   - 4-6 nodes with `sleep 10` style work and distinct log output.
   - Useful for validating status, logs, duration, and cost displays.

## UX Notes

- Keep operational UI compact and work-focused.
- Avoid large marketing-style cards or decorative layouts.
- Prefer tables/forms/inline helper text for component port editing.
- Use clear labels such as `输入端口`, `输出端口`, `输出文件`, `Join 输入`.

## Verification Strategy

- Unit tests for port serialization and fan-in validation helpers.
- Frontend tests for component port form behavior and sample loading.
- Backend/transpiler tests if new persistence or transformation behavior is introduced.
- Browser smoke:
  - Create component with multiple outputs.
  - Build fan-out/fan-in pipeline with distinct join inputs.
  - Load each standard example and save/run at least one.
