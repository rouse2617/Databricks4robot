package cdc

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestKafkaSource_Run_DecodesAndRoutes verifies that a successful batch is
// decoded and dispatched to the router.
func TestKafkaSource_Run_DecodesAndRoutes(t *testing.T) {
	handler := &recordingHandler{}
	runtime := &Runtime{
		Config: RuntimeConfig{Enabled: true, SourceDriver: "kafka"},
		Handlers: map[string]BatchHandler{
			"data4cyber.public.assets": handler,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var pollCalls int
	customPoller := &funcPoller{
		pollFn: func(ctx context.Context) ([]KafkaRecord, error) {
			pollCalls++
			if pollCalls == 1 {
				return []KafkaRecord{
					{
						Topic: "data4cyber.public.assets",
						Key:   map[string]any{"asset_id": "a1"},
						Value: []byte(`{"payload":{"op":"u","before":null,"after":{"asset_id":"a1"},"source":{"table":"assets"}}}`),
					},
				}, nil
			}
			cancel()
			return nil, context.Canceled
		},
	}

	source := &KafkaSource{Poller: customPoller}
	err := source.Run(ctx, runtime)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if len(handler.batches) != 1 || len(handler.batches[0]) != 1 {
		t.Fatalf("unexpected handler batches: %+v", handler.batches)
	}
	if handler.batches[0][0].Table != "assets" {
		t.Fatalf("unexpected event: %+v", handler.batches[0][0])
	}
}

// TestKafkaSource_Run_GracefulShutdown verifies that cancelling the context
// causes Run to return context.Canceled without processing further records.
func TestKafkaSource_Run_GracefulShutdown(t *testing.T) {
	handler := &recordingHandler{}
	runtime := &Runtime{
		Config: RuntimeConfig{Enabled: true, SourceDriver: "kafka"},
		Handlers: map[string]BatchHandler{
			"topic.assets": handler,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	poller := &funcPoller{
		pollFn: func(ctx context.Context) ([]KafkaRecord, error) {
			// Cancel on first call so Run exits immediately.
			cancel()
			return nil, context.Canceled
		},
	}

	source := &KafkaSource{Poller: poller}
	err := source.Run(ctx, runtime)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled on shutdown, got %v", err)
	}
	// No events should have been dispatched.
	if len(handler.batches) != 0 {
		t.Fatalf("expected no batches on shutdown, got %d", len(handler.batches))
	}
}

// TestKafkaSource_Run_ClosesPoller verifies that Run calls Close() on the
// poller when it exits, if the poller implements io.Closer.
func TestKafkaSource_Run_ClosesPoller(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	closed := false
	customPoller := &closablePollerWithFunc{
		closeFn: func() error {
			closed = true
			return nil
		},
		pollFn: func(ctx context.Context) ([]KafkaRecord, error) {
			cancel()
			return nil, context.Canceled
		},
	}

	source := &KafkaSource{Poller: customPoller}
	runtime := &Runtime{
		Config:   RuntimeConfig{Enabled: true},
		Handlers: map[string]BatchHandler{},
	}
	_ = source.Run(ctx, runtime)

	if !closed {
		t.Fatal("expected poller.Close() to be called on Run exit")
	}
}

// TestKafkaSource_Run_RetryOnTransientError verifies that transient poll errors
// trigger exponential backoff retries and that Run eventually succeeds after
// the transient errors resolve.
func TestKafkaSource_Run_RetryOnTransientError(t *testing.T) {
	handler := &recordingHandler{}
	runtime := &Runtime{
		Config: RuntimeConfig{Enabled: true, SourceDriver: "kafka"},
		Handlers: map[string]BatchHandler{
			"topic.assets": handler,
		},
	}

	// Override retryDelays with very short durations for the test.
	origDelays := retryDelays
	retryDelays = []time.Duration{1 * time.Millisecond, 2 * time.Millisecond, 5 * time.Millisecond, 10 * time.Millisecond}
	defer func() { retryDelays = origDelays }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	callCount := 0
	poller := &funcPoller{
		pollFn: func(ctx context.Context) ([]KafkaRecord, error) {
			callCount++
			switch callCount {
			case 1, 2:
				// Simulate transient errors.
				return nil, errors.New("broker unavailable")
			case 3:
				// Return a successful batch.
				return []KafkaRecord{
					{
						Topic: "topic.assets",
						Key:   map[string]any{"asset_id": "a2"},
						Value: []byte(`{"payload":{"op":"c","before":null,"after":{"asset_id":"a2"},"source":{"table":"assets"}}}`),
					},
				}, nil
			default:
				// Cancel so Run exits cleanly.
				cancel()
				return nil, context.Canceled
			}
		},
	}

	source := &KafkaSource{Poller: poller}
	err := source.Run(ctx, runtime)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled after retry success, got %v", err)
	}
	// The successful batch (call 3) should have been dispatched.
	if len(handler.batches) != 1 {
		t.Fatalf("expected 1 batch after retry, got %d", len(handler.batches))
	}
	// Verify we retried: callCount should be at least 4 (2 errors + 1 success + 1 cancel).
	if callCount < 4 {
		t.Fatalf("expected at least 4 poll calls (2 errors + 1 success + 1 cancel), got %d", callCount)
	}
}

// funcPoller is a test helper that delegates Poll to a function.
type funcPoller struct {
	pollFn func(ctx context.Context) ([]KafkaRecord, error)
}

func (f *funcPoller) Poll(ctx context.Context) ([]KafkaRecord, error) {
	return f.pollFn(ctx)
}

// closablePollerWithFunc is a test helper that implements both KafkaPoller and
// io.Closer via injected functions.
type closablePollerWithFunc struct {
	pollFn  func(ctx context.Context) ([]KafkaRecord, error)
	closeFn func() error
}

func (c *closablePollerWithFunc) Poll(ctx context.Context) ([]KafkaRecord, error) {
	return c.pollFn(ctx)
}

func (c *closablePollerWithFunc) Close() error {
	return c.closeFn()
}
