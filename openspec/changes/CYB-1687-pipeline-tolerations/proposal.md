## Why

DataBrew pipelines can request GPU or other specialized resources, but today they cannot express Kubernetes tolerations end to end. In tainted GKE node pools this blocks workflow pods from scheduling, even when a component already requests matching GPU resources.

The current gap is operationally significant:

- GPU components cannot land on tainted GPU pools
- component authors must rely on out-of-band cluster changes instead of DataBrew modelled scheduling intent
- users cannot see or edit tolerations in either component defaults or per-node overrides

## What Changes

Introduce a first-class DataBrew toleration model through the pipeline authoring and transpilation flow.

### Backend

- Add a DataBrew-owned `Toleration` struct to transpiler-facing pipeline models
- Allow both `Component` and `Node` to carry tolerations
- Pass node tolerations into generated Argo container/script templates
- Normalize component tolerations from component JSON resources
- Add compute-tier preset injection for `gpu-l4` that adds only the GPU toleration

### Frontend

- Add component-level toleration editing in Component Manager
- Add node-level toleration editing in NodeConfigPanel
- Ensure newly dragged nodes inherit component default tolerations

### Guardrails

- Do not expose raw `corev1.Toleration` in API/UI-facing models
- Do not hardcode environment-specific tolerations like `environment=dev` in component presets
- Reuse existing JSON / JSONB component storage; no new database migration in this change

## Impact

- GPU-oriented components can carry scheduling intent directly in DataBrew
- per-node override remains available for exceptional cases
- Argo pod specs become compatible with tainted node pools without manual cluster-side exceptions
