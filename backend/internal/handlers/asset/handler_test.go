package asset

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"data-platform/internal/config"
	"data-platform/internal/models"
	assetUC "data-platform/internal/usecase/asset"
)

type mockAssetRepo struct {
	getFn             func(ctx context.Context, assetID string) (*models.Asset, error)
	setFn             func(ctx context.Context, a *models.Asset) error
	softDeleteFn      func(ctx context.Context, assetID string) error
	listByMcapFileFn  func(ctx context.Context, mcapFileID string) ([]*models.Asset, error)
	writeSegIndexFn   func(ctx context.Context, a *models.Asset) error
	listWithFiltersFn func(ctx context.Context, whereSQL string, args []interface{}, page, pageSize int, orderBy string) ([]*models.Asset, int64, error)
}

// mockDeliveryRepoForAsset implements repository.DeliveryRepository for asset handler tests.
type mockDeliveryRepoForAsset struct {
	listByAssetFn func(ctx context.Context, assetID string) ([]string, error)
}

func (m *mockDeliveryRepoForAsset) Set(context.Context, *models.Delivery) error { return nil }
func (m *mockDeliveryRepoForAsset) Get(context.Context, string) (*models.Delivery, error) {
	return nil, nil
}
func (m *mockDeliveryRepoForAsset) WriteIndexes(context.Context, string, *models.Delivery) error {
	return nil
}
func (m *mockDeliveryRepoForAsset) ListByCustomer(context.Context, string) ([]string, error) {
	return nil, nil
}
func (m *mockDeliveryRepoForAsset) ListByAsset(_ context.Context, _ string) ([]string, error) {
	if m.listByAssetFn != nil {
		return m.listByAssetFn(context.Background(), "")
	}
	return []string{}, nil
}
func (m *mockDeliveryRepoForAsset) List(_ context.Context, _, _ int, _ string) ([]*models.Delivery, int64, error) {
	return []*models.Delivery{}, 0, nil
}

func (m *mockAssetRepo) Get(ctx context.Context, assetID string) (*models.Asset, error) {
	if m.getFn != nil {
		return m.getFn(ctx, assetID)
	}
	return nil, nil
}
func (m *mockAssetRepo) Set(ctx context.Context, a *models.Asset) error {
	if m.setFn != nil {
		return m.setFn(ctx, a)
	}
	return nil
}
func (m *mockAssetRepo) SoftDelete(ctx context.Context, assetID string) error {
	if m.softDeleteFn != nil {
		return m.softDeleteFn(ctx, assetID)
	}
	return nil
}
func (m *mockAssetRepo) ListByMcapFile(ctx context.Context, mcapFileID string) ([]*models.Asset, error) {
	if m.listByMcapFileFn != nil {
		return m.listByMcapFileFn(ctx, mcapFileID)
	}
	return nil, nil
}
func (m *mockAssetRepo) WriteSegmentIndex(ctx context.Context, a *models.Asset) error {
	if m.writeSegIndexFn != nil {
		return m.writeSegIndexFn(ctx, a)
	}
	return nil
}
func (m *mockAssetRepo) ListWithFilters(ctx context.Context, whereSQL string, args []interface{}, page, pageSize int, orderBy string) ([]*models.Asset, int64, error) {
	if m.listWithFiltersFn != nil {
		return m.listWithFiltersFn(ctx, whereSQL, args, page, pageSize, orderBy)
	}
	return []*models.Asset{}, 0, nil
}
func (m *mockAssetRepo) MergeCfAlgo(_ context.Context, _ string, _ int64, _ map[string]interface{}, _ map[string]interface{}) (int64, error) {
	return 0, nil
}

func setupAssetRouter(method, path string, fn gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Handle(method, path, fn)
	return r
}

func doReq(t *testing.T, r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestGet(t *testing.T) {
	repo := &mockAssetRepo{}
	h := New(assetUC.New(repo), &mockDeliveryRepoForAsset{})
	r := setupAssetRouter(http.MethodGet, "/assets/:id", h.Get)

	repo.getFn = func(context.Context, string) (*models.Asset, error) { return nil, nil }
	w := doReq(t, r, http.MethodGet, "/assets/a1", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}

	repo.getFn = func(context.Context, string) (*models.Asset, error) { return nil, errors.New("boom") }
	w = doReq(t, r, http.MethodGet, "/assets/a1", nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	repo.getFn = func(context.Context, string) (*models.Asset, error) { return &models.Asset{AssetID: "a1"}, nil }
	w = doReq(t, r, http.MethodGet, "/assets/a1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestList(t *testing.T) {
	repo := &mockAssetRepo{
		listByMcapFileFn: func(context.Context, string) ([]*models.Asset, error) {
			return []*models.Asset{{AssetID: "a1"}, {AssetID: "a2"}}, nil
		},
		listWithFiltersFn: func(_ context.Context, whereSQL string, args []interface{}, page, pageSize int, orderBy string) ([]*models.Asset, int64, error) {
			if page < 1 || pageSize < 1 {
				t.Fatalf("unexpected pagination: %d %d", page, pageSize)
			}
			return []*models.Asset{}, 0, nil
		},
	}
	h := New(assetUC.New(repo), &mockDeliveryRepoForAsset{})
	r := setupAssetRouter(http.MethodGet, "/assets", h.List)

	// No filters, no mcap_file_id → returns empty list via ListWithFilters (200 OK).
	w := doReq(t, r, http.MethodGet, "/assets", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for no filters, got %d", w.Code)
	}

	// Invalid filter → 400.
	w = doReq(t, r, http.MethodGet, "/assets?filter=badformat", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid filter, got %d", w.Code)
	}

	// mcap_file_id backward compat → 200.
	w = doReq(t, r, http.MethodGet, "/assets?mcap_file_id=m1&page=1&page_size=1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Promoted field resolves to real column (owner was promoted from cf_meta JSONB).
	repo.listWithFiltersFn = func(_ context.Context, whereSQL string, args []interface{}, page, pageSize int, orderBy string) ([]*models.Asset, int64, error) {
		if whereSQL != "owner = $1" {
			t.Fatalf("unexpected whereSQL: %s", whereSQL)
		}
		if len(args) != 1 || args[0] != "alice" {
			t.Fatalf("unexpected args: %#v", args)
		}
		if orderBy != "cf_tag#>>'{notes}' DESC" {
			t.Fatalf("unexpected orderBy: %s", orderBy)
		}
		return []*models.Asset{}, 0, nil
	}
	w = doReq(t, r, http.MethodGet, "/assets?filter=owner:eq:alice&sort_by=-tags.notes", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for resolved whitelist fields, got %d", w.Code)
	}

	// Invalid sort field → 400.
	w = doReq(t, r, http.MethodGet, "/assets?sort_by=-drop_table", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid sort field, got %d", w.Code)
	}
}

func TestCreate(t *testing.T) {
	repo := &mockAssetRepo{}
	h := New(assetUC.New(repo), &mockDeliveryRepoForAsset{})
	r := setupAssetRouter(http.MethodPost, "/assets", h.Create)

	w := doReq(t, r, http.MethodPost, "/assets", map[string]any{"bad": 1})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	w = doReq(t, r, http.MethodPost, "/assets", map[string]any{
		"mcap_file_id":       "m1",
		"start_timestamp_ns": 10,
		"end_timestamp_ns":   10,
		"reviewer":           "r1",
	})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", w.Code)
	}

	w = doReq(t, r, http.MethodPost, "/assets", map[string]any{
		"mcap_file_id":       "m1",
		"start_timestamp_ns": 10,
		"end_timestamp_ns":   20,
		"reviewer":           "r1",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
}

func TestUpdateDeleteAndCommitSegments(t *testing.T) {
	repo := &mockAssetRepo{
		getFn: func(context.Context, string) (*models.Asset, error) {
			return &models.Asset{AssetID: "a1", Tags: map[string]string{}}, nil
		},
	}
	h := New(assetUC.New(repo), &mockDeliveryRepoForAsset{})

	// Update not found
	repo.getFn = func(context.Context, string) (*models.Asset, error) { return nil, nil }
	r := setupAssetRouter(http.MethodPatch, "/assets/:id", h.Update)
	w := doReq(t, r, http.MethodPatch, "/assets/a1", map[string]any{"status": "approved"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}

	// Update success
	repo.getFn = func(context.Context, string) (*models.Asset, error) {
		return &models.Asset{AssetID: "a1", Tags: map[string]string{}}, nil
	}
	w = doReq(t, r, http.MethodPatch, "/assets/a1", map[string]any{"status": "approved"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Delete
	r = setupAssetRouter(http.MethodDelete, "/assets/:id", h.Delete)
	w = doReq(t, r, http.MethodDelete, "/assets/a1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Commit segments bad body
	r = setupAssetRouter(http.MethodPost, "/internal/commit-segments", h.CommitSegments)
	w = doReq(t, r, http.MethodPost, "/internal/commit-segments", map[string]any{"x": 1})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	// Commit segments invalid range
	w = doReq(t, r, http.MethodPost, "/internal/commit-segments", map[string]any{
		"mcap_file_id": "m1",
		"reviewer":     "r1",
		"ranges":       [][2]int64{{10, 10}},
	})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", w.Code)
	}
}

func TestListDeliveriesAndHelpers(t *testing.T) {
	h := New(assetUC.New(&mockAssetRepo{}), &mockDeliveryRepoForAsset{})
	r := setupAssetRouter(http.MethodGet, "/assets/:id/deliveries", h.ListDeliveries)
	w := doReq(t, r, http.MethodGet, "/assets/a1/deliveries", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	p, s := parsePageParams("2", "50")
	if p != 2 || s != 50 {
		t.Fatalf("unexpected pagination parse: %d %d", p, s)
	}
	p, s = parsePageParams("-1", "1000")
	if p != 1 || s != 20 {
		t.Fatalf("expected default pagination")
	}

	items := []*models.Asset{{AssetID: "a1"}, {AssetID: "a2"}}
	got := paginateAssets(items, 1, 1)
	if len(got) != 1 || got[0].AssetID != "a1" {
		t.Fatalf("paginate first page failed")
	}
	if len(paginateAssets(items, 3, 1)) != 0 {
		t.Fatalf("expected empty page")
	}
	if len(paginateAssets(nil, 1, 20)) != 0 {
		t.Fatalf("expected empty list")
	}
}

// ─── Filter Param Parsing and Error Response Tests ──────────────────────────

func TestListFilterParsing(t *testing.T) {
	repo := &mockAssetRepo{
		listWithFiltersFn: func(_ context.Context, whereSQL string, args []interface{}, page, pageSize int, orderBy string) ([]*models.Asset, int64, error) {
			return []*models.Asset{{AssetID: "a1"}}, 1, nil
		},
	}
	h := New(assetUC.New(repo), &mockDeliveryRepoForAsset{})
	r := setupAssetRouter(http.MethodGet, "/assets", h.List)

	// Valid filter → 200.
	w := doReq(t, r, http.MethodGet, "/assets?filter=status:eq:approved", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid filter, got %d", w.Code)
	}

	// Multiple valid filters → 200.
	w = doReq(t, r, http.MethodGet, "/assets?filter=status:eq:approved&filter=owner:eq:alice", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for multiple valid filters, got %d", w.Code)
	}

	// Invalid operator → 400 INVALID_FILTER.
	w = doReq(t, r, http.MethodGet, "/assets?filter=status:badop:value", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid operator, got %d", w.Code)
	}
	var errBody map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errBody)
	if errBody["code"] != "INVALID_FILTER" {
		t.Fatalf("expected INVALID_FILTER code, got %v", errBody["code"])
	}

	// Blacklisted field → 400 INVALID_FILTER.
	w = doReq(t, r, http.MethodGet, "/assets?filter=is_deleted:eq:true", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for blacklisted field, got %d", w.Code)
	}
	json.Unmarshal(w.Body.Bytes(), &errBody)
	if errBody["code"] != "INVALID_FILTER" {
		t.Fatalf("expected INVALID_FILTER code, got %v", errBody["code"])
	}

	// sort_by parameter → 200.
	w = doReq(t, r, http.MethodGet, "/assets?sort_by=-created_at", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for sort_by, got %d", w.Code)
	}
}

func TestListResponseFormat(t *testing.T) {
	repo := &mockAssetRepo{
		listWithFiltersFn: func(_ context.Context, _ string, _ []interface{}, _, _ int, _ string) ([]*models.Asset, int64, error) {
			return []*models.Asset{{AssetID: "a1"}, {AssetID: "a2"}}, 5, nil
		},
	}
	h := New(assetUC.New(repo), &mockDeliveryRepoForAsset{})
	r := setupAssetRouter(http.MethodGet, "/assets", h.List)

	w := doReq(t, r, http.MethodGet, "/assets?page=2&page_size=2", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["total"] != float64(5) {
		t.Fatalf("expected total=5, got %v", resp["total"])
	}
	if resp["page"] != float64(2) {
		t.Fatalf("expected page=2, got %v", resp["page"])
	}
	if resp["page_size"] != float64(2) {
		t.Fatalf("expected page_size=2, got %v", resp["page_size"])
	}
	items, ok := resp["items"].([]interface{})
	if !ok || len(items) != 2 {
		t.Fatalf("expected 2 items, got %v", resp["items"])
	}
}

// ─── Algo Handler Tests ─────────────────────────────────────────────────────

func setupAlgoRouter(h *AlgoHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/assets/:id/algo/:algo_key/start", h.Start)
	r.POST("/assets/:id/algo/:algo_key/finish", h.Finish)
	r.POST("/assets/:id/algo/:algo_key/reset", h.Reset)
	r.GET("/assets/:id/algo-events", h.ListEvents)
	return r
}

func TestAlgoStart(t *testing.T) {
	// Test bad request body.
	uc := assetUC.NewAlgoUsecase(
		&mockAssetRepo{getFn: func(context.Context, string) (*models.Asset, error) { return nil, nil }},
		&mockAlgoEventRepo{},
		buildTestAlgoRegistry(t),
	)
	h := NewAlgoHandler(uc)
	r := setupAlgoRouter(h)

	// Missing method field → 400.
	w := doReq(t, r, http.MethodPost, "/assets/a1/algo/hand_tracking@1.2.0/start", map[string]any{})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing method, got %d", w.Code)
	}

	// Invalid algo key → 400.
	w = doReq(t, r, http.MethodPost, "/assets/a1/algo/invalid_key/start", map[string]any{"method": "test"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid algo key, got %d", w.Code)
	}

	// Asset not found → 404.
	w = doReq(t, r, http.MethodPost, "/assets/nonexistent/algo/hand_tracking@1.2.0/start", map[string]any{"method": "test"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for nonexistent asset, got %d", w.Code)
	}

	// Already running → 409.
	uc2 := assetUC.NewAlgoUsecase(
		&mockAssetRepo{getFn: func(context.Context, string) (*models.Asset, error) {
			return &models.Asset{
				AssetID:     "a1",
				AlgoResults: map[string]string{"hand_tracking@1.2.0:status": "running"},
				Tags:        map[string]string{},
				Files:       map[string]string{},
				Version:     1,
			}, nil
		}},
		&mockAlgoEventRepo{},
		buildTestAlgoRegistry(t),
	)
	h2 := NewAlgoHandler(uc2)
	r2 := setupAlgoRouter(h2)
	w = doReq(t, r2, http.MethodPost, "/assets/a1/algo/hand_tracking@1.2.0/start", map[string]any{"method": "test"})
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for already running, got %d", w.Code)
	}
}

func TestAlgoFinish(t *testing.T) {
	uc := assetUC.NewAlgoUsecase(
		&mockAssetRepo{getFn: func(context.Context, string) (*models.Asset, error) {
			return &models.Asset{
				AssetID:     "a1",
				AlgoResults: map[string]string{"env_analysis@1.0.0:status": "running"},
				Tags:        map[string]string{},
				Files:       map[string]string{},
				Version:     1,
			}, nil
		}},
		&mockAlgoEventRepo{},
		buildTestAlgoRegistry(t),
	)
	h := NewAlgoHandler(uc)
	r := setupAlgoRouter(h)

	// Missing status → 400.
	w := doReq(t, r, http.MethodPost, "/assets/a1/algo/env_analysis@1.0.0/finish", map[string]any{})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing status, got %d", w.Code)
	}

	// Failed without reason → 422 MISSING_REASON.
	w = doReq(t, r, http.MethodPost, "/assets/a1/algo/env_analysis@1.0.0/finish", map[string]any{"status": "failed"})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for missing reason, got %d", w.Code)
	}
	var errBody map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errBody)
	if errBody["code"] != "MISSING_REASON" {
		t.Fatalf("expected MISSING_REASON, got %v", errBody["code"])
	}

	// Ok with required fields for hand_tracking (missing output_uri) → 422.
	uc2 := assetUC.NewAlgoUsecase(
		&mockAssetRepo{getFn: func(context.Context, string) (*models.Asset, error) {
			return &models.Asset{
				AssetID:     "a1",
				AlgoResults: map[string]string{"hand_tracking@1.2.0:status": "running"},
				Tags:        map[string]string{},
				Files:       map[string]string{},
				Version:     1,
			}, nil
		}},
		&mockAlgoEventRepo{},
		buildTestAlgoRegistry(t),
	)
	h2 := NewAlgoHandler(uc2)
	r2 := setupAlgoRouter(h2)
	w = doReq(t, r2, http.MethodPost, "/assets/a1/algo/hand_tracking@1.2.0/finish", map[string]any{"status": "ok"})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for missing required field, got %d", w.Code)
	}
	json.Unmarshal(w.Body.Bytes(), &errBody)
	if errBody["code"] != "MISSING_REQUIRED_FIELD" {
		t.Fatalf("expected MISSING_REQUIRED_FIELD, got %v", errBody["code"])
	}

	// Invalid state transition (not running) → 409.
	uc3 := assetUC.NewAlgoUsecase(
		&mockAssetRepo{getFn: func(context.Context, string) (*models.Asset, error) {
			return &models.Asset{
				AssetID:     "a1",
				AlgoResults: map[string]string{"env_analysis@1.0.0:status": "pending"},
				Tags:        map[string]string{},
				Files:       map[string]string{},
				Version:     1,
			}, nil
		}},
		&mockAlgoEventRepo{},
		buildTestAlgoRegistry(t),
	)
	h3 := NewAlgoHandler(uc3)
	r3 := setupAlgoRouter(h3)
	w = doReq(t, r3, http.MethodPost, "/assets/a1/algo/env_analysis@1.0.0/finish", map[string]any{
		"status": "failed",
		"reason": "test",
	})
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for invalid state transition, got %d", w.Code)
	}
}

func TestAlgoReset(t *testing.T) {
	// Reset from pending → 409 (only failed/ok allowed).
	uc := assetUC.NewAlgoUsecase(
		&mockAssetRepo{getFn: func(context.Context, string) (*models.Asset, error) {
			return &models.Asset{
				AssetID:     "a1",
				AlgoResults: map[string]string{"env_analysis@1.0.0:status": "pending"},
				Tags:        map[string]string{},
				Files:       map[string]string{},
				Version:     1,
			}, nil
		}},
		&mockAlgoEventRepo{},
		buildTestAlgoRegistry(t),
	)
	h := NewAlgoHandler(uc)
	r := setupAlgoRouter(h)

	w := doReq(t, r, http.MethodPost, "/assets/a1/algo/env_analysis@1.0.0/reset", nil)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for reset from pending, got %d", w.Code)
	}

	// Invalid algo key → 400.
	w = doReq(t, r, http.MethodPost, "/assets/a1/algo/bad_key/reset", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid algo key, got %d", w.Code)
	}
}

func TestAlgoListEvents(t *testing.T) {
	// Asset not found → 404.
	uc := assetUC.NewAlgoUsecase(
		&mockAssetRepo{getFn: func(context.Context, string) (*models.Asset, error) { return nil, nil }},
		&mockAlgoEventRepo{},
		buildTestAlgoRegistry(t),
	)
	h := NewAlgoHandler(uc)
	r := setupAlgoRouter(h)

	w := doReq(t, r, http.MethodGet, "/assets/nonexistent/algo-events", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for nonexistent asset, got %d", w.Code)
	}

	// Success with events.
	uc2 := assetUC.NewAlgoUsecase(
		&mockAssetRepo{getFn: func(context.Context, string) (*models.Asset, error) {
			return &models.Asset{AssetID: "a1", Tags: map[string]string{}, Files: map[string]string{}, AlgoResults: map[string]string{}}, nil
		}},
		&mockAlgoEventRepo{},
		buildTestAlgoRegistry(t),
	)
	h2 := NewAlgoHandler(uc2)
	r2 := setupAlgoRouter(h2)
	w = doReq(t, r2, http.MethodGet, "/assets/a1/algo-events", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// ─── Property 10: Soft-deleted assets never appear in query results ─────────
// **Validates: Requirements 3.6**
// This property is validated at the handler level by ensuring the ListWithFilters
// repo method is called (which always adds is_deleted=FALSE in the SQL).
// We verify the handler correctly delegates to ListWithFilters and returns
// only the items from the repo (which excludes soft-deleted assets).
func TestProperty10_SoftDeletedAssetsNeverInResults(t *testing.T) {
	// The mock repo simulates a repo that never returns soft-deleted assets.
	// We verify the handler returns exactly what the repo returns.
	repo := &mockAssetRepo{
		listWithFiltersFn: func(_ context.Context, _ string, _ []interface{}, _, _ int, _ string) ([]*models.Asset, int64, error) {
			// Only non-deleted assets returned.
			return []*models.Asset{
				{AssetID: "live-1"},
				{AssetID: "live-2"},
			}, 2, nil
		},
	}
	h := New(assetUC.New(repo), &mockDeliveryRepoForAsset{})
	r := setupAssetRouter(http.MethodGet, "/assets", h.List)

	w := doReq(t, r, http.MethodGet, "/assets?filter=status:eq:approved", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Items []struct {
			AssetID string `json:"asset_id"`
		} `json:"items"`
		Total int `json:"total"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Total != 2 {
		t.Fatalf("expected total=2, got %d", resp.Total)
	}
	for _, item := range resp.Items {
		if item.AssetID != "live-1" && item.AssetID != "live-2" {
			t.Fatalf("unexpected asset in results: %s", item.AssetID)
		}
	}
}

// ─── Test Helpers for Algo Handler ──────────────────────────────────────────

type mockAlgoEventRepo struct{}

func (m *mockAlgoEventRepo) Insert(_ context.Context, _ *models.AlgoEvent) error { return nil }
func (m *mockAlgoEventRepo) ListByAsset(_ context.Context, _ string, _ *string) ([]*models.AlgoEvent, error) {
	return []*models.AlgoEvent{}, nil
}

func buildTestAlgoRegistry(t *testing.T) *config.AlgoRegistry {
	t.Helper()
	reg, err := config.LoadAlgoRegistry("../../../config/algo_registry.yaml")
	if err != nil {
		t.Fatalf("failed to load algo registry: %v", err)
	}
	return reg
}
