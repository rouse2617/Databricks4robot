# Dashboard Spec Delta

## Modified Requirements

### Requirement: Dashboard Unavailable Metrics

When a subset of dashboard metrics is unavailable, the dashboard SHALL keep available metrics visible and show a user-facing unavailable state for missing metrics.

#### Scenario: Lakehouse backend is not configured

- **Given** lakehouse-backed dashboard APIs return an unavailable/configuration error
- **When** the dashboard renders
- **Then** available asset/event summary data remains visible
- **And** the primary alert explains that lakehouse metrics are temporarily unavailable
- **And** raw backend error strings are not the primary visible message
