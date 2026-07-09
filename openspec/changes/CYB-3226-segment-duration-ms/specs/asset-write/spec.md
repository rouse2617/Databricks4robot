# Asset Write Spec Delta — CYB-3226 Segment duration & write gates

## New Requirements

### Requirement: Persist a correct duration on every asset write
The system SHALL persist a `duration_ms` consistent with the asset's time range,
regardless of whether the caller supplied duration in milliseconds, in seconds,
or not at all.

#### Scenario: Segment created with seconds only
- **Given** a create request whose usecase sets `duration_sec` but not `duration_ms`
- **When** the asset is written
- **Then** the stored `duration_ms` equals `round(duration_sec * 1000)` and is > 0.

#### Scenario: Neither duration field provided
- **Given** a create request that sets neither `duration_ms` nor `duration_sec` but has `end_timestamp_ns > start_timestamp_ns`
- **When** the asset is written
- **Then** the stored `duration_ms` equals `(end_timestamp_ns - start_timestamp_ns) / 1_000_000`.

#### Scenario: duration_ms explicitly provided
- **Given** a create request that already sets `duration_ms` (e.g. child assets)
- **When** the asset is written
- **Then** the stored `duration_ms` is unchanged.

### Requirement: Validate the asset time range at write time
The system SHALL reject asset writes whose time range is empty, inverted, or
sub-millisecond.

#### Scenario: Inverted or empty range
- **Given** a create request with `end_timestamp_ns <= start_timestamp_ns`
- **When** the system validates the request
- **Then** it returns `422 INVALID_STATE` and does not create the asset.

#### Scenario: Sub-millisecond span
- **Given** a create request with `0 < (end - start) < 1_000_000 ns` (rounds to `duration_ms = 0`)
- **When** the system validates the request
- **Then** it returns `422 INVALID_STATE` and does not create the asset.

### Requirement: Warn on segment/parent time-range inconsistency
The system SHALL log a warning (without rejecting) when a segment's relative
offset range falls outside its parent raw_mcap's known duration.

#### Scenario: Segment offset exceeds parent duration
- **Given** a segment whose parent raw_mcap has a known `file_duration_ms > 0`
- **When** the segment's relative `[start,end]` offset exceeds `[0, file_duration_ms]` beyond tolerance
- **Then** the system logs a warning and still accepts the write (no rejection in this change).
