# Decisions — CYB-1538 Pipeline Run UX Output

## 2026-06-02 — Explicit no-asset run payload
- **Context**: Omitting `asset_ids` lets downstream persistence see a nil slice.
- **Decision**: Frontend run actions send `asset_ids: []` and the API helper uses `/pipeline-runs/template/:id`.
- **Rationale**: No-asset runs are a valid product flow and need an explicit representation.

## 2026-06-02 — Output files are conditional
- **Context**: UI-created nodes often have an output port even when the container does not write an output file.
- **Decision**: The transpiler declares file output parameters only when the output is consumed by another node or the component command/source references `/tmp/outputs/<name>`.
- **Rationale**: This prevents unused default ports from turning successful containers into Argo failures while preserving DAG dataflow behavior.
