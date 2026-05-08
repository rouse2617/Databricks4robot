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
	// Commit acknowledges the record after application-level handling succeeds.
	// It should be nil only in tests or transports that do not require explicit
	// offset management.
	Commit func(context.Context) error
}

// KafkaPoller abstracts the transport-specific polling loop. A later slice can
// back this with a real Kafka client without changing Runtime or consumers.
type KafkaPoller interface {
	Poll(ctx context.Context) ([]KafkaRecord, error)
}

// KafkaSource decodes Debezium-style records from a Kafka-compatible bus and
// dispatches them into the runtime router by topic.
type KafkaSource struct {
	Poller                  KafkaPoller
	ErrorSink               ErrorSink
	MaxConsecutiveFailures  int
	ContinueOnCommitFailure bool
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
	maxFailures := s.MaxConsecutiveFailures
	if maxFailures <= 0 {
		maxFailures = 20
	}
	consecutiveFailures := 0
	CDCConsecutiveFailures.Set(0)

	recordFailure := func(stage FailureStage, err error) error {
		CDCHandlerFailuresTotal.WithLabelValues(string(stage)).Inc()
		consecutiveFailures++
		CDCConsecutiveFailures.Set(float64(consecutiveFailures))
		if consecutiveFailures >= maxFailures {
			return fmt.Errorf("kafka source exceeded consecutive failure threshold (%d): %w", maxFailures, err)
		}
		return nil
	}

	resetFailures := func() {
		consecutiveFailures = 0
		CDCConsecutiveFailures.Set(0)
	}

	for {
		select {
		case <-ctx.Done():
			slog.Info("kafka source: context cancelled, exiting")
			return ctx.Err()
		default:
		}

		records, err := s.Poller.Poll(ctx)
		if err != nil {
			// Clean exit on context cancellation or deadline.
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				slog.Info("kafka source: poll error (context)", "err", err)
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
			CDCKafkaRetriesTotal.Inc()
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

		if len(records) == 0 {
			// No records this poll, continue
			slog.Info("kafka source: no records this poll")
			continue
		}

		slog.Info("kafka source: received records", "count", len(records))

		for _, record := range records {
			slog.Info("kafka source: decoding event", "topic", record.Topic)
			event, err := DecodeDebeziumMessage(record.Value, record.Key)
			if err != nil {
				slog.Error("kafka source: decode error", "topic", record.Topic, "err", err)
				if parkErr := s.parkFailedRecord(ctx, record, FailureStageDecode, err); parkErr != nil {
					if fatal := recordFailure(FailureStageDecode, parkErr); fatal != nil {
						return fatal
					}
					continue
				}
				if fatal := recordFailure(FailureStageDecode, fmt.Errorf("kafka source decode topic %s: %w", record.Topic, err)); fatal != nil {
					return fatal
				}
				continue
			}
			slog.Info("kafka source: dispatching event", "topic", record.Topic, "table", event.Table, "op", event.Op)
			if err := router.HandleTopicBatch(ctx, record.Topic, []ChangeEvent{event}); err != nil {
				if parkErr := s.parkFailedRecord(ctx, record, FailureStageRoute, err); parkErr != nil {
					if fatal := recordFailure(FailureStageRoute, parkErr); fatal != nil {
						return fatal
					}
					continue
				}
				if fatal := recordFailure(FailureStageRoute, err); fatal != nil {
					return fatal
				}
				continue
			}
			if record.Commit != nil {
				if err := record.Commit(ctx); err != nil {
					wrapped := fmt.Errorf("kafka source commit topic %s: %w", record.Topic, err)
					if s.ErrorSink != nil {
						if sinkErr := s.ErrorSink.Write(ctx, NewFailedRecord(FailureStageCommit, record, wrapped)); sinkErr != nil {
							CDCDLQWriteFailuresTotal.Inc()
							if fatal := recordFailure(FailureStageCommit, fmt.Errorf("write failed commit record to dlq: %w", sinkErr)); fatal != nil {
								return fatal
							}
						} else {
							CDCDLQWritesTotal.Inc()
						}
					}
					if fatal := recordFailure(FailureStageCommit, wrapped); fatal != nil {
						return fatal
					}
					if !s.ContinueOnCommitFailure {
						return wrapped
					}
					continue
				}
			}
			resetFailures()
		}
	}
}

func (s *KafkaSource) parkFailedRecord(ctx context.Context, record KafkaRecord, stage FailureStage, rootErr error) error {
	if s.ErrorSink != nil {
		if err := s.ErrorSink.Write(ctx, NewFailedRecord(stage, record, rootErr)); err != nil {
			CDCDLQWriteFailuresTotal.Inc()
			return fmt.Errorf("write failed record to dlq: %w", err)
		}
		CDCDLQWritesTotal.Inc()
	}
	if record.Commit != nil {
		if err := record.Commit(ctx); err != nil {
			return fmt.Errorf("commit failed poison record after %s failure: %w", stage, err)
		}
	}
	return nil
}
