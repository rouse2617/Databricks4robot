# CYB-2100 Argo-Only Run Platform

Unified DataBrew Run model on Argo Workflows; Tekton excluded from product runtime.

## Phases

- Phase 0: Pipeline designer split, UUID nodes, save/run validation, runId operations
- Phase 1: `databrew_runs` + `/api/v1/runs`
- Phase 2: `/runs/:runId` Run Inspector routes
- Phase 3: Pods API + RunPodsPanel
- Phase 4: Component build dialog + `/runs/component-build`
- Phase 5: RAG build + `/runs/rag-build`
