package outbox

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"cloud.google.com/go/pubsub"
)

// Publisher sends outbox payloads to a single Pub/Sub topic with message ordering.
type Publisher struct {
	mode        string
	client      *pubsub.Client
	topic       *pubsub.Topic
	internalBus *InMemoryBus
}

// NewPublisher creates a Pub/Sub client and enables ordering on the topic handle.
func NewPublisher(ctx context.Context, projectID, topicName string) (*Publisher, error) {
	projectID = strings.TrimSpace(projectID)
	topicName = strings.TrimSpace(topicName)
	if projectID == "" || topicName == "" {
		return nil, fmt.Errorf("outbox publisher: project id and topic name are required")
	}
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("outbox publisher: new client: %w", err)
	}
	topic := client.Topic(topicName)
	// Message ordering is intentionally disabled. The ES subscriber
	// (es_subscriber.go) is idempotent: it dedupes events per asset_id
	// within a batch, takes max(event_seq), rebuilds the doc from PG,
	// and writes with ExternalVersion=event_seq so any out-of-order
	// retry is rejected by ES's version conflict. Ordering would force
	// serial publish per key and caps relay throughput at ~200 ev/s.
	topic.EnableMessageOrdering = false
	return &Publisher{mode: "pubsub", client: client, topic: topic}, nil
}

// NewKafkaPublisher creates a Kafka producer for outbox events.
func NewKafkaPublisher(brokers []string, topicName string) (*Publisher, error) {
	return nil, fmt.Errorf("outbox kafka publisher is not compiled in this build (brokers=%q topic=%q)", strings.Join(brokers, ","), topicName)
}

// NewInternalPublisher creates an in-process publisher for local/default mode.
func NewInternalPublisher(bus *InMemoryBus) (*Publisher, error) {
	if bus == nil {
		return nil, fmt.Errorf("outbox internal publisher: bus is required")
	}
	return &Publisher{mode: "internal", internalBus: bus}, nil
}

// Publish sends one message; orderingKey should be asset_id when present.
func (p *Publisher) Publish(ctx context.Context, orderingKey string, data []byte) (PublishReceipt, error) {
	if p == nil {
		return nil, fmt.Errorf("outbox publisher: nil")
	}
	if orderingKey == "" {
		orderingKey = "_na"
	}
	switch p.mode {
	case "pubsub":
		if p.topic == nil {
			return nil, fmt.Errorf("outbox publisher: nil topic")
		}
		return p.topic.Publish(ctx, &pubsub.Message{
			Data:        data,
			OrderingKey: orderingKey,
		}), nil
	case "kafka":
		return nil, fmt.Errorf("outbox kafka publisher is not compiled in this build")
	case "internal":
		if p.internalBus == nil {
			return nil, fmt.Errorf("outbox internal publisher: nil bus")
		}
		return p.internalBus.publish(ctx, data)
	default:
		return nil, fmt.Errorf("outbox publisher: unknown mode %q", p.mode)
	}
}

// ResumePublishAfterError must be called after a failed publish for orderingKey
// so subsequent publishes on that key are not blocked.
func (p *Publisher) ResumePublishAfterError(orderingKey string) {
	if p == nil || p.topic == nil {
		return
	}
	if orderingKey == "" {
		orderingKey = "_na"
	}
	p.topic.ResumePublish(orderingKey)
}

// Close releases Pub/Sub client resources.
func (p *Publisher) Close() error {
	if p == nil {
		return nil
	}
	switch p.mode {
	case "pubsub":
		p.topic.Stop()
		return p.client.Close()
	case "kafka":
		return nil
	case "internal":
		return nil
	default:
		slog.Warn("outbox publisher close skipped", "mode", p.mode)
		return nil
	}
}
