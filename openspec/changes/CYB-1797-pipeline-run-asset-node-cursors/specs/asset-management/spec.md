## MODIFIED Requirements

### Requirement: Pipeline run asset-node listing paginates consistently
- **Before**: Asset-node listing could filter subsequent pages by row ID while ordering by asset, node, cost, duration, or status, causing skipped or duplicated rows.
- **After**: The system SHALL paginate pipeline run asset-node results using cursor semantics that match the active ordering.
- **Reason**: Pipeline execution diagnostics require complete and repeatable per-asset/per-node visibility.

**Priority**: P1 (High)
**Rationale**: Incorrect pagination can hide failed or costly nodes from operational review.

#### Scenario: default ordering has no missing or duplicate rows
- **Given** a pipeline run has more asset-node rows than one page
- **When** a user fetches every page using the returned cursor
- **Then** each row appears exactly once
- **And** the combined pages contain the same rows as an unpaginated ordered read

#### Scenario: alternate ordering has matching cursor semantics
- **Given** a pipeline run has asset-node rows with cost, duration, or status ties
- **When** a user fetches pages using a non-default ordering
- **Then** the cursor continues from the last returned row for that same ordering
- **And** tied rows remain deterministic across pages

#### Scenario: cursor does not match requested ordering
- **Given** a cursor was generated for one ordering
- **When** a user sends that cursor with a different ordering
- **Then** the system MUST reject the request as invalid
- **And** the system SHALL NOT silently return a partially incorrect page
