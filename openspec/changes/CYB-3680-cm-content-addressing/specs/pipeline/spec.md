# Pipeline Spec Delta — CYB-3680

## MODIFIED Requirements

### Requirement: Runtime-config ConfigMaps are shared and crash-safe
- **Before**: The system SHALL create one `runtime-config-<run-id>` ConfigMap
  per run, owned by its Workflow and created after it — duplicating identical
  content across a batch and leaving a crash window where a pod mounts a
  ConfigMap that will never exist.
- **After**: The system SHALL name runtime-config ConfigMaps by content hash,
  create them idempotently BEFORE the Workflow, share them across runs
  (owner-less), persist the bytes in `runtime_config_blobs` as the source of
  truth, and reclaim them only when their sliding-reference age exceeds the
  configured TTL (default 35d). Legacy per-run ConfigMaps keep their owner
  cascade untouched.
- **Reason**: 100k-batch object explosion (P2) + the FailedMount stuck-pod
  class (original production incident).

#### Scenario: Batch shares one ConfigMap
- **Given** 1000 runs whose runtime config bytes are identical
- **When** they are deployed
- **Then** exactly one content-addressed ConfigMap exists and all runs mount it

#### Scenario: Ensure failure precedes the Workflow
- **Given** the ConfigMap cannot be created
- **When** Deploy runs
- **Then** no Workflow is created (no pod can ever mount a missing config)

#### Scenario: Active configs are never reclaimed
- **Given** a config still being ensured by active batches
- **When** the janitor sweeps
- **Then** its refreshed last-referenced timestamp keeps it alive; only
  configs unreferenced for the TTL are deleted, and deletion is recoverable
  from the DB blob
