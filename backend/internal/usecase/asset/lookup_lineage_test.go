package asset

// CYB-4305: batch asset lineage lookup — usecase tests.
//
// Exercises depth=1 (no ES call) and depth=all (ES mock) paths, plus the
// stats logic (has_parent / is_root / orphan / is_current + relation_type
// aggregation). Repo-level SQL is covered in internal/postgres/repos_test.go.

import (
	"context"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// stubLineageBatchRepo returns pre-canned ES projections keyed by asset_id.
// Records the ids it was called with so a test can assert the depth=1 path
// never hits it.
type stubLineageBatchRepo struct {
	byID   map[string]repository.AssetLineageProjection
	called [][]string
}

func (s *stubLineageBatchRepo) LineageDocsByAssetID(_ context.Context, ids []string) (map[string]repository.AssetLineageProjection, error) {
	s.called = append(s.called, append([]string(nil), ids...))
	out := make(map[string]repository.AssetLineageProjection, len(ids))
	for _, id := range ids {
		if p, ok := s.byID[id]; ok {
			out[id] = p
		}
	}
	return out, nil
}

func seedLineageRepo(t *testing.T, repo *mockAssetRepo, a *models.Asset) {
	t.Helper()
	if err := repo.Set(context.Background(), a); err != nil {
		t.Fatalf("seed: %v", err)
	}
}

// TestLookupLineageDepth1DoesNotHitES covers the fast path: depth=1 should
// resolve and answer with pointer fields from PG only, no ES round trip.
func TestLookupLineageDepth1DoesNotHitES(t *testing.T) {
	repo := newMockAssetRepo()
	seedLineageRepo(t, repo, &models.Asset{
		AssetID:        "aaaaaaaa",
		LogicalAssetID: "logical1",
		ParentAssetID:  "parent01",
		RootAssetID:    "root0001",
		IsCurrent:      true,
		Revision:       3,
	})
	seedLineageRepo(t, repo, &models.Asset{
		AssetID: "bbbbbbbb", RootAssetID: "bbbbbbbb", // self-root
	})

	esStub := &stubLineageBatchRepo{byID: map[string]repository.AssetLineageProjection{}}
	uc := New(repo)
	uc.SetLineageBatchRepo(esStub)

	req := models.AssetLineageBatchRequest{IDs: []string{"aaaaaaaa", "bbbbbbbb"}}
	resp, err := uc.LookupLineage(context.Background(), req, false)
	if err != nil {
		t.Fatalf("LookupLineage err: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(resp.Items))
	}
	// depth=1: upstream/downstream/relation pointers must all be nil so the
	// JSON payload omits them entirely.
	for i, it := range resp.Items {
		if it.UpstreamIDs != nil || it.DownstreamIDs != nil || it.RelationTypes != nil {
			t.Fatalf("items[%d] depth=1 must not carry upstream/downstream/relation, got %#v", i, it)
		}
	}
	if len(esStub.called) != 0 {
		t.Fatalf("depth=1 must NOT call ES, got %d calls", len(esStub.called))
	}
	// relation_type_counts must be nil so json omitempty drops it.
	if resp.Stats.RelationTypeCounts != nil {
		t.Fatalf("depth=1 stats.RelationTypeCounts must be nil, got %#v", resp.Stats.RelationTypeCounts)
	}
}

// TestLookupLineageDepthAllOverlaysESProjection covers the full path: the
// ES mock is called for the resolved asset_id set, and its projection lands
// on the item + rolls into relation_type_counts.
func TestLookupLineageDepthAllOverlaysESProjection(t *testing.T) {
	repo := newMockAssetRepo()
	seedLineageRepo(t, repo, &models.Asset{AssetID: "aaaaaaaa", RootAssetID: "aaaaaaaa"})
	seedLineageRepo(t, repo, &models.Asset{AssetID: "bbbbbbbb", ParentAssetID: "aaaaaaaa", RootAssetID: "aaaaaaaa"})

	esStub := &stubLineageBatchRepo{byID: map[string]repository.AssetLineageProjection{
		"aaaaaaaa": {
			UpstreamIDs:   nil,
			DownstreamIDs: []string{"bbbbbbbb", "cccccccc"},
			RelationTypes: []string{"derive", "split"},
		},
		"bbbbbbbb": {
			UpstreamIDs:   []string{"aaaaaaaa"},
			DownstreamIDs: nil,
			RelationTypes: []string{"derive"},
		},
	}}
	uc := New(repo)
	uc.SetLineageBatchRepo(esStub)

	req := models.AssetLineageBatchRequest{IDs: []string{"aaaaaaaa", "bbbbbbbb"}}
	resp, err := uc.LookupLineage(context.Background(), req, true)
	if err != nil {
		t.Fatalf("LookupLineage err: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(resp.Items))
	}
	if len(esStub.called) != 1 || len(esStub.called[0]) != 2 {
		t.Fatalf("expected 1 ES call with 2 ids, got %#v", esStub.called)
	}
	// aaaaaaaa: empty upstream (nil normalized to []), 2 downstream, 2 relations.
	a := resp.Items[0]
	if a.UpstreamIDs == nil || len(*a.UpstreamIDs) != 0 {
		t.Fatalf("aaaaaaaa upstream = %#v, want empty slice", a.UpstreamIDs)
	}
	if a.DownstreamIDs == nil || len(*a.DownstreamIDs) != 2 {
		t.Fatalf("aaaaaaaa downstream = %#v", a.DownstreamIDs)
	}
	if a.RelationTypes == nil || len(*a.RelationTypes) != 2 {
		t.Fatalf("aaaaaaaa relation_types = %#v", a.RelationTypes)
	}
	// Relation-type union across items: derive=2 (one on each), split=1 (a only).
	if got := resp.Stats.RelationTypeCounts; got == nil || (*got)["derive"] != 2 || (*got)["split"] != 1 {
		t.Fatalf("relation_type_counts = %#v, want derive=2 split=1", got)
	}
}

// TestLookupLineageMissingIDsPopulated: unknown ids land in missing_ids and
// stay out of items/stats.
func TestLookupLineageMissingIDsPopulated(t *testing.T) {
	repo := newMockAssetRepo()
	seedLineageRepo(t, repo, &models.Asset{AssetID: "aaaaaaaa"})

	uc := New(repo)
	req := models.AssetLineageBatchRequest{IDs: []string{"aaaaaaaa", "missing1", "missing2"}}
	resp, err := uc.LookupLineage(context.Background(), req, false)
	if err != nil {
		t.Fatalf("LookupLineage err: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(resp.Items))
	}
	if len(resp.MissingIDs) != 2 {
		t.Fatalf("missing = %v, want 2", resp.MissingIDs)
	}
	if resp.Stats.MissingCount != 2 || resp.Stats.MatchedCount != 1 {
		t.Fatalf("stats = %#v", resp.Stats)
	}
}

// TestLookupLineageIsRootCountBothConventions: is_root must count both
// "root_asset_id IS NULL" and "root_asset_id == asset_id" cases as one root
// class (see decisions.md — the historical NULL bucket and the modern
// self-root convention both mean "top of its tree").
func TestLookupLineageIsRootCountBothConventions(t *testing.T) {
	repo := newMockAssetRepo()
	// null-root convention (older ingest left root NULL for self-roots).
	seedLineageRepo(t, repo, &models.Asset{AssetID: "aaaaaaaa"})
	// self-root convention (newer ingest writes root_asset_id = asset_id).
	seedLineageRepo(t, repo, &models.Asset{AssetID: "bbbbbbbb", RootAssetID: "bbbbbbbb"})
	// Not a root: root points at another asset.
	seedLineageRepo(t, repo, &models.Asset{AssetID: "cccccccc", ParentAssetID: "aaaaaaaa", RootAssetID: "aaaaaaaa"})

	uc := New(repo)
	req := models.AssetLineageBatchRequest{IDs: []string{"aaaaaaaa", "bbbbbbbb", "cccccccc"}}
	resp, err := uc.LookupLineage(context.Background(), req, false)
	if err != nil {
		t.Fatalf("LookupLineage err: %v", err)
	}
	if resp.Stats.IsRootCount != 2 {
		t.Fatalf("is_root_count = %d, want 2", resp.Stats.IsRootCount)
	}
	// aaaaaaaa is the "null parent + null root" case → also counts as orphan
	// (see decisions.md). bbbbbbbb has self-root so parent is nil but root
	// is set — NOT an orphan. cccccccc has both parent and root pointing at
	// aaaaaaaa — has_parent, not root, not orphan.
	if resp.Stats.OrphanCount != 1 {
		t.Fatalf("orphan_count = %d, want 1 (aaaaaaaa only)", resp.Stats.OrphanCount)
	}
	if resp.Stats.HasParentCount != 1 {
		t.Fatalf("has_parent_count = %d, want 1", resp.Stats.HasParentCount)
	}
}

// TestLookupLineageOrphanCountStrict: orphan requires BOTH parent+root null.
// A row with parent=null but root=self is NOT an orphan. A row with parent
// set but root also set is a normal descendant.
func TestLookupLineageOrphanCountStrict(t *testing.T) {
	repo := newMockAssetRepo()
	seedLineageRepo(t, repo, &models.Asset{AssetID: "aaaaaaaa"})                                                    // orphan (both null)
	seedLineageRepo(t, repo, &models.Asset{AssetID: "bbbbbbbb", RootAssetID: "bbbbbbbb"})                            // self-root, NOT orphan
	seedLineageRepo(t, repo, &models.Asset{AssetID: "cccccccc", ParentAssetID: "root9999", RootAssetID: "root9999"}) // normal
	seedLineageRepo(t, repo, &models.Asset{AssetID: "dddddddd", ParentAssetID: "root9999", RootAssetID: "root9999"}) // normal

	uc := New(repo)
	req := models.AssetLineageBatchRequest{IDs: []string{"aaaaaaaa", "bbbbbbbb", "cccccccc", "dddddddd"}}
	resp, err := uc.LookupLineage(context.Background(), req, false)
	if err != nil {
		t.Fatalf("LookupLineage err: %v", err)
	}
	if resp.Stats.OrphanCount != 1 {
		t.Fatalf("orphan_count = %d, want 1", resp.Stats.OrphanCount)
	}
	// Root assets: aaaaaaaa (null root) + bbbbbbbb (self-root). cccccccc /
	// dddddddd have a non-null non-self root → NOT root.
	if resp.Stats.IsRootCount != 2 {
		t.Fatalf("is_root_count = %d, want 2", resp.Stats.IsRootCount)
	}
}

// TestLookupLineageIsCurrentCount: only items with IsCurrent=true count.
func TestLookupLineageIsCurrentCount(t *testing.T) {
	repo := newMockAssetRepo()
	seedLineageRepo(t, repo, &models.Asset{AssetID: "aaaaaaaa", IsCurrent: true})
	seedLineageRepo(t, repo, &models.Asset{AssetID: "bbbbbbbb", IsCurrent: false})
	seedLineageRepo(t, repo, &models.Asset{AssetID: "cccccccc", IsCurrent: true})

	uc := New(repo)
	req := models.AssetLineageBatchRequest{IDs: []string{"aaaaaaaa", "bbbbbbbb", "cccccccc"}}
	resp, err := uc.LookupLineage(context.Background(), req, false)
	if err != nil {
		t.Fatalf("LookupLineage err: %v", err)
	}
	if resp.Stats.IsCurrentCount != 2 {
		t.Fatalf("is_current_count = %d, want 2", resp.Stats.IsCurrentCount)
	}
}

// TestLookupLineageDepthAllWithoutESRepo degrades gracefully when the ES
// reader isn't wired: depth=all still returns 200 with empty upstream /
// downstream / relation slices per item so the frontend shape stays stable.
func TestLookupLineageDepthAllWithoutESRepo(t *testing.T) {
	repo := newMockAssetRepo()
	seedLineageRepo(t, repo, &models.Asset{AssetID: "aaaaaaaa"})

	uc := New(repo) // no SetLineageBatchRepo
	req := models.AssetLineageBatchRequest{IDs: []string{"aaaaaaaa"}}
	resp, err := uc.LookupLineage(context.Background(), req, true)
	if err != nil {
		t.Fatalf("LookupLineage err: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(resp.Items))
	}
	it := resp.Items[0]
	if it.UpstreamIDs == nil || it.DownstreamIDs == nil || it.RelationTypes == nil {
		t.Fatalf("depth=all without ES must emit empty slices, got %#v", it)
	}
	if len(*it.UpstreamIDs) != 0 || len(*it.DownstreamIDs) != 0 || len(*it.RelationTypes) != 0 {
		t.Fatalf("depth=all without ES must emit empty slices, got %#v", it)
	}
	// Stats' relation_type_counts must be set (even if empty) so the response
	// carries the map key when depth=all.
	if resp.Stats.RelationTypeCounts == nil {
		t.Fatalf("depth=all stats.RelationTypeCounts must be non-nil, got nil")
	}
}
