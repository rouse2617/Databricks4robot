## ADDED Requirements

### Requirement: Component registry page
The system SHALL provide a sidebar-accessible component registry page where authenticated users can search, list, create, edit, and delete custom pipeline components.

**Priority**: P1 (High)
**Rationale**: Pipeline users need reusable components beyond the currently hardcoded defaults, and registry management must not require editing the canvas.

#### Scenario: List and search registered components
- **Given** the backend has pipeline components with names, images, types, descriptions, and creation times
- **When** the user opens `/components` and enters a name search
- **Then** the page displays a table with Name, Image, Type, Description, CreatedAt, and Actions filtered by the search term

#### Scenario: Empty registry state
- **Given** the backend returns no components for the current search
- **When** the user views the component registry
- **Then** the page shows an empty table state without crashing

### Requirement: Component form modal
The system SHALL provide a validated component form modal for creating and editing name, image, type, command, args, env, and description.

**Priority**: P1 (High)
**Rationale**: Component definitions need structured fields so users can safely configure reusable pipeline steps.

#### Scenario: Create a valid component
- **Given** the user opens the new component modal
- **When** the user enters required name and image values, selects a type, optionally adds command/args/env/description, and submits
- **Then** the backend persists the component and the table refreshes with the created row

#### Scenario: Required field validation
- **Given** the user opens the new component modal
- **When** the user submits without required name or image values
- **Then** the form blocks submission and shows field validation feedback

#### Scenario: Edit an existing component
- **Given** the user clicks Edit on an existing component row
- **When** the modal opens
- **Then** the form is populated with that component's existing values and saves updates back to the backend

### Requirement: Component deletion
The system SHALL require confirmation before deleting a component and remove deleted custom components from the registry list after backend success.

**Priority**: P1 (High)
**Rationale**: Deleting component definitions is destructive and should be deliberate.

#### Scenario: Confirm component deletion
- **Given** the user clicks Delete on a component row
- **When** the user confirms the Popconfirm
- **Then** the backend deletes the component and the row disappears from the table

#### Scenario: Cancel component deletion
- **Given** the user clicks Delete on a component row
- **When** the user cancels the Popconfirm
- **Then** no delete request is sent and the row remains visible

### Requirement: Component registry API
The system SHALL expose authenticated CRUD API endpoints for pipeline components under `/api/v1/pipeline-components`.

**Priority**: P1 (High)
**Rationale**: The standalone registry page needs a stable backend-backed API surface that matches the product domain.

#### Scenario: CRUD API happy path
- **Given** an authenticated client sends a valid create request
- **When** the client lists, updates, and deletes the created component
- **Then** each operation returns the expected status and response body, and the deleted component no longer appears

#### Scenario: Invalid component create request
- **Given** an authenticated client sends a create request missing required fields
- **When** the backend validates the request
- **Then** it returns a 400 response without creating a component
