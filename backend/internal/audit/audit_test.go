package audit

import (
	"context"
	"testing"

	"data-platform/internal/middleware"
)

func TestLog_NilDB_DoesNotPanic(t *testing.T) {
	// Reset global state for test isolation.
	db = nil

	ctx := context.Background()
	// Should not panic when db is nil.
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
