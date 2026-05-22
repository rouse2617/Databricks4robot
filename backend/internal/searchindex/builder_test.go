package searchindex

import (
	"context"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// ── stub repositories ────────────────────────────────────────────────────────

type stubAssetRepo struct {
	asset *models.Asset
}

func (s *stubAssetRepo) Get(ctx context.Context, id string) (*models.Asset, error) {
	return s.GetAll(ctx, id)
}
func (s *stubAssetRepo) GetAll(_ context.Context, _ string) (*models.Asset, error) {
	return s.asset, nil
}
func (s *stubAssetRepo) InsertNew(context.Context, *models.Asset) error { return nil }
func (s *stubAssetRepo) Set(context.Context, *models.Asset) error       { return nil }
func (s *stubAssetRepo) SoftDelete(context.Context, string) error       { return nil }
func (s *stubAssetRepo) ListByMcapFile(context.Context, string) ([]*models.Asset, error) {
	return nil, nil
}
func (s *stubAssetRepo) WriteSegmentIndex(context.Context, *models.Asset) error { return nil }
func (s *stubAssetRepo) ListWithFilters(_ context.Context, _ string, _ []interface{}, page, pageSize int, _ filter.OrderByClause) ([]*models.Asset, int64, error) {
	return nil, 0, nil
}

type stubTagRepo struct {
	tags []*models.AssetTag
}

func (s *stubTagRepo) Upsert(context.Context, repository.AssetTagUpsertInput) error {
	return nil
}
func (s *stubTagRepo) ListByAsset(_ context.Context, _ string) ([]*models.AssetTag, error) {
	return s.tags, nil
}
func (s *stubTagRepo) Delete(context.Context, string, string, string) error { return nil }

type stubAlgoRepo struct {
	algos []*models.AssetAlgoLatest
}

func (s *stubAlgoRepo) Upsert(context.Context, *models.AssetAlgoLatest) error { return nil }
func (s *stubAlgoRepo) GetByAlgo(context.Context, string, string) (*models.AssetAlgoLatest, error) {
	return nil, nil
}
func (s *stubAlgoRepo) ListByAsset(_ context.Context, _ string) ([]*models.AssetAlgoLatest, error) {
	return s.algos, nil
}

// ── tests ────────────────────────────────────────────────────────────────────

func TestBuild_LifecycleStateIsPrimary(t *testing.T) {
	now := time.Now().UTC()
	b := &Builder{
		Assets: &stubAssetRepo{asset: &models.Asset{
			AssetID:        "abcd1234",
			McapFileID:     "m-1",
			AssetType:      "segment",
			LifecycleState: "ready",
			Status:         models.AssetStatusApproved,
			CreatedAt:      now,
			UpdatedAt:      now,
		}},
		Tags:  &stubTagRepo{},
		Algos: &stubAlgoRepo{},
	}

	doc, ok, err := b.Build(context.Background(), "abcd1234")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true")
	}

	// lifecycle_state should use the model's LifecycleState value.
	if got := doc["lifecycle_state"]; got != "ready" {
		t.Errorf("lifecycle_state = %q, want %q", got, "ready")
	}
	// status should still be present (dual-write window).
	if got := doc["status"]; got != "approved" {
		t.Errorf("status = %q, want %q", got, "approved")
	}
}

func TestBuild_LifecycleStateFallsBackToStatus(t *testing.T) {
	now := time.Now().UTC()
	b := &Builder{
		Assets: &stubAssetRepo{asset: &models.Asset{
			AssetID:        "zzzz8888",
			McapFileID:     "m-2",
			AssetType:      "segment",
			LifecycleState: "", // not yet backfilled
			Status:         models.AssetStatusRejected,
			CreatedAt:      now,
			UpdatedAt:      now,
		}},
		Tags:  &stubTagRepo{},
		Algos: &stubAlgoRepo{},
	}

	doc, ok, err := b.Build(context.Background(), "zzzz8888")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true")
	}

	// When lifecycle_state is empty, fall back to status for aggregation safety.
	if got := doc["lifecycle_state"]; got != "rejected" {
		t.Errorf("lifecycle_state = %q, want %q (fallback from status)", got, "rejected")
	}
	if got := doc["status"]; got != "rejected" {
		t.Errorf("status = %q, want %q", got, "rejected")
	}
}

func TestBuild_VersionFieldsWhenLogicalAssetSet(t *testing.T) {
	now := time.Now().UTC()
	b := &Builder{
		Assets: &stubAssetRepo{asset: &models.Asset{
			AssetID:        "newrev01",
			LogicalAssetID: "logical1",
			Revision:       2,
			IsCurrent:      true,
			McapFileID:     "m-1",
			AssetType:      "clip",
			LifecycleState: "ready",
			CreatedAt:      now,
			UpdatedAt:      now,
		}},
		Tags:  &stubTagRepo{},
		Algos: &stubAlgoRepo{},
	}
	doc, ok, err := b.Build(context.Background(), "newrev01")
	if err != nil || !ok {
		t.Fatalf("Build: ok=%v err=%v", ok, err)
	}
	if doc["logical_asset_id"] != "logical1" || doc["revision"] != int64(2) || doc["is_current"] != true {
		t.Fatalf("version fields: %#v", doc)
	}
}

func TestBuild_NilAssetReturnsNotOk(t *testing.T) {
	b := &Builder{
		Assets: &stubAssetRepo{asset: nil},
		Tags:   &stubTagRepo{},
		Algos:  &stubAlgoRepo{},
	}

	_, ok, err := b.Build(context.Background(), "missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected ok=false for nil asset")
	}
}
