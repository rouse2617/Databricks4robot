package models

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// TestSyncLegacyFields_DurationConversion verifies DurationSec is computed from DurationMs.
func TestSyncLegacyFields_DurationConversion(t *testing.T) {
	a := Asset{DurationMs: 12500, AssetType: "segment"}
	a.SyncLegacyFields()
	if a.DurationSec != 12.5 {
		t.Fatalf("expected DurationSec=12.5, got %f", a.DurationSec)
	}
}

// TestSyncLegacyFields_SegTypeMirror verifies SegType mirrors AssetType.
func TestSyncLegacyFields_SegTypeMirror(t *testing.T) {
	a := Asset{AssetType: "clip", DurationMs: 0}
	a.SyncLegacyFields()
	if a.SegType != "clip" {
		t.Fatalf("expected SegType='clip', got %q", a.SegType)
	}
}

// TestSyncLegacyFields_ZeroDuration verifies zero DurationMs produces zero DurationSec.
func TestSyncLegacyFields_ZeroDuration(t *testing.T) {
	a := Asset{DurationMs: 0, AssetType: "segment"}
	a.SyncLegacyFields()
	if a.DurationSec != 0 {
		t.Fatalf("expected DurationSec=0, got %f", a.DurationSec)
	}
}

// Property 11: API response backward compatibility
//
// For any asset with both legacy and new fields populated, the JSON-serialized
// API response SHALL contain all of: "status" AND "lifecycle_state",
// "duration_sec" AND "duration_ms", "type" AND "asset_type", plus
// "algo_results", "tags", and "files" map fields.
//
// **Validates: Requirements 14.3, 19.1, 19.2, 19.3, 19.4**
func TestProperty11_APIResponseBackwardCompatibility(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate an asset with both legacy and new fields populated.
		assetType := rapid.SampledFrom([]string{"segment", "clip", "frame_set", "derived_asset"}).Draw(t, "asset_type")
		lifecycleState := rapid.SampledFrom([]string{"created", "processing", "ready", "rejected", "delivered", "archived", "superseded"}).Draw(t, "lifecycle_state")
		status := rapid.SampledFrom([]AssetStatus{AssetStatusApproved, AssetStatusRejected, AssetStatusSuperseded, AssetStatusArchived}).Draw(t, "status")
		durationMs := rapid.Int64Range(0, 3_600_000).Draw(t, "duration_ms")

		// Generate non-empty maps for legacy fields
		tagCount := rapid.IntRange(1, 5).Draw(t, "tag_count")
		tags := make(map[string]string, tagCount)
		for i := 0; i < tagCount; i++ {
			k := rapid.StringMatching(`[a-z]{1,10}`).Draw(t, "tag_key")
			v := rapid.StringMatching(`[a-z]{1,10}`).Draw(t, "tag_val")
			tags[k] = v
		}

		algoCount := rapid.IntRange(1, 3).Draw(t, "algo_count")
		algoResults := make(map[string]string, algoCount)
		for i := 0; i < algoCount; i++ {
			k := rapid.StringMatching(`[a-z]{1,8}@[0-9]\.[0-9]:[a-z]+`).Draw(t, "algo_key")
			v := rapid.StringMatching(`[a-z]{1,10}`).Draw(t, "algo_val")
			algoResults[k] = v
		}

		fileCount := rapid.IntRange(1, 3).Draw(t, "file_count")
		files := make(map[string]string, fileCount)
		for i := 0; i < fileCount; i++ {
			k := rapid.StringMatching(`[a-z]{1,8}`).Draw(t, "file_key")
			v := rapid.StringMatching(`gs://[a-z]{1,20}`).Draw(t, "file_val")
			files[k] = v
		}

		now := time.Now().UTC().Truncate(time.Second)
		a := Asset{
			AssetID:        "test-asset-id",
			McapFileID:     "test-mcap-id",
			Status:         status,
			SegType:        assetType,
			DurationMs:     durationMs,
			AssetType:      assetType,
			LifecycleState: lifecycleState,
			AlgoResults:    algoResults,
			Tags:           tags,
			Files:          files,
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		// Sync legacy fields from new typed fields
		a.SyncLegacyFields()

		// Serialize to JSON
		data, err := json.Marshal(a)
		if err != nil {
			t.Fatalf("json.Marshal failed: %v", err)
		}
		jsonStr := string(data)

		// Verify all required fields are present in the JSON output.
		requiredKeys := []string{
			`"status"`,
			`"lifecycle_state"`,
			`"duration_sec"`,
			`"duration_ms"`,
			`"asset_type"`,
			`"algo_results"`,
			`"tags"`,
			`"files"`,
		}
		for _, key := range requiredKeys {
			if !strings.Contains(jsonStr, key) {
				t.Fatalf("JSON missing required key %s in: %s", key, jsonStr)
			}
		}

		// Also verify "type" is present (SegType field with json:"type")
		// SegType uses omitempty, but SyncLegacyFields sets it from AssetType,
		// so it should be present when AssetType is non-empty.
		if assetType != "" && !strings.Contains(jsonStr, `"type"`) {
			t.Fatalf("JSON missing required key \"type\" in: %s", jsonStr)
		}

		// Verify DurationSec is consistent with DurationMs
		var decoded map[string]interface{}
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("json.Unmarshal failed: %v", err)
		}
		gotDurationSec, ok := decoded["duration_sec"].(float64)
		if !ok {
			t.Fatalf("duration_sec not a float64 in decoded JSON")
		}
		expectedDurationSec := float64(durationMs) / 1000.0
		if gotDurationSec != expectedDurationSec {
			t.Fatalf("duration_sec mismatch: got %f, want %f", gotDurationSec, expectedDurationSec)
		}
	})
}
