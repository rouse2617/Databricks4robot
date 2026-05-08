package postgres

import (
	"context"
	"fmt"

	"data-platform/internal/audit"
)

// AuditSink persists audit events to the audit_events table.
type AuditSink struct {
	c *Client
}

// NewAuditSink wires an audit.Sink backed by Postgres.
func NewAuditSink(c *Client) *AuditSink { return &AuditSink{c: c} }

// WriteEvent implements audit.Sink.
func (s *AuditSink) WriteEvent(ctx context.Context, e audit.Event) error {
	const q = `
INSERT INTO audit_events (actor, action, resource_type, resource_ids, request_summary, request_id)
VALUES ($1, $2, $3, $4, $5, $6)`
	if err := s.c.db.Exec(ctx, q,
		e.Actor, e.Action, e.ResourceType, e.ResourceIDs, []byte(e.RequestSummary), e.RequestID,
	); err != nil {
		return fmt.Errorf("postgres AuditSink.WriteEvent: %w", err)
	}
	return nil
}
