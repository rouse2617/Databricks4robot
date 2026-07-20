# Tasks

- [x] Canonical projection hash (length-prefixed, order-independent) + name
      derivation (`runtime-config-<hex32>`, ≤63 chars).
- [x] Deploy ordering flip: ensure CM (owner-less) → blob upsert → submit
      Workflow; owner-lookup/rollback block deleted; obsolete wfClient guard
      removed.
- [x] Store: content-addressed labels/annotation; AlreadyExists → rate-limited
      last-referenced touch (shared cache via factory); legacy path unchanged.
- [x] Janitor: cluster-wide sweep by label, delete only expired; ambiguity →
      keep; hourly loop + boot eager sweep (default cluster; per-cluster rides
      CYB-3678/3681).
- [x] `runtime_config_blobs` migration + repo + best-effort upsert wiring.
- [x] Tests (new code 100%): hash canonicalization/boundary/determinism, name
      limits, blob persist (all branches), store CA/legacy/refresh/rate-limit/
      patch-failure, janitor sweep matrix + delete-failure + list-error +
      eager-start, Deploy ensure-failure & store-resolve-failure abort BEFORE
      workflow (ordering proof); 4 legacy Deploy tests updated to owner-less
      contract.
- [x] Gates: gofmt (own files), go vet, full suite green.
- [ ] Migration to dev, PR → merge, CI/CD watch, live verify (shared CM across
      two runs + blob row + workflows succeed).
