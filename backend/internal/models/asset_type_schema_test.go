package models

import (
	"encoding/json"
	"testing"
)

func TestSchemaRegistry_GetSchema(t *testing.T) {
	reg := NewSchemaRegistry()
	schema := reg.GetSchema("dataset")
	if len(schema) == 0 {
		t.Fatal("expected dataset schema")
	}
	if !json.Valid(schema) {
		t.Fatal("dataset schema is not valid JSON")
	}
	if got := reg.GetSchema("unknown"); got != nil {
		t.Fatalf("expected nil schema for unknown type, got %s", string(got))
	}
}

func TestSchemaRegistry_ValidateDataset(t *testing.T) {
	reg := NewSchemaRegistry()
	valid := map[string]interface{}{
		"format":            "parquet",
		"record_count":      float64(10),
		"size_bytes":        int64(2048),
		"annotation_status": "raw",
		"time_range": map[string]interface{}{
			"start": "2026-05-28T00:00:00Z",
			"end":   "2026-05-28T01:00:00Z",
		},
		"source": "gs://bucket/raw",
	}
	if err := reg.Validate("dataset", valid); err != nil {
		t.Fatalf("valid dataset metadata rejected: %v", err)
	}
	if err := reg.Validate("dataset", map[string]interface{}{"format": "jsonl"}); err == nil {
		t.Fatal("expected invalid format to be rejected")
	}
	if err := reg.Validate("dataset", map[string]interface{}{"record_count": 1.5}); err == nil {
		t.Fatal("expected non-integer record_count to be rejected")
	}
}

func TestSchemaRegistry_ValidateAnnotationResult(t *testing.T) {
	reg := NewSchemaRegistry()
	valid := map[string]interface{}{
		"tool":           "label-studio",
		"schema_version": "v1",
		"annotators":     []interface{}{"alice", "bob"},
		"quality_score":  0.95,
		"coverage":       1.0,
		"artifact_uri":   "gs://bucket/annotations.json",
	}
	if err := reg.Validate("annotation_result", valid); err != nil {
		t.Fatalf("valid annotation_result metadata rejected: %v", err)
	}
	if err := reg.Validate("annotation_result", map[string]interface{}{"coverage": 1.2}); err == nil {
		t.Fatal("expected coverage > 1 to be rejected")
	}
	if err := reg.Validate("annotation_result", map[string]interface{}{"annotators": []interface{}{"alice", 42}}); err == nil {
		t.Fatal("expected non-string annotator to be rejected")
	}
}

func TestSchemaRegistry_ValidateUnknownTypeNoop(t *testing.T) {
	reg := NewSchemaRegistry()
	if err := reg.Validate("legacy_type", map[string]interface{}{"anything": []interface{}{1}}); err != nil {
		t.Fatalf("unknown types should fall back outside registry, got %v", err)
	}
}
