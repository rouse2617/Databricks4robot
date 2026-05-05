package cdc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"
)

var ErrKafkaPollUnimplemented = errors.New("kafka source poller is not implemented yet")

// retryDelays defines the exponential backoff sequence for transient errors.
// The last value is the cap: any subsequent retry uses the same delay.
var retryDelays = []time.Duration{1 * time.Second, 2 * time.Second, 5 * time.Second, 10 * time.Second}

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

	// Close the poller on exit if it implements io.Closer.
	defer func() {
		if closer, ok := s.Poller.(io.Closer); ok {
			if err := closer.Close(); err != nil {
				slog.Error("kafka source: poller close error", "err", err)
			}
		}
	}()

	retryCount := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		records, err := s.Poller.Poll(ctx)
		if err != nil {
			// Clean exit on context cancellation or deadline.
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			// Transient error: log and retry with exponential backoff.
			idx := retryCount
			if idx >= len(retryDelays) {
				idx = len(retryDelays) - 1
			}
			delay := retryDelays[idx]
			slog.Warn("kafka source: transient poll error, retrying",
				"err", err,
				"retry", retryCount+1,
				"backoff", delay,
			)
			retryCount++
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			continue
		}

		// Successful poll resets the retry counter.
		retryCount = 0

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
