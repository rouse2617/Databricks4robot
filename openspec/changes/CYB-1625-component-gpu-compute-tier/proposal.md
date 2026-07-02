# Proposal — CYB-1625

## Why
Pipeline components can already declare CPU, memory, and disk resources, but GPU workloads need an explicit GPU request and a compute tier marker for later scheduling and cost policy.

## What Changes
- Add `gpu` and `computeTier` to pipeline component resource metadata.
- Preserve these fields through component CRUD, pipeline canvas serialization, and saved pipeline restore.
- Emit Kubernetes `nvidia.com/gpu` limits when a pipeline node declares `resources.gpu`.
- Document the new resource fields in OpenAPI and the API guide.

## Impact
- **Affected code**: pipeline component frontend, pipeline JSON contract, backend transpiler, OpenAPI, API guide
- **New APIs**: No new endpoint. Existing component and pipeline payloads accept additional `resources` fields.
- **Dependencies**: None

## Scope
- **In scope**: GPU quantity, compute tier metadata, frontend create/edit/detail support, Argo manifest GPU limit generation.
- **Out of scope**: GPU node pool selection, quota admission, actual billing allocation, or scheduler policy enforcement.

## Success Criteria
- [ ] Component create/update can store `resources.gpu` and `resources.computeTier`.
- [ ] Dragging a component into the designer preserves GPU and compute tier on the node.
- [ ] Saving/restoring a pipeline preserves GPU and compute tier.
- [ ] A node with `gpu: "1"` emits `nvidia.com/gpu: "1"` in the Argo template limits.
