package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadDLQFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dlq.jsonl")
	line := `{"time":"2026-05-07T00:00:00Z","stage":"decode","topic":"topic.assets","key":{"asset_id":"a1"},"value_base64":"e30=","error":"bad payload"}`
	if err := os.WriteFile(path, []byte(line+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}
	records, err := readDLQFile(path)
	if err != nil {
		t.Fatalf("readDLQFile error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Topic != "topic.assets" {
		t.Fatalf("unexpected topic: %s", records[0].Topic)
	}
}
