package middleware

import (
	"context"
	"log/slog"
)

type ctxKey string

const requestIDKey ctxKey = "request_id"

// WithRequestID returns a new context carrying the request ID.
func WithRequestID(ctx context.Context, rid string) context.Context {
	return context.WithValue(ctx, requestIDKey, rid)
}

// RequestIDFromContext extracts the request ID from context.
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// L returns a slog.Logger with the request_id from context attached.
// Use this in usecase/repo layers to get traced logging:
//
//	log := middleware.L(ctx)
//	log.Info("doing something", "asset_id", id)
func L(ctx context.Context) *slog.Logger {
	rid := RequestIDFromContext(ctx)
	if rid == "" {
		return slog.Default()
	}
	return slog.Default().With("request_id", rid)
}
