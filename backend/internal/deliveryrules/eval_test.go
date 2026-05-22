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
