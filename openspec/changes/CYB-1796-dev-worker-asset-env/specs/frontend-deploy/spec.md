## MODIFIED Requirements

### Requirement: Dev Worker frontend environment identity
- **Before**: The system SHALL serve the frontend app from the dev Worker, but the served bundle may identify itself as `production` when built through a generic production-mode Vite build.
- **After**: The system SHALL serve the dev Worker frontend from a bundle that identifies the environment as `dev`.
- **Reason**: Dev QA relies on the version marker to distinguish dev from production and to confirm the correct Worker asset bundle is being exercised.

**Priority**: P1 (High)
**Rationale**: Incorrect environment identity undermines deploy verification and can hide stale Worker asset deployments.

#### Scenario: Dev Worker marker shows dev
- **Given** the frontend is deployed to `cyber-databrew-dev`
- **When** a user opens any authenticated dev app page
- **Then** the frontend version marker reports the environment as `dev`

#### Scenario: Served asset matches fresh dev build
- **Given** a dev Worker deploy has completed
- **When** the served root HTML is inspected
- **Then** it references the freshly built `index-*.js` asset for that deploy
- **And** that asset contains the dev environment marker

#### Scenario: Production Worker scope is unchanged
- **Given** the dev Worker deployment path is used
- **When** the dev Worker is deployed
- **Then** production Worker behavior and production environment identity are not changed
