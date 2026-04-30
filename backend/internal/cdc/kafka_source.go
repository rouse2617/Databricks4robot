package cdc

import (
	"context"
	"errors"
	"fmt"
)

var ErrKafkaPollUnimplemented = errors.New("kafka source poller is not implemented yet")

type KafkaRecord struct {
	Topic string
	Key   map[string]any
	Value []byte
}

// KafkaPoller abstracts the transport-specific polling loop. A later slice can
// back this with a real Kafka client without changing Runtime or consumers.
type KafkaPoller interface {
	Poll(ctx context.Context) ([]KafkaRecord, error)
}

// KafkaSource decodes Debezium-style records from a Kafka-compatible bus and
// dispatches them into the runtime router by topic.
type KafkaSource struct {
	Poller KafkaPoller
}

func (s *KafkaSource) Run(ctx context.Context, router Router) error {
	if s.Poller == nil {
		return ErrKafkaPollUnimplemented
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		records, err := s.Poller.Poll(ctx)
		if err != nil {
			return err
		}
		for _, record := range records {
			event, err := DecodeDebeziumMessage(record.Value, record.Key)
			if err != nil {
				return fmt.Errorf("kafka source decode topic %s: %w", record.Topic, err)
			}
			if err := router.HandleTopicBatch(ctx, record.Topic, []ChangeEvent{event}); err != nil {
				return err
			}
		}
	}
}
