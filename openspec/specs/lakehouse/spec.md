# lakehouse Specification

## Purpose

Provide query access to BigQuery-over-Iceberg lakehouse for analytical workloads on asset data.

## Requirements

### Requirement: Lakehouse query

The system SHALL proxy analytical queries to BigQuery via `/api/v1/lakehouse/*` endpoints.

### Requirement: Report generation

The system SHALL support report export from lakehouse query results.

### Requirement: Access control

The system SHALL enforce `X-Grace-Token` authentication on all lakehouse endpoints.
