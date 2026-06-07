# Decisions — CYB-1797

## 2026-06-07 — Use composite cursor design
- **Context**: The current asset-node list cursor filters by row ID while results are ordered by business fields.
- **Decision**: Use opaque composite cursors tied to the requested order mode.
- **Alternatives**: Force all cursor pagination to `id ASC`, or switch to offset pagination.
- **Rationale**: Composite cursors preserve existing sort behavior while fixing skipped and duplicate rows across pages.
