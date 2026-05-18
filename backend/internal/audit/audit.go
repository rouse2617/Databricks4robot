package audit

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/CyberOrigin2077/cyber-databrew/internal/middleware"
)

// Sink is the storage-agnostic interface used to persist audit events.
//
// The package keeps a single global sink so handlers can call audit.Log()
// without threading a dependency through every layer. Implementations live
// in the storage packages (e.g. internal/postgres) and a NoopSink is used
// when the active backend has no audit storage.
type Sink interface {
	WriteEvent(ctx context.Context, e Event) error
}

// Event is a single audit record. resource_ids is JSON-serialized as a string
// array; summary is serialized to a JSON object.
type Event struct {
	Actor          string
	Action         string
	ResourceType   string
	ResourceIDs    []string
	RequestSummary json.RawMessage
	RequestID      string
}

// NoopSink discards all events; used when the active backend doesn't persist audit.
type NoopSink struct{}

// WriteEvent implements Sink.
func (NoopSink) WriteEvent(_ context.Context, _ Event) error { return nil }

// sink holds the configured sink. Defaults to NoopSink so Log() never panics
// even before Init has been called.
var sink Sink = NoopSink{}

// Init sets the audit sink. Must be called once at startup.
func Init(s Sink) {
	if s == nil {
		s = NoopSink{}
	}
	sink = s
}

// Log records an audit event. It extracts actor and request_id from context.
// Errors are logged but never returned — audit must not break the request flow.
func Log(ctx context.Context, action, resourceType string, resourceIDs []string, summary map[string]any) {
	actor := "dev-token" // Phase 0 static token; will be replaced by OIDC subject in Phase 0.5.
	requestID := middleware.RequestIDFromContext(ctx)

	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		summaryJSON = []byte("{}")
	}

	if err := sink.WriteEvent(ctx, Event{
		Actor:          actor,
		Action:         action,
		ResourceType:   resourceType,
		ResourceIDs:    resourceIDs,
		RequestSummary: summaryJSON,
		RequestID:      requestID,
	}); err != nil {
		slog.Error("audit: failed to write event", "action", action, "err", err)
	}
}
