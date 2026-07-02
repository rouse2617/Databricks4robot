# Frontend Spec Delta

## Changed Requirements

### Requirement: Pipeline pages should not emit avoidable framework warnings during normal use

Pipeline authoring and execution pages SHOULD keep browser console clear of known avoidable framework warnings during normal navigation and core interactions.

#### Scenario: Opening Pipeline pages

- **Given** the frontend is running against a local or dev backend
- **When** a user opens Pipeline design, saved pipeline, components, or execution pages
- **Then** the browser console should not show avoidable AntD usage warnings or React Flow node type stability warnings caused by application code.
