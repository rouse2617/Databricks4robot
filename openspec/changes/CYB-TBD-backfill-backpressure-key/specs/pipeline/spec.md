# pipeline Specification Delta

## MODIFIED Requirements

### Requirement: Cluster-scoped dispatch backpressure observation

- **Before**: The system SHALL record the active (pending+running) workflow
  count per namespace and gate batch dispatch on it. Two clusters that dispatch
  into the same namespace name shared one observation, so one cluster's count
  overwrote the other's.
- **After**: The system SHALL record and read the active-workflow backpressure
  observation per `(cluster, namespace)`. A cluster's count SHALL NOT be
  overwritten by another cluster that happens to use the same namespace name.
- **Reason**: Namespace-only keying let same-named namespaces on different
  clusters collide, causing over-dispatch to a saturated cluster and false
  backpressure on an idle one.

**Priority**: P1 (High)
**Rationale**: Dispatch still functions, but wrong backpressure input degrades
control-plane protection and cross-cluster throughput.

#### Scenario: Same namespace name on two clusters stays independent

- **Given** cluster-A and cluster-B both dispatch into a namespace named `argo`
- **And** the watcher observes 200 active workflows on cluster-A/`argo` and 10 on cluster-B/`argo`
- **When** the submitter evaluates backpressure for a target on cluster-B/`argo` with a ceiling of 100
- **Then** it reads 10 (cluster-B's own count) and dispatch proceeds
- **And** a target on cluster-A/`argo` reads 200 and dispatch is deferred

#### Scenario: Default-cluster observation resolves to one key

- **Given** a run and a target both resolve to the default cluster (empty `cluster_id`)
- **When** the watcher records the count and the submitter reads it
- **Then** both use the same canonical `default` cluster key and backpressure is effective

#### Scenario: Unknown observation still fails open

- **Given** no watcher observation exists yet for a target's `(cluster, namespace)`
- **When** the submitter evaluates backpressure
- **Then** dispatch proceeds (fail open), unchanged from before
