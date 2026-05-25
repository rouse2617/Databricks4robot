## ADDED Requirements

### Requirement: 分层创建 clip
The system SHALL create a child clip asset under a parent segment via layered API.

**Priority**: P0 (Critical)
**Rationale**: clip 是分层 API 最常用的类型之一，通用 Create 无法自动写 asset_relations 边。

#### Scenario: 在 segment 下创建 clip 成功
- **Given** 存在 asset_type=segment 的资产 seg_001
- **When** POST /api/v1/assets/seg_001/clips with `{start_timestamp_ns, end_timestamp_ns, split_method: "manual"}`
- **Then** 返回 201 + asset_id，asset_type 为 clip，parent_asset_id = seg_001，asset_relations 写入 split_from 边

#### Scenario: 父不是 segment 时创建 clip 被拒绝
- **Given** 存在 asset_type=task 的资产 task_001
- **When** POST /api/v1/assets/task_001/clips
- **Then** 返回 422，提示 L2 不变式违反

### Requirement: 分层创建 action
The system SHALL create a child action asset under a parent segment (L2) or task (L3).

**Priority**: P1 (High)
**Rationale**: action 是 L2/L3 双重语义类型，需区分父子关系。

#### Scenario: 在 segment 下创建 L2 action 成功
- **Given** 存在 asset_type=segment 的资产
- **When** POST /api/v1/assets/seg_001/actions
- **Then** 返回 201，asset_type=action，parent=seg_001

#### Scenario: 在 task 下创建 L3 action 成功
- **Given** 存在 asset_type=task 的资产
- **When** POST /api/v1/assets/task_001/actions
- **Then** 返回 201，asset_type=action，parent=task_001

### Requirement: 分层创建 frame
The system SHALL create a child frame asset under a parent segment.

**Priority**: P1 (High)
**Rationale**: frame 作为采样产物需通过 API 创建。

#### Scenario: 在 segment 下创建 frame 成功
- **Given** 存在 asset_type=segment 的资产
- **When** POST /api/v1/assets/seg_001/frames
- **Then** 返回 201，asset_type=frame，parent=seg_001

### Requirement: 分层创建 task
The system SHALL create a child task asset under a parent segment or task.

**Priority**: P1 (High)
**Rationale**: task 支持递归嵌套（subtask）。

#### Scenario: 在 segment 下创建 task 成功
- **Given** 存在 asset_type=segment 的资产
- **When** POST /api/v1/assets/seg_001/tasks
- **Then** 返回 201，asset_type=task，parent=seg_001

#### Scenario: task 下创建 subtask 成功
- **Given** 存在 asset_type=task 的资产 task_001
- **When** POST /api/v1/assets/task_001/tasks
- **Then** 返回 201，asset_type=task，parent=task_001（L5 允许 task→task 递归）

### Requirement: split_method 决定边类型
The system SHALL infer `asset_relations.relation_type` from `split_method` per the decision table in §3.2.1.

**Priority**: P0 (Critical)
**Rationale**: 边类型决定 lineage 查询的准确性；算法产物必须标记为 derived_from。

#### Scenario: algo: 前缀 → derived_from
- **Given** split_method = "algo:hand_track@2.0", run_id = "R001"
- **When** POST layered API
- **Then** asset_relations type = 'derived_from'，metadata 带 run_id

#### Scenario: manual → split_from
- **Given** split_method = "manual"
- **When** POST layered API
- **Then** asset_relations type = 'split_from'

#### Scenario: 缺失 split_method → split_from
- **Given** split_method 未提供
- **When** POST layered API
- **Then** asset_relations type = 'split_from'（保守默认）

#### Scenario: 非法父类型返回 422
- **Given** 父资产类型不符合 L1-L7（clip 的父不是 segment）
- **When** POST layered API
- **Then** 返回 422 HierarchyViolation
