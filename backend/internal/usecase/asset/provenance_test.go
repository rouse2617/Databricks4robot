package asset

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

type provenanceAssetRepo struct {
	asset     *models.Asset
	byLogical []*models.Asset
}

func (m *provenanceAssetRepo) Get(ctx context.Context, assetID string) (*models.Asset, error) {
	return m.GetAll(ctx, assetID)
}
func (m *provenanceAssetRepo) GetAll(_ context.Context, assetID string) (*models.Asset, error) {
	if m.asset != nil && m.asset.AssetID == assetID {
		return m.asset, nil
	}
	return nil, ErrNotFound
}
func (m *provenanceAssetRepo) InsertNew(context.Context, *models.Asset) error { return nil }
func (m *provenanceAssetRepo) Set(context.Context, *models.Asset) error       { return nil }
func (m *provenanceAssetRepo) SoftDelete(context.Context, string) error       { return nil }
func (m *provenanceAssetRepo) ListByMcapFile(context.Context, string) ([]*models.Asset, error) {
	return nil, nil
}
func (m *provenanceAssetRepo) ListByLogicalAssetID(_ context.Context, logicalID string) ([]*models.Asset, error) {
	if m.byLogical != nil {
		return m.byLogical, nil
	}
	return nil, nil
}
func (m *provenanceAssetRepo) WriteSegmentIndex(context.Context, *models.Asset) error { return nil }
func (m *provenanceAssetRepo) ListDescendants(_ context.Context, _ string) ([]*models.Asset, error) {
	return nil, nil
}
func (m *provenanceAssetRepo) ListWithFilters(context.Context, string, []interface{}, int, int, filter.OrderByClause) ([]*models.Asset, int64, error) {
	return nil, 0, nil
}

type provenanceEventRepo struct {
	promoted []*models.AssetEvent
}

func (m *provenanceEventRepo) Append(context.Context, repository.AssetEventAppendInput) error {
	return nil
}
func (m *provenanceEventRepo) ListPending(context.Context, int) ([]*models.AssetEvent, error) {
	return nil, nil
}
func (m *provenanceEventRepo) ListPendingSafe(context.Context, time.Duration, int) ([]*models.AssetEvent, error) {
	return nil, nil
}
func (m *provenanceEventRepo) ListByAsset(context.Context, string, repository.AssetEventListOptions) ([]*models.AssetEvent, error) {
	return nil, nil
}
func (m *provenanceEventRepo) ListVersionPromotedByLogical(context.Context, string) ([]*models.AssetEvent, error) {
	return m.promoted, nil
}
func (m *provenanceEventRepo) ListGlobal(context.Context, repository.AssetEventListOptions) ([]*models.AssetEvent, error) {
	return nil, nil
}
func (m *provenanceEventRepo) MarkPublished(context.Context, []int64) error { return nil }
func (m *provenanceEventRepo) MarkFailed(context.Context, int64, string) error {
	return nil
}
func (m *provenanceEventRepo) CountPending(context.Context) (int64, error) { return 0, nil }
func (m *provenanceEventRepo) CountPendingClaimable(context.Context, time.Duration) (int64, error) {
	return 0, nil
}
func (m *provenanceEventRepo) CountProcessing(context.Context) (int64, error)    { return 0, nil }
func (m *provenanceEventRepo) OldestPendingAge(context.Context) (float64, error) { return 0, nil }
func (m *provenanceEventRepo) PublishStateCounts(context.Context) (map[string]int64, error) {
	return nil, nil
}
func (m *provenanceEventRepo) ListBetweenSeq(context.Context, int64, int64, int) ([]*models.AssetEvent, error) {
	return nil, nil
}

func TestGetProvenance_LegacySingleRevision(t *testing.T) {
	created := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	uc := New(&provenanceAssetRepo{
		asset: &models.Asset{
			AssetID:   "aaaaaaaa",
			CreatedAt: created,
			IsCurrent: true,
		},
	})
	res, err := uc.GetProvenance(context.Background(), "aaaaaaaa")
	if err != nil {
		t.Fatalf("GetProvenance: %v", err)
	}
	if len(res.VersionHistory) != 1 || res.VersionHistory[0].Version != 1 {
		t.Fatalf("version_history: %+v", res.VersionHistory)
	}
}

func TestGetProvenance_MultiRevisionWithPromoteEvent(t *testing.T) {
	t1 := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC)
	logical := "logical1"
	payload, _ := json.Marshal(map[string]any{
		"run_id": "run-99",
		"reason": "algo rerun",
	})
	assetRepo := &provenanceAssetRepo{
		asset: &models.Asset{
			AssetID:        "bbbbbbbb",
			LogicalAssetID: logical,
			Revision:       2,
			IsCurrent:      true,
			CreatedAt:      t2,
		},
		byLogical: []*models.Asset{
			{AssetID: "aaaaaaaa", LogicalAssetID: logical, Revision: 1, IsCurrent: false, CreatedAt: t1},
			{AssetID: "bbbbbbbb", LogicalAssetID: logical, Revision: 2, IsCurrent: true, CreatedAt: t2},
		},
	}
	eventRepo := &provenanceEventRepo{
		promoted: []*models.AssetEvent{{
			AssetID:      "bbbbbbbb",
			EventType:    "version_promoted",
			OccurredAt:   t2,
			EventPayload: payload,
		}},
	}
	uc := New(assetRepo)
	uc.eventRepo = eventRepo
	res, err := uc.GetProvenance(context.Background(), "bbbbbbbb")
	if err != nil {
		t.Fatalf("GetProvenance: %v", err)
	}
	if len(res.Revisions) != 2 {
		t.Fatalf("revisions: %+v", res.Revisions)
	}
	if len(res.VersionHistory) != 2 {
		t.Fatalf("version_history: %+v", res.VersionHistory)
	}
	v2 := res.VersionHistory[1]
	if v2.ByRunID != "run-99" || v2.Reason != "algo rerun" {
		t.Fatalf("v2 history: %+v", v2)
	}
}
