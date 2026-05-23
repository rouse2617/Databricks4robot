package deliveryrules

import (
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

func TestMatches_tagEq(t *testing.T) {
	snap := AssetSnapshot{
		AssetID: "abcd1234",
		TagsByKey: map[string][]string{
			"compliance.pii": {"true"},
		},
	}
	dsl := &QueryDSL{
		Where: []Predicate{{Field: "tag.compliance.pii", Op: "eq", Value: "true"}},
	}
	hit, err := Matches(snap, dsl)
	if err != nil || !hit {
		t.Fatalf("expected hit, err=%v hit=%v", err, hit)
	}
}

func TestMatches_assetTypeFilter(t *testing.T) {
	snap := AssetSnapshot{AssetID: "a", AssetType: "clip", TagsByKey: map[string][]string{"x": {"1"}}}
	dsl := &QueryDSL{
		AssetTypes: []string{"action"},
		Where:      []Predicate{{Field: "tag.x", Op: "exists"}},
	}
	hit, _ := Matches(snap, dsl)
	if hit {
		t.Fatal("expected no hit when asset_type mismatch")
	}
}

func TestBuildSnapshot_multiSource(t *testing.T) {
	a := &models.Asset{AssetID: "z", AssetType: "clip", LifecycleState: "ready"}
	tags := []*models.AssetTag{
		{TagKey: "sensitive", TagValue: "true", SourceType: "human"},
		{TagKey: "sensitive", TagValue: "false", SourceType: "algo"},
	}
	snap := BuildSnapshot(a, tags)
	if len(snap.TagsByKey["sensitive"]) != 2 {
		t.Fatalf("got %v", snap.TagsByKey["sensitive"])
	}
}

// TestBuildSnapshot_logicalAllMerge simulates the merge that
// loadSnapshotLogicalAll performs (CYB-1051).
func TestBuildSnapshot_logicalAllMerge(t *testing.T) {
	// Current revision has tag "quality=high"
	current := &models.Asset{AssetID: "rev1", AssetType: "clip", LifecycleState: "ready", LogicalAssetID: "log1"}
	currentTags := []*models.AssetTag{
		{TagKey: "quality", TagValue: "high"},
	}
	base := BuildSnapshot(current, currentTags)

	// Simulate tags from a second revision: "quality=low", "sensitive=true"
	rev2Tags := []*models.AssetTag{
		{TagKey: "quality", TagValue: "low"},
		{TagKey: "sensitive", TagValue: "true"},
	}

	// Merge tags the same way loadSnapshotLogicalAll does.
	merged := map[string][]string{}
	for k, vs := range base.TagsByKey {
		merged[k] = append(merged[k], vs...)
	}
	for _, t := range rev2Tags {
		merged[t.TagKey] = append(merged[t.TagKey], t.TagValue)
	}

	logical := AssetSnapshot{
		AssetID:        base.AssetID,
		AssetType:      base.AssetType,
		LifecycleState: base.LifecycleState,
		TagsByKey:      merged,
	}

	// "quality" should have both "high" and "low" from two revisions.
	if len(logical.TagsByKey["quality"]) != 2 {
		t.Fatalf("expected 2 quality values, got %v", logical.TagsByKey["quality"])
	}
	// "sensitive" should exist from rev2 even though current revision lacks it.
	if len(logical.TagsByKey["sensitive"]) != 1 || logical.TagsByKey["sensitive"][0] != "true" {
		t.Fatalf("expected sensitive=true, got %v", logical.TagsByKey["sensitive"])
	}

	// A rule checking "tag.sensitive eq true" should match on the logical snapshot.
	dsl := &QueryDSL{
		Where: []Predicate{{Field: "tag.sensitive", Op: "eq", Value: "true"}},
	}
	hit, err := Matches(logical, dsl)
	if err != nil || !hit {
		t.Fatalf("expected hit on logical snapshot, err=%v hit=%v", err, hit)
	}

	// But should NOT match on the base (single-revision) snapshot.
	hitBase, err := Matches(base, dsl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hitBase {
		t.Fatal("expected no hit on base snapshot (current revision has no sensitive tag)")
	}
}
