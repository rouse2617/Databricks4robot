## MODIFIED Requirements

### Requirement: Algorithm lifecycle
- **Before**: The system SHALL track algorithm execution state per asset using a state machine driven by `backend/config/algo_registry.yaml`.
- **After**: The system SHALL let users initiate asset processing from asset discovery/detail through Pipeline while preserving existing algorithm state tracking for legacy algorithm surfaces.
- **Reason**: Pipeline is becoming the processing execution surface, so asset users need a direct path from selected assets to Pipeline runs.

**Priority**: P0 (Critical)
**Rationale**: Users should not leave the asset page to manually copy asset IDs into Pipeline when the asset selection already expresses the intended processing batch.

#### Scenario: Launch Pipeline from selected assets
- **Given** the user has selected one or more assets in asset discovery
- **When** the user chooses the asset processing action
- **Then** the system opens Pipeline with those assets already selected for the next run

#### Scenario: Clear selected assets before running
- **Given** Pipeline was opened from selected assets
- **When** the user clears the asset selection in the run dialog
- **Then** the system treats the submission as an explicit no-asset run and labels it accordingly

#### Scenario: Large selected batch is explicit
- **Given** the user selected a large number of assets
- **When** the user opens Pipeline processing from that selection
- **Then** the system shows the selected count before submission so the user can confirm or clear the batch

### Requirement: Asset detail processing action
The system SHALL provide a single-asset Pipeline processing action on asset detail pages.

**Priority**: P1 (High)
**Rationale**: Asset detail is where users inspect an individual asset and naturally decide to process or reprocess it.

#### Scenario: Run Pipeline for current asset
- **Given** the user is viewing an asset detail page
- **When** the user chooses to run Pipeline for that asset
- **Then** the system opens Pipeline with only that asset selected

#### Scenario: Return to asset after inspecting execution
- **Given** an execution detail shows a bound input asset
- **When** the user selects that asset link
- **Then** the system navigates to the corresponding asset detail page
