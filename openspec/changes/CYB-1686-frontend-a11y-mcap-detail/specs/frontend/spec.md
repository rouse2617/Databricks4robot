# Frontend Spec Delta

## Modified Requirements

### Requirement: Form controls are identifiable

Audited AntD form controls SHALL expose stable `id` and/or `name` attributes on the underlying interactive control so browser accessibility checks and assistive technology can identify the field.

#### Scenario: Pipeline node configuration modal

- **Given** the node configuration modal is rendered
- **When** a user or audit tool inspects the form controls
- **Then** the editable controls expose stable identifiers matching their form purpose

#### Scenario: Delivery and execution filters

- **Given** delivery creation or workflow execution filters are rendered
- **When** a user or audit tool inspects inputs/selectors
- **Then** the controls expose stable identifiers and keep existing behavior

### Requirement: MCAP file detail route

The `/mcap-files/:id` route SHALL render a functional MCAP detail page for the route parameter.

#### Scenario: MCAP file exists

- **Given** the route is `/mcap-files/{id}`
- **When** the frontend loads the page
- **Then** it requests `/api/v1/mcap-files/{id}`
- **And** displays MCAP metadata
- **And** loads related assets by `mcap_file_id`

#### Scenario: MCAP file fails to load

- **Given** the route is `/mcap-files/{id}`
- **When** the MCAP API returns an error
- **Then** the page displays an error state with retry affordance
