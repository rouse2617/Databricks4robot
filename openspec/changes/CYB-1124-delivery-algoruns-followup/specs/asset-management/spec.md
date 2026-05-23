## MODIFIED Requirements

### Requirement: Algo run list pagination contract
- **Before**: The system SHALL normalize algo-runs list pagination for data retrieval but may return pagination metadata copied from the raw request.
- **After**: The system SHALL return pagination metadata that matches the effective normalized page used to retrieve algo-runs.
- **Reason**: Returning raw invalid pagination values creates a response envelope that contradicts the actual result slice.

**Priority**: P1 (High)
**Rationale**: Operators and UI pagination controls rely on the response envelope to describe the visible result page.

#### Scenario: Invalid page values are normalized in the response
- **Given** a client sends invalid pagination values below the supported range
- **When** the client lists algo runs
- **Then** the response SHALL report the normalized first page metadata and return the corresponding result slice

#### Scenario: Oversized page values are normalized in the response
- **Given** a client sends a page size larger than the supported maximum
- **When** the client lists algo runs
- **Then** the response SHALL report the default or supported effective page size rather than the unsupported requested value

#### Scenario: Valid page values are preserved
- **Given** a client sends valid pagination values
- **When** the client lists algo runs
- **Then** the response SHALL report the same valid pagination values used for the query
