## Overview

This change adds toleration support at two authoring levels:

1. component defaults
2. per-node overrides

The transpiler emits node-level tolerations into Argo templates so Kubernetes pod scheduling honors the authored intent.

## Model design

Use a DataBrew-owned toleration model rather than importing `corev1.Toleration` into API-facing structs.

```go
type Toleration struct {
  Key               string `json:"key,omitempty" yaml:"key,omitempty"`
  Operator          string `json:"operator,omitempty" yaml:"operator,omitempty"`
  Value             string `json:"value,omitempty" yaml:"value,omitempty"`
  Effect            string `json:"effect,omitempty" yaml:"effect,omitempty"`
  TolerationSeconds *int64 `json:"tolerationSeconds,omitempty" yaml:"tolerationSeconds,omitempty"`
}
```

Add it to:

- `transpiler.Component`
- `transpiler.Node`
- frontend component/node types
- component API resources payload

## Preset policy

`computeTier=gpu-l4` may inject a default GPU toleration:

```yaml
- key: nvidia.com/gpu
  operator: Equal
  value: present
  effect: NoSchedule
```

Environment-specific tolerations such as `environment=dev` are explicitly out of scope for this change. Those belong to later execution-target or cluster-profile logic.

## Merge semantics

- component defaults populate newly authored nodes
- node config can add or edit tolerations after placement
- transpiler uses node tolerations only
- if a node is created from a component, inherited tolerations become part of node data so later component edits do not silently mutate existing authored pipelines

## Backend flow

1. component create/update normalizes tolerations from JSON resources
2. frontend palette/node creation copies component tolerations into node data
3. pipeline submit serializes node tolerations
4. transpiler converts DataBrew tolerations into `corev1.Toleration`
5. generated Argo `Template` sets `tmpl.Tolerations`

## Validation

Validation in this change remains conservative:

- require non-empty `operator` / `effect` values only if provided
- allow multiple tolerations
- trim whitespace
- do not introduce deep Kubernetes admission emulation in V1

## Test strategy

### Backend

- transpiler tests for container and script templates carrying tolerations
- component normalization tests for toleration JSON and `gpu-l4` preset injection

### Frontend

- component manager serialization/deserialization tests
- node config save test for toleration editing
- node inheritance test when creating a node from a registered component
