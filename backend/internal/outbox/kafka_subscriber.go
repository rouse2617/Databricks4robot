package outbox

import (
	"context"
	"errors"
	"strings"
)

// KafkaSubscriber consumes events from Kafka.
type KafkaSubscriber struct {
	brokers []string
	topic   string
	groupID string
}

// NewKafkaSubscriber creates a Kafka-backed outbox subscriber.
func NewKafkaSubscriber(brokers []string, topic, groupID string) (*KafkaSubscriber, error) {
	trimmed := make([]string, 0, len(brokers))
	for _, b := range brokers {
		if s := strings.TrimSpace(b); s != "" {
			trimmed = append(trimmed, s)
		}
	}
	if len(trimmed) == 0 || strings.TrimSpace(topic) == "" || strings.TrimSpace(groupID) == "" {
		return nil, errors.New("outbox kafka subscriber: brokers/topic/group_id are required")
	}
	return &KafkaSubscriber{brokers: trimmed, topic: topic, groupID: groupID}, nil
}

func (s *KafkaSubscriber) Receive(ctx context.Context, handler func(context.Context, []byte) error) error {
	if s == nil {
		return errors.New("outbox kafka subscriber: nil")
	}
	return errors.New("outbox kafka subscriber is not compiled in this build")
}

func (s *KafkaSubscriber) Close() error {
	return nil
}
