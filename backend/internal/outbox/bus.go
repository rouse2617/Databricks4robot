package outbox

import (
	"context"
	"time"
)

// PublishReceipt models an async publish acknowledgement.
type PublishReceipt interface {
	Get(ctx context.Context) (string, error)
}

// EventPublisher publishes outbox events to a transport.
type EventPublisher interface {
	Publish(ctx context.Context, orderingKey string, data []byte) (PublishReceipt, error)
	ResumePublishAfterError(orderingKey string)
	Close() error
}

// EventSubscriber consumes outbox events from a transport.
type EventSubscriber interface {
	Receive(ctx context.Context, handler func(context.Context, []byte) error) error
	Close() error
}

// BatchEventSubscriber is an OPTIONAL extension that lets a transport deliver
// messages in short-window batches and ack them as a unit. Subscribers that
// don't implement this interface should be invoked through the legacy
// single-message Receive path; the ESSubscriber feature-detects this at
// runtime so PubSub / Kafka transports keep working unchanged.
//
// Semantics:
//   - The handler is invoked once per batch with the underlying payloads.
//   - Per-routing-key ordering is preserved by the implementation: a single
//     batch may span different routing keys, but two events sharing a key
//     are delivered to the same worker, in original publish order.
//   - The handler's returned error is applied to ALL messages in the batch
//     ("at-least-once"). On error, the publisher's pr.Get(ctx) sees the same
//     error for every message in the batch and the relay retries each event
//     by event_seq. Implementations MUST NOT split partial successes across
//     individual acks.
type BatchEventSubscriber interface {
	EventSubscriber
	ReceiveBatch(
		ctx context.Context,
		batchSize int,
		waitFor time.Duration,
		handler func(context.Context, [][]byte) error,
	) error
}

type immediateReceipt struct {
	id  string
	err error
}

func (r immediateReceipt) Get(context.Context) (string, error) {
	return r.id, r.err
}

// Compile-time assertion: InternalSubscriber must satisfy BatchEventSubscriber
// so feature-detection in ESSubscriber works.
var _ BatchEventSubscriber = (*InternalSubscriber)(nil)
