// Package outbox — bronze_sink.go implements the staging writer for the
// two-stage Iceberg ingestion pipeline (§5.6.2 of data-platform-design.md).
//
// The Bronze Sink writes batches of asset_events as JSONL files to a staging
// directory. File names encode the event_seq range for idempotent MERGE INTO
// by the downstream PyIceberg CronJob.
//
// File naming convention:
//
//	staging/events_{minSeq}_{maxSeq}_{timestamp}.jsonl
//
// The PyIceberg CronJob lists new files in the staging directory, reads them,
// and performs MERGE INTO bronze.asset_events USING staging ON event_seq
// (idempotent dedup). After successful merge, the CronJob deletes the
// consumed staging files.
package outbox

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"data-platform/internal/models"
)

// BronzeSinkConfig holds configuration for the staging parquet writer.
type BronzeSinkConfig struct {
	// Enabled controls whether the Bronze Sink writes staging files.
	// Defaults to false (disabled).
	Enabled bool

	// StagingDir is the local directory where staging JSONL files are written.
	// In production this maps to a MinIO/S3 bucket mount or is replaced by
	// direct S3 writes.
	// Defaults to "/tmp/iceberg-staging" if empty.
	StagingDir string
}

// BronzeSink writes outbox event batches to staging JSONL files for
// downstream PyIceberg MERGE INTO bronze.
type BronzeSink struct {
	stagingDir string
}

// stagingEvent is the JSON schema written to staging files. It mirrors the
// asset_events table columns needed by the Bronze Iceberg table.
type stagingEvent struct {
	EventID              string          `json:"event_id"`
	EventSeq             int64           `json:"event_seq"`
	EventType            string          `json:"event_type"`
	AggregateType        string          `json:"aggregate_type"`
	PayloadSchemaVersion string          `json:"payload_schema_version"`
	AssetID              string          `json:"asset_id"`
	McapFileID           string          `json:"mcap_file_id"`
	TenantID             string          `json:"tenant_id"`
	ProjectID            string          `json:"project_id"`
	EventSource          string          `json:"event_source"`
	PublishState         string          `json:"publish_state"`
	EventPayload         json.RawMessage `json:"event_payload"`
	OccurredAt           string          `json:"occurred_at"`
	CreatedAt            string          `json:"created_at"`
}

// NewBronzeSink creates a BronzeSink. Returns nil if cfg.Enabled is false.
func NewBronzeSink(cfg BronzeSinkConfig) *BronzeSink {
	if !cfg.Enabled {
		return nil
	}
	dir := cfg.StagingDir
	if dir == "" {
		dir = "/tmp/iceberg-staging"
	}
	return &BronzeSink{stagingDir: dir}
}

// WriteBatch writes a batch of events to a staging JSONL file.
// The file name encodes the event_seq range for downstream idempotent merge.
// Returns the path of the written file, or empty string if the batch is empty.
func (s *BronzeSink) WriteBatch(events []*models.AssetEvent) (string, error) {
	if s == nil || len(events) == 0 {
		return "", nil
	}

	// Ensure staging directory exists.
	if err := os.MkdirAll(s.stagingDir, 0o755); err != nil {
		return "", fmt.Errorf("bronze sink: mkdir staging: %w", err)
	}

	// Determine event_seq range.
	minSeq := events[0].EventSeq
	maxSeq := events[0].EventSeq
	for _, e := range events[1:] {
		if e.EventSeq < minSeq {
			minSeq = e.EventSeq
		}
		if e.EventSeq > maxSeq {
			maxSeq = e.EventSeq
		}
	}

	// File name: events_{minSeq}_{maxSeq}_{unixNano}.jsonl
	filename := fmt.Sprintf("events_%d_%d_%d.jsonl", minSeq, maxSeq, time.Now().UnixNano())
	path := filepath.Join(s.stagingDir, filename)

	// Write to a temp file first, then rename for atomicity.
	tmpPath := path + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return "", fmt.Errorf("bronze sink: create temp file: %w", err)
	}

	enc := json.NewEncoder(f)
	for _, e := range events {
		se := stagingEvent{
			EventID:              e.EventID,
			EventSeq:             e.EventSeq,
			EventType:            e.EventType,
			AggregateType:        e.AggregateType,
			PayloadSchemaVersion: e.PayloadSchemaVersion,
			AssetID:              e.AssetID,
			McapFileID:           e.McapFileID,
			TenantID:             e.TenantID,
			ProjectID:            e.ProjectID,
			EventSource:          e.EventSource,
			PublishState:         e.PublishState,
			EventPayload:         e.EventPayload,
			OccurredAt:           e.OccurredAt.Format(time.RFC3339Nano),
			CreatedAt:            e.CreatedAt.Format(time.RFC3339Nano),
		}
		if err := enc.Encode(se); err != nil {
			f.Close()
			os.Remove(tmpPath)
			return "", fmt.Errorf("bronze sink: encode event %d: %w", e.EventSeq, err)
		}
	}

	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("bronze sink: close temp file: %w", err)
	}

	// Atomic rename.
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("bronze sink: rename staging file: %w", err)
	}

	slog.Info("bronze sink: wrote staging file",
		"path", path,
		"events", len(events),
		"min_seq", minSeq,
		"max_seq", maxSeq,
	)
	return path, nil
}
