## ADDED Requirements

### Requirement: Deploy panel supports multiple config file source modes
The system SHALL let the deploy panel represent config files from three mutually exclusive source modes: saved platform config, local file upload, and inline draft editing.

**Priority**: P0 (Critical)
**Rationale**: Users need one deploy-time flow that covers reusable platform configs and one-off runtime files without leaving the deploy surface.

#### Scenario: User chooses a saved platform config
- **Given** the deploy panel is open
- **When** the user chooses the saved-config source mode
- **Then** the deploy panel shows selectable configs from the platform library
- **And** the user can review the chosen config metadata in the same panel

#### Scenario: User chooses a local file
- **Given** the deploy panel is open
- **When** the user chooses the upload source mode and selects a local file
- **Then** the deploy panel shows the file name and draft file metadata
- **And** the chosen file is treated as deploy-time draft input rather than an existing platform config

#### Scenario: User writes config content inline
- **Given** the deploy panel is open
- **When** the user chooses the inline-edit source mode
- **Then** the deploy panel shows an editor for direct file content entry
- **And** the summary identifies the source as inline draft content

### Requirement: Deploy panel captures config mount destination
The system SHALL let users define the in-container mount destination for the chosen config file from within the deploy panel.

**Priority**: P1 (High)
**Rationale**: Selecting a file without defining where it lands in the container leaves the runtime contract incomplete.

#### Scenario: User specifies mount destination for a saved config
- **Given** the user has selected a deploy-time config file
- **When** the user enters a mount path and target filename
- **Then** the deploy panel summary shows both the source and the target destination

#### Scenario: Missing mount destination is surfaced before deploy review
- **Given** the deploy panel requires a config file to be mounted
- **When** the user has chosen a file source but left the mount destination incomplete
- **Then** the UI surfaces that the destination is incomplete
- **And** the panel does not present the draft as fully ready
