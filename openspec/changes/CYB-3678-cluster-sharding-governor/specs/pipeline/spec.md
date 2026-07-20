# Pipeline Spec Delta — CYB-3678

## MODIFIED Requirements

### Requirement: Dispatch is per-cluster isolated and rate-governed
- **Before**: One global submitter cycle SHALL dispatch all clusters
  sequentially with a fixed concurrency and no rate protection, so one slow
  cluster stalls all and submission can outrun the Argo controller.
- **After**: The system SHALL shard dispatch by target cluster (own goroutine,
  advisory lock, token bucket sized to controller consumption, AIMD
  concurrency with a floor), SHALL self-kick while backlog remains, and SHALL
  skip a cluster's remaining cycle after consecutive transient failures.
- **Reason**: C5/C13/C16 (head-of-line, controller overrun, operator-storm).

#### Scenario: Slow cluster is isolated
- **Given** batches on clusters A and B where B's channel is held elsewhere
- **When** a cycle runs
- **Then** A dispatches fully and B's items stay pending for B's holder

### Requirement: Transient failures retry bounded; permanent fail fast
- **Before**: Any non-AlreadyExists submit error SHALL immediately fail the
  item — a network blip mass-fails innocent items; a poison item that stays
  pending retries forever.
- **After**: Only explicitly-permanent errors SHALL fail immediately; all
  other errors SHALL leave the item pending with a durable attempt counter,
  dead-lettering at the cap. The DLQ SHALL be listable and explicitly
  retryable (fresh cap, governed re-entry).
- **Reason**: review P1-1 (misclassification mass-failure) + C8 (poison).

#### Scenario: Network blip does not fail the batch
- **Given** a transient connection failure during submit
- **When** the cycle runs
- **Then** the item stays pending and is retried next cycle, failing only
  after the attempt cap

#### Scenario: DLQ revive is storm-safe
- **Given** 1000 dead-lettered items
- **When** the operator calls dlq:retry
- **Then** they rejoin the normal governed pipeline (bucket + AIMD), not a
  bypass path
