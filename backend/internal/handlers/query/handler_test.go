package query

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	assetUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/asset"
)

type stubAssetRepo struct {
	lastWhereSQL string
	lastArgs     []interface{}
	lastPage     int
	lastPageSize int
	lastOrderBy  string
}

func (r *stubAssetRepo) Get(ctx context.Context, id string) (*models.Asset, error) {
	return r.GetAll(ctx, id)
}
func (r *stubAssetRepo) GetAll(context.Context, string) (*models.Asset, error) { return nil, nil }
func (r *stubAssetRepo) InsertNew(context.Context, *models.Asset) error        { return nil }
func (r *stubAssetRepo) Set(context.Context, *models.Asset) error              { return nil }
func (r *stubAssetRepo) SoftDelete(context.Context, string) error              { return nil }
func (r *stubAssetRepo) ListByMcapFile(context.Context, string) ([]*models.Asset, error) {
	return nil, nil
}
func (r *stubAssetRepo) ListByLogicalAssetID(context.Context, string) ([]*models.Asset, error) {
	return nil, nil
}
func (r *stubAssetRepo) WriteSegmentIndex(context.Context, *models.Asset) error { return nil }

func (r *stubAssetRepo) ListWithFilters(_ context.Context, whereSQL string, args []interface{}, page, pageSize int, orderBy filter.OrderByClause) ([]*models.Asset, int64, error) {
	r.lastWhereSQL = whereSQL
	r.lastArgs = append([]interface{}{}, args...)
	r.lastPage = page
	r.lastPageSize = pageSize
	r.lastOrderBy = orderBy.SQL
	return []*models.Asset{{
		AssetID:        "aset0001",
		McapFileID:     "mcap0001",
		AssetType:      "segment",
		LifecycleState: "ready",
		DurationMs:     60000,
		Owner:          "alice",
		Reviewer:       "bob",
		CreatedAt:      time.Unix(0, 0).UTC(),
		UpdatedAt:      time.Unix(0, 0).UTC(),
		Tags:           map[string]string{},
		AlgoResults:    map[string]string{},
		Files:          map[string]string{},
		LifecycleMeta:  map[string]interface{}{},
	}}, 1, nil
}

func TestValidate_ReturnsNormalizedQueryAndDebugPlan(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubAssetRepo{}
	h := New(assetUC.New(repo), nil, nil, nil)
	r := gin.New()
	r.POST("/queries/validate", h.Validate)

	body := map[string]any{
		"schema_version": "v1",
		"scope":          map[string]any{"resource": "assets"},
		"where": map[string]any{
			"and": []map[string]any{
				{"pred": map[string]any{"field": "owner", "op": "eq", "value": "alice"}},
				{"pred": map[string]any{"field": "delivery_count", "op": "gte", "value": 1}},
			},
		},
		"sort": []map[string]any{{"field": "created_at", "direction": "desc"}},
		"page": map[string]any{"page": 2, "page_size": 30},
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/queries/validate", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Valid             bool     `json:"valid"`
		Warnings          []string `json:"warnings"`
		FieldCapabilities []struct {
			Field   string   `json:"field"`
			Engines []string `json:"engines"`
		} `json:"field_capabilities"`
		DebugPlan struct {
			Steps []struct {
				Engine string `json:"engine"`
				Mode   string `json:"mode"`
			} `json:"steps"`
		} `json:"debug_plan"`
		NormalizedQuery struct {
			SchemaVersion string `json:"schema_version"`
			Page          struct {
				Page     int `json:"page"`
				PageSize int `json:"page_size"`
			} `json:"page"`
		} `json:"normalized_query"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Valid {
		t.Fatalf("expected valid=true")
	}
	if resp.NormalizedQuery.SchemaVersion != "v1" {
		t.Fatalf("unexpected schema version: %q", resp.NormalizedQuery.SchemaVersion)
	}
	if resp.NormalizedQuery.Page.Page != 2 || resp.NormalizedQuery.Page.PageSize != 30 {
		t.Fatalf("unexpected normalized page: %+v", resp.NormalizedQuery.Page)
	}
	if len(resp.Warnings) != 0 {
		t.Fatalf("expected empty warnings, got %#v", resp.Warnings)
	}
	if len(resp.FieldCapabilities) != 3 {
		t.Fatalf("expected 3 field capabilities, got %d", len(resp.FieldCapabilities))
	}
	if len(resp.DebugPlan.Steps) != 1 || resp.DebugPlan.Steps[0].Engine != "postgres" || resp.DebugPlan.Steps[0].Mode != "filter" {
		t.Fatalf("unexpected debug plan: %+v", resp.DebugPlan)
	}
}

func TestRun_UsesTreeWhereAndReturnsDebugPlan(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubAssetRepo{}
	h := New(assetUC.New(repo), nil, nil, nil)
	r := gin.New()
	r.POST("/queries/run", h.Run)

	body := map[string]any{
		"schema_version": "v1",
		"scope":          map[string]any{"resource": "assets"},
		"select":         map[string]any{"fields": []string{"asset_id", "owner"}},
		"where": map[string]any{
			"pred": map[string]any{"field": "owner", "op": "eq", "value": "alice"},
		},
		"sort": []map[string]any{{"field": "created_at", "direction": "desc"}},
		"page": map[string]any{"page": 1, "page_size": 20},
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/queries/run", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	if repo.lastPage != 1 || repo.lastPageSize != 20 || repo.lastOrderBy != "created_at DESC" {
		t.Fatalf("unexpected repo call: page=%d pageSize=%d orderBy=%q", repo.lastPage, repo.lastPageSize, repo.lastOrderBy)
	}
	if repo.lastWhereSQL == "" {
		t.Fatalf("expected whereSQL to be populated")
	}
	if !strings.Contains(repo.lastWhereSQL, "is_current") {
		t.Fatalf("expected default current-only SQL, got %q", repo.lastWhereSQL)
	}
	if len(repo.lastArgs) != 1 || repo.lastArgs[0] != "alice" {
		t.Fatalf("unexpected repo args: %#v", repo.lastArgs)
	}

	var resp struct {
		Items     []map[string]any `json:"items"`
		Total     int              `json:"total"`
		DebugPlan struct {
			Steps []struct {
				Engine string `json:"engine"`
				Mode   string `json:"mode"`
			} `json:"steps"`
		} `json:"debug_plan"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Total != 1 || len(resp.Items) != 1 {
		t.Fatalf("unexpected run response: %+v", resp)
	}
	if len(resp.DebugPlan.Steps) != 1 || resp.DebugPlan.Steps[0].Engine != "postgres" {
		t.Fatalf("unexpected debug plan: %+v", resp.DebugPlan)
	}
}

func TestRun_IncludeHistorySkipsCurrentOnlySQL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubAssetRepo{}
	h := New(assetUC.New(repo), nil, nil, nil)
	r := gin.New()
	r.POST("/queries/run", h.Run)

	body := map[string]any{
		"schema_version": "v1",
		"scope":          map[string]any{"resource": "assets"},
		"where": map[string]any{
			"pred": map[string]any{"field": "owner", "op": "eq", "value": "alice"},
		},
		"page": map[string]any{"page": 1, "page_size": 10},
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/queries/run?include_history=true", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	if strings.Contains(repo.lastWhereSQL, "is_current") {
		t.Fatalf("include_history should not add current-only filter, got %q", repo.lastWhereSQL)
	}
}

func TestValidate_UsesConfiguredFieldCapabilities(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubAssetRepo{}
	dir := t.TempDir()
	path := filepath.Join(dir, "query_field_registry.yaml")
	content := `
schema_version: 1
resources:
  assets:
    fields:
      - field: owner
        filter_engines: [postgres, elasticsearch]
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	reg, err := config.LoadQueryFieldRegistry(path)
	if err != nil {
		t.Fatalf("LoadQueryFieldRegistry: %v", err)
	}

	h := New(assetUC.New(repo), reg, nil, nil)
	r := gin.New()
	r.POST("/queries/validate", h.Validate)

	body := map[string]any{
		"schema_version": "v1",
		"scope":          map[string]any{"resource": "assets"},
		"where": map[string]any{
			"pred": map[string]any{"field": "owner", "op": "eq", "value": "alice"},
		},
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/queries/validate", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		FieldCapabilities []struct {
			Field   string   `json:"field"`
			Engines []string `json:"engines"`
		} `json:"field_capabilities"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.FieldCapabilities) != 1 {
		t.Fatalf("unexpected field capabilities: %+v", resp.FieldCapabilities)
	}
	if got, want := resp.FieldCapabilities[0].Engines, []string{"elasticsearch", "postgres"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("unexpected engines: got=%v want=%v", got, want)
	}
}

var _ repository.AssetRepository = (*stubAssetRepo)(nil)
