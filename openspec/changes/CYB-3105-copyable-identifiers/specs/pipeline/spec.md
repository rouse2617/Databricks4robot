# Pipeline Spec Delta — CYB-3105

## ADDED Requirements

### Requirement: Identifier fields expose one-click copy

The run execution list and the batch job detail view SHALL provide a one-click copy
control for identifier fields, so users do not have to manually select text.

#### Scenario: Subtask/run name and asset_id are copyable

- **Given** the executions list (or a batch's subtask table) shows a run
- **When** the user hovers the run 名称 or its 资产/asset_id
- **Then** a one-click copy control SHALL be available
- **And** activating it SHALL copy the full value to the clipboard

#### Scenario: Batch header identifiers are copyable

- **Given** the batch job detail header
- **When** the user views the 批次 ID and 模板 fields
- **Then** each SHALL provide a one-click copy control that copies the full value
