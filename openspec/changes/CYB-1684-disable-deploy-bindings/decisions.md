# CYB-1684 Decisions

## 2026-06-05 - Submit PR before dev UI verification

- **Context**: The current Chrome DevTools MCP session is not authenticated to the Cloudflare Access-protected dev UI, so the agent cannot reproduce the user's exact canvas deployment flow in browser.
- **Decision**: Submit the backend fix with targeted regression coverage, full backend tests, pre-commit, and quick local CI. Dev deployment/UI verification remains pending after the PR is merged and deployed.
- **Alternatives**: Wait for an authenticated browser session before opening the PR.
- **Rationale**: The Argo error is explained by `SkipOutputArtifacts` skipping upstream output declarations while explicit `arg.From` / `env.From` still generated downstream task arguments. The regression test covers that exact invalid-spec path.
