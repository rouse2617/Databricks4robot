# Spec Delta — Pipeline UI

## ADDED Requirements

### Requirement: Auto-added pipeline nodes do not overlap
The pipeline canvas SHALL place components added by clicking the component palette with enough spacing that consecutive nodes do not visually overlap.

#### Scenario: Multiple components are added from the palette
- **Given** the user adds multiple components without manually dragging them
- **When** the components appear on the canvas
- **Then** each node SHALL be placed on a stable grid with horizontal or wrapped vertical spacing.

### Requirement: Execution list separates pipeline identity from run identity
The execution list SHALL show workflow/run identity separately from pipeline template identity.

#### Scenario: Pipeline run has template metadata
- **Given** an execution row has a template id and template version
- **When** the row is rendered
- **Then** the `名称` column SHALL show the workflow/run identity
- **And** the `流水线` column SHALL show `templateId : version` with the template name as secondary text.
