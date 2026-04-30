package outbox

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"data-platform/internal/models"
)

func TestBronzeSink_Disabled(t *testing.T) {
	sink := NewBronzeSink(BronzeSinkConfig{Enabled: false})
	if sink != nil {
		t.Fatal("expected nil sink when disabled")
	}
}

func TestBronzeSink_WriteBatch(t *testing.T) {
	dir := t.TempDir()
	sink := NewBronzeSink(BronzeSinkConfig{Enabled: true, StagingDir: dir})
	if sink == nil {
		t.Fatal("expected non-nil sink")
	}

	now := time.Now()
	events := []*models.AssetEvent{
		{
			EventID:              "evt-001",
			EventSeq:             10,
			EventType:            "asset_created",
			AggregateType:        "asset",
			PayloadSchemaVersion: "v1",
			AssetID:              "asset-001",
			EventSource:          "backend",
			PublishState:         "pending",
			EventPayload:         json.RawMessage(`{"key":"val"}`),
			OccurredAt:           now,
			CreatedAt:            now,
		},
		{
			EventID:              "evt-002",
			EventSeq:             15,
			EventType:            "tag_upserted",
			AggregateType:        "asset",
			PayloadSchemaVersion: "v1",
			AssetID:              "asset-001",
			EventSource:          "backend",
			PublishState:         "pending",
			EventPayload:         json.RawMessage(`{}`),
			OccurredAt:           now,
			CreatedAt:            now,
		},
	}

	path, err := sink.WriteBatch(events)
	if err != nil {
		t.Fatalf("WriteBatch: %v", err)
	}
	if path == "" {
		t.Fatal("expected non-empty path")
	}

	// Verify file name contains seq range.
	base := filepath.Base(path)
	if !strings.HasPrefix(base, "events_10_15_") {
		t.Errorf("unexpected filename: %s", base)
	}
	if !strings.HasSuffix(base, ".jsonl") {
		t.Errorf("expected .jsonl suffix: %s", base)
	}

	// Read and verify content.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read staging file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	var first stagingEvent
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("unmarshal first line: %v", err)
	}
	if first.EventSeq != 10 {
		t.Errorf("first event_seq = %d, want 10", first.EventSeq)
	}
	if first.EventType != "asset_created" {
		t.Errorf("first event_type = %s, want asset_created", first.EventType)
	}
}

func TestBronzeSink_EmptyBatch(t *testing.T) {
	dir := t.TempDir()
	sink := NewBronzeSink(BronzeSinkConfig{Enabled: true, StagingDir: dir})

	path, err := sink.WriteBatch(nil)
	if err != nil {
		t.Fatalf("WriteBatch nil: %v", err)
	}
	if path != "" {
		t.Errorf("expected empty path for nil batch, got %s", path)
	}

	path, err = sink.WriteBatch([]*models.AssetEvent{})
	if err != nil {
		t.Fatalf("WriteBatch empty: %v", err)
	}
	if path != "" {
		t.Errorf("expected empty path for empty batch, got %s", path)
	}
}
