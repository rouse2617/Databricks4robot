## 2026-06-03 — Skip dev deploy before PR
- **Context**: This change touches `Frontend/` runtime code. The standard gate asks for Cloud Run dev deploy and MCP verification before commit.
- **Decision**: User explicitly instructed to skip deploy for now and push a PR to `dev` first.
- **Alternatives**: Continue local Docker build, push image, deploy frontend dev, then ask for commit approval.
- **Rationale**: The latest user instruction takes precedence for this handoff. Local lint, targeted tests, build, and local Chrome MCP verification were completed and recorded in `tasks.md` / Linear.
