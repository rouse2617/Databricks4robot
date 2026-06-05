# Pipeline Designer Spec Delta

## New Blank Pipeline Names

### Requirement

The pipeline designer SHALL initialize a new blank pipeline with a generated, timestamp-based name instead of a hard-coded repeated name.

#### Scenario: Open a blank designer

- **Given** the user opens the pipeline designer without a template id, session edit, imported pipeline, or example selection
- **When** the page renders
- **Then** the pipeline name field contains a generated name matching `pipeline-YYYYMMDD-HHMMSS`
- **And** the name is used by save and deploy unless the user edits it.

#### Scenario: Load an existing named pipeline

- **Given** the user opens the designer from a saved pipeline, session edit, import, or example
- **When** the pipeline is loaded onto the canvas
- **Then** the loaded pipeline name replaces the generated blank default.
