## ADDED Requirements

### Requirement: 提交前失败的运行 UI 准确反映"未创建 workflow"
The frontend SHALL distinguish, in the run detail degraded-mode banner and the batch node overview, a run/batch that failed before ever creating a runtime workflow from one whose workflow existed but is no longer available. It SHALL NOT imply TTL cleanup or "still syncing" for a terminal run/batch that never produced a workflow or nodes.

**Priority**: P2 (Medium)
**Rationale**: A run rejected before submission (e.g. by the resource guard) has no workflow and no nodes; wording implying TTL cleanup or in-progress syncing misleads users into thinking data is missing or still loading.

#### Scenario: Run detail banner for a pre-submission failure
- **Given** a run is terminally failed (`Failed`/`Error`) and never obtained an Argo workflow UID
- **When** the user opens its detail page (served from the DataBrew ledger)
- **Then** the banner states the workflow was never created and points to the diagnosis, rather than implying TTL cleanup

#### Scenario: Batch node overview for a terminal batch with no nodes
- **Given** a batch job has reached a terminal status and produced no node progress
- **When** the user views the node overview
- **Then** it shows a terminal empty state rather than "节点状态仍在同步中"

#### Scenario: Live/misclassified cases unchanged
- **Given** a run is still running, or a workflow genuinely existed and was TTL-cleaned
- **When** the user views the detail page or node overview
- **Then** the existing wording (in-progress / ledger-fallback) is unchanged
