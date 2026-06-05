## 2026-06-05 — Treat incomplete Argo resource data as non-estimable
- **Context**: Existing estimated cost could explode for short runs because the estimator used memory-derived `resourcesDuration` values as if they were direct machine runtime seconds and also defaulted missing metadata to a GPU node price.
- **Decision**: Restrict estimation to explicit pricing metadata plus CPU/GPU runtime duration only. Memory-only or metadata-free rows now return no estimate.
- **Alternatives**: Keep a default non-GPU machine profile, or continue inferring runtime from the largest resource duration value.
- **Rationale**: Returning no estimate is safer than showing absurd totals and preserves trust in the execution list.

## 2026-06-05 — Partial dev verification after backend deploy
- **Context**: The backend revision deployed successfully and `/readyz` returned healthy, but authenticated spot-checking of a representative `pipeline-runs` record stalled while fetching runtime auth material from gcloud in this shell.
- **Decision**: Keep the implementation and backend tests, record the deploy revision, and leave the representative run verification task unchecked in OpenSpec.
- **Alternatives**: Block the PR entirely until a full authenticated cost smoke completes.
- **Rationale**: The code and deploy are ready, and the remaining gap is a verification follow-up rather than an unresolved implementation defect.
