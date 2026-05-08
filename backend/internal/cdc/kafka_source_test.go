package cdc

import (
	"context"
	"errors"
	"strings"
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
	var commitCalls int
	customPoller := &funcPoller{
		pollFn: func(ctx context.Context) ([]KafkaRecord, error) {
			pollCalls++
			if pollCalls == 1 {
				return []KafkaRecord{
					{
						Topic: "data4cyber.public.assets",
						Key:   map[string]any{"asset_id": "a1"},
						Value: []byte(`{"payload":{"op":"u","before":null,"after":{"asset_id":"a1"},"source":{"table":"assets"}}}`),
						Commit: func(context.Context) error {
							commitCalls++
							return nil
						},
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
	if commitCalls != 1 {
		t.Fatalf("expected 1 successful commit, got %d", commitCalls)
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

func TestKafkaSource_Run_HandlerErrorParksToDLQAndCommitsPoisonRecord(t *testing.T) {
	expectedErr := errors.New("handler failed")
	runtime := &Runtime{
		Config: RuntimeConfig{Enabled: true, SourceDriver: "kafka"},
		Handlers: map[string]BatchHandler{
			"topic.assets": BatchHandlerFunc(func(context.Context, []ChangeEvent) error {
				return expectedErr
			}),
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var pollCalls int
	var commitCalls int
	sink := &fakeErrorSink{}
	poller := &funcPoller{
		pollFn: func(ctx context.Context) ([]KafkaRecord, error) {
			pollCalls++
			if pollCalls > 1 {
				cancel()
				return nil, context.Canceled
			}
			return []KafkaRecord{
				{
					Topic: "topic.assets",
					Key:   map[string]any{"asset_id": "a3"},
					Value: []byte(`{"payload":{"op":"u","before":null,"after":{"asset_id":"a3"},"source":{"table":"assets"}}}`),
					Commit: func(context.Context) error {
						commitCalls++
						return nil
					},
				},
			}, nil
		},
	}

	source := &KafkaSource{Poller: poller, ErrorSink: sink}
	err := source.Run(ctx, runtime)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
	if commitCalls != 1 {
		t.Fatalf("expected poison record commit after handler error, got %d", commitCalls)
	}
	if len(sink.records) != 1 {
		t.Fatalf("expected 1 dlq record, got %d", len(sink.records))
	}
	if sink.records[0].Stage != FailureStageRoute {
		t.Fatalf("expected route stage record, got %s", sink.records[0].Stage)
	}
}

func TestKafkaSource_Run_CommitErrorFailsRun(t *testing.T) {
	expectedErr := errors.New("commit failed")
	handler := &recordingHandler{}
	runtime := &Runtime{
		Config: RuntimeConfig{Enabled: true, SourceDriver: "kafka"},
		Handlers: map[string]BatchHandler{
			"topic.assets": handler,
		},
	}

	poller := &funcPoller{
		pollFn: func(ctx context.Context) ([]KafkaRecord, error) {
			return []KafkaRecord{
				{
					Topic: "topic.assets",
					Key:   map[string]any{"asset_id": "a4"},
					Value: []byte(`{"payload":{"op":"u","before":null,"after":{"asset_id":"a4"},"source":{"table":"assets"}}}`),
					Commit: func(context.Context) error {
						return expectedErr
					},
				},
			}, nil
		},
	}

	source := &KafkaSource{Poller: poller}
	err := source.Run(context.Background(), runtime)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected commit error %v, got %v", expectedErr, err)
	}
	if len(handler.batches) != 1 {
		t.Fatalf("expected handler to process batch before commit failure, got %d", len(handler.batches))
	}
}

func TestKafkaSource_Run_StopsAfterConsecutiveFailureThreshold(t *testing.T) {
	runtime := &Runtime{
		Config: RuntimeConfig{Enabled: true, SourceDriver: "kafka"},
		Handlers: map[string]BatchHandler{
			"topic.assets": BatchHandlerFunc(func(context.Context, []ChangeEvent) error {
				return errors.New("route failed")
			}),
		},
	}
	poller := &funcPoller{
		pollFn: func(context.Context) ([]KafkaRecord, error) {
			return []KafkaRecord{
				{
					Topic: "topic.assets",
					Key:   map[string]any{"asset_id": "a9"},
					Value: []byte(`{"payload":{"op":"u","before":null,"after":{"asset_id":"a9"},"source":{"table":"assets"}}}`),
					Commit: func(context.Context) error {
						return nil
					},
				},
			}, nil
		},
	}
	source := &KafkaSource{
		Poller:                 poller,
		ErrorSink:              &fakeErrorSink{},
		MaxConsecutiveFailures: 3,
	}
	err := source.Run(context.Background(), runtime)
	if err == nil || !strings.Contains(err.Error(), "exceeded consecutive failure threshold") {
		t.Fatalf("expected threshold error, got %v", err)
	}
}

type BatchHandlerFunc func(context.Context, []ChangeEvent) error

func (f BatchHandlerFunc) HandleBatch(ctx context.Context, events []ChangeEvent) error {
	return f(ctx, events)
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

type fakeErrorSink struct {
	records []FailedRecord
}

func (f *fakeErrorSink) Write(_ context.Context, record FailedRecord) error {
	f.records = append(f.records, record)
	return nil
}
