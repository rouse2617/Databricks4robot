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

## 2026-06-18 — Node config binding locks an explicit version
- **Context**: Browser regression showed the node config selector displayed entries with their current version but did not expose a way to bind an older ready version.
- **Decision**: Add an explicit version selector after choosing a config. New bindings default to the config current version, but users can choose any ready historical version. Saved nodes persist `{configId, version}`.
- **Alternatives**: Always bind `currentVersion`, or add a separate "latest" floating mode.
- **Rationale**: Runtime reproducibility requires a node to resolve the exact selected config content. A floating latest mode would make saved templates and historical runs drift when config authors publish a new version.

## 2026-06-18 — Config versions need human-readable comparison
- **Context**: The config library exposed `v1`, `v2`, etc. and per-version content viewing, but users could not easily identify or compare many historical versions before binding one to a node.
- **Decision**: Reuse the existing per-version content API to compare any two config versions in the frontend, and display summary, timestamp, and content hash in both config-library comparison controls and node version selectors.
- **Alternatives**: Add a backend diff endpoint immediately, or keep only single-version content viewing.
- **Rationale**: Frontend diff avoids a new HTTP contract while making historical versions auditable enough for node-level runtime selection. A backend diff endpoint can still be added later if large files or binary config formats require server-side handling.
