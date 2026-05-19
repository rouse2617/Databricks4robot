# asset-management Specification

## Purpose

Manage lifecycle of MCAP-oriented data assets: creation, metadata, tagging, algorithm execution tracking, and event emission.

## Requirements

### Requirement: Asset CRUD

The system SHALL provide create, read, update, list operations for assets with metadata fields (name, source, tags, status).

### Requirement: Tag management

The system SHALL validate tags against `backend/config/tag_registry.yaml` definitions (type, allowed values, max_length).

### Requirement: Algorithm lifecycle

The system SHALL track algorithm execution state per asset using a state machine driven by `backend/config/algo_registry.yaml`.

### Requirement: Event emission

The system SHALL emit `asset_events` to the outbox table within the same transaction as state changes.
