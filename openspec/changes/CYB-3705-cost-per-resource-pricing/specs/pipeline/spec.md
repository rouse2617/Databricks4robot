## MODIFIED Requirements

### Requirement: 流水线 Pod 成本按资源分项单价估算

The system SHALL estimate a leaf Pod node's cost by summing each Argo `resourcesDuration` resource against a per-resource unit rate, rather than multiplying one dominant resource's duration by a whole-instance hourly rate.

For each leaf Pod node with `resourcesDuration` `rd`:

```
cost = Σ_r  rd[r] × unit_rate[r][provisioning] / 3600
```

where `r ∈ {cpu, nvidia.com/gpu, memory}`, and:

- `rd["cpu"]` is core-seconds (`cores × wall-seconds`) and SHALL be priced by a per-vCPU-hour rate — NOT treated as wall-clock seconds.
- `rd["nvidia.com/gpu"]` is GPU-seconds and SHALL be priced by a per-GPU-hour (accelerator) rate. GPU cost SHALL be added only when `nvidia.com/gpu > 0`.
- `rd["memory"]` is in Argo's 100Mi base unit and SHALL be priced by a small per-100Mi-hour rate.
- `provisioning` SHALL be taken from the node's resolved `cloud.google.com/gke-provisioning` label (`spot` | `standard`) when available, and SHALL default to `standard` when unresolved.
- The result SHALL be multiplied by `calibration_factor`.
- The function SHALL return a non-nil cost (possibly `0`) whenever at least one resource has a positive duration, and `nil` only when no positive resource duration exists, so the cost-snapshot-missing check does not loop.

#### Scenario: CPU-only step on an unresolved cluster is priced by CPU rate, not GPU

- **GIVEN** a leaf Pod node with `resourcesDuration = {cpu: 20132}` and no resolvable machine type (e.g. a `cyber-clust` delivery step)
- **WHEN** the estimator prices it
- **THEN** the cost SHALL be `20132 × cpu_unit_rate[standard] / 3600` (a CPU-tier price, order ~$0.17)
- **AND** the cost SHALL NOT include any GPU component

#### Scenario: GPU step includes GPU-seconds priced at the accelerator rate

- **GIVEN** a leaf Pod node with `resourcesDuration = {cpu: C, nvidia.com/gpu: G}` where `G > 0`
- **WHEN** the estimator prices it
- **THEN** the cost SHALL be `C × cpu_unit_rate / 3600 + G × gpu_unit_rate / 3600`
- **AND** it SHALL remain the same order of magnitude as the pre-change whole-instance estimate for a single-GPU pod (regression protection)

#### Scenario: spot provisioning uses the discounted unit rate

- **GIVEN** two identical `resourcesDuration` maps, one with `provisioning: spot` and one with `provisioning: standard`
- **WHEN** the estimator prices them
- **THEN** the spot cost SHALL be strictly lower than the standard cost (≈ 1/3)

### Requirement: Run 总成本仅累加叶子 Pod 节点

The system SHALL compute a run's total estimated cost by summing only leaf Pod nodes' stored per-node cost, excluding aggregate nodes (DAG / Steps / StepGroup / TaskGroup / Retry) whose `resourcesDuration` is a rollup of their children. (Unchanged from CYB-3073; restated to guard against regression by this change.)

#### Scenario: aggregate DAG rollup node is not double-counted

- **GIVEN** a run with two leaf Pod nodes and one DAG rollup node each carrying an `estimatedCostUsd`
- **WHEN** the run total is computed
- **THEN** the total SHALL equal the sum of the two Pod nodes only, excluding the DAG node
