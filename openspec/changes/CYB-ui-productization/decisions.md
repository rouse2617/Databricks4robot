# Decisions — CYB-ui-productization

## 2026-06-16

- User approved full push in worktree `feat/CYB-ui-productization` without separate OpenSpec checkpoint pause.
- Vocabulary rule: 业务概念中文，技术标识（Commit、Tag、PG/ES、Outbox）保留英文并可选 tooltip。
- Status colors: cancelled → default (grey); warning reserved for blocked/degraded/retrying.
- No new i18n library — central `productVocabulary.ts` + extend existing `statusLabels.ts`.
