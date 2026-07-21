## MODIFIED Requirements

### Requirement: Run 状态不得由瞬时观测从失败误升为成功

The system SHALL NOT transition a persisted run from a terminal failure (`Failed`/`Error`) directly to `Succeeded`. A genuine recovery always transits through an active phase (Argo retry/resubmit resets the workflow to `Running` before it can reach `Succeeded`), so a direct `Failed/Error → Succeeded` change (with no intervening active status) is spurious and SHALL be rejected, keeping the existing terminal-failure status.

Additionally, when deriving run status from a fetched Argo workflow, the system SHALL NOT map a workflow to `Succeeded` while that workflow is active (phase `Running`/empty) or has any `Failed`/`Error` leaf (Pod) node; such a workflow SHALL map to `Running` (to be re-observed).

#### Scenario: mid-retry read does not promote a failed run to succeeded

- **GIVEN** a run persisted as `Failed` whose Argo workflow has just been reset by a retry (active/mid-retry, failed leaf being re-run)
- **WHEN** a read-path reconcile observes the workflow and would derive `Succeeded`
- **THEN** the run SHALL remain `Failed` (the direct `Failed → Succeeded` promotion is rejected)
- **AND** the retry entry point SHALL still consider the run retryable

#### Scenario: genuine retry success is still recorded

- **GIVEN** a `Failed` run that is retried and the workflow re-runs to real success
- **WHEN** observations arrive as `Failed → Running` then `Running → Succeeded`
- **THEN** the run SHALL end as `Succeeded` (transit through the active phase is allowed)

#### Scenario: a genuinely succeeded run is unaffected

- **GIVEN** a run whose workflow completed with all leaf nodes `Succeeded`
- **WHEN** its status is derived/persisted
- **THEN** the run SHALL be `Succeeded`

### Requirement: 已误标为成功的失败 run 依 Argo 权威纠回

The system SHALL correct a run that is persisted as `Succeeded` but whose durable asset-node ledger contains a `Failed`/`Error` leaf back to the terminal failure status inferred from the ledger. This correction SHALL occur on the bounded single-run read path and SHALL NOT require manual data mutation.

#### Scenario: stuck succeeded run is recovered to failed on read

- **GIVEN** a run persisted as `Succeeded` whose asset-node ledger has a `Failed` leaf (a run stuck by the prior revival bug)
- **WHEN** its detail is read
- **THEN** the run status SHALL be corrected to `Failed` (terminal→terminal), and the retry control SHALL become available again
