## 2026-06-18 — Correct config scope to pipeline node
- **Context**: Browser regression showed the new runtime-config UI was implemented in the saved-pipeline deploy modal as one deploy-level config. The user clarified that config-library entries should be configurable on each node/component.
- **Decision**: Model CYB-2178 as node-level saved config binding. The normal product UI moves to the node configuration panel; deploy-level `configSelection` remains compatibility behavior only.
- **Alternatives**: Keep deploy-level config and add node config later, or support upload/inline drafts per node immediately.
- **Rationale**: Per-node saved config binding matches how different components need different task parameters and avoids one global config leaking across the DAG.

## 2026-06-18 — OpenSpec approved
- **Context**: The OpenSpec checkpoint defined separate scopes: deploy-level asset IDs injected into every Pod, and node-level config-library bindings mounted only on selected nodes.
- **Decision**: User approved the OpenSpec in chat with "ok"; implementation can proceed.
- **Alternatives**: Keep refining spec before runtime code.
- **Rationale**: The product model is now clear enough to implement without continuing the earlier deploy-level config mismatch.

## 2026-06-18 — Runtime config projection lifecycle
- **Context**: Node configs are projected into Kubernetes as run-scoped ConfigMaps. The user called out that large-scale runs could leave many ConfigMaps behind if cleanup relies on humans.
- **Decision**: Generate a deterministic run-scoped ConfigMap name for the Workflow manifest, create the ConfigMap only for real deploys after Argo Workflow creation, and attach an ownerReference to the Argo Workflow UID. Dry-run only renders the manifest and does not create a ConfigMap.
- **Alternatives**: Inline config content into Argo parameters/env vars, use one long-lived ConfigMap per library config, or create unowned ConfigMaps and clean them later.
- **Rationale**: OwnerReferences make the projection behave like part of the Argo task lifecycle: when Workflow TTL removes the Workflow, Kubernetes garbage collection removes the runtime ConfigMap. This avoids unbounded ConfigMap buildup while keeping per-node file mounts simple for component authors.

## 2026-06-18 — Local frontend runtime probe
- **Context**: The user asked to keep the frontend local for faster debugging, while still validating against the deployed dev backend and real Kubernetes pods.
- **Decision**: Use local Vite on `http://127.0.0.1:5177` with the Cloud Run dev backend, create a probe image/config/component, and validate the node-level config binding through Chrome DevTools MCP plus live Pod logs.
- **Alternatives**: Deploy the frontend through CF before this probe, or use API-only deploy verification.
- **Rationale**: Local frontend keeps iteration fast and still exercises the real backend, Argo workflow, ConfigMap projection, asset env injection, and UI workflow. CF frontend deploy remains required before commit if this frontend diff is finalized.

## 2026-06-18 — Dirty snapshot excludes React Flow interaction state
- **Context**: Chrome regression showed that saving a pipeline and then switching tabs could still show the "unsaved changes" modal. The saved DSL had not changed; the difference came from transient React Flow node state such as `selected`.
- **Decision**: Build dirty-check snapshots from stable persisted fields only: pipeline name, node id/type/position/data, and edge id/source/target/handles/data.
- **Alternatives**: Clear dirty only imperatively after save, or ignore dirty checks while navigating away from the design tab.
- **Rationale**: The dirty check should model persisted pipeline content, not hover/selection/drag runtime state. This keeps real edits protected while preventing false-positive leave prompts.
