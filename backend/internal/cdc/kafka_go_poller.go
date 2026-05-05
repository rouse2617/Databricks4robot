package cdc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
		return nil, nil
	}

	records := make([]KafkaRecord, 0, p.maxBatch)
	// rawMsgs tracks the kafka.Message values so we can commit them all at once.
	rawMsgs := make([]kafka.Message, 0, p.maxBatch)

	for len(records) < p.maxBatch {
		polledAny := false
		for _, nr := range p.readers {
			readCtx, cancel := context.WithTimeout(ctx, p.pollTimeout)
			msg, err := nr.reader.FetchMessage(readCtx)
			cancel()
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
					if ctx.Err() != nil {
						return nil, ctx.Err()
					}
					continue
				}
				return nil, fmt.Errorf("kafka-go poll topic %s: %w", nr.topic, err)
			}
			polledAny = true
			key, err := decodeKafkaKey(msg.Key)
			if err != nil {
				return nil, fmt.Errorf("kafka-go decode key topic %s: %w", msg.Topic, err)
			}
			records = append(records, KafkaRecord{
				Topic: msg.Topic,
				Key:   key,
				Value: msg.Value,
			})
			rawMsgs = append(rawMsgs, msg)
			if len(records) >= p.maxBatch {
				break
			}
		}
		if !polledAny {
			break
		}
	}

	// Batch-commit all fetched messages before returning so offsets are only
	// advanced after the caller has received the full batch.
	if len(rawMsgs) > 0 {
		// Group messages by reader so each reader commits its own messages.
		byTopic := make(map[string][]kafka.Message, len(p.readers))
		for _, m := range rawMsgs {
			byTopic[m.Topic] = append(byTopic[m.Topic], m)
		}
		for _, nr := range p.readers {
			msgs, ok := byTopic[nr.topic]
			if !ok {
				continue
			}
			if err := nr.reader.CommitMessages(ctx, msgs...); err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return nil, ctx.Err()
				}
				return nil, fmt.Errorf("kafka-go commit topic %s: %w", nr.topic, err)
			}
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
