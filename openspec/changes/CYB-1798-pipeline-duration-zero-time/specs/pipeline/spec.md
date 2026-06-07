# Pipeline Spec Delta — CYB-1798

## Modified Requirements

### Pipeline asset-node duration is absent when timestamps are incomplete
The system SHALL report asset-node duration as absent when either start or finish timestamp is nil, zero, or invalid.

#### Scenario: zero start timestamp
- **Given** a pipeline node has a zero start timestamp and a non-zero finish timestamp
- **When** the system builds asset-node cost summary rows
- **Then** the corresponding `durationSeconds` value is absent

#### Scenario: valid timestamps
- **Given** a pipeline node has non-zero start and finish timestamps
- **And** finish is not before start
- **When** the system builds asset-node cost summary rows
- **Then** the corresponding `durationSeconds` value is the elapsed seconds
