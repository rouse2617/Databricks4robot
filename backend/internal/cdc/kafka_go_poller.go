package cdc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaGoPoller is a minimal real Kafka-compatible poller backed by
// github.com/segmentio/kafka-go. It is intentionally simple and aimed at local
// CDC bring-up, not production-grade retry/commit tuning.
type KafkaGoPoller struct {
	readers     []namedReader
	pollTimeout time.Duration
	maxBatch    int
}

type namedReader struct {
	topic  string
	reader *kafka.Reader
}

func NewKafkaGoPoller(brokers []string, groupID string, topics []string, pollTimeout time.Duration, maxBatch int) *KafkaGoPoller {
	if pollTimeout <= 0 {
		pollTimeout = 1 * time.Second
	}
	if maxBatch <= 0 {
		maxBatch = 100
	}
	readers := make([]namedReader, 0, len(topics))
	for _, topic := range topics {
		if topic == "" {
			continue
		}
		readers = append(readers, namedReader{
			topic: topic,
			reader: kafka.NewReader(kafka.ReaderConfig{
				Brokers:  brokers,
				GroupID:  groupID,
				Topic:    topic,
				MinBytes: 1,
				MaxBytes: 10e6,
				MaxWait:  pollTimeout,
			}),
		})
	}
	return &KafkaGoPoller{
		readers:     readers,
		pollTimeout: pollTimeout,
		maxBatch:    maxBatch,
	}
}

func (p *KafkaGoPoller) Poll(ctx context.Context) ([]KafkaRecord, error) {
	if len(p.readers) == 0 {
		slog.Warn("kafka-go poller: no readers configured")
		return nil, nil
	}

	records := make([]KafkaRecord, 0, p.maxBatch)
	slog.Info("kafka-go poller: starting poll", "num_readers", len(p.readers), "topics", func() []string {
		topics := make([]string, len(p.readers))
		for i, r := range p.readers {
			topics[i] = r.topic
		}
		return topics
	}())

	for len(records) < p.maxBatch {
		polledAny := false
		for _, nr := range p.readers {
			readCtx, cancel := context.WithTimeout(ctx, p.pollTimeout)
			msg, err := nr.reader.ReadMessage(readCtx)
			cancel()
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
					if ctx.Err() != nil {
						slog.Warn("kafka-go poller: context cancelled", "topic", nr.topic, "ctx_err", ctx.Err())
						return nil, ctx.Err()
					}
					// Continue to next reader on timeout
					continue
				}
				slog.Error("kafka-go poller: fetch error", "topic", nr.topic, "err", err)
				return nil, fmt.Errorf("kafka-go poll topic %s: %w", nr.topic, err)
			}

			slog.Info("kafka-go poller: received message", "topic", msg.Topic, "offset", msg.Offset, "size", len(msg.Value))
			polledAny = true
			key, err := decodeKafkaKey(msg.Key)
			if err != nil {
				slog.Error("kafka-go poller: decode key error", "topic", msg.Topic, "err", err)
				return nil, fmt.Errorf("kafka-go decode key topic %s: %w", msg.Topic, err)
			}
			records = append(records, KafkaRecord{
				Topic: msg.Topic,
				Key:   key,
				Value: msg.Value,
			})
			slog.Info("kafka-go poller: record added", "topic", msg.Topic, "key", key)
			if len(records) >= p.maxBatch {
				break
			}
		}
		if !polledAny {
			break
		}
	}

	return records, nil
}

func decodeKafkaKey(raw []byte) (map[string]any, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err == nil {
		return parsed, nil
	}
	return map[string]any{"_raw": strings.TrimSpace(string(raw))}, nil
}

func (p *KafkaGoPoller) Close() error {
	var firstErr error
	for _, nr := range p.readers {
		if err := nr.reader.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
