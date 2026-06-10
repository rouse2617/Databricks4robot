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
	sub, err := NewInternalSubscriber(bus.SubscribeOrClosed(), 4)
	if err != nil {
		t.Fatalf("NewInternalSubscriber: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
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
		receipt, perr := bus.publish(publishCtx, evJSON)
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
	sub, err := NewInternalSubscriber(bus.SubscribeOrClosed(), 4)
	if err != nil {
		t.Fatalf("NewInternalSubscriber: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
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
		receipt, perr := bus.publish(context.Background(), raw)
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
	sub, err := NewInternalSubscriber(bus.SubscribeOrClosed(), 1) // serial path
	if err != nil {
		t.Fatalf("NewInternalSubscriber: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
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
		receipt, perr := bus.publish(context.Background(), raw)
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
	sub, err := NewInternalSubscriber(bus.SubscribeOrClosed(), 1)
	if err != nil {
		t.Fatalf("NewInternalSubscriber: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
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
		receipt, perr := bus.publish(context.Background(), raw)
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
	// Fan-out only enqueues to active subscribers; register one and never
	// read so the channel depth grows by exactly 1 per publish.
	_, _ = bus.Subscribe()

	for i := 0; i < 2; i++ {
		raw, _ := json.Marshal(map[string]any{"event_seq": int64(i + 1), "asset_id": "depth"})
		if _, err := bus.publish(context.Background(), raw); err != nil {
			t.Fatalf("publish %d failed: %v", i, err)
		}
	}
	// In fan-out mode the gauge tracks the per-subscriber channel length
	// (set in publish). The exact prior value is not deterministic because
	// earlier tests in this binary may have left residual gauge state; we
	// only assert the publish loop populated the gauge with at least 2
	// messages sitting in the channel we just subscribed.
	after := readGaugeValue(t, metrics.OutboxInternalBusDepth)
	if after < 2 {
		t.Fatalf("expected bus depth gauge to reflect >=2 enqueued messages, got %v", after)
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

// TestInMemoryBus_FanOut_DeliversEachMessageToAllSubscribers verifies the
// fan-out / broadcast contract: every published event reaches every
// registered subscriber. The pre-refactor bug (C3) was a single shared
// channel acting as a competing-consumer queue; this test would have failed
// against that implementation.
func TestInMemoryBus_FanOut_DeliversEachMessageToAllSubscribers(t *testing.T) {
	bus := NewInMemoryBus(16)
	subA, unsubA := bus.Subscribe()
	defer unsubA()
	subB, unsubB := bus.Subscribe()
	defer unsubB()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// drain reads n messages from ch and signals completion.
	// t.Fatalf is forbidden on non-test goroutines; signal via doneCh.
	drain := func(ch <-chan internalMessage, n int, id string, errCh chan<- string, doneCh chan<- struct{}) {
		for i := 0; i < n; i++ {
			select {
			case <-ctx.Done():
				errCh <- id + ": ctx done"
				return
			case msg := <-ch:
				msg.ack <- nil
			}
		}
		doneCh <- struct{}{}
	}

	const total = 6
	errCh := make(chan string, 2)
	doneCh := make(chan struct{}, 2)
	go drain(subA, total, "subA", errCh, doneCh)
	go drain(subB, total, "subB", errCh, doneCh)

	for i := 0; i < total; i++ {
		receipt, err := bus.publish(ctx, []byte{byte(i)})
		if err != nil {
			t.Fatalf("publish %d: %v", i, err)
		}
		if _, err := receipt.Get(ctx); err != nil {
			t.Fatalf("receipt.Get %d: %v", i, err)
		}
	}

	for i := 0; i < 2; i++ {
		select {
		case e := <-errCh:
			t.Fatalf("drain failed: %s", e)
		case <-doneCh:
			// one subscriber finished its 6 messages
		case <-time.After(2 * time.Second):
			t.Fatalf("drain timed out")
		}
	}
}

// TestInMemoryBus_UnsubscribeStopsDelivery ensures callers that drop their
// subscription stop receiving events (no leak into a closed channel).
func TestInMemoryBus_UnsubscribeStopsDelivery(t *testing.T) {
	bus := NewInMemoryBus(4)
	sub, unsub := bus.Subscribe()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	go func() { _ = sub }() // pin reference

	unsub()

	// After unsubscribe, publish should not block. If it does, the ctx
	// timeout below trips and the test fails.
	if _, err := bus.publish(ctx, []byte("after-unsub")); err != nil {
		t.Fatalf("publish after unsubscribe: %v", err)
	}
}

// TestInMemoryBus_FanOut_NoSilentLoss proves that every subscriber receives
// every message — this is the fix for C3. The pre-refactor bus used a single
// shared channel where two subscribers would split the stream (competing
// consumer), silently losing events on both sides.
//
// We simulate the pre-fix behavior with a raw chan (competing consumer)
// and the post-fix behavior with InMemoryBus.Subscribe (fan-out), then
// assert that fan-out delivers all events to both subscribers.
func TestInMemoryBus_FanOut_NoSilentLoss(t *testing.T) {
	const total = 24

	// ── Pre-fix: competing consumer (single shared channel) ──
	competing := make(chan internalMessage, 1024)
	receivedA := 0
	receivedB := 0
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-competing:
				if !ok {
					return
				}
				receivedA++
				msg.ack <- nil
			}
		}
	}()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-competing:
				if !ok {
					return
				}
				receivedB++
				msg.ack <- nil
			}
		}
	}()
	for i := 0; i < total; i++ {
		msg := internalMessage{data: []byte{byte(i)}, ack: make(chan error, 1)}
		competing <- msg
	}
	time.Sleep(100 * time.Millisecond)

	// ── Post-fix: fan-out bus with Subscribe ──
	bus := NewInMemoryBus(16)
	chA, unsubA := bus.Subscribe()
	chB, unsubB := bus.Subscribe()

	fanOutA := 0
	fanOutB := 0
	doneA := make(chan struct{})
	doneB := make(chan struct{})

	go func() {
		for fanOutA < total {
			select {
			case <-ctx.Done():
				return
			case msg := <-chA:
				fanOutA++
				msg.ack <- nil
			}
		}
		close(doneA)
	}()
	go func() {
		for fanOutB < total {
			select {
			case <-ctx.Done():
				return
			case msg := <-chB:
				fanOutB++
				msg.ack <- nil
			}
		}
		close(doneB)
	}()

	for i := 0; i < total; i++ {
		receipt, err := bus.publish(ctx, []byte{byte(i)})
		if err != nil {
			t.Fatalf("publish %d: %v", i, err)
		}
		if _, err := receipt.Get(ctx); err != nil {
			t.Fatalf("receipt.Get %d: %v", i, err)
		}
	}

	<-doneA
	<-doneB
	unsubA()
	unsubB()

	// Competing consumer: messages split roughly 50/50, each < total.
	if receivedA+receivedB != total {
		t.Fatalf("competing consumer: expected %d total deliveries, got A=%d B=%d sum=%d",
			total, receivedA, receivedB, receivedA+receivedB)
	}

	// Fan-out: both subscribers received ALL messages — the fix.
	if fanOutA != total {
		t.Fatalf("fan-out subscriber A got %d/%d — lost messages!", fanOutA, total)
	}
	if fanOutB != total {
		t.Fatalf("fan-out subscriber B got %d/%d — lost messages!", fanOutB, total)
	}
}
