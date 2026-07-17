# Decisions

## autonomous-approved (opt-loop, no human in the loop)

This change was designed, self-reviewed, and approved autonomously under the
standing "全自主无限自迭代" authorization. Rationale: frontend-only,
behavior-preserving internal refactor with no spec/API/contract delta; the
regression surface (12 `request()` call sites, core to the resource-pool
feature) is fully covered by preserving the exact public contract + new unit
tests + dev smoke.

## D1 — Delegate `request()` to axios, rather than migrate 29 call sites

Two viable directions to "converge two clients":

- (A) Make `request()` the one wrapper and rewrite the 17 `apiClient` call
  sites onto it — large, touches unrelated features, high regression surface.
- (B) Keep both public surfaces but route `request()`'s transport through the
  shared `apiClient` — one file changed, zero call-site churn.

Chose **(B)**. It captures the actual duplication (transport + auth wiring)
with minimal blast radius, and leaves an ergonomic single-API migration as an
optional follow-up.

## D2 — Preserve behavior exactly (no opportunistic "improvements")

Investigation established the real error contract: `ApiError extends Error`,
and the app-wide `describeApiError`/`extractApiErrorMessage` helpers treat it
via the generic `Error` branch (message-only). No caller does
`instanceof ApiError` or reads `.status`/`.code`. Preserved-contract checklist:

- Throws `ApiError(status, code, message)` for HTTP errors, with `code`/
  `message` parsed from the `{code,message,error}` envelope (same as the fetch
  version). `.message` stays the backend message.
- `204` and non-`application/json` responses resolve to `undefined`.
- Mutations (`POST`/`PUT`/`DELETE`) still send `X-Requested-With`.
- Pipeline `401`s still do **not** trigger the global logout — the axios call
  sets `skipAuthRedirect: true` (matches the fetch version, which never
  dispatched `UNAUTHORIZED_EVENT`).

## D3 — Network/timeout errors now propagate as `AxiosError`

The fetch version let a network failure reject as a raw `TypeError`; the axios
version rejects with an `AxiosError` that has no `response`. `request()`
re-throws it unchanged (only HTTP errors are wrapped in `ApiError`). This is a
strict improvement: `describeApiError` recognizes `AxiosError` network/timeout
codes and yields a proper "网络连接失败 / 请求超时" message instead of the raw
"Failed to fetch". No caller branches on the thrown type for the no-response
case, so this is safe.
