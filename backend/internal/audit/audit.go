package audit

import (
	"context"
	"encoding/json"
	"log/slog"

	"data-platform/internal/middleware"
	"data-platform/internal/postgres"
)

// db holds the postgres client used for writing audit events.
var db *postgres.Client

// Init sets the postgres client for the audit package.
// Must be called once at startup before any Log calls.
func Init(pgClient *postgres.Client) {
	db = pgClient
}

// Log records an audit event. It extracts actor and request_id from context.
// Errors are logged but never returned — audit must not break the request flow.
func Log(ctx context.Context, action, resourceType string, resourceIDs []string, summary map[string]any) {
	if db == nil {
		slog.Warn("audit: not initialized, skipping", "action", action)
		return
	}

	actor := "dev-token" // Phase 0 static token; will be replaced by OIDC subject in Phase 0.5
	requestID := middleware.RequestIDFromContext(ctx)

	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		summaryJSON = []byte("{}")
	}

	const q = `
INSERT INTO audit_events (actor, action, resource_type, resource_ids, request_summary, request_id)
VALUES ($1, $2, $3, $4, $5, $6)`

	if err := db.Exec(ctx, q, actor, action, resourceType, resourceIDs, summaryJSON, requestID); err != nil {
		slog.Error("audit: failed to write event", "action", action, "err", err)
	}
}
