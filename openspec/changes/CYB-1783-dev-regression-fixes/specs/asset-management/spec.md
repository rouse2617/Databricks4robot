## MODIFIED Requirements

### Requirement: Asset action annotations use the documented route
- **Before**: The frontend and OpenAPI used `/api/v1/assets/{id}/action-annotations`, but the backend only registered `/api/v1/assets/{id}/actions`, causing the Action timeline tab to render a 404 error.
- **After**: The system SHALL serve the documented `/action-annotations` route with the same response envelope and authorization behavior as the existing action list route, while keeping `/actions` backward compatible.
- **Reason**: Users opening asset detail should see an empty or populated Action timeline, not a route-mismatch 404.

**Priority**: P1 (High)
**Rationale**: The Action timeline is a visible asset detail tab and the route mismatch breaks a documented/public API path.

#### Scenario: action timeline loads through documented route
- **Given** asset `CYB10A01` exists
- **When** the asset detail page requests `/api/v1/assets/CYB10A01/action-annotations?limit=200`
- **Then** the backend returns 200 with the action list envelope

#### Scenario: legacy actions route remains compatible
- **Given** an existing client calls `/api/v1/assets/CYB10A01/actions`
- **When** the request is valid
- **Then** the backend continues returning the same action list envelope as before
