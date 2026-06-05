## MODIFIED Requirements

### Requirement: Estimated run cost must remain conservative and credible

- **Before**: The system MAY estimate run cost from Argo `resourcesDuration` using a default GPU machine profile when node metadata is absent.
- **After**: The system SHALL only apply machine pricing when the node metadata is sufficient to identify a pricing profile, and SHALL avoid inflated estimated cost when that metadata is missing or ambiguous.
- **Reason**: Operators must be able to trust estimated cost as an approximate operational signal; absurd totals break that trust.

**Priority**: P1 (High)
**Rationale**: Cost is displayed directly in pipeline execution records and is used to evaluate run quality and resource policy.

#### Scenario: Short run without explicit instance metadata
- **Given** a pipeline run contains nodes with Argo `resourcesDuration`
- **And** the node metadata does not explicitly identify instance type or GPU usage
- **When** DataBrew computes estimated run cost
- **Then** the system does not apply GPU pricing by default
- **And** the returned total remains conservative rather than inflated

#### Scenario: Explicit GPU node keeps GPU pricing
- **Given** a pipeline run node includes explicit GPU instance metadata
- **When** DataBrew computes estimated node and run cost
- **Then** the system applies the configured GPU pricing for that node

#### Scenario: Execution list shows corrected estimated cost
- **Given** a pipeline run has corrected backend estimated cost
- **When** the user opens the pipeline execution list or run detail
- **Then** the displayed estimated cost matches the corrected backend value
- **And** the UI does not show impossible total cost for short runs
