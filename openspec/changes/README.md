# OpenSpec active changes

Each runtime PR links a directory: `openspec/changes/CYB-{id}-{slug}/`

## Required files

| File | Bug | Feature |
|------|-----|---------|
| `proposal.md` | yes | yes |
| `tasks.md` | yes | yes |
| `specs/<module>/spec.md` | yes (delta) | yes |
| `design.md` | optional | **required** |
| `context-files.md` | recommended | recommended |
| `decisions.md` | when needed | when needed |

See [`docs/agents/spec-driven-workflow.md`](../../docs/agents/spec-driven-workflow.md).

**`context-files.md`** lists repo paths agents must read before coding (curated context). One path per line; optional `# reason`. Alternative: a `## Context files` section inside `tasks.md` with the same paths.

---

## `proposal.md` template (copy into new changes)

```markdown
# Proposal — CYB-{id}

## Why
{一句话：用户痛点 / 业务需求 / 线上问题}

## What Changes

### New Capabilities
- {能力 1：用模块名定位，如 asset-management}
- {能力 2}

### Modified Capabilities
- {被修改的已有能力}

## Impact
- **Affected code**: `backend/internal/{package}`, `Frontend/src/components/{path}`
- **New APIs**: {如有，列出路径和方法}
- **Dependencies**: {新增或变更的依赖}

## Scope
- **In scope**: {明确包含的}
- **Out of scope**: {明确不做的}

## Success Criteria
- [ ] {可验证的验收条件}
```

---

## `context-files.md` template (copy into new changes)

```markdown
# Context files — CYB-{id}

# One repo-relative path per line. Read before implementation.
# Optional trailing comment after #

docs/agents/deploy-verification.md  # §6 regression scope
Frontend/src/hooks/assets/useAssetsDiscoveryReducer.ts
backend/internal/handlers/asset/handler.go
```

---

## `tasks.md` template (copy into new changes)

```markdown
# Tasks — CYB-{id}

## Context files
<!-- Optional if you use context-files.md instead -->
- `path/to/file` — why read

## Implementation
- [ ] [backend] …
- [ ] [Frontend] …

## API contract sync (mandatory if HTTP API added/changed — same PR)
See [`docs/agents/AI-RULES.md` § API contract sync](../../docs/agents/AI-RULES.md#api-contract-sync-mandatory).

- [ ] `api/openapi.yaml` — paths + schemas
- [ ] `docs/review/api-guide.md` — curl + errors
- [ ] `sdk/src/asset_sdk/` + `client.py` (+ `sdk/tests/unit/` if SDK touched)
- [ ] `scripts/api-guide-smoke.sh` or `scripts/smoke-<feature>-dev.sh`
- [ ] `openspec/changes/CYB-{id}-*/specs/*/spec.md` — behavior delta
- [ ] [Frontend] `src/api/` or hooks — only if UI calls the API

## Local verification (Tier S/M/L per AI-RULES)
- [ ] …

## Deploy verification (before commit — runtime only)
- [ ] Build + push with **git SHA tag** and `cloudrun-dev-latest` (see `docs/agents/deploy-before-commit.md`)
- [ ] Deploy using `IMAGE=…:<sha>`; record **image tag**, **revision**, dev URL in this file or PR

### Frontend（仅当本 change 修改 `Frontend/` 代码 — 必填）
- [ ] Chrome DevTools MCP on dev: targeted path per proposal/tasks
- [ ] §6.1 regression pages: __/11 (or reduced set per deploy-verification.md)
- [ ] Screenshot: `deploy-verify-<scenario>.png` in this change dir
- [ ] Console: no new errors

### Backend (if `backend/` changed)
- [ ] L1 smoke (`healthz`, core APIs) or L2 `api-guide-smoke.sh` per risk

## PR
- [ ] PR template filled; Linear CYB-{id} linked
```

---

## `specs/<module>/spec.md` delta template (copy into new changes)

```markdown
## ADDED Requirements

### Requirement: {新行为名称}
The system SHALL {描述新增的系统行为}。

**Priority**: P0 (Critical) / P1 (High) / P2 (Nice-to-have)
**Rationale**: {为什么需要这个行为}

#### Scenario: {happy-path 场景名}
- **Given** {前置条件}
- **When** {触发动作}
- **Then** {预期结果}

#### Scenario: {error-path 场景名}
- **Given** {异常前置条件}
- **When** {触发动作}
- **Then** {错误预期结果}

## MODIFIED Requirements

### Requirement: {已有行为名称}
- **Before**: The system SHALL {旧行为}。
- **After**: The system SHALL {新行为}。
- **Reason**: {为什么改}

#### Scenario: {修改后的典型场景}
- **Given** {前置条件}
- **When** {触发动作}
- **Then** {预期结果}

## REMOVED Requirements

### Requirement: {被移除的行为名称}
- **Was**: The system SHALL {被移除的行为}。
- **Reason**: {为什么移除}
```

---

## Deploy evidence screenshots

Store under the change directory, e.g. `deploy-verify-algo-status-failed.png`. Safe to commit for review traceability (no secrets in images).

---

## Quality reference

For detailed quality rules and anti-patterns, see [`docs/agents/spec-writing-skill.md`](../../docs/agents/spec-writing-skill.md).
