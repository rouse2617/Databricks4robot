# Design — CYB-1625

## Resource Shape
Component and pipeline node resources use the existing JSON object:

```json
{
  "cpu": "4000m",
  "memory": "16Gi",
  "disk": "50Gi",
  "gpu": "1",
  "computeTier": "gpu-l4"
}
```

`gpu` is a Kubernetes quantity string. `computeTier` is DataBrew metadata used by future target selection, admission, and cost policy.

## Kubernetes Mapping
When `gpu` is set, DataBrew emits only a container limit:

```yaml
resources:
  limits:
    nvidia.com/gpu: "1"
```

Kubernetes treats GPU as a limit-only extended resource. CPU, memory, and ephemeral storage keep the existing request=limit behavior.

## Non-Goals
- Do not add node selectors or tolerations yet.
- Do not enforce quota or GPU availability yet.
- Do not add database columns; existing `resources` JSONB stores the new fields.
