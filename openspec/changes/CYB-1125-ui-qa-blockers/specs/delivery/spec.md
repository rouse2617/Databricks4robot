## ADDED Requirements

### Requirement: Delivery creation failures remain actionable in the modal

The delivery creation modal SHALL preserve user input and show field-specific, actionable errors when the backend rejects a delivery request.

#### Scenario: Backend rejects a manual asset ID

- **GIVEN** the user manually enters asset IDs in the delivery creation modal
- **AND** the backend rejects one of the asset IDs
- **WHEN** the delivery request fails
- **THEN** the modal remains open
- **AND** the entered asset IDs and customer ID remain visible
- **AND** the asset ID field shows what needs to be fixed
- **AND** the request ID is shown when the backend provides one
