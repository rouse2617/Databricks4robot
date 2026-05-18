---
name: linear-progress-update
description: Write and post standardized Linear progress updates with parent/sub-issue structure and commit traceability. Use when user asks to update Linear progress, asks for "模板", wants summary in parent issue, or wants detailed delivery notes in sub-issue.
---

# Linear Progress Update

Use this skill to keep Linear updates consistent:
- parent issue: short summary + pointer
- sub-issue: detailed delivery record

## Quick rules

1. Detailed progress goes to sub-issue (delivery facts, validation, metrics, commit).
2. Parent issue stays concise (status, key result, link to sub-issue).
3. Always include commit hash when code changed.
4. Keep language plain; avoid over-technical wording unless user asks.

## Workflow

### 1) Confirm issue structure
- Parent issue exists (for milestone/epic tracking).
- Sub-issue exists for this delivery.
- If no sub-issue, create one and set `parentId` to parent issue.

### 2) Post detailed update on sub-issue
Use the "Sub-Issue Detailed Update" template from `TEMPLATE.md`.

Must include:
- Problem
- What changed
- Measured result (before/after)
- Validation status
- Commit info

### 3) Post brief note on parent issue
Use the "Parent-Issue Brief Update" template from `TEMPLATE.md`.

Must include:
- one-line status
- key metric/result
- sub-issue link
- commit hash

### 4) Keep parent clean
- Prefer one concise parent comment per delivery.
- If a detailed comment was posted to parent by mistake, replace it with a pointer-style summary.

## Style guide

- Prefer bullets over long paragraphs.
- Use concrete numbers (`13.2s -> 0.7s`, `18.7x`).
- Use neutral, direct wording:
  - "问题"
  - "动作"
  - "结果"
  - "验证"
  - "状态"

## Output contract

When done, report back:
- parent issue id/url
- sub-issue id/url
- comment posted summary (1-2 lines)
- commit hash included
