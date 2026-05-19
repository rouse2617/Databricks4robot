# Spec-Driven Workflow (OpenSpec)

## Overview

Every runtime change must go through an OpenSpec change before code is written. This ensures traceability from requirement to implementation.

**OpenSpec in this repo is plain Markdown under `openspec/` — no global CLI install.** Clone the repository and you already have the spec library and conventions (`openspec/config.yaml`). Agents create and edit files under `openspec/changes/` like any other docs.

**Writing quality:** When authoring change artifacts, follow [`spec-writing-skill.md`](spec-writing-skill.md) (proposal, delta, design, tasks, self-check).

## Directory structure

```
openspec/
├── config.yaml              # Project config (modules, conventions)
├── specs/                   # Current system behavior (source of truth)
│   ├── asset-management/spec.md
│   ├── delivery/spec.md
│   ├── lakehouse/spec.md
│   └── search/spec.md
└── changes/                 # Active change proposals
    └── CYB-{id}-{slug}/     # Linear id (CYB-xxx); DAT-xxx accepted by CI legacy
        ├── proposal.md      # Why + What
        ├── design.md        # How (Feature: mandatory; Bug: optional)
        ├── tasks.md         # Implementation checklist
        ├── context-files.md # Must-read paths before coding (recommended)
        ├── decisions.md     # Agent decision log (append-only; see AI-RULES.md)
        └── specs/
            └── {module}/spec.md  # Delta: ADDED/MODIFIED/REMOVED
```

## Change lifecycle

1. **Create**: Make a directory under `openspec/changes/` named `CYB-{id}-{slug}` (see [`openspec/changes/README.md`](../../openspec/changes/README.md) for `tasks.md` template)
2. **Propose**: Write `proposal.md` (goal, scope, approach)
3. **Specify**: Write spec delta showing ADDED/MODIFIED/REMOVED requirements
4. **Design** (Feature only): Write `design.md` (technical approach, architecture decisions)
5. **Plan**: Write `tasks.md` (implementation checklist with checkboxes)
6. **Implement**: Write code following the tasks
7. **Archive** (after merge): Merge delta into `openspec/specs/` in the same PR or a follow-up commit (see below). Do **not** require `openspec` CLI or `/opsx:archive`.

## Spec delta format

```markdown
## ADDED Requirements

### Requirement: {name}
The system SHALL {behavior description}.

## MODIFIED Requirements

### Requirement: {name}
- **Before**: The system SHALL {old behavior}.
- **After**: The system SHALL {new behavior}.
- **Reason**: {why this changed}

## REMOVED Requirements

### Requirement: {name}
- **Was**: The system SHALL {removed behavior}.
- **Reason**: {why removed}
```

## CI enforcement

**Scope:** `openspec-gate` and `commitlint` run only when the PR **head branch** contains a Linear id (`CYB-123` / `DAT-456`), e.g. `feat/CYB-58-*`. They **do not** run on migration or refactor branches such as `refactor/cyber-databrew-repo-migration` (exempt by design).

The `openspec-gate` workflow validates:
- PR includes an OpenSpec change-id (branch name or PR body)
- `openspec/changes/{id}/` directory exists
- Required files present (based on change type from label)
- At least 1 spec delta file exists
- **Optional:** `npx @fission-ai/openspec validate <change-id>` on touched change dirs (format/lint for deltas only — **not** required to author changes via CLI)

## 合并后归档（无 CLI，在仓库内完成）

Feature 合并后，把 `changes/` 里的需求 delta **合入** 对应模块的主 spec，然后删除或移走 change 目录：

1. 打开 `openspec/changes/CYB-xxx-.../specs/<module>/spec.md`
2. 将 `ADDED` / `MODIFIED` / `REMOVED` 条目合并进 `openspec/specs/<module>/spec.md`（含 `#### Scenario:`、Priority、Rationale — 与 change delta 同格式；基线简版将逐步替换）
3. 删除 `openspec/changes/CYB-xxx-.../`（或移到 `openspec/changes/archive/CYB-xxx-.../` 若团队希望保留历史）
4. 在同一 PR 或紧接的 `chore: archive openspec CYB-xxx` PR 中提交

Agent 可在合并后自动执行上述步骤，无需安装任何 OpenSpec 包。

## Exemptions

PRs with these labels skip the openspec-gate:
- `docs-only`
- `infra-ci-only`
- `hotfix-approved` (must backfill within T+2 business days)
