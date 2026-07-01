package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/metrics"
)

func TestInternalSubscriberReceiveBatch_BatchesAcrossKeysAndAcksAll(t *testing.T) {
	bus := NewInMemoryBus(64)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go bus.Run(ctx)
	sub, err := NewInternalSubscriber(bus, 4)
	if err != nil {
		t.Fatalf("NewInternalSubscriber: %v", err)
	}

	defer cancel()

	var (
		mu         sync.Mutex
		batches    [][][]byte
		messagesIn = 12
	)
	go func() {
		_ = sub.ReceiveBatch(ctx, 16, 50*time.Millisecond, func(_ context.Context, payloads [][]byte) error {
			cp := make([][]byte, len(payloads))
			for i, p := range payloads {
				cp[i] = append([]byte(nil), p...)
			}
			mu.Lock()
			batches = append(batches, cp)
			mu.Unlock()
			return nil
		})
	}()

	type pubResult struct {
		assetID string
		err     error
	}
	resultCh := make(chan pubResult, messagesIn)

	publishCtx, publishCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer publishCancel()

	for i := 0; i < messagesIn; i++ {
		assetID := "asset-" + asciiInt(i%4)
		evJSON, _ := json.Marshal(map[string]any{
			"event_seq": int64(i + 1),
			"asset_id":  assetID,
		})
		receipt, perr := bus.Publish(publishCtx, evJSON)
		if perr != nil {
			t.Fatalf("publish %d failed: %v", i, perr)
		}
		go func(id string) {
			_, gerr := receipt.Get(publishCtx)
			resultCh <- pubResult{assetID: id, err: gerr}
		}(assetID)
	}

	totalsByAsset := map[string]int{}
	deadline := time.After(3 * time.Second)
	got := 0
	for got < messagesIn {
		select {
		case r := <-resultCh:
			if r.err != nil {
				t.Fatalf("publish %s ack returned error: %v", r.assetID, r.err)
			}
			totalsByAsset[r.assetID]++
			got++
		case <-deadline:
			t.Fatalf("timed out waiting for %d acks; got %d", messagesIn, got)
		}
	}

	if got != messagesIn {
		t.Fatalf("expected %d acks, got %d", messagesIn, got)
	}
	for k, v := range totalsByAsset {
		if v <= 0 {
			t.Fatalf("missing acks for %s: %d", k, v)
		}
	}

	// Cancel and let the worker drain.
	cancel()
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	batchCount := len(batches)
	totalDelivered := 0
	for _, b := range batches {
		totalDelivered += len(b)
	}
	mu.Unlock()

	if batchCount == 0 {
		t.Fatalf("expected at least one batch invocation, got 0")
	}
	if totalDelivered != messagesIn {
		t.Fatalf("expected handler to see %d total payloads across batches, got %d", messagesIn, totalDelivered)
	}
}

func TestInternalSubscriberReceiveBatch_PerAssetOrderingPreservedWithinWorker(t *testing.T) {
	bus := NewInMemoryBus(32)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go bus.Run(ctx)
	sub, err := NewInternalSubscriber(bus, 4)
	if err != nil {
		t.Fatalf("NewInternalSubscriber: %v", err)
	}

	defer cancel()

	const n = 8
	const assetID = "asset-fixed"

	var (
		mu              sync.Mutex
		seenForAsset    []int64
		batchesObserved int
		done            = make(chan struct{})
	)
	go func() {
		_ = sub.ReceiveBatch(ctx, n, 30*time.Millisecond, func(_ context.Context, payloads [][]byte) error {
			mu.Lock()
			batchesObserved++
			for _, p := range payloads {
				var ev struct {
					Seq     int64  `json:"event_seq"`
					AssetID string `json:"asset_id"`
				}
				if err := json.Unmarshal(p, &ev); err != nil {
					mu.Unlock()
					return err
				}
				if ev.AssetID == assetID {
					seenForAsset = append(seenForAsset, ev.Seq)
				}
			}
			if len(seenForAsset) >= n {
				select {
				case <-done:
				default:
					close(done)
				}
			}
			mu.Unlock()
			return nil
		})
	}()

	for i := 0; i < n; i++ {
		ev := map[string]any{"event_seq": int64(i + 1), "asset_id": assetID}
		raw, _ := json.Marshal(ev)
		receipt, perr := bus.Publish(context.Background(), raw)
		if perr != nil {
			t.Fatalf("publish %d: %v", i, perr)
		}
		go func() { _, _ = receipt.Get(context.Background()) }()
	}

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for all events for the asset")
	}

	cancel()

	mu.Lock()
	defer mu.Unlock()
	if len(seenForAsset) != n {
		t.Fatalf("expected %d events for asset, got %d", n, len(seenForAsset))
	}
	for i := 1; i < len(seenForAsset); i++ {
		if seenForAsset[i] <= seenForAsset[i-1] {
			t.Fatalf("ordering violated for asset: %v", seenForAsset)
		}
	}
}

func TestInternalSubscriberReceiveBatch_HandlerErrorAcksAllWithSameError(t *testing.T) {
	bus := NewInMemoryBus(8)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go bus.Run(ctx)
	sub, err := NewInternalSubscriber(bus, 1) // serial path
	if err != nil {
		t.Fatalf("NewInternalSubscriber: %v", err)
	}

	defer cancel()

	wantErr := errors.New("batch boom")
	go func() {
		_ = sub.ReceiveBatch(ctx, 4, 20*time.Millisecond, func(_ context.Context, _ [][]byte) error {
			return wantErr
		})
	}()

	const n = 3
	results := make(chan error, n)
	for i := 0; i < n; i++ {
		raw, _ := json.Marshal(map[string]any{"event_seq": int64(i + 1), "asset_id": "a1"})
		receipt, perr := bus.Publish(context.Background(), raw)
		if perr != nil {
			t.Fatalf("publish %d: %v", i, perr)
		}
		go func() {
			_, gerr := receipt.Get(context.Background())
			results <- gerr
		}()
	}

	for i := 0; i < n; i++ {
		select {
		case e := <-results:
			if e == nil || e.Error() != wantErr.Error() {
				t.Fatalf("expected handler error to be acked back to publisher, got %v", e)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for ack %d", i)
		}
	}
}

func TestInternalSubscriberReceiveBatch_FallbackWhenBatchSizeOne(t *testing.T) {
	bus := NewInMemoryBus(4)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go bus.Run(ctx)
	sub, err := NewInternalSubscriber(bus, 1)
	if err != nil {
		t.Fatalf("NewInternalSubscriber: %v", err)
	}

	defer cancel()

	var (
		mu    sync.Mutex
		got   int
		sizes []int
	)
	go func() {
		_ = sub.ReceiveBatch(ctx, 1, 0, func(_ context.Context, payloads [][]byte) error {
			mu.Lock()
			got++
			sizes = append(sizes, len(payloads))
			mu.Unlock()
			return nil
		})
	}()

	const n = 3
	for i := 0; i < n; i++ {
		raw, _ := json.Marshal(map[string]any{"event_seq": int64(i + 1), "asset_id": "a1"})
		receipt, perr := bus.Publish(context.Background(), raw)
		if perr != nil {
			t.Fatalf("publish %d: %v", i, perr)
		}
		go func() { _, _ = receipt.Get(context.Background()) }()
	}

	deadline := time.After(2 * time.Second)
	for {
		mu.Lock()
		c := got
		mu.Unlock()
		if c >= n {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for %d invocations, got %d", n, c)
		case <-time.After(10 * time.Millisecond):
		}
	}

	mu.Lock()
	defer mu.Unlock()
	for i, s := range sizes {
		if s != 1 {
			t.Fatalf("BatchSize=1 fallback should deliver 1 payload per call; call %d had %d", i, s)
		}
	}
}

func TestInMemoryBusPublish_UpdatesDepthMetric(t *testing.T) {
	bus := NewInMemoryBus(8)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Need at least one subscriber so publish() takes the fan-out branch.
	// The consumer deliberately blocks on a never-closed channel so the
	// subscriber's queue stays full and the depth gauge reflects pending
	// messages.
	block := make(chan struct{})
	defer close(block)
	sub, err := NewInternalSubscriber(bus, 1)
	if err != nil {
		t.Fatalf("NewInternalSubscriber: %v", err)
	}
	defer sub.Close()
	go func() {
		_ = sub.Receive(ctx, func(_ context.Context, _ []byte) error {
			<-block
			return nil
		})
	}()

	before := readGaugeValue(t, metrics.OutboxInternalBusDepth)
	for i := 0; i < 2; i++ {
		raw, _ := json.Marshal(map[string]any{"event_seq": int64(i + 1), "asset_id": "depth"})
		if _, err := bus.Publish(context.Background(), raw); err != nil {
			t.Fatalf("publish %d failed: %v", i, err)
		}
	}
	after := readGaugeValue(t, metrics.OutboxInternalBusDepth)
	if after < before+1 {
		t.Fatalf("expected bus depth gauge to be ≥ before+1 (one event queued in slow consumer's channel), before=%v after=%v", before, after)
	}
}

// asciiInt returns a single-digit string representation for small ints; small
// helper to keep test data terse without bringing in strconv.
func asciiInt(i int) string {
	if i < 0 || i > 9 {
		return "?"
	}
	return string(rune('0' + i))
}

// TestInMemoryBus_FanOutEverySubscriberSeesEveryEvent exercises the regression
// that Blocker #3 of PR #270 flagged: pre-PR the bus used a per-subscriber
// private channel; a refactor collapsed that into a single shared channel so
// 3 subscribers (asset ES, algo_run ES, delivery/lineage projector) competed
// for events and each silently dropped ~2/3 of the stream. This test asserts
// the original fan-out contract.
func TestInMemoryBus_FanOutEverySubscriberSeesEveryEvent(t *testing.T) {
	bus := NewInMemoryBus(64)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go bus.Run(ctx)

	const subscribers = 3
	const events = 25

	subs := make([]*InternalSubscriber, subscribers)
	for i := range subs {
		var err error
		subs[i], err = NewInternalSubscriber(bus, 1)
		if err != nil {
			t.Fatalf("subscriber %d: %v", i, err)
		}
	}

	var (
		mu   sync.Mutex
		seen [subscribers][]int64
	)
	done := make(chan struct{}, subscribers)
	for i := range subs {
		i := i
		go func() {
			_ = subs[i].Receive(ctx, func(_ context.Context, data []byte) error {
				var ev struct {
					Seq int64 `json:"event_seq"`
				}
				_ = json.Unmarshal(data, &ev)
				mu.Lock()
				seen[i] = append(seen[i], ev.Seq)
				ready := true
				for j := 0; j < subscribers; j++ {
					if len(seen[j]) < events {
						ready = false
						break
					}
				}
				mu.Unlock()
				if ready {
					select {
					case done <- struct{}{}:
					default:
					}
				}
				return nil
			})
		}()
	}

	for i := 0; i < events; i++ {
		raw, _ := json.Marshal(map[string]any{"event_seq": int64(i + 1), "asset_id": "a1"})
		receipt, err := bus.Publish(ctx, raw)
		if err != nil {
			t.Fatalf("publish %d: %v", i, err)
		}
		go func() { _, _ = receipt.Get(ctx) }()
	}

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out: subscribers did not all see %d events", events)
	}

	mu.Lock()
	defer mu.Unlock()
	for i := 0; i < subscribers; i++ {
		if len(seen[i]) != events {
			t.Errorf("subscriber %d received %d events, want %d (fan-out broken — one subscriber is starving)",
				i, len(seen[i]), events)
		}
		// Each subscriber must see the SAME event sequence; no events
		// lost to a competing-consumer race.
		for j, seq := range seen[i] {
			if seq != int64(j+1) {
				t.Errorf("subscriber %d event %d has seq=%d, want %d",
					i, j, seq, j+1)
			}
		}
	}
}
