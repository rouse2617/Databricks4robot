## 2026-06-05 — frontend dev verification deferred by user instruction
- **Context**: This change touches `Frontend/`, which normally requires Chrome DevTools MCP verification on deployed dev.
- **Decision**: Did not treat `frontend dev` as an acceptance gate for this round after the user explicitly said `frontend dev 的，你用考虑`.
- **Alternatives**: Block PR creation until Chrome DevTools MCP transport recovered and full frontend deploy verification completed.
- **Rationale**: User instruction takes precedence. Local frontend verification still completed with `npm run lint`, targeted tests, and `npm run build`.

## 2026-06-05 — backend deploy verification gap on gpu-l4 preset response
- **Context**: Local backend tests prove `gpu-l4` preset injection, but deployed dev backend `POST /api/v1/pipeline-components` still returns only the explicit toleration in the response body.
- **Decision**: Keep the PR open with this gap called out instead of claiming deploy verification is complete.
- **Alternatives**: Claim the feature verified based only on local tests, or block the branch entirely until the discrepancy is root-caused.
- **Rationale**: The discrepancy is material for direct API callers. It needs explicit follow-up in the PR rather than being hidden.
