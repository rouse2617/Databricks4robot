package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"hash/fnv"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/metrics"
	"golang.org/x/sync/errgroup"
)

// InMemoryBus is an in-process event bus for default local transport mode.
type InMemoryBus struct {
	ch chan internalMessage
}

// NewInMemoryBus creates a buffered in-process bus.
func NewInMemoryBus(buffer int) *InMemoryBus {
	if buffer <= 0 {
		buffer = 1024
	}
	return &InMemoryBus{ch: make(chan internalMessage, buffer)}
}

type internalMessage struct {
	data []byte
	ack  chan error
}

func (b *InMemoryBus) publish(ctx context.Context, data []byte) (PublishReceipt, error) {
	if b == nil || b.ch == nil {
		return nil, errors.New("outbox internal bus is not initialized")
	}
	msg := internalMessage{
		data: append([]byte(nil), data...),
		ack:  make(chan error, 1),
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case b.ch <- msg:
		metrics.OutboxInternalBusDepth.Set(float64(len(b.ch)))
		return internalReceipt{ack: msg.ack}, nil
	}
}

type internalReceipt struct {
	ack <-chan error
}

func (r internalReceipt) Get(ctx context.Context) (string, error) {
	if r.ack == nil {
		return "", errors.New("outbox internal receipt: nil ack channel")
	}
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case err := <-r.ack:
		return "", err
	}
}

// InternalSubscriber consumes events from an in-process bus.
type InternalSubscriber struct {
	bus     *InMemoryBus
	workers int // <=1 means serial single-handler loop (legacy behaviour).
}

// NewInternalSubscriber creates a subscriber for internal bus mode.
// workers controls parallel handlers: routing keys that match orderingKeyFor (asset / mcap / _na)
// hash to the same worker queue so events for one asset stay ordered.
func NewInternalSubscriber(bus *InMemoryBus, workers int) (*InternalSubscriber, error) {
	if bus == nil {
		return nil, errors.New("outbox internal subscriber: bus is required")
	}
	if workers < 1 {
		workers = 1
	}
	return &InternalSubscriber{bus: bus, workers: workers}, nil
}

func (s *InternalSubscriber) Receive(ctx context.Context, handler func(context.Context, []byte) error) error {
	if s == nil || s.bus == nil || s.bus.ch == nil {
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
						metrics.OutboxInternalBusDepth.Set(float64(len(s.bus.ch)))
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
			case msg := <-s.bus.ch:
				metrics.OutboxInternalBusDepth.Set(float64(len(s.bus.ch)))
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
		case msg := <-s.bus.ch:
			metrics.OutboxInternalBusDepth.Set(float64(len(s.bus.ch)))
			err := handler(ctx, msg.data)
			select {
			case msg.ack <- err:
				metrics.OutboxInternalBusDepth.Set(float64(len(s.bus.ch)))
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

// ReceiveBatch consumes the in-process bus and delivers messages to handler in
// short-window batches. Per-asset ordering is preserved because the same
// routing-key dispatch (FNV(asset_id) % workers) used by Receive is applied
// here too, so every event for a given asset always lands on the same worker.
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
	if s == nil || s.bus == nil || s.bus.ch == nil {
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
			case msg := <-s.bus.ch:
				metrics.OutboxInternalBusDepth.Set(float64(len(s.bus.ch)))
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
	return runBatchWorker(ctx, s.bus.ch, batchSize, waitFor, handler)
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
		select {
		case <-ctx.Done():
			return ctx.Err()
		case first = <-source:
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
			case m := <-source:
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
		case m := <-source:
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
