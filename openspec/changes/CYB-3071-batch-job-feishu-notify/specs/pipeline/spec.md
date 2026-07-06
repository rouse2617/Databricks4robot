## ADDED Requirements

### Requirement: 批量任务终态飞书通知
The system SHALL send a Feishu text notification exactly once when a batch (backfill) job transitions from a non-terminal status to a terminal status (`completed` or `failed`), regardless of outcome, including a summary of total/succeeded/failed item counts and a link to the job detail page.

**Priority**: P1 (High)
**Rationale**: Users currently have no way to learn that a batch job finished without repeatedly reloading the page.

#### Scenario: Job completes and a summary notification is sent
- **Given** a batch job is `running` and a Feishu webhook URL is configured
- **When** the job's status transitions to `completed` or `failed`
- **Then** the system sends exactly one Feishu message containing the job name, terminal status, total/succeeded/failed counts, and a link to the job detail page

#### Scenario: Concurrent observation of the same transition does not double-send
- **Given** multiple backend instances observe the same batch job crossing into a terminal status at nearly the same time
- **When** each instance attempts to claim the notification for that job
- **Then** only one instance successfully sends the notification, and the others no-op

#### Scenario: Feature is disabled without a configured webhook
- **Given** no Feishu webhook URL is configured for batch job notifications
- **When** a batch job transitions to a terminal status
- **Then** the system records the terminal transition normally but sends no network request and raises no error
