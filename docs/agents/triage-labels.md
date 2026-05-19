# Triage Labels

The [mattpocock/skills](https://github.com/mattpocock/skills) `triage` skill uses five canonical triage roles. Map them to **Linear labels** (create in Linear if missing).

| Role in skill | Linear label (default) | Meaning |
|---------------|------------------------|---------|
| `needs-triage` | `needs-triage` | Maintainer needs to evaluate |
| `needs-info` | `needs-info` | Waiting on reporter |
| `ready-for-agent` | `ready-for-agent` | Fully specified, AFK-ready for an agent |
| `ready-for-human` | `ready-for-human` | Needs human implementation |
| `wontfix` | `wontfix` | Will not be actioned |

When a skill mentions a role (e.g. “apply the AFK-ready triage label”), use the **Linear label** from the right column via `save_issue` → `labels`.

Edit this table if your Linear workspace uses different label names.
