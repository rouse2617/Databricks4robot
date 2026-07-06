# Runtime OS Spec Delta — CYB-3073

## MODIFIED Requirements

### Requirement: Pipeline run estimated cost sums leaf nodes only
- **Before**: The system SHALL compute a run's estimated total cost by summing the estimated cost of all workflow nodes.
- **After**: The system SHALL compute a run's estimated total cost by summing only leaf (Pod) nodes, excluding aggregate nodes (DAG, Steps, StepGroup) whose resource duration is a rollup of their children, so the total is not double-counted and matches the per-step breakdown.
- **Reason**: The DAG root node carries a rollup `resourcesDuration`; summing it on top of the child pods over-counts (observed ~70% inflation).

#### Scenario: Total equals sum of steps
- **Given** a run whose per-step (Pod) estimated costs sum to X
- **When** the run's total estimated cost is computed
- **Then** the total equals X (the aggregate DAG/Steps nodes are not added)

### Requirement: Per-node cost is priced by the node's real machine type
- **Before**: The system SHALL estimate per-node cost using a fixed default GPU node-pool hourly rate when the node carries no embedded instance type.
- **After**: The system SHALL estimate per-node cost using the node's actual machine type, accelerator, and provisioning (resolved from where the node's pod ran), so CPU-only work is priced at CPU rates and GPU work at GPU rates.
- **Reason**: Defaulting every node to the GPU pool over-estimates CPU-only steps by an order of magnitude and makes the CPU price tiers dead config.

#### Scenario: CPU step priced at CPU rate
- **Given** a step that ran on a CPU node pool
- **When** its cost is estimated
- **Then** the cost uses that CPU pool's hourly rate, not the GPU pool rate

#### Scenario: Node profile lookup failure degrades gracefully
- **Given** the node's machine type cannot be resolved (lookup miss or K8s unavailable)
- **When** its cost is estimated
- **Then** the system falls back to the previous default behavior and never errors or crashes
