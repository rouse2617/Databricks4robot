## ADDED Requirements

### Requirement: workflow operations map drives workflow actions
The workflow UI SHALL expose Argo workflow operations from a shared declarative operations map so list-row actions and detail-page actions use the same labels, icons, phase enablement, and API callers.

**Priority**: P1 (High)
**Rationale**: Operators need consistent workflow actions across list and detail surfaces, and action availability must follow workflow phase.

#### Scenario: detail page operations match workflow phase
- **Given** a workflow detail page has loaded a workflow phase
- **When** the page renders its header actions
- **Then** retry is enabled only for Failed/Error workflows
- **And** suspend is enabled only for Running workflows
- **And** stop and terminate are enabled for Running/Pending workflows
- **And** resume is enabled only for Suspended workflows
- **And** delete is always available when a workflow is loaded

#### Scenario: list row operations are phase-filtered
- **Given** the workflow list contains workflows in different phases
- **When** a user opens a row operation dropdown
- **Then** the dropdown includes only operations enabled for that row's phase.

### Requirement: workflow status utilities are shared
The workflow UI SHALL share phase colors, Ant Design status tags, duration formatting, workflow label rendering, and linkified text utilities across workflow list and detail pages.

**Priority**: P2 (Nice-to-have)
**Rationale**: Shared utilities prevent list/detail drift and make Argo workflow metadata easier to scan.

#### Scenario: labels render as tags
- **Given** a workflow response includes metadata labels
- **When** the workflow list or detail surface renders those labels
- **Then** each label appears as an Ant Design tag with key/value text.

#### Scenario: workflow messages linkify URLs
- **Given** a workflow or node message contains a URL
- **When** the message is rendered
- **Then** the URL is rendered as an external link without changing surrounding text.
