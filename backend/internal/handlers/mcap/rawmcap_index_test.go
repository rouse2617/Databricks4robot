package mcap

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// rawMcapAssetRepoStub is a minimal AssetRepository: InsertNew succeeds so the
// placeholder raw_mcap is "created"; everything else is a no-op.
type rawMcapAssetRepoStub struct{}

func (rawMcapAssetRepoStub) Get(context.Context, string) (*models.Asset, error) { return nil, nil }
func (rawMcapAssetRepoStub) GetAll(context.Context, string) (*models.Asset, error) {
	return nil, nil
}
func (rawMcapAssetRepoStub) FindExistingIDs(context.Context, []string) (map[string]struct{}, error) {
	return map[string]struct{}{}, nil
}
func (rawMcapAssetRepoStub) InsertNew(context.Context, *models.Asset) error { return nil }
func (rawMcapAssetRepoStub) Set(context.Context, *models.Asset) error       { return nil }
func (rawMcapAssetRepoStub) SoftDelete(context.Context, string) error       { return nil }
func (rawMcapAssetRepoStub) ListByMcapFile(context.Context, string) ([]*models.Asset, error) {
	return nil, nil
}
func (rawMcapAssetRepoStub) ListByLogicalAssetID(context.Context, string) ([]*models.Asset, error) {
	return nil, nil
}
func (rawMcapAssetRepoStub) WriteSegmentIndex(context.Context, *models.Asset) error { return nil }
func (rawMcapAssetRepoStub) ListWithFilters(context.Context, string, []interface{}, int, int, filter.OrderByClause) ([]*models.Asset, int64, error) {
	return nil, 0, nil
}
func (rawMcapAssetRepoStub) ListDescendants(context.Context, string) ([]*models.Asset, error) {
	return nil, nil
}

// CYB-3297 Phase D: creating an mcap file must emit an asset-scoped
// asset_created event (with asset_id set) for the placeholder raw_mcap, so the
// ES subscriber indexes it immediately. The mcap_file_created event has no
// asset_id and is dropped by the subscriber; without the asset event, a newly
// ingested raw_mcap is invisible in search until the next full reindex.
func TestCreateFile_EmitsAssetCreatedEventForRawMcap(t *testing.T) {
	repo := &mockMcapRepo{}
	repo.setFn = func(context.Context, *models.McapFile) error { return nil }

	var appended []repository.AssetEventAppendInput
	eventRepo := &mcapEventRepo{
		appendFn: func(_ context.Context, in repository.AssetEventAppendInput) error {
			appended = append(appended, in)
			return nil
		},
	}

	h := New(repo)
	h.SetAssetRepo(rawMcapAssetRepoStub{})
	h.SetEventRepo(eventRepo)

	r := setupMcapRouter(http.MethodPost, "/mcap-files", h.CreateFile)
	w := doMcapReq(t, r, http.MethodPost, "/mcap-files", map[string]any{
		"mcap_file_id": "abcd1234",
		"gcs_path":     "gs://bucket/a.mcap",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create expected 201, got %d body=%s", w.Code, w.Body.String())
	}

	var assetEvt *repository.AssetEventAppendInput
	for i := range appended {
		if appended[i].EventType == "asset_created" {
			assetEvt = &appended[i]
			break
		}
	}
	if assetEvt == nil {
		t.Fatalf("expected an asset_created event; got %d events: %+v", len(appended), appended)
	}
	if assetEvt.AssetID != "abcd1234" {
		t.Errorf("asset_created AssetID = %q, want abcd1234 (empty AssetID is dropped by ES subscriber)", assetEvt.AssetID)
	}
	if assetEvt.AggregateType != "asset" {
		t.Errorf("asset_created AggregateType = %q, want asset", assetEvt.AggregateType)
	}
}

func (rawMcapAssetRepoStub) LookupCosts(context.Context, []string, time.Time, time.Time, bool) ([]repository.AssetCostRow, error) {
	return nil, nil
}

func (rawMcapAssetRepoStub) LookupDurations(context.Context, []string, int64, int64) ([]repository.DurationRow, error) {
	return nil, nil
}
