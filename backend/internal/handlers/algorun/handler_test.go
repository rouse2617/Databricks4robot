package algorun

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	algorunUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/algorun"
)

type mockAlgoRunRepo struct {
	listFn func(ctx context.Context, filter repository.AlgoRunListFilter) ([]*models.AlgoRun, int64, error)
}

func (m *mockAlgoRunRepo) Insert(context.Context, *models.AlgoRun) error { return nil }
func (m *mockAlgoRunRepo) Get(context.Context, string) (*models.AlgoRun, error) {
	return nil, nil
}
func (m *mockAlgoRunRepo) Exists(context.Context, string) (bool, error)   { return false, nil }
func (m *mockAlgoRunRepo) Start(context.Context, string, time.Time) error { return nil }
func (m *mockAlgoRunRepo) Finish(context.Context, string, repository.AlgoRunFinishPatch) error {
	return nil
}
func (m *mockAlgoRunRepo) List(ctx context.Context, filter repository.AlgoRunListFilter) ([]*models.AlgoRun, int64, error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return []*models.AlgoRun{}, 0, nil
}
func (m *mockAlgoRunRepo) Cancel(context.Context, string, string, time.Time) error {
	return nil
}
func (m *mockAlgoRunRepo) GetAffectedAssets(context.Context, string) ([]*repository.AffectedAsset, error) {
	return nil, nil
}

type mockAssetRepo struct {
	existing map[string]struct{}
}

func (m *mockAssetRepo) FindExistingIDs(_ context.Context, assetIDs []string) (map[string]struct{}, error) {
	out := make(map[string]struct{})
	for _, assetID := range assetIDs {
		if _, ok := m.existing[assetID]; ok {
			out[assetID] = struct{}{}
		}
	}
	return out, nil
}
func (m *mockAssetRepo) Get(context.Context, string) (*models.Asset, error)    { return nil, nil }
func (m *mockAssetRepo) GetAll(context.Context, string) (*models.Asset, error) { return nil, nil }
func (m *mockAssetRepo) InsertNew(context.Context, *models.Asset) error        { return nil }
func (m *mockAssetRepo) Set(context.Context, *models.Asset) error              { return nil }
func (m *mockAssetRepo) SoftDelete(context.Context, string) error              { return nil }
func (m *mockAssetRepo) ListByMcapFile(context.Context, string) ([]*models.Asset, error) {
	return nil, nil
}
func (m *mockAssetRepo) ListByLogicalAssetID(context.Context, string) ([]*models.Asset, error) {
	return nil, nil
}
func (m *mockAssetRepo) WriteSegmentIndex(context.Context, *models.Asset) error { return nil }
func (m *mockAssetRepo) ListWithFilters(context.Context, string, []interface{}, int, int, filter.OrderByClause) ([]*models.Asset, int64, error) {
	return nil, 0, nil
}
func (m *mockAssetRepo) ListDescendants(context.Context, string) ([]*models.Asset, error) {
	return nil, nil
}

func TestList_ReturnsNormalizedPaginationMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &mockAlgoRunRepo{}
	h := New(algorunUC.New(repo))
	r := gin.New()
	r.GET("/algo-runs", h.List)

	req := httptest.NewRequest(http.MethodGet, "/algo-runs?page=-1&page_size=999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Page     int `json:"page"`
		PageSize int `json:"page_size"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Page != 1 || body.PageSize != 50 {
		t.Fatalf("expected normalized page/page_size 1/50, got %d/%d", body.Page, body.PageSize)
	}
}

func TestCreate_ReturnsAssetValidationDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uc := algorunUC.New(&mockAlgoRunRepo{})
	uc.SetAssetRepo(&mockAssetRepo{existing: map[string]struct{}{"asset-1": {}}})
	h := New(uc)
	r := gin.New()
	r.POST("/algo-runs", h.Create)

	body := `{"run_id":"R001abc123def456","algo_name":"hand_track","algo_version":"2.0","triggered_by":"manual:test","input_asset_ids":["asset-1","missing","missing"]}`
	req := httptest.NewRequest(http.MethodPost, "/algo-runs", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Code    string         `json:"code"`
		Details map[string]any `json:"details"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != "INVALID_ARGUMENT" {
		t.Fatalf("code=%q", resp.Code)
	}
	if resp.Details["field"] != "input_asset_ids" {
		t.Fatalf("field detail=%v", resp.Details["field"])
	}
	missing, ok := resp.Details["missing_asset_ids"].([]any)
	if !ok || len(missing) != 1 || missing[0] != "missing" {
		t.Fatalf("missing detail=%#v", resp.Details["missing_asset_ids"])
	}
	duplicates, ok := resp.Details["duplicate_asset_ids"].([]any)
	if !ok || len(duplicates) != 1 || duplicates[0] != "missing" {
		t.Fatalf("duplicate detail=%#v", resp.Details["duplicate_asset_ids"])
	}
}
