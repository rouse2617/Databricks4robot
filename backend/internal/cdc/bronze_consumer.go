package cdc

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"data-platform/internal/models"
)

// BronzeEventWriter is the minimal capability needed to materialize Bronze
// staging files. The existing outbox.BronzeSink satisfies this interface.
type BronzeEventWriter interface {
	WriteBatch(events []*models.AssetEvent) (string, error)
}

// BronzeConsumer accepts CDC events for asset_events and materializes only the
// business-event INSERT stream into Bronze staging batches.
type BronzeConsumer struct {
	Sink                 BronzeEventWriter
	IncludeSnapshotReads bool
}

func (c *BronzeConsumer) HandleBatch(ctx context.Context, events []ChangeEvent) error {
	_ = ctx // reserved for future cancellation-aware batching / telemetry

	var batch []*models.AssetEvent
	for _, event := range events {
		if event.Table != "asset_events" {
			continue
		}
		if event.Op != OperationCreate && !(c.IncludeSnapshotReads && event.Op == OperationRead) {
			continue
		}
		ae, err := toAssetEvent(event)
		if err != nil {
			return err
		}
		batch = append(batch, ae)
	}

	if len(batch) == 0 {
		return nil
	}
	CDCBatchProcessedTotal.WithLabelValues("bronze").Add(float64(len(batch)))
	_, err := c.Sink.WriteBatch(batch)
	return err
}

func toAssetEvent(event ChangeEvent) (*models.AssetEvent, error) {
	row := event.After
	if row == nil {
		return nil, fmt.Errorf("cdc bronze consumer: missing after payload for %s", event.Table)
	}

	eventSeq, err := int64Field(row, "event_seq")
	if err != nil {
		return nil, fmt.Errorf("cdc bronze consumer: %w", err)
	}

	occurredAt, err := timeField(row, "occurred_at")
	if err != nil {
		return nil, fmt.Errorf("cdc bronze consumer: %w", err)
	}
	createdAt, err := timeField(row, "created_at")
	if err != nil {
		return nil, fmt.Errorf("cdc bronze consumer: %w", err)
	}

	payload, err := rawJSONField(row, "event_payload")
	if err != nil {
		return nil, fmt.Errorf("cdc bronze consumer: %w", err)
	}

	ae := &models.AssetEvent{
		EventID:              stringField(row, "event_id"),
		EventSeq:             eventSeq,
		EventType:            stringField(row, "event_type"),
		AggregateType:        stringField(row, "aggregate_type"),
		PayloadSchemaVersion: stringField(row, "payload_schema_version"),
		AssetID:              stringField(row, "asset_id"),
		McapFileID:           stringField(row, "mcap_file_id"),
		TenantID:             stringField(row, "tenant_id"),
		ProjectID:            stringField(row, "project_id"),
		EventSource:          stringField(row, "event_source"),
		PublishState:         stringField(row, "publish_state"),
		EventPayload:         payload,
		OccurredAt:           occurredAt,
		CreatedAt:            createdAt,
	}
	return ae, nil
}

func stringField(row map[string]any, key string) string {
	raw, ok := row[key]
	if !ok || raw == nil {
		return ""
	}
	return fmt.Sprint(raw)
}

func int64Field(row map[string]any, key string) (int64, error) {
	raw, ok := row[key]
	if !ok || raw == nil {
		return 0, fmt.Errorf("missing %s", key)
	}
	switch v := raw.(type) {
	case int64:
		return v, nil
	case int32:
		return int64(v), nil
	case int:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid %s: %w", key, err)
		}
		return n, nil
	default:
		return 0, fmt.Errorf("invalid %s type %T", key, raw)
	}
}

func timeField(row map[string]any, key string) (time.Time, error) {
	raw, ok := row[key]
	if !ok || raw == nil {
		return time.Time{}, fmt.Errorf("missing %s", key)
	}
	switch v := raw.(type) {
	case time.Time:
		return v.UTC(), nil
	case string:
		ts, err := time.Parse(time.RFC3339Nano, v)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid %s: %w", key, err)
		}
		return ts.UTC(), nil
	default:
		return time.Time{}, fmt.Errorf("invalid %s type %T", key, raw)
	}
}

func rawJSONField(row map[string]any, key string) (json.RawMessage, error) {
	raw, ok := row[key]
	if !ok || raw == nil {
		return json.RawMessage(`{}`), nil
	}
	switch v := raw.(type) {
	case json.RawMessage:
		return v, nil
	case []byte:
		return json.RawMessage(v), nil
	case string:
		return json.RawMessage(v), nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("invalid %s: %w", key, err)
		}
		return json.RawMessage(b), nil
	}
}
