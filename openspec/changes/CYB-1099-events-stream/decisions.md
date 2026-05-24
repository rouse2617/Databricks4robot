## 2026-05-23 — OpenSpec approval and SDK scope
- **Context**: CYB-1099 adds a public SSE endpoint, which normally triggers SDK consideration.
- **Decision**: Proceed after user confirmed "可以,开始弄吧"; keep SDK and Frontend out of scope for this issue while still syncing OpenAPI, api-guide, smoke coverage, and behavior spec.
- **Alternatives**: Add SDK streaming helper and Frontend consumption in the same branch.
- **Rationale**: The issue acceptance is backend SSE stream behavior; no current UI consumes the endpoint, and curl/EventSource clients can use the documented SSE contract directly.
