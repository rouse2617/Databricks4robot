package outbox

import (
	"context"
	"errors"

	"cloud.google.com/go/pubsub"
)

// PubSubSubscriber consumes events from Google Pub/Sub.
type PubSubSubscriber struct {
	client  *pubsub.Client
	subName string
}

// NewPubSubSubscriber creates a Pub/Sub-backed outbox subscriber.
func NewPubSubSubscriber(ctx context.Context, projectID, subName string) (*PubSubSubscriber, error) {
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &PubSubSubscriber{client: client, subName: subName}, nil
}

func (s *PubSubSubscriber) Receive(ctx context.Context, handler func(context.Context, []byte) error) error {
	if s == nil || s.client == nil || s.subName == "" {
		return errors.New("outbox pubsub subscriber: incomplete wiring")
	}
	sub := s.client.Subscription(s.subName)
	sub.ReceiveSettings.MaxOutstandingMessages = 64
	sub.ReceiveSettings.NumGoroutines = 4
	return sub.Receive(ctx, func(msgCtx context.Context, msg *pubsub.Message) {
		if err := handler(msgCtx, msg.Data); err != nil {
			msg.Nack()
			return
		}
		msg.Ack()
	})
}

func (s *PubSubSubscriber) Close() error {
	if s == nil || s.client == nil {
		return nil
	}
	return s.client.Close()
}
