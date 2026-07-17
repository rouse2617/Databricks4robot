# Unify the frontend HTTP transport onto one axios client

## Why

The frontend carries two independent HTTP client implementations:

- `src/api/client.ts` — an axios instance (`apiClient`) with baseURL, dev-token
  injection, `withCredentials`, and a global `401 → logout` response
  interceptor. ~17 references.
- `src/api/pipelineClient.ts` — a hand-rolled `fetch` wrapper `request<T>()`
  plus an `ApiError` class. ~12 references (pipeline / run / workflow / batch /
  component APIs).

Two transports means duplicated auth wiring (the dev-token header logic is
copy-pasted), divergent request defaults, and two places to change whenever
auth/credentials/headers evolve.

## What changes

Reimplement `pipelineClient.request()` on top of the shared `apiClient`
(axios). `request()` keeps its **exact public contract** — same signature,
still throws `ApiError(status, code, message)` parsed from the backend
envelope, still resolves `204` / non-JSON bodies to `undefined`. No call site
changes; no HTTP API/behavior contract change.

Net effect: one transport owns baseURL, dev-token injection, and credentials;
`pipelineClient` drops its duplicated dev-token logic.

## Scope

- Frontend only. No backend, no API surface, no openapi/sdk delta.
- Behavior-preserving (see `decisions.md` for the preserved-contract checklist).
- Out of scope: migrating the 29 call sites to a single ergonomic API, and
  unifying `401` handling across both client styles (pipeline calls keep their
  current non-logout behavior via `skipAuthRedirect`). Those are follow-ups.
