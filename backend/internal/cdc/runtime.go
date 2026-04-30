package cdc

import (
	"context"
	"errors"
	"fmt"
	"slices"
)

type ConsumerKind string

const (
	ConsumerKindBronzeEvents     ConsumerKind = "bronze_events"
	ConsumerKindSearchProjection ConsumerKind = "search_projection"
)

type TopicMapping struct {
	Table    string
	Topic    string
	Consumer ConsumerKind
}

type RuntimeConfig struct {
	Enabled      bool
	SourceDriver string
	Mappings     []TopicMapping
}

var (
	ErrNoCDCSource  = errors.New("cdc runtime has no source adapter")
	ErrNoCDCHandler = errors.New("cdc runtime has no topic handlers")
)

// BatchHandler processes a batch of normalized change events for one topic.
type BatchHandler interface {
	HandleBatch(ctx context.Context, events []ChangeEvent) error
}

// Source is the pluggable transport adapter. A future Debezium/Kafka adapter
// or a managed CDC client can implement this without changing the consumers.
type Source interface {
	Run(ctx context.Context, router Router) error
}

// Router dispatches CDC batches by topic to application-level handlers.
type Router interface {
	HandleTopicBatch(ctx context.Context, topic string, events []ChangeEvent) error
}

type Runtime struct {
	Config   RuntimeConfig
	Source   Source
	Handlers map[string]BatchHandler
}

func (r *Runtime) Validate() error {
	if !r.Config.Enabled {
		return nil
	}
	if r.Source == nil {
		return ErrNoCDCSource
	}
	if len(r.Handlers) == 0 {
		return ErrNoCDCHandler
	}
	return nil
}

func (r *Runtime) Run(ctx context.Context) error {
	if !r.Config.Enabled {
		return nil
	}
	if err := r.Validate(); err != nil {
		return err
	}
	return r.Source.Run(ctx, r)
}

func (r *Runtime) HandleTopicBatch(ctx context.Context, topic string, events []ChangeEvent) error {
	handler, ok := r.Handlers[topic]
	if !ok {
		return fmt.Errorf("cdc runtime: no handler registered for topic %q", topic)
	}
	return handler.HandleBatch(ctx, events)
}

func (c RuntimeConfig) TopicsFor(kind ConsumerKind) []string {
	topics := make([]string, 0, len(c.Mappings))
	for _, mapping := range c.Mappings {
		if mapping.Consumer != kind {
			continue
		}
		if mapping.Topic == "" || slices.Contains(topics, mapping.Topic) {
			continue
		}
		topics = append(topics, mapping.Topic)
	}
	return topics
}
