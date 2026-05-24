## ADDED Requirements

### Requirement: Admin search settings degrade cleanly when unsupported

The settings UI SHALL avoid surfacing console/resource errors for search admin widgets when deployed admin search capabilities are unavailable, and SHALL either hide or disable unsupported widgets with clear empty/unavailable states.

#### Scenario: Reindex job endpoints are unavailable

- **GIVEN** the deployed API returns not found for search reindex job history
- **WHEN** the user opens `/settings`
- **THEN** the settings page remains usable
- **AND** no user-facing 404 error is shown for background job history loading
- **AND** unsupported reindex controls are hidden, disabled, or marked unavailable

#### Scenario: User cancels reindex confirmation

- **GIVEN** the reindex confirmation modal is open
- **WHEN** the user clicks cancel or closes the modal
- **THEN** the modal closes
- **AND** no destructive reindex request is sent

### Requirement: Search sync state and degraded query warnings are understandable

The UI SHALL present search index sync mode and degraded search execution warnings using the actual runtime status and user-facing copy.

#### Scenario: Outbox ES subscriber is running

- **GIVEN** Elasticsearch is connected
- **AND** the Outbox ES subscriber is running
- **WHEN** the user opens the search sync status surface
- **THEN** the UI shows Outbox automatic sync instead of manual or no-sync status

#### Scenario: Query falls back to PostgreSQL

- **GIVEN** an asset query returns a backend warning that Elasticsearch is unavailable and PostgreSQL fallback was used
- **WHEN** the user views the assets result warning
- **THEN** the warning is shown in Chinese
- **AND** it explains that the query still ran but may be slower
