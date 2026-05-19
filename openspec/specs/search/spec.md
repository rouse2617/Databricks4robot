# search Specification

## Purpose

Provide full-text and structured search over asset metadata via Elasticsearch.

## Requirements

### Requirement: Asset search

The system SHALL support searching assets by name, tags, status, and free-text via `/api/v1/search/assets`.

### Requirement: Pagination

The system SHALL return paginated results with `page`, `page_size`, and `next_token`.

### Requirement: Sync consistency

The system SHALL maintain near-real-time consistency between PostgreSQL and Elasticsearch via the outbox relay.
