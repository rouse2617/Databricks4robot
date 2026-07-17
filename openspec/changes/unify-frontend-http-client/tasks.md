# Tasks

- [x] Reimplement `request<T>()` in `src/api/pipelineClient.ts` to delegate to
      `apiClient` (axios) instead of raw `fetch`.
- [x] Preserve the public contract: `ApiError(status, code, message)` from the
      backend envelope; `204` / non-JSON → `undefined`; `X-Requested-With` on
      mutations; `skipAuthRedirect` so pipeline `401`s do not trigger logout.
- [x] Drop the now-duplicated `DEV_ACCESS_TOKEN` header logic (apiClient's
      request interceptor owns it).
- [x] Add `src/api/pipelineClient.test.ts` covering success/204/non-JSON,
      ApiError mapping (with and without envelope), network-error passthrough,
      and the mutation header.
- [x] Local gates: `npm run lint`, affected `vitest`, `npm run build`.
- [ ] Post-merge: dev smoke on a `request()`-backed surface (pool CRUD list).
