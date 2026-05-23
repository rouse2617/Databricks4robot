## ADDED Requirements

### Requirement: Compliance lineage traversal
The system SHALL provide read-only lineage traversal from a starting asset across recorded asset dependency relationships.

**Priority**: P1 (High)
**Rationale**: Compliance operators need to understand how an asset depends on upstream sources and which downstream assets may be affected by it.

#### Scenario: Trace upstream dependencies
- **Given** relation data connects a starting asset to parent assets
- **When** a client requests upstream lineage
- **Then** the response SHALL contain parent-side related assets with direction and depth metadata

#### Scenario: Trace downstream dependents
- **Given** relation data connects a starting asset to child assets
- **When** a client requests downstream lineage
- **Then** the response SHALL contain child-side related assets with direction and depth metadata

#### Scenario: Trace both directions
- **Given** relation data exists on both sides of a starting asset
- **When** a client requests bidirectional lineage
- **Then** the response SHALL contain both upstream and downstream nodes without mutating catalog state

#### Scenario: Default dependency relation types
- **Given** relation data includes dependency edges and version-only edges
- **When** a client searches lineage without a relation type filter
- **Then** the system SHALL traverse dependency/structural relation types by default
- **And** the system SHALL NOT silently include unrelated relation types in the default result

#### Scenario: Explicit relation type filter
- **Given** a client provides supported relation types
- **When** the client searches lineage
- **Then** the system SHALL limit traversal to those relation types

### Requirement: Lineage depth bounding
The system SHALL bound recursive lineage traversal by a validated maximum depth.

**Priority**: P1 (High)
**Rationale**: Asset relation graphs can be deep or cyclic, so lineage search must avoid unbounded recursion and oversized responses.

#### Scenario: Default bounded traversal
- **Given** a client omits an explicit depth
- **When** the client searches lineage
- **Then** the system SHALL use a documented default depth instead of traversing without a bound

#### Scenario: Requested depth limits recursion
- **Given** related assets exist beyond the requested depth
- **When** the client searches lineage with a smaller depth
- **Then** the response SHALL include only nodes within the requested depth

#### Scenario: Cyclic relations do not repeat forever
- **Given** relation data contains a cycle
- **When** the client searches lineage
- **Then** the system SHALL stop traversal without repeating the same path indefinitely

### Requirement: Lineage empty results
The system SHALL return a successful empty lineage response when no related assets match the request.

**Priority**: P1 (High)
**Rationale**: Absence of related lineage is a valid compliance result and should not be confused with a server or validation failure.

#### Scenario: No related assets
- **Given** the starting asset has no matching upstream or downstream relations
- **When** the client searches lineage
- **Then** the response SHALL contain an empty node list and a zero count

### Requirement: Lineage search validation
The system SHALL reject malformed lineage search filters with explicit client errors.

**Priority**: P1 (High)
**Rationale**: Invalid traversal inputs should not produce confusing empty results or expensive database queries.

#### Scenario: Missing starting asset
- **Given** a client omits the starting asset identifier
- **When** the client searches lineage
- **Then** the system SHALL return `400 INVALID_ARGUMENT`

#### Scenario: Invalid direction
- **Given** a client provides an unsupported traversal direction
- **When** the client searches lineage
- **Then** the system SHALL return `400 INVALID_ARGUMENT`

#### Scenario: Invalid depth
- **Given** a client provides a non-integer or non-positive depth
- **When** the client searches lineage
- **Then** the system SHALL return `400 INVALID_ARGUMENT`
