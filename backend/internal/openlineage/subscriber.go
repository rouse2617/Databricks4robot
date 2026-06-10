package openlineage

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/outbox"
)

type Subscriber struct {
	Source  outbox.EventSubscriber
	Builder Builder
	Emitter *Emitter
}

func (s *Subscriber) Run(ctx context.Context) error {
	if s == nil || s.Source == nil || s.Emitter == nil {
		return errors.New("openlineage subscriber: incomplete wiring")
	}
	return s.Source.Receive(ctx, s.HandleData)
}

func (s *Subscriber) HandleData(ctx context.Context, data []byte) error {
	var ev models.AssetEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		slog.Error("openlineage subscriber: failed to unmarshal event, skipping malformed message", "err", err)
		return nil
	}
	event, ok, err := s.Builder.Build(ev)
	if err != nil {
		// A malformed payload must not become a poison pill:
		// returning the error NACKs and causes infinite redelivery.
		slog.Warn("openlineage subscriber: failed to build event, skipping", "err", err)
		return nil
	}
	if !ok {
		slog.Debug("openlineage subscriber: skipping unsupported event", "event_seq", ev.EventSeq, "event_type", ev.EventType)
		return nil
	}
	return s.Emitter.Emit(ctx, event)
}

func (s *Subscriber) Close() error {
	if s == nil || s.Source == nil {
		return nil
	}
	return s.Source.Close()
}
