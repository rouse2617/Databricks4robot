# UI design handoff — CYB-1017

Full spec (wireframes, visual tokens, Figma file structure, interaction spec):

**[`docs/review/unified-asset-catalog/design/asset-versioning-provenance-ui.md`](../../../docs/review/unified-asset-catalog/design/asset-versioning-provenance-ui.md)**

## Shipped vs designed

| Item | CYB-1017 (shipped) | Design doc (next) |
|------|-------------------|-------------------|
| Header version switch | Ant `Select` | `VersionControl` dropdown + logical id copy |
| Version history | API only | Tab「版本与溯源」+ timeline |
| Non-current warning | — | `VersionHistoryBanner` |
| Overview `revision` | Shows legacy `version` | Use `revision` + `is_current` |

Implementation phases: **P0 polish** → **P1 tab** → **P2 diff** (see design doc §8).
