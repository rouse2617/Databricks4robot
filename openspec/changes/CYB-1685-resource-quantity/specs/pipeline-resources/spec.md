# Spec Delta — Pipeline Resources

## ADDED Requirements

### Requirement: Resource quantities reject unsafe memory and disk bare numbers
The system SHALL reject memory and disk resource quantities that are bare numbers or non-positive, and require explicit Kubernetes quantity units such as `Mi`, `Gi`, `M`, or `G`.

**Priority**: P0 (Critical)
**Rationale**: Kubernetes interprets bare memory quantities as bytes, so accepting `1` creates misleading resource requests and can produce failed or nonsensical workflow pods.

#### Scenario: Component form rejects bare memory value
- **Given** a user is creating or editing a pipeline component
- **When** the user enters `1` in the memory field and submits the form
- **Then** the UI SHALL block submission and explain that memory must include a unit such as `512Mi` or `1Gi`.

#### Scenario: Component form accepts valid memory value
- **Given** a user is creating or editing a pipeline component
- **When** the user enters `512Mi` or `1Gi` in the memory field and submits the form
- **Then** the UI SHALL allow submission if all other fields are valid.

#### Scenario: API rejects bypassed unsafe memory value
- **Given** a client bypasses the UI and submits component or pipeline resources with memory `1`
- **When** the backend validates the request
- **Then** the backend SHALL reject the request with a validation error instead of storing or submitting the unsafe value.

#### Scenario: API rejects non-positive resource values
- **Given** a client bypasses the UI and submits CPU `0`, memory `0Gi`, or disk `0Gi`
- **When** the backend validates the request
- **Then** the backend SHALL reject the request with a validation error.

### Requirement: CPU and GPU resource quantities keep valid Kubernetes semantics
The system SHALL allow positive Kubernetes CPU quantities and non-negative integer GPU counts while rejecting malformed CPU or GPU values.

**Priority**: P1 (High)
**Rationale**: CPU `1` is valid Kubernetes syntax while GPU counts should remain whole numbers, so validation must not incorrectly apply memory rules to every resource type.

#### Scenario: CPU accepts cores and millicores
- **Given** a user is editing component or node resources
- **When** the user enters CPU `500m` or `1`
- **Then** the system SHALL treat the value as valid.

#### Scenario: GPU rejects fractional count
- **Given** a user is editing component or node resources
- **When** the user enters GPU `0.5`
- **Then** the system SHALL block submission and explain that GPU must be a non-negative integer count.
