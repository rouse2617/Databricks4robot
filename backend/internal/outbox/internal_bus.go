package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"hash/fnv"
	"sync"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/metrics"
	"golang.org/x/sync/errgroup"
)

// InMemoryBus is an in-process event bus for default local transport mode.
//
// It is a fan-out / broadcast bus: every subscriber receives every published
// event. This mirrors the semantics of a Pub/Sub topic with one subscription
// per subscriber, or a Kafka topic with one consumer group per subscriber —
// the "competing consumer" anti-pattern is avoided at the transport layer so
// downstream consumers (asset ES, algo_run ES, delivery eligibility projector,
// openlineage emitter, …) can each filter the full event stream without
// stealing events from siblings.
type InMemoryBus struct {
	mu      sync.RWMutex
	subs    []chan internalMessage
	bufSize int
}

// NewInMemoryBus creates a fan-out in-process bus. The buffer size applies
// to each subscriber's channel — a slow subscriber will block the publisher
// once its channel is full (intentional back-pressure so a stuck handler
// surfaces immediately rather than silently dropping events).
func NewInMemoryBus(buffer int) *InMemoryBus {
	if buffer <= 0 {
		buffer = 1024
	}
	return &InMemoryBus{bufSize: buffer}
}

// SubscribeOrClosed is a convenience for tests and one-shot consumers that
// don't need explicit unsubscribe semantics. Returns the private channel
// only; cleanup happens via garbage collection when the bus is dropped.
func (b *InMemoryBus) SubscribeOrClosed() <-chan internalMessage {
	ch, _ := b.Subscribe()
	return ch
}

// Subscribe registers a new subscriber and returns its private channel plus
// an unsubscribe function. Each subscriber must call unsubscribe when done so
// the bus stops fanning out to it and the channel is GC'd.
func (b *InMemoryBus) Subscribe() (<-chan internalMessage, func()) {
	if b == nil {
		ch := make(chan internalMessage)
		close(ch)
		return ch, func() {}
	}
	ch := make(chan internalMessage, b.bufSize)
	b.mu.Lock()
	b.subs = append(b.subs, ch)
	b.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			b.mu.Lock()
			for i, sub := range b.subs {
				if sub == ch {
					b.subs = append(b.subs[:i], b.subs[i+1:]...)
					break
				}
			}
			b.mu.Unlock()
			close(ch)
		})
	}
	return ch, unsubscribe
}

// subscriberCount returns the number of currently registered subscribers.
// Used for back-pressure and metrics.
func (b *InMemoryBus) subscriberCount() int {
	if b == nil {
		return 0
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subs)
}

type internalMessage struct {
	data []byte
	ack  chan error
}

// publish fans the message out to every subscribed channel. The returned
// receipt resolves only when EVERY subscriber has acked; if any one fails,
// the receipt's first error is returned and the relay retries the entire
// event_seq (at-least-once delivery per subscriber).
func (b *InMemoryBus) publish(ctx context.Context, data []byte) (PublishReceipt, error) {
	if b == nil {
		return nil, errors.New("outbox internal bus is not initialized")
	}
	b.mu.RLock()
	subs := make([]chan internalMessage, len(b.subs))
	copy(subs, b.subs)
	b.mu.RUnlock()

	if len(subs) == 0 {
		// No subscribers yet (startup race) — return an already-resolved
		// receipt so the relay doesn't deadlock waiting for an ack that
		// will never arrive. The event will be visible to the next poll.
		return immediateReceipt{}, nil
	}

	acks := make([]chan error, len(subs))
	for i, ch := range subs {
		payload := append([]byte(nil), data...)
		ack := make(chan error, 1)
		acks[i] = ack
		msg := internalMessage{data: payload, ack: ack}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case ch <- msg:
			metrics.OutboxInternalBusDepth.Set(float64(len(ch)))
		}
	}
	return fanoutReceipt{acks: acks}, nil
}

// fanoutReceipt aggregates per-subscriber acks into a single relay-facing
// receipt. The first non-nil error short-circuits and is returned to the
// relay so the publisher retries the underlying event_seq.
type fanoutReceipt struct {
	acks []chan error
}

func (r fanoutReceipt) Get(ctx context.Context) (string, error) {
	for _, ack := range r.acks {
		if ack == nil {
			continue
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case err := <-ack:
			if err != nil {
				return "", err
			}
		}
	}
	return "", nil
}

// InternalSubscriber consumes events from a single subscriber channel
// (obtained from InMemoryBus.Subscribe). It is fully isolated from siblings:
// events published to the bus while this subscriber was offline are NOT
// queued for it. Producers must tolerate subscriber re-registration.
type InternalSubscriber struct {
	ch      <-chan internalMessage
	workers int // <=1 means serial single-handler loop (legacy behaviour).
}

// NewInternalSubscriber creates a subscriber bound to a single channel.
// workers controls parallel handlers: routing keys that match orderingKeyFor
// (asset / mcap / _na) hash to the same worker queue so events for one asset
// stay ordered.
func NewInternalSubscriber(ch <-chan internalMessage, workers int) (*InternalSubscriber, error) {
	if ch == nil {
		return nil, errors.New("outbox internal subscriber: channel is required")
	}
	if workers < 1 {
		workers = 1
	}
	return &InternalSubscriber{ch: ch, workers: workers}, nil
}

func (s *InternalSubscriber) Receive(ctx context.Context, handler func(context.Context, []byte) error) error {
	if s == nil || s.ch == nil {
		return errors.New("outbox internal subscriber is not initialized")
	}
	if s.workers <= 1 {
		return s.receiveSerial(ctx, handler)
	}
	w := s.workers
	workerChs := make([]chan internalMessage, w)
	for i := range workerChs {
		workerChs[i] = make(chan internalMessage, 256)
	}
	g, gctx := errgroup.WithContext(ctx)

	for wi := 0; wi < w; wi++ {
		wi := wi
		g.Go(func() error {
			for {
				select {
				case <-gctx.Done():
					return gctx.Err()
				case msg := <-workerChs[wi]:
					err := handler(gctx, msg.data)
					select {
					case msg.ack <- err:
					case <-gctx.Done():
						return gctx.Err()
					}
				}
			}
		})
	}

	g.Go(func() error {
		for {
			select {
			case <-gctx.Done():
				return gctx.Err()
			case msg, ok := <-s.ch:
				if !ok {
					return nil
				}
				idx := routingWorkerIndex(msg.data, w)
				select {
				case workerChs[idx] <- msg:
				case <-gctx.Done():
					return gctx.Err()
				}
			}
		}
	})

	return g.Wait()
}

func (s *InternalSubscriber) receiveSerial(ctx context.Context, handler func(context.Context, []byte) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-s.ch:
			if !ok {
				return nil
			}
			err := handler(ctx, msg.data)
			select {
			case msg.ack <- err:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

// ReceiveBatch consumes a single subscriber channel and delivers messages to
// handler in short-window batches. Per-asset ordering is preserved because
// the same routing-key dispatch (FNV(asset_id) % workers) used by Receive is
// applied here too, so every event for a given asset always lands on the
// same worker.
//
// Each worker:
//  1. Blocks on the first message in its queue.
//  2. Drains up to batchSize-1 additional messages without blocking, bounded
//     by waitFor (a fresh timer per batch). waitFor == 0 means "drain whatever
//     is already buffered without waiting; flush as soon as the worker would
//     block on an empty queue".
//  3. Invokes handler ONCE with the slice of payloads.
//  4. Acks every message in the batch with the handler's returned error.
//
// At-least-once: handler-level errors fail the entire batch — every original
// publisher.Publish receipt observes the same error and the relay will retry
// each event_seq independently. Callers that want partial success must either
// re-shape the error or implement their own per-message retry logic.
func (s *InternalSubscriber) ReceiveBatch(
	ctx context.Context,
	batchSize int,
	waitFor time.Duration,
	handler func(context.Context, [][]byte) error,
) error {
	if s == nil || s.ch == nil {
		return errors.New("outbox internal subscriber is not initialized")
	}
	if handler == nil {
		return errors.New("outbox internal subscriber: nil batch handler")
	}
	if batchSize <= 1 {
		// Caller asked for batching but configured size 1 → fall back to
		// the legacy single-message path with a 1-element slice so callers
		// can treat both paths uniformly.
		return s.Receive(ctx, func(c context.Context, data []byte) error {
			return handler(c, [][]byte{data})
		})
	}
	if waitFor < 0 {
		waitFor = 0
	}

	if s.workers <= 1 {
		return s.receiveBatchSerial(ctx, batchSize, waitFor, handler)
	}

	w := s.workers
	workerChs := make([]chan internalMessage, w)
	for i := range workerChs {
		workerChs[i] = make(chan internalMessage, 256)
	}
	g, gctx := errgroup.WithContext(ctx)

	for wi := 0; wi < w; wi++ {
		ch := workerChs[wi]
		g.Go(func() error {
			return runBatchWorker(gctx, ch, batchSize, waitFor, handler)
		})
	}

	g.Go(func() error {
		for {
			select {
			case <-gctx.Done():
				return gctx.Err()
			case msg, ok := <-s.ch:
				if !ok {
					return nil
				}
				idx := routingWorkerIndex(msg.data, w)
				select {
				case workerChs[idx] <- msg:
				case <-gctx.Done():
					return gctx.Err()
				}
			}
		}
	})

	return g.Wait()
}

func (s *InternalSubscriber) receiveBatchSerial(
	ctx context.Context,
	batchSize int,
	waitFor time.Duration,
	handler func(context.Context, [][]byte) error,
) error {
	return runBatchWorker(ctx, s.ch, batchSize, waitFor, handler)
}

// runBatchWorker drains source up to batchSize, bounded by waitFor (per batch),
// then invokes handler and acks every message with the handler's error.
func runBatchWorker(
	ctx context.Context,
	source <-chan internalMessage,
	batchSize int,
	waitFor time.Duration,
	handler func(context.Context, [][]byte) error,
) error {
	for {
		// Block on the first message: a worker that has nothing to do should
		// not flush an empty batch.
		var first internalMessage
		var ok bool
		select {
		case <-ctx.Done():
			return ctx.Err()
		case first, ok = <-source:
			if !ok {
				return nil
			}
		}

		batch := make([]internalMessage, 0, batchSize)
		batch = append(batch, first)

		if err := drainBatch(ctx, source, &batch, batchSize, waitFor); err != nil {
			return err
		}

		payloads := make([][]byte, len(batch))
		for i, m := range batch {
			payloads[i] = m.data
		}
		err := handler(ctx, payloads)
		for _, m := range batch {
			select {
			case m.ack <- err:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

// drainBatch fills batch up to batchSize from source, honoring waitFor.
// waitFor == 0 means non-blocking drain (return as soon as source is empty);
// waitFor > 0 starts a fresh timer that bounds total drain time per batch.
func drainBatch(
	ctx context.Context,
	source <-chan internalMessage,
	batch *[]internalMessage,
	batchSize int,
	waitFor time.Duration,
) error {
	if waitFor == 0 {
		for len(*batch) < batchSize {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case m, ok := <-source:
				if !ok {
					return nil
				}
				*batch = append(*batch, m)
			default:
				return nil
			}
		}
		return nil
	}

	timer := time.NewTimer(waitFor)
	defer timer.Stop()
	for len(*batch) < batchSize {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case m, ok := <-source:
			if !ok {
				return nil
			}
			*batch = append(*batch, m)
		case <-timer.C:
			return nil
		}
	}
	return nil
}

func routingWorkerIndex(data []byte, workers int) int {
	if workers <= 1 {
		return 0
	}
	key := routingKeyFromPayload(data)
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return int(h.Sum32() % uint32(workers))
}

func routingKeyFromPayload(data []byte) string {
	var peek struct {
		AssetID    string `json:"asset_id"`
		McapFileID string `json:"mcap_file_id"`
	}
	if err := json.Unmarshal(data, &peek); err != nil {
		return "_na"
	}
	if peek.AssetID != "" {
		return peek.AssetID
	}
	if peek.McapFileID != "" {
		return "mcap:" + peek.McapFileID
	}
	return "_na"
}

func (s *InternalSubscriber) Close() error {
	return nil
}
