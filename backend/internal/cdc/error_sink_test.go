package cdc

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileErrorSink_Write(t *testing.T) {
	dir := t.TempDir()
	sink := NewFileErrorSink(FileErrorSinkConfig{Enabled: true, Dir: dir})
	record := FailedRecord{
		Stage:        FailureStageDecode,
		Topic:        "topic.assets",
		Key:          map[string]any{"asset_id": "a1"},
		ValueBase64:  "e30=",
		ErrorMessage: "decode failed",
	}
	if err := sink.Write(context.Background(), record); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir returned error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 file in dlq dir, got %d", len(entries))
	}
	content, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !strings.Contains(string(content), `"topic":"topic.assets"`) {
		t.Fatalf("expected topic in dlq line, got %s", string(content))
	}
}
