## ADDED Requirements

### Requirement: Asset context preserved through run submission
The system SHALL preserve asset IDs passed from asset pages through Pipeline run confirmation and submission.

**Priority**: P0 (Critical)
**Rationale**: Asset-driven processing is only trustworthy if the selected assets are visible and unchanged at the point where the user starts the run.

#### Scenario: Incoming asset IDs prefill the run dialog
- **Given** Pipeline is opened with asset context from an asset page
- **When** the user opens the run dialog
- **Then** the run dialog shows the selected asset count and selected asset IDs before submission

#### Scenario: User changes selected assets before running
- **Given** Pipeline is opened with asset context from an asset page
- **When** the user adds or removes assets in the run dialog
- **Then** the run submission uses the updated asset selection

#### Scenario: No-asset run remains explicit
- **Given** the run dialog has no selected assets
- **When** the user submits the run
- **Then** the system labels the run as a no-asset/debug run instead of implying assets were processed

### Requirement: Execution detail input asset links
The system SHALL display input asset bindings on Pipeline execution detail pages when they are available.

**Priority**: P0 (Critical)
**Rationale**: Users need to answer "which assets did this run process?" from the execution page without reading labels or manifests.

#### Scenario: Asset-bound execution shows linked assets
- **Given** a Pipeline execution was submitted with input assets
- **When** the user opens the execution detail
- **Then** the system shows those input assets as clickable business objects

#### Scenario: Legacy or external execution has no asset binding
- **Given** a workflow was submitted without DataBrew asset bindings
- **When** the user opens the execution detail
- **Then** the system shows a clear product-facing empty state instead of backend placeholder text

### Requirement: Pipeline product empty states
The system SHALL describe unavailable events, cost, product, and lineage data in product-facing language.

**Priority**: P1 (High)
**Rationale**: Backend-oriented placeholders make the product look broken even when the absence is expected for legacy, external, or no-asset runs.

#### Scenario: Events are unavailable for historical workflow
- **Given** an execution has no DataBrew run events
- **When** the user opens the event timeline
- **Then** the system explains that events are unavailable for this historical or external workflow

#### Scenario: Cost is unavailable
- **Given** an execution has no estimated cost data
- **When** the user opens cost or node diagnostics
- **Then** the system shows that cost was not recorded for this run without displaying it as zero
