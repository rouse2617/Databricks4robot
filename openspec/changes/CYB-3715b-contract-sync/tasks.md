# CYB-3715b tasks

- [x] `api/openapi.yaml` — insert 7 flatten fields in `Asset` schema after `env`/`task` and before `version`
- [x] `docs/review/api-guide.md` — update 顶层字段 section with the 7 fields + example filter
- [x] `scripts/api-guide-smoke.sh` — add CYB-3715 regression pack (loop over `camera_model` + `source_platform`, assert 200 + total > 0)
- [x] Tier L — grep for lint hooks; run smoke against dev to prove the assertion passes
- [x] Commit + push + open PR base=dev

## Not in this PR

- Subscriber ES doc projection (CYB-3715c — separate PR / branch)
- Frontend facet chips (CYB-3715d)
