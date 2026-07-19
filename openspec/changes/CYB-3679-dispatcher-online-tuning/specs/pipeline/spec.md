# Pipeline Spec Delta — CYB-3679

## ADDED Requirements

### Requirement: Dispatcher tuning is per-cluster, online, and range-guarded
- The system SHALL read per-cluster dispatch settings (max_concurrency,
  submit_batch, rate_per_sec, paused) from dispatcher_configs at the start of
  every submit cycle; an edit SHALL take effect within one tick without a
  deploy. Values outside the guard ranges SHALL be rejected with 400.
  A failed config read SHALL keep the previous snapshot.

#### Scenario: Pause bites within one tick
- **Given** a running batch on cluster A
- **When** an operator sets paused=true for A
- **Then** the next cycle dispatches nothing for A and
  backend_dispatcher_channel_paused{cluster="A"} reports 1

#### Scenario: Fat-fingered rate is rejected
- **Given** a PUT with rate_per_sec=10000
- **When** the admin API validates it
- **Then** the request fails 400 and the stored config is unchanged

### Requirement: Operators can see configured vs effective and why
- The status API and UI SHALL show each cluster's configured concurrency,
  the governor's live effective concurrency, and a suppression reason
  ("paused" | "aimd_backoff") when effective < configured.

#### Scenario: AIMD backoff is visible
- **Given** a cluster whose governor halved concurrency under distress
- **When** the operator opens the 调度调参 page
- **Then** the row shows effective < configured with a 背压降速中 chip
