package cdc

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

type fakeBronzeSink struct {
	batches [][]*models.AssetEvent
}

func (f *fakeBronzeSink) WriteBatch(events []*models.AssetEvent) (string, error) {
	copied := make([]*models.AssetEvent, len(events))
	copy(copied, events)
	f.batches = append(f.batches, copied)
	return "staging/events.jsonl", nil
}

func TestBronzeConsumer_HandleBatch_ConsumesInsertAssetEvents(t *testing.T) {
	sink := &fakeBronzeSink{}
	consumer := &BronzeConsumer{Sink: sink}
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)

	err := consumer.HandleBatch(context.Background(), []ChangeEvent{
		{
			Table: "asset_events",
			Op:    OperationCreate,
			After: map[string]any{
				"event_id":               "e1",
				"event_seq":              int64(101),
				"event_type":             "asset_created",
				"aggregate_type":         "asset",
				"payload_schema_version": "v1",
				"asset_id":               "a1",
				"mcap_file_id":           "m1",
				"tenant_id":              "t1",
				"project_id":             "p1",
				"event_source":           "backend",
				"publish_state":          "pending",
				"event_payload":          map[string]any{"asset_id": "a1"},
				"occurred_at":            now.Format(time.RFC3339Nano),
				"created_at":             now.Format(time.RFC3339Nano),
			},
		},
		{
			Table: "asset_events",
			Op:    OperationUpdate,
			After: map[string]any{
				"event_id":      "e1",
				"publish_state": "published",
			},
		},
		{
			Table: "assets",
			Op:    OperationUpdate,
			After: map[string]any{"asset_id": "a1"},
		},
	})
	if err != nil {
		t.Fatalf("HandleBatch returned error: %v", err)
	}
	if len(sink.batches) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(sink.batches))
	}
	if len(sink.batches[0]) != 1 {
		t.Fatalf("expected 1 event in batch, got %d", len(sink.batches[0]))
	}
	got := sink.batches[0][0]
	if got.EventID != "e1" || got.EventSeq != 101 || got.EventType != "asset_created" {
		t.Fatalf("unexpected event: %+v", got)
	}
	var payload map[string]any
	if err := json.Unmarshal(got.EventPayload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload["asset_id"] != "a1" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestBronzeConsumer_HandleBatch_IncludesSnapshotReadsWhenEnabled(t *testing.T) {
	sink := &fakeBronzeSink{}
	consumer := &BronzeConsumer{
		Sink:                 sink,
		IncludeSnapshotReads: true,
	}
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)

	err := consumer.HandleBatch(context.Background(), []ChangeEvent{
		{
			Table: "asset_events",
			Op:    OperationRead,
			After: map[string]any{
				"event_id":               "e2",
				"event_seq":              "102",
				"event_type":             "asset_updated",
				"aggregate_type":         "asset",
				"payload_schema_version": "v1",
				"event_source":           "backend",
				"publish_state":          "pending",
				"event_payload":          `{"asset_id":"a2"}`,
				"occurred_at":            now,
				"created_at":             now,
			},
		},
	})
	if err != nil {
		t.Fatalf("HandleBatch returned error: %v", err)
	}
	if len(sink.batches) != 1 || len(sink.batches[0]) != 1 {
		t.Fatalf("expected one snapshot-derived event, got %+v", sink.batches)
	}
	if sink.batches[0][0].EventSeq != 102 {
		t.Fatalf("expected event_seq=102, got %+v", sink.batches[0][0])
	}
}

func TestBronzeConsumer_HandleBatch_PreservesBatchOrder(t *testing.T) {
	sink := &fakeBronzeSink{}
	consumer := &BronzeConsumer{Sink: sink}
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)

	err := consumer.HandleBatch(context.Background(), []ChangeEvent{
		{
			Table: "asset_events",
			Op:    OperationCreate,
			After: map[string]any{
				"event_id":               "e10",
				"event_seq":              int64(10),
				"event_type":             "asset_created",
				"aggregate_type":         "asset",
				"payload_schema_version": "v1",
				"event_source":           "backend",
				"publish_state":          "pending",
				"event_payload":          `{"asset_id":"a10"}`,
				"occurred_at":            now,
				"created_at":             now,
			},
		},
		{
			Table: "asset_events",
			Op:    OperationCreate,
			After: map[string]any{
				"event_id":               "e11",
				"event_seq":              int64(11),
				"event_type":             "asset_updated",
				"aggregate_type":         "asset",
				"payload_schema_version": "v1",
				"event_source":           "backend",
				"publish_state":          "pending",
				"event_payload":          `{"asset_id":"a11"}`,
				"occurred_at":            now,
				"created_at":             now,
			},
		},
	})
	if err != nil {
		t.Fatalf("HandleBatch returned error: %v", err)
	}
	if len(sink.batches) != 1 || len(sink.batches[0]) != 2 {
		t.Fatalf("unexpected batches: %+v", sink.batches)
	}
	if sink.batches[0][0].EventSeq != 10 || sink.batches[0][1].EventSeq != 11 {
		t.Fatalf("expected stable event order, got %+v", sink.batches[0])
	}
}

func TestBronzeConsumer_HandleBatch_IgnoresSnapshotReadsWhenDisabled(t *testing.T) {
	sink := &fakeBronzeSink{}
	consumer := &BronzeConsumer{Sink: sink}
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)

	err := consumer.HandleBatch(context.Background(), []ChangeEvent{
		{
			Table: "asset_events",
			Op:    OperationRead,
			After: map[string]any{
				"event_id":               "e20",
				"event_seq":              int64(20),
				"event_type":             "asset_created",
				"aggregate_type":         "asset",
				"payload_schema_version": "v1",
				"event_source":           "backend",
				"publish_state":          "pending",
				"event_payload":          `{"asset_id":"a20"}`,
				"occurred_at":            now,
				"created_at":             now,
			},
		},
	})
	if err != nil {
		t.Fatalf("HandleBatch returned error: %v", err)
	}
	if len(sink.batches) != 0 {
		t.Fatalf("expected no batches when snapshot reads are disabled, got %+v", sink.batches)
	}
}
