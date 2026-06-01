## MODIFIED Requirements

### Requirement: Standard API error envelope
- **Before**: The system SHALL return standard API errors from most handlers, while some compatibility routes MAY return ad-hoc JSON error bodies.
- **After**: The system SHALL return JSON API errors with `code`, `message`, `request_id`, and optional `details` for documented REST failures.
- **Reason**: Users and SDK integrations need stable fields for branching, support escalation, and diagnostics.

#### Scenario: Auth request body is invalid
- **Given** a client calls an auth compatibility endpoint with an invalid request body
- **When** the backend rejects the request
- **Then** the response includes a stable error `code`, human-readable `message`, and `request_id`

#### Scenario: SDK receives a standard API error
- **Given** the backend returns a standard JSON error envelope
- **When** the Python SDK handles the response
- **Then** the raised exception exposes `code`, `message`, `http_status`, `request_id`, and `details`

### Requirement: OpenAPI error schema consistency
- **Before**: The OpenAPI document MAY reference multiple error schema names for the same standard error body.
- **After**: The OpenAPI document SHALL reference a valid canonical error schema for standard JSON errors.
- **Reason**: Generated API reference pages and generated models must describe the same error fields the backend and SDK use.

#### Scenario: API reference renders error responses
- **Given** an endpoint documents a 4xx or 5xx JSON error
- **When** the OpenAPI document is processed by docs tooling
- **Then** the referenced error schema exists and includes `code`, `message`, `request_id`, and optional `details`

#### Scenario: Unknown schema references are checked
- **Given** the OpenAPI document contains error response references
- **When** a reviewer searches for obsolete error schema names
- **Then** no endpoint references an undefined standard error schema

### Requirement: Documented routes are registered at runtime
- **Before**: The SDK and OpenAPI MAY include public route paths that are not mounted by the runtime router.
- **After**: The runtime router SHALL mount public route paths that are present in the SDK and OpenAPI unless they are explicitly documented as unavailable.
- **Reason**: Users following the generated reference or SDK should not encounter 404s caused only by missing route registration.

#### Scenario: SDK calls a documented delivery lifecycle route
- **Given** a delivery lifecycle method exists in the Python SDK and OpenAPI
- **When** the backend starts with the corresponding handler configured
- **Then** the runtime router registers the matching HTTP route

#### Scenario: SDK calls documented event or audit routes
- **Given** event stream and audit search paths exist in OpenAPI
- **When** the backend starts with the corresponding handler configured
- **Then** the runtime router registers those paths instead of returning router-level 404s
