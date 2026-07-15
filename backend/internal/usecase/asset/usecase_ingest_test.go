package asset

import (
	"context"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// ─── Test Helpers ───────────────────────────────────────────────────────────

func buildTestAlgoRegistry(t *testing.T) *config.AlgoRegistry {
	t.Helper()
	reg, err := config.LoadAlgoRegistry("../../../config/algo_registry.yaml")
	if err != nil {
		t.Fatalf("failed to load algo registry: %v", err)
	}
	return reg
}

// ─── Task 11.1: Lifecycle Field Defaults ────────────────────────────────────

func TestCreate_LifecycleDefaults(t *testing.T) {
	repo := newMockAssetRepo()
	algoReg := buildTestAlgoRegistry(t)
	uc := NewFull(repo, nil, algoReg)
	ctx := context.Background()

	asset, err := uc.Create(ctx, CreateInput{
		McapFileID:       "mcap-test-001",
		StartTimestampNs: 1000,
		EndTimestampNs:   1001000,
		Reviewer:         "tester",
		Owner:            "owner",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify lifecycle defaults in LifecycleMeta.
	checks := []struct {
		key      string
		expected interface{}
	}{
		{"retention_tier", "standard"},
		{"archive_after_days", 90},
		{"delete_after_days", 365},
		{"total_size_bytes", 0},
	}
	for _, c := range checks {
		val, ok := asset.LifecycleMeta[c.key]
		if !ok {
			t.Errorf("lifecycle field %q not set", c.key)
			continue
		}
		// Compare as int for numeric values (Go map stores int directly).
		switch expected := c.expected.(type) {
		case int:
			got, ok := val.(int)
			if !ok {
				t.Errorf("lifecycle field %q: expected int, got %T", c.key, val)
				continue
			}
			if got != expected {
				t.Errorf("lifecycle field %q: expected %d, got %d", c.key, expected, got)
			}
		case string:
			got, ok := val.(string)
			if !ok {
				t.Errorf("lifecycle field %q: expected string, got %T", c.key, val)
				continue
			}
			if got != expected {
				t.Errorf("lifecycle field %q: expected %q, got %q", c.key, expected, got)
			}
		}
	}

	// last_accessed_at should be nil.
	val, ok := asset.LifecycleMeta["last_accessed_at"]
	if !ok {
		t.Error("lifecycle field last_accessed_at not set")
	} else if val != nil {
		t.Errorf("lifecycle field last_accessed_at: expected nil, got %v", val)
	}
}

// CYB-3267 Bug 2: Create sets duration_ms directly from the timestamp span
// (the mock repo does not run prepAssetForWrite, so this isolates the
// create-site assignment). Without the fix, DurationMs would be 0 here.
func TestCreate_SetsDurationMsFromSpan(t *testing.T) {
	repo := newMockAssetRepo()
	uc := NewFull(repo, nil, buildTestAlgoRegistry(t))
	ctx := context.Background()

	asset, err := uc.Create(ctx, CreateInput{
		McapFileID:       "mcap-test-001",
		StartTimestampNs: 1_000_000_000,
		EndTimestampNs:   3_000_000_000, // span = 2s
		Reviewer:         "tester",
		Owner:            "owner",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if asset.DurationMs != 2000 {
		t.Errorf("DurationMs = %d, want 2000", asset.DurationMs)
	}
	if asset.DurationSec != 2 {
		t.Errorf("DurationSec = %v, want 2", asset.DurationSec)
	}
}

func TestCreate_DefaultAssetTypeWhenMissing(t *testing.T) {
	repo := newMockAssetRepo()
	uc := New(repo)
	ctx := context.Background()

	asset, err := uc.Create(ctx, CreateInput{
		McapFileID:       "mcap-test-assettype-001",
		StartTimestampNs: 1000,
		EndTimestampNs:   1001000,
		Reviewer:         "tester",
		Owner:            "owner",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if asset.AssetType != "segment" {
		t.Fatalf("expected asset_type=segment, got %q", asset.AssetType)
	}
	if asset.SegType != "segment" {
		t.Fatalf("expected type=segment, got %q", asset.SegType)
	}
}

func TestCreate_TrimAndMirrorAssetTypeFields(t *testing.T) {
	repo := newMockAssetRepo()
	uc := New(repo)
	ctx := context.Background()

	asset, err := uc.Create(ctx, CreateInput{
		McapFileID:       "mcap-test-assettype-002",
		StartTimestampNs: 1000,
		EndTimestampNs:   1001000,
		Reviewer:         "tester",
		Owner:            "owner",
		AssetType:        " clip ",
		SegType:          "   ",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if asset.AssetType != "clip" {
		t.Fatalf("expected trimmed asset_type=clip, got %q", asset.AssetType)
	}
	if asset.SegType != "clip" {
		t.Fatalf("expected mirrored type=clip, got %q", asset.SegType)
	}
}

func TestCreate_DatasetAllowsMissingMcapAndValidatesMetadata(t *testing.T) {
	repo := newMockAssetRepo()
	uc := New(repo)
	ctx := context.Background()

	asset, err := uc.Create(ctx, CreateInput{
		StartTimestampNs: 1000,
		EndTimestampNs:   1001000,
		Reviewer:         "tester",
		AssetType:        "dataset",
		Metadata: map[string]interface{}{
			"format":            "csv",
			"record_count":      float64(12),
			"annotation_status": "raw",
		},
	})
	if err != nil {
		t.Fatalf("Create dataset failed: %v", err)
	}
	if asset.AssetType != "dataset" {
		t.Fatalf("expected asset_type=dataset, got %q", asset.AssetType)
	}

	_, err = uc.Create(ctx, CreateInput{
		StartTimestampNs: 1000,
		EndTimestampNs:   1001000,
		Reviewer:         "tester",
		AssetType:        "dataset",
		Metadata:         map[string]interface{}{"format": "jsonl"},
	})
	if err == nil {
		t.Fatal("expected invalid dataset metadata to fail")
	}
}

func TestGetAssetTypeSchema(t *testing.T) {
	uc := New(newMockAssetRepo())
	if schema, ok := uc.GetAssetTypeSchema("annotation_result"); !ok || len(schema) == 0 {
		t.Fatal("expected annotation_result schema")
	}
	if schema, ok := uc.GetAssetTypeSchema("unknown"); ok || schema != nil {
		t.Fatal("expected unknown schema to be absent")
	}
}

func TestCommitSegments_LifecycleDefaults(t *testing.T) {
	repo := newMockAssetRepo()
	algoReg := buildTestAlgoRegistry(t)
	uc := NewFull(repo, nil, algoReg)
	ctx := context.Background()

	ids, err := uc.CommitSegments(ctx, CommitSegmentsInput{
		McapFileID: "mcap-test-002",
		Ranges:     [][2]int64{{1000000, 2000000}, {3000000, 4000000}},
		Reviewer:   "tester",
		Owner:      "owner",
	})
	if err != nil {
		t.Fatalf("CommitSegments failed: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 assets, got %d", len(ids))
	}

	for _, id := range ids {
		asset, err := repo.Get(ctx, id)
		if err != nil || asset == nil {
			t.Fatalf("failed to get asset %s: %v", id, err)
		}
		if asset.LifecycleMeta == nil {
			t.Fatalf("asset %s: LifecycleMeta is nil", id)
		}
		if v, ok := asset.LifecycleMeta["retention_tier"]; !ok || v != "standard" {
			t.Errorf("asset %s: retention_tier expected 'standard', got %v", id, v)
		}
		if v, ok := asset.LifecycleMeta["total_size_bytes"]; !ok || v != 0 {
			t.Errorf("asset %s: total_size_bytes expected 0, got %v", id, v)
		}
	}
}

// ─── Task 11.2: Algorithm State Initialization ──────────────────────────────

func TestCreate_AlgoStateInitialization(t *testing.T) {
	repo := newMockAssetRepo()
	algoReg := buildTestAlgoRegistry(t)
	uc := NewFull(repo, nil, algoReg)
	ctx := context.Background()

	asset, err := uc.Create(ctx, CreateInput{
		McapFileID:       "mcap-test-003",
		StartTimestampNs: 1000,
		EndTimestampNs:   1001000,
		Reviewer:         "tester",
		Owner:            "owner",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	allAlgos := algoReg.GetAllAlgorithms()

	// Verify all algorithms have their status initialized.
	for algoName, def := range allAlgos {
		for _, ver := range def.Versions {
			algoKey := algoName + "@" + ver
			statusKey := algoKey + ":" + models.AlgoFieldStatus
			status, ok := asset.AlgoResults[statusKey]
			if !ok {
				t.Errorf("algo %s: status not initialized", algoKey)
				continue
			}

			if len(def.DependsOn) == 0 {
				// No dependencies → should be pending.
				if status != string(models.AlgoStatusPending) {
					t.Errorf("algo %s (no deps): expected pending, got %q", algoKey, status)
				}
			} else {
				// Has dependencies → should be blocked.
				if status != string(models.AlgoStatusBlocked) {
					t.Errorf("algo %s (has deps): expected blocked, got %q", algoKey, status)
				}
			}
		}
	}
}

func TestCreate_AlgoStateInitialization_ActionAnnotationBlocked(t *testing.T) {
	repo := newMockAssetRepo()
	algoReg := buildTestAlgoRegistry(t)
	uc := NewFull(repo, nil, algoReg)
	ctx := context.Background()

	asset, err := uc.Create(ctx, CreateInput{
		McapFileID:       "mcap-test-004",
		StartTimestampNs: 1000,
		EndTimestampNs:   1001000,
		Reviewer:         "tester",
		Owner:            "owner",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// action_annotation@1.0.0 depends on hand_tracking@1.2.0, head_tracking@1.0.0, body_tracking@1.0.0
	// → should be blocked.
	statusKey := "action_annotation@1.0.0:" + models.AlgoFieldStatus
	status := asset.AlgoResults[statusKey]
	if status != string(models.AlgoStatusBlocked) {
		t.Errorf("action_annotation@1.0.0: expected blocked, got %q", status)
	}

	// env_analysis@1.0.0 has no dependencies → should be pending.
	statusKey = "env_analysis@1.0.0:" + models.AlgoFieldStatus
	status = asset.AlgoResults[statusKey]
	if status != string(models.AlgoStatusPending) {
		t.Errorf("env_analysis@1.0.0: expected pending, got %q", status)
	}

	// hand_tracking@1.2.0 has no dependencies → should be pending.
	statusKey = "hand_tracking@1.2.0:" + models.AlgoFieldStatus
	status = asset.AlgoResults[statusKey]
	if status != string(models.AlgoStatusPending) {
		t.Errorf("hand_tracking@1.2.0: expected pending, got %q", status)
	}
}

func TestCommitSegments_AlgoStateInitialization(t *testing.T) {
	repo := newMockAssetRepo()
	algoReg := buildTestAlgoRegistry(t)
	uc := NewFull(repo, nil, algoReg)
	ctx := context.Background()

	ids, err := uc.CommitSegments(ctx, CommitSegmentsInput{
		McapFileID: "mcap-test-005",
		Ranges:     [][2]int64{{1000000, 2000000}},
		Reviewer:   "tester",
		Owner:      "owner",
	})
	if err != nil {
		t.Fatalf("CommitSegments failed: %v", err)
	}

	asset, _ := repo.Get(ctx, ids[0])

	// Verify action_annotation is blocked and env_analysis is pending.
	aaStatus := asset.AlgoResults["action_annotation@1.0.0:"+models.AlgoFieldStatus]
	if aaStatus != string(models.AlgoStatusBlocked) {
		t.Errorf("action_annotation@1.0.0: expected blocked, got %q", aaStatus)
	}
	envStatus := asset.AlgoResults["env_analysis@1.0.0:"+models.AlgoFieldStatus]
	if envStatus != string(models.AlgoStatusPending) {
		t.Errorf("env_analysis@1.0.0: expected pending, got %q", envStatus)
	}
}

// ─── Task 11.2: files["raw_mcap"] Written ───────────────────────────────

func TestCreate_CfFilesRawMcap(t *testing.T) {
	repo := newMockAssetRepo()
	algoReg := buildTestAlgoRegistry(t)
	uc := NewFull(repo, nil, algoReg)
	ctx := context.Background()

	asset, err := uc.Create(ctx, CreateInput{
		McapFileID:       "mcap-test-006",
		StartTimestampNs: 1000,
		EndTimestampNs:   1001000,
		Reviewer:         "tester",
		Owner:            "owner",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// files["raw_mcap"] should be set to the mcap_file_id.
	rawMcap, ok := asset.Files["raw_mcap"]
	if !ok {
		t.Fatal("files[raw_mcap] not set")
	}
	if rawMcap != "mcap-test-006" {
		t.Errorf("files[raw_mcap]: expected 'mcap-test-006', got %q", rawMcap)
	}
}

func TestCommitSegments_CfFilesRawMcap(t *testing.T) {
	repo := newMockAssetRepo()
	algoReg := buildTestAlgoRegistry(t)
	uc := NewFull(repo, nil, algoReg)
	ctx := context.Background()

	ids, err := uc.CommitSegments(ctx, CommitSegmentsInput{
		McapFileID: "mcap-test-007",
		Ranges:     [][2]int64{{1000000, 2000000}, {3000000, 4000000}},
		Reviewer:   "tester",
		Owner:      "owner",
	})
	if err != nil {
		t.Fatalf("CommitSegments failed: %v", err)
	}

	for _, id := range ids {
		asset, _ := repo.Get(ctx, id)
		rawMcap, ok := asset.Files["raw_mcap"]
		if !ok {
			t.Errorf("asset %s: files[raw_mcap] not set", id)
			continue
		}
		if rawMcap != "mcap-test-007" {
			t.Errorf("asset %s: files[raw_mcap] expected 'mcap-test-007', got %q", id, rawMcap)
		}
	}
}

// ─── Backward Compatibility: No AlgoRegistry ────────────────────────────────

func TestCreate_WithoutAlgoRegistry_NoAlgoStates(t *testing.T) {
	repo := newMockAssetRepo()
	// Use New() without algo registry — backward compatible.
	uc := New(repo)
	ctx := context.Background()

	asset, err := uc.Create(ctx, CreateInput{
		McapFileID:       "mcap-test-008",
		StartTimestampNs: 1000,
		EndTimestampNs:   1001000,
		Reviewer:         "tester",
		Owner:            "owner",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Without algo registry, AlgoResults should be empty (no algo states initialized).
	if len(asset.AlgoResults) != 0 {
		t.Errorf("expected empty AlgoResults without algo registry, got %d entries", len(asset.AlgoResults))
	}

	// Files should not have raw_mcap without algo registry.
	if _, ok := asset.Files["raw_mcap"]; ok {
		t.Error("expected no raw_mcap in Files without algo registry")
	}

	// But lifecycle defaults should still be set.
	if asset.LifecycleMeta == nil {
		t.Fatal("LifecycleMeta should be set even without algo registry")
	}
	if v, ok := asset.LifecycleMeta["retention_tier"]; !ok || v != "standard" {
		t.Errorf("retention_tier expected 'standard', got %v", v)
	}
}
