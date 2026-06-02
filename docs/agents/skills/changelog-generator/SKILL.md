---
name: changelog-generator
description: Automatically creates user-facing changelogs from git commits by analyzing commit history, categorizing changes, and transforming technical commits into clear, customer-friendly release notes.
---

# Changelog Generator

Transform technical git commits into polished, user-friendly changelogs.

**Core principle:** Commits are for developers; changelogs are for users. Translate technical details into customer language.

Adapted from [ComposioHQ/awesome-codex-skills](https://github.com/ComposioHQ/awesome-codex-skills/blob/master/changelog-generator/SKILL.md).

---

## When to use

- Preparing release notes for a new version
- Creating weekly or monthly product update summaries
- Documenting changes for customers
- Writing changelog entries for app store submissions
- Generating update notifications
- Creating internal release documentation
- Maintaining a public changelog or product updates page

---

## Process

### 1. Scan git history

Analyze commits from a specific time period or between versions. Use `git log` with appropriate range flags (`--since`, `--after`, tag ranges).

### 2. Categorize changes

Group commits into logical categories:

| Category | git log patterns |
|----------|-----------------|
| New Features | `feat:`, `feat(`, new capabilities |
| Improvements | `perf:`, `refactor:`, `style:`, UX polish |
| Bug Fixes | `fix:`, `hotfix:`, `fix(` |
| Breaking Changes | `BREAKING CHANGE`, breaking in body |
| Security | `security`, `vuln`, `auth` changes |
| Docs / Infra | `docs:`, `ci:`, `chore:`, `build:` |

### 3. Translate technical → user-friendly

- Replace internal jargon with user-facing language
- Describe what the user can now do, not what was changed
- Remove commit hashes, file paths, and internal identifiers
- Expand abbreviations; use full, readable sentences

### 4. Filter noise

Exclude commits that are invisible to users:
- Internal refactoring with no behavior change
- Test-only changes (`test:`, test fixtures)
- CI/CD pipeline adjustments
- Dependency bumps with no user impact
- Merge commits

### 5. Format professionally

Output structured markdown:

```markdown
# Updates — <Version or Date Range>

## New Features

- **Feature Name**: What users can now do. (one sentence per feature)

## Improvements

- **Area**: What's better and why it matters. (one sentence)

## Bug Fixes

- Fixed issue where <symptom>. (one sentence)

## Breaking Changes (if any)

- **Change**: What changed and migration path.
```

### 6. Apply brand voice

If a `CHANGELOG_STYLE.md` exists at the repo root, follow its tone, formatting, and category conventions. Otherwise default to:
- Plain, friendly language (no marketing fluff)
- Active voice
- One sentence per item
- No emojis unless the project convention uses them

---

## Usage patterns

| User says | Action |
|-----------|--------|
| "Create a changelog from commits since last release" | `git describe --tags --abbrev=0` → range from that tag to HEAD |
| "Generate changelog for the past week" | `git log --since="1 week ago"` |
| "Create release notes for version 2.5.0" | `git log v2.4.0..v2.5.0` (or last tag to specified tag) |
| "Create a changelog between March 1 and March 15" | `git log --since="2026-03-01" --until="2026-03-15"` |

---

## Integration with cyber-databrew workflow

- Run from the repository root
- Use `git log` with conventional commit prefixes for categorization
- If the task is part of a CYB issue, save the changelog to `openspec/changes/CYB-{id}-{slug}/changelog.md`
- For release notes, save to `docs/review/release-notes/` or the project root `CHANGELOG.md`
- Do NOT commit changelog drafts without user review — always present the draft first and ask for edits

---

## Related

- [`spec-writing-skill.md`](../../spec-writing-skill.md) — OpenSpec change artifacts that may reference changelog entries
- [`deploy-verification.md`](../../deploy-verification.md) — verify behavior described in changelog actually works on dev
