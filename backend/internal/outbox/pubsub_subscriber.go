package outbox

import (
	"context"
	"errors"
	"strings"

	"cloud.google.com/go/pubsub"
)

// PubSubSubscriber consumes events from Google Pub/Sub.
type PubSubSubscriber struct {
	client  *pubsub.Client
	subName string
}

// NewPubSubSubscriber creates a Pub/Sub-backed outbox subscriber.
func NewPubSubSubscriber(ctx context.Context, projectID, subName string) (*PubSubSubscriber, error) {
	if err := ValidatePubSubEventSourceConfig(projectID, subName); err != nil {
		return nil, err
	}
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &PubSubSubscriber{client: client, subName: subName}, nil
}

func ValidatePubSubEventSourceConfig(projectID, subName string) error {
	if strings.TrimSpace(projectID) == "" || strings.TrimSpace(subName) == "" {
		return errors.New("outbox pubsub event source: PUBSUB_PROJECT and subscription are required")
	}
	return nil
}

type pubSubAckNacker interface {
	Ack()
	Nack()
}

func (s *PubSubSubscriber) Receive(ctx context.Context, handler func(context.Context, []byte) error) error {
	if s == nil || s.client == nil || s.subName == "" {
		return errors.New("outbox pubsub subscriber: incomplete wiring")
	}
	sub := s.client.Subscription(s.subName)
	sub.ReceiveSettings.MaxOutstandingMessages = 64
	sub.ReceiveSettings.NumGoroutines = 4
	return sub.Receive(ctx, func(msgCtx context.Context, msg *pubsub.Message) {
		handlePubSubEventMessage(msgCtx, msg.Data, msg, handler)
	})
}

func handlePubSubEventMessage(ctx context.Context, data []byte, msg pubSubAckNacker, handler func(context.Context, []byte) error) {
	if err := handler(ctx, data); err != nil {
		msg.Nack()
		return
	}
	msg.Ack()
}

func (s *PubSubSubscriber) Close() error {
	if s == nil || s.client == nil {
		return nil
	}
	return s.client.Close()
}
