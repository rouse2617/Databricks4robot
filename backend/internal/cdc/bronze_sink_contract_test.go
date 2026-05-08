package cdc

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"data-platform/internal/models"
)

func TestBronzeSinkContract_WritesOrderedJSONLLinesAndNoTmpResidue(t *testing.T) {
	dir := t.TempDir()
	sink := NewBronzeSink(BronzeSinkConfig{Enabled: true, StagingDir: dir})
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)

	events := []*models.AssetEvent{
		{
			EventID:              "evt-100",
			EventSeq:             100,
			EventType:            "asset_created",
			AggregateType:        "asset",
			PayloadSchemaVersion: "v1",
			AssetID:              "a100",
			EventSource:          "backend",
			PublishState:         "pending",
			EventPayload:         json.RawMessage(`{"asset_id":"a100"}`),
			OccurredAt:           now,
			CreatedAt:            now,
		},
		{
			EventID:              "evt-101",
			EventSeq:             101,
			EventType:            "asset_updated",
			AggregateType:        "asset",
			PayloadSchemaVersion: "v1",
			AssetID:              "a101",
			EventSource:          "backend",
			PublishState:         "pending",
			EventPayload:         json.RawMessage(`{"asset_id":"a101"}`),
			OccurredAt:           now,
			CreatedAt:            now,
		},
	}

	path, err := sink.WriteBatch(events)
	if err != nil {
		t.Fatalf("WriteBatch returned error: %v", err)
	}

	matches, err := filepath.Glob(filepath.Join(dir, "*.tmp"))
	if err != nil {
		t.Fatalf("Glob tmp: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("expected no tmp residue, got %v", matches)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open written file: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var rows []stagingEvent
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var row stagingEvent
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatalf("unmarshal line: %v", err)
		}
		rows = append(rows, row)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan written file: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].EventSeq != 100 || rows[1].EventSeq != 101 {
		t.Fatalf("expected stable event order, got %+v", rows)
	}
	if rows[0].OccurredAt == "" || rows[0].CreatedAt == "" {
		t.Fatalf("expected timestamp fields to be populated, got %+v", rows[0])
	}
}
