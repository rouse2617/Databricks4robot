# Spec Delta — asset-management (CYB-3267)

## MODIFIED Requirements

### Requirement: 资产 Action 列表读取
The system SHALL return an asset's actions on `GET /assets/:id/actions`,
including each action's `task_id`, without error.

- **Before**: The read errored (HTTP 500) because the row scan skipped the `task_id` column, leaving the SELECT column count and Scan destination count mismatched.
- **After**: The read succeeds; `task_id` is populated and adjacent fields (`tenant_id`, `project_id`) are not shifted.
- **Reason**: Bug 1 — `scanAction` was missing a destination; the Action 时间轴 tab was unusable for every segment.

**Priority**: P0 (Critical)
**Rationale**: Every action read 500s; blocks the Action timeline UI and any consumer of the actions list.

#### Scenario: 列出含 task_id 的 action
- **Given** 某资产存在一条带 `task_id` 的 action
- **When** 调用 `GET /assets/:id/actions`
- **Then** 返回 200，列表包含该 action，其 `task_id` 正确，且 `tenant_id`/`project_id` 未错位

#### Scenario: 无 action 的资产
- **Given** 某资产没有任何 action
- **When** 调用 `GET /assets/:id/actions`
- **Then** 返回 200 且列表为空（不再 500）

### Requirement: 资产创建时长一致性
The system SHALL set an asset's `duration_ms` at create time so that
`duration_sec` and `duration_ms` are consistent in the create response.

- **Before**: `Create` set only `duration_sec`; `duration_ms` was derived downstream by canonicalization (CYB-3226).
- **After**: `Create` sets `duration_ms` directly from the timestamp span, matching child-asset creation; the derived `duration_sec` is unchanged.
- **Reason**: Bug 2 — consistency/robustness (the `duration_sec=0` symptom was already resolved by CYB-3226).

**Priority**: P2 (Nice-to-have)
**Rationale**: Defensive alignment of the create site with `CreateChildAsset`; avoids relying on the float round-trip.

#### Scenario: 创建资产返回正确时长
- **Given** 一个 `end - start = 2,000,000,000 ns`（2 秒）的创建请求
- **When** 调用 `POST /assets`
- **Then** 返回资产的 `duration_ms = 2000` 且 `duration_sec = 2`
