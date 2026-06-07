## ADDED Requirements

### Requirement: Global frontend build marker
The system SHALL show a global frontend build marker in the app shell so users can identify the loaded UI build from normal application pages.

**Priority**: P1 (High)
**Rationale**: QA and regression reports need to tie screenshots and browser sessions to the exact frontend bundle deployed to the Worker.

#### Scenario: Build marker visible on app pages
- **Given** a user opens an authenticated Cyber Databrew page
- **When** the app shell finishes rendering
- **Then** a lower-corner build marker is visible without navigating to a diagnostics page

#### Scenario: Build marker includes build identity
- **Given** the frontend bundle was built with version and build reference metadata
- **When** the user views the build marker
- **Then** the marker shows the package version and a short commit/build reference

#### Scenario: Full build details available
- **Given** the build marker is visible
- **When** the user hovers or opens the marker details
- **Then** the UI exposes the full build reference and environment label when available

#### Scenario: Marker does not block workflow controls
- **Given** the user is viewing a dense workflow or pipeline page
- **When** they interact with normal page controls
- **Then** the marker stays visually secondary and does not prevent access to primary controls
