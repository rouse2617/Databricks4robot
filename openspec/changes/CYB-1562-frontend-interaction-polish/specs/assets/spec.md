# Assets Spec Delta

## Modified Requirements

### Requirement: Asset Search and Batch Interaction Clarity

The asset management page SHALL make filtering and batch actions understandable without requiring backend field-name knowledge.

#### Scenario: User opens search help

- **Given** the asset management page is loaded
- **When** the user opens search help
- **Then** the UI shows at least one directly usable search example
- **And** technical field names are secondary to user-facing labels where possible

#### Scenario: User selects assets

- **Given** one or more assets are selected
- **When** batch actions are displayed
- **Then** disabled actions explain why they are unavailable
- **And** actions that affect all filtered results communicate the affected scope
