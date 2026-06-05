# Pipeline Spec Delta

## Modified Requirements

### Requirement: Pipeline templates can pin an active version

The UI SHALL allow users to select a historical template version as the active version for future runs.

#### Scenario: Set older template version active

- **Given** a pipeline template has multiple versions
- **When** the user opens version history and selects "set active version" for a historical version
- **Then** the template list shows that version as active
- **And** future run dialogs default to that active version instead of the latest version

#### Scenario: Latest remains default when no active version is pinned

- **Given** a pipeline template has no active version
- **When** the user opens the run dialog
- **Then** the run dialog defaults to the latest version
