# Issue tracker: Linear

Issues and PRDs for this repo live in **Linear**. Team process requires a **Linear Issue** for every deliverable change (see `CLAUDE.md` → Team process).

Use the **Linear MCP** (`user-linear`) in Cursor, or the Linear web UI / `gh` is **not** used for issue tracking.

## Conventions

- **Create an issue**: `save_issue` with `title`, `team`, and `description` (Markdown). Link the GitHub PR in the description when one exists.
- **Read an issue**: `get_issue` with issue ID or identifier (e.g. `CYB-978`).
- **Issue ID prefix**: `CYB-` (Cyberorigin / DataBrew team). Branch and OpenSpec dirs use the same id: `fix/CYB-978-slug`.
- **List / search**: `list_issues` with filters (team, state, assignee, labels).
- **Update**: `save_issue` with `id` plus fields to change (`state`, `labels`, `description`, etc.).
- **Comments**: `save_comment` / `list_comments` on the issue.

Infer team and project from existing issues in the repo’s Linear workspace when unsure; ask the user if ambiguous.

## When a skill says "publish to the issue tracker"

Create or update a **Linear issue** with enough context for a reviewer or AFK agent: goal, approach, acceptance criteria, and links to related ADRs or `docs/review/*` docs.

## When a skill says "fetch the relevant ticket"

Use `get_issue` (and `list_comments` if thread context matters).

## GitHub

Pull requests and code review stay on **GitHub** (`CyberOrigin2077/cyber-databrew`). Linear Issues trace *what* to build; GitHub PRs trace *how* it landed.
