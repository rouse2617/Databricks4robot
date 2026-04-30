package cdc

import (
	"context"
	"errors"
	"testing"
)

type fakeKafkaPoller struct {
	calls   int
	records []KafkaRecord
}

var errTestStop = errors.New("stop after first poll")

func (f *fakeKafkaPoller) Poll(ctx context.Context) ([]KafkaRecord, error) {
	f.calls++
	if f.calls > 1 {
		return nil, errTestStop
	}
	return f.records, nil
}

func TestKafkaSource_Run_DecodesAndRoutes(t *testing.T) {
	handler := &recordingHandler{}
	runtime := &Runtime{
		Config: RuntimeConfig{Enabled: true, SourceDriver: "kafka"},
		Handlers: map[string]BatchHandler{
			"data4cyber.public.assets": handler,
		},
	}

	source := &KafkaSource{
		Poller: &fakeKafkaPoller{
			records: []KafkaRecord{
				{
					Topic: "data4cyber.public.assets",
					Key:   map[string]any{"asset_id": "a1"},
					Value: []byte(`{"payload":{"op":"u","before":null,"after":{"asset_id":"a1"},"source":{"table":"assets"}}}`),
				},
			},
		},
	}

	err := source.Run(context.Background(), runtime)
	if !errors.Is(err, errTestStop) {
		t.Fatalf("expected test stop error after first poll, got %v", err)
	}
	if len(handler.batches) != 1 || len(handler.batches[0]) != 1 {
		t.Fatalf("unexpected handler batches: %+v", handler.batches)
	}
	if handler.batches[0][0].Table != "assets" {
		t.Fatalf("unexpected event: %+v", handler.batches[0][0])
	}
}
