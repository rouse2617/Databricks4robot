# Spec Delta — Pipeline Version Management

## ADDED: Pipeline Version History

### Given a user viewing the pipeline management page
**When** the user clicks the version badge on a pipeline card
**Then** a drawer opens showing all historical versions of that pipeline
**And** each version entry displays: version number, step count, save timestamp, and optional description
**And** versions are listed in reverse chronological order (latest first)

### Given a pipeline with multiple versions
**When** the user views the version history drawer
**Then** they can distinguish versions by metadata (not just a number)
**And** user-provided descriptions (if any) are shown beneath each version

## ADDED: Execution List Version Filter

### Given a user viewing the pipeline execution list
**When** the user selects a version from the "模板版本" dropdown
**And** clicks "应用"
**Then** only executions matching that template version are displayed

### Given multiple versions exist for a pipeline
**When** the user opens the version filter dropdown
**Then** all distinct versions across visible executions are listed

## ADDED: Version Description on Save

### Given a user editing a pipeline template
**When** the user clicks "保存"
**Then** they are presented with an optional text input for a version description
**And** the description is saved alongside the template metadata
**And** the description appears in the version history drawer
