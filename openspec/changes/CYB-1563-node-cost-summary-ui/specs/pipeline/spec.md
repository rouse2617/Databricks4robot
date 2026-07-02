# Pipeline Execution Node Cost Summary

## ADDED Requirements

### Requirement: Workflow detail shows node duration and cost summary

The workflow execution detail page SHALL show a run-level node summary table
before the DAG or timeline view.

#### Scenario: User opens a workflow detail with nodes

- **Given** a workflow execution has one or more nodes
- **When** the user opens the workflow detail page
- **Then** the page shows a "节点耗时 / 花费" summary
- **And** each row shows node name, status, pod count, duration, estimated cost,
  start time, and finish time when available

#### Scenario: Cost data is not available yet

- **Given** the backend does not provide node estimated cost
- **When** the node summary renders
- **Then** the cost cell shows a clear pending value
- **And** the rest of the node summary remains usable

#### Scenario: Terminal node is missing finished time

- **Given** a node is in a terminal phase such as Succeeded, Failed, or Error
- **And** the backend response has `startedAt` but no `finishedAt`
- **When** the node summary calculates duration
- **Then** it does not calculate duration against the current clock
- **And** it uses `estimatedDuration` when available, otherwise shows an empty
  duration value

#### Scenario: User needs to inspect a node

- **Given** a node appears in the summary table
- **When** the user clicks the node row or an action
- **Then** the existing node detail or logs UI opens for that node

#### Scenario: User wants to identify bottlenecks

- **Given** multiple nodes have durations or estimated costs
- **When** the user sorts the summary table by duration or cost
- **Then** the table orders nodes by the selected metric
