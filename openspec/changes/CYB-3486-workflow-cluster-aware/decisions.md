# Decisions — CYB-3486 workflow handler cluster-aware

## 2026-07-16 — Proceed to implementation without an OpenSpec confirmation checkpoint
- **Context**: AI-RULES prescribes an OpenSpec checkpoint (stop and confirm
  before editing runtime code). The user supplied a complete root-cause analysis
  and implementation directive in-chat and asked to "deliver one PR, base=dev …
  open the PR and wait for review".
- **Decision**: Write the OpenSpec artifacts and proceed directly to
  implementation without pausing for a separate "OpenSpec OK?" confirmation.
- **Alternatives**: Stop after proposal+tasks and wait for「OpenSpec OK，继续」.
- **Rationale**: Rule precedence #1 (explicit user instruction in the current
  message) — the user already authored the design and requested the PR.
  OpenSpec gate is `main`-only (`branches: [main]`); this PR targets `dev`, so
  the gate does not run. The three prior CYB-3486 sub-PRs (#435/#436/#437) left
  no change dir under `openspec/changes/`.

## 2026-07-16 — Reuse the /runs/* resolution logic rather than share a new abstraction
- **Context**: `usecase/pipeline` already has `resolveRunClusterID` /
  `resolveArgoClientForRun` with a target-id → cluster cache.
- **Decision**: Replicate the small resolver inside the workflow handler package
  (`cluster_routing.go`) keyed off the injected `argo.ClientFactory` +
  `ExecutionTargetRepository`, instead of extracting a shared type and
  refactoring the usecase.
- **Alternatives**: Extract a shared `RunClientResolver` used by both usecase
  and handler.
- **Rationale**: Keeps blast radius off the proven `/runs/*` path (no regression
  risk), self-contained, matches the existing in-handler cluster resolution
  precedent (elastic_quota / resource_quota use `h.k8sFactory` + clusterId).
  A shared abstraction can wait for a third consumer.

## 2026-07-16 — Defer per-cluster Pod terminal exec
- **Context**: `/workflows/:name/nodes/:nodeId/terminal-sessions` attach uses
  `k8s.ExecClient`, which needs a `*rest.Config` (SPDY). The current
  `k8s.ClientFactory` exposes only typed/dynamic clients per cluster, not the
  rest.Config.
- **Decision**: Route the terminal-session *validation* GetWorkflow by cluster,
  but leave the exec attach on the default-cluster `ExecClient`; document as a
  follow-up.
- **Alternatives**: Extend `k8s.ClientFactory` with a `RestConfigForCluster`
  seam now.
- **Rationale**: Stated priority is logs (read path), not the web terminal.
  Extending the k8s factory is new infrastructure beyond this fix's scope; defer
  until a real non-default-cluster terminal need exists.
