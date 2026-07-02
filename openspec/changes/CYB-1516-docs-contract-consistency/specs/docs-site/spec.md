## MODIFIED Requirements

### Requirement: Documentation examples match current SDK and API surfaces
- **Before**: Documentation MAY include examples for SDK methods or API routes that are not currently implemented or registered.
- **After**: Documentation SHALL prefer examples that match current SDK methods and registered backend API routes.
- **Reason**: Users should be able to copy examples from public docs without hitting immediate `AttributeError` or 404 failures.

#### Scenario: User copies a search example
- **Given** a user follows a public documentation example for finding assets
- **When** they run the example against the current Python SDK
- **Then** the example uses a supported SDK method and documented API route

#### Scenario: User copies a pipeline component example
- **Given** a user follows a public documentation example for listing pipeline components
- **When** they run the example against the current Python SDK
- **Then** the call only passes parameters supported by the SDK method signature

### Requirement: SDK error handling guidance is actionable
- **Before**: Public docs MAY show only broad exception classes without explaining stable error fields.
- **After**: Public docs SHALL show how SDK users inspect `code`, `message`, `http_status`, `request_id`, and `details`.
- **Reason**: Users need immediate, field-level guidance to distinguish validation, not-found, auth, conflict, and server errors.

#### Scenario: User handles a not-found error
- **Given** an SDK call raises a typed API exception
- **When** the user reads the docs
- **Then** they can see which exception fields are stable and how to log `request_id`

#### Scenario: User receives validation details
- **Given** an API response includes structured `details`
- **When** the SDK raises an exception
- **Then** the docs explain that `details` is available without parsing the message string
