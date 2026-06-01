## 2026-06-01 - Skip deploy before PR
- **Context**: The frontend runtime change normally requires Cloud Run dev deploy plus Chrome DevTools MCP verification before commit and push.
- **Decision**: Per the user's explicit instruction, skip deploy for now and open the PR to `dev` with local verification evidence.
- **Alternatives**: Finish Cloud Run dev deployment and dev-browser verification before opening the PR.
- **Rationale**: The current user instruction has higher precedence and requested a PR first; deploy verification remains documented as not run.
