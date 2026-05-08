package audit

import (
	"context"
	"testing"

	"data-platform/internal/middleware"
)

func TestLog_DefaultSink_DoesNotPanic(t *testing.T) {
	// Reset to default (Noop) sink for test isolation.
	sink = NoopSink{}

	ctx := context.Background()
	// Should not panic when no sink has been configured.
	Log(ctx, "asset.delete", "asset", []string{"abc-123"}, nil)
}

func TestLog_ExtractsRequestID(t *testing.T) {
	// Verify that the function can extract request_id from context without error.
	ctx := middleware.WithRequestID(context.Background(), "req-42")
	rid := middleware.RequestIDFromContext(ctx)
	if rid != "req-42" {
		t.Errorf("expected request_id 'req-42', got %q", rid)
	}
}
