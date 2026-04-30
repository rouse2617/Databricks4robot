package cdc

import (
	"context"
	"fmt"
)

type TopicBatch struct {
	Topic  string
	Events []ChangeEvent
}

// InMemorySource is a test/demo source adapter that replays predefined topic
// batches into the runtime router.
type InMemorySource struct {
	Batches []TopicBatch
}

func (s *InMemorySource) Run(ctx context.Context, router Router) error {
	for _, batch := range s.Batches {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if batch.Topic == "" {
			return fmt.Errorf("in-memory cdc source: empty topic")
		}
		if err := router.HandleTopicBatch(ctx, batch.Topic, batch.Events); err != nil {
			return err
		}
	}
	return nil
}
