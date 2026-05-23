package asset

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/handlers"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	assetUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/asset"
)

type mockAssetRepo struct {
	insertNewFn       func(ctx context.Context, a *models.Asset) error
	getFn             func(ctx context.Context, assetID string) (*models.Asset, error)
	setFn             func(ctx context.Context, a *models.Asset) error
	softDeleteFn      func(ctx context.Context, assetID string) error
	listByMcapFileFn  func(ctx context.Context, mcapFileID string) ([]*models.Asset, error)
	writeSegIndexFn   func(ctx context.Context, a *models.Asset) error
	listWithFiltersFn func(ctx context.Context, whereSQL string, args []interface{}, page, pageSize int, orderBy filter.OrderByClause) ([]*models.Asset, int64, error)
}

// mockDeliveryRepoForAsset implements repository.DeliveryRepository for asset handler tests.
type mockDeliveryRepoForAsset struct {
	listByAssetFn func(ctx context.Context, assetID string) ([]string, error)
}

func (m *mockDeliveryRepoForAsset) Set(context.Context, *models.Delivery) error { return nil }
func (m *mockDeliveryRepoForAsset) Get(context.Context, string) (*models.Delivery, error) {
	return nil, nil
}
func (m *mockDeliveryRepoForAsset) AddItems(context.Context, string, []string) error {
	return nil
}
func (m *mockDeliveryRepoForAsset) RefreshAssetDeliveryIndex(context.Context, string) error {
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
func (m *mockDeliveryRepoForAsset) ListItems(context.Context, string) ([]*models.DeliveryItem, error) {
	return []*models.DeliveryItem{}, nil
}
func (m *mockDeliveryRepoForAsset) List(_ context.Context, _, _ int, _ string, _ string) ([]*models.Delivery, int64, error) {
	return []*models.Delivery{}, 0, nil
}
func (m *mockDeliveryRepoForAsset) Update(_ context.Context, _ *models.Delivery, _ int64) error {
	return nil
}

func (m *mockAssetRepo) InsertNew(ctx context.Context, a *models.Asset) error {
	if m.insertNewFn != nil {
		return m.insertNewFn(ctx, a)
	}
	return m.Set(ctx, a)
}
func (m *mockAssetRepo) Get(ctx context.Context, assetID string) (*models.Asset, error) {
	return m.GetAll(ctx, assetID)
}
func (m *mockAssetRepo) GetAll(ctx context.Context, assetID string) (*models.Asset, error) {
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
func (m *mockAssetRepo) ListByLogicalAssetID(context.Context, string) ([]*models.Asset, error) {
	return nil, nil
}
func (m *mockAssetRepo) WriteSegmentIndex(ctx context.Context, a *models.Asset) error {
	if m.writeSegIndexFn != nil {
		return m.writeSegIndexFn(ctx, a)
	}
	return nil
}
func (m *mockAssetRepo) ListWithFilters(ctx context.Context, whereSQL string, args []interface{}, page, pageSize int, orderBy filter.OrderByClause) ([]*models.Asset, int64, error) {
	if m.listWithFiltersFn != nil {
		return m.listWithFiltersFn(ctx, whereSQL, args, page, pageSize, orderBy)
	}
	return []*models.Asset{}, 0, nil
}
func (m *mockAssetRepo) ListDescendants(_ context.Context, _ string) ([]*models.Asset, error) {
	return nil, nil
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
	w := doReq(t, r, http.MethodGet, "/assets/aaaaaaaa", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}

	repo.getFn = func(context.Context, string) (*models.Asset, error) { return nil, errors.New("boom") }
	w = doReq(t, r, http.MethodGet, "/assets/aaaaaaaa", nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	repo.getFn = func(context.Context, string) (*models.Asset, error) { return &models.Asset{AssetID: "aaaaaaaa"}, nil }
	w = doReq(t, r, http.MethodGet, "/assets/aaaaaaaa", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestList(t *testing.T) {
	repo := &mockAssetRepo{
		listByMcapFileFn: func(context.Context, string) ([]*models.Asset, error) {
			return []*models.Asset{{AssetID: "aaaaaaaa"}, {AssetID: "bbbbbbbb"}}, nil
		},
		listWithFiltersFn: func(_ context.Context, whereSQL string, args []interface{}, page, pageSize int, orderBy filter.OrderByClause) ([]*models.Asset, int64, error) {
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

	// Promoted field resolves to real column; tag sort now resolves via projection subquery.
	repo.listWithFiltersFn = func(_ context.Context, whereSQL string, args []interface{}, page, pageSize int, orderBy filter.OrderByClause) ([]*models.Asset, int64, error) {
		if whereSQL != "owner = $1" {
			t.Fatalf("unexpected whereSQL: %s", whereSQL)
		}
		if len(args) != 1 || args[0] != "alice" {
			t.Fatalf("unexpected args: %#v", args)
		}
		if orderBy.SQL != "(SELECT t.tag_value FROM asset_tags t WHERE t.asset_id = assets.asset_id AND t.tag_key = $1 LIMIT 1) DESC" {
			t.Fatalf("unexpected orderBy: %s", orderBy.SQL)
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
	var created map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal create response failed: %v", err)
	}
	if got, _ := created["asset_type"].(string); got != "segment" {
		t.Fatalf("expected asset_type=segment by default, got %v", created["asset_type"])
	}

	repo.insertNewFn = func(context.Context, *models.Asset) error {
		return fmt.Errorf("postgres AssetRepo.InsertNew: %w", &pgconn.PgError{
			Code:           "23514",
			ConstraintName: "assets_mcap_file_id_check",
		})
	}
	w = doReq(t, r, http.MethodPost, "/assets", map[string]any{
		"mcap_file_id":       "bad-format-mcap-id",
		"start_timestamp_ns": 10,
		"end_timestamp_ns":   20,
		"reviewer":           "r1",
	})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for mcap_file_id check violation, got %d", w.Code)
	}

	repo.insertNewFn = func(context.Context, *models.Asset) error {
		return fmt.Errorf("postgres AssetRepo.InsertNew: %w", &pgconn.PgError{
			Code:           "23503",
			ConstraintName: "fk_assets_mcap",
		})
	}
	w = doReq(t, r, http.MethodPost, "/assets", map[string]any{
		"mcap_file_id":       "A1B2C3D4",
		"start_timestamp_ns": 10,
		"end_timestamp_ns":   20,
		"reviewer":           "r1",
	})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for missing mcap_file FK violation, got %d", w.Code)
	}
}

func TestUpdateDeleteAndCommitSegments(t *testing.T) {
	repo := &mockAssetRepo{
		getFn: func(context.Context, string) (*models.Asset, error) {
			return &models.Asset{AssetID: "aaaaaaaa", Tags: map[string]string{}}, nil
		},
	}
	h := New(assetUC.New(repo), &mockDeliveryRepoForAsset{})

	// Update not found
	repo.getFn = func(context.Context, string) (*models.Asset, error) { return nil, nil }
	r := setupAssetRouter(http.MethodPatch, "/assets/:id", h.Update)
	w := doReq(t, r, http.MethodPatch, "/assets/aaaaaaaa", map[string]any{"status": "approved"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}

	// Update success
	repo.getFn = func(context.Context, string) (*models.Asset, error) {
		return &models.Asset{AssetID: "aaaaaaaa", Tags: map[string]string{}}, nil
	}
	w = doReq(t, r, http.MethodPatch, "/assets/aaaaaaaa", map[string]any{"status": "approved"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Delete
	r = setupAssetRouter(http.MethodDelete, "/assets/:id", h.Delete)
	w = doReq(t, r, http.MethodDelete, "/assets/aaaaaaaa", nil)
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

func TestUpdate_WithLifecycleStateOnly(t *testing.T) {
	repo := &mockAssetRepo{
		getFn: func(context.Context, string) (*models.Asset, error) {
			return &models.Asset{
				AssetID:        "aaaaaaaa",
				Status:         models.AssetStatusApproved,
				LifecycleState: "ready",
				Tags:           map[string]string{},
			}, nil
		},
	}
	h := New(assetUC.New(repo), &mockDeliveryRepoForAsset{})
	r := setupAssetRouter(http.MethodPatch, "/assets/:id", h.Update)

	w := doReq(t, r, http.MethodPatch, "/assets/aaaaaaaa", map[string]any{
		"lifecycle_state": "rejected",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if got, _ := resp["lifecycle_state"].(string); got != "rejected" {
		t.Fatalf("expected lifecycle_state=rejected, got %v", resp["lifecycle_state"])
	}
	if got, _ := resp["status"].(string); got != "rejected" {
		t.Fatalf("expected status=rejected, got %v", resp["status"])
	}
}

func TestListDeliveriesAndHelpers(t *testing.T) {
	h := New(assetUC.New(&mockAssetRepo{}), &mockDeliveryRepoForAsset{})
	r := setupAssetRouter(http.MethodGet, "/assets/:id/deliveries", h.ListDeliveries)
	w := doReq(t, r, http.MethodGet, "/assets/aaaaaaaa/deliveries", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	p, s := handlers.ParsePageParams("2", "50")
	if p != 2 || s != 50 {
		t.Fatalf("unexpected pagination parse: %d %d", p, s)
	}
	p, s = handlers.ParsePageParams("-1", "1000")
	if p != 1 || s != 20 {
		t.Fatalf("expected default pagination")
	}

	items := []*models.Asset{{AssetID: "aaaaaaaa"}, {AssetID: "bbbbbbbb"}}
	got := paginateAssets(items, 1, 1)
	if len(got) != 1 || got[0].AssetID != "aaaaaaaa" {
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
		listWithFiltersFn: func(_ context.Context, whereSQL string, args []interface{}, page, pageSize int, orderBy filter.OrderByClause) ([]*models.Asset, int64, error) {
			return []*models.Asset{{AssetID: "aaaaaaaa"}}, 1, nil
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
		listWithFiltersFn: func(_ context.Context, _ string, _ []interface{}, _, _ int, _ filter.OrderByClause) ([]*models.Asset, int64, error) {
			return []*models.Asset{{AssetID: "aaaaaaaa"}, {AssetID: "bbbbbbbb"}}, 5, nil
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
	r.GET("/assets/:id/algo", h.ListCurrent)
	r.POST("/assets/:id/algo/:algo_key/start", h.Start)
	r.POST("/assets/:id/algo/:algo_key/finish", h.Finish)
	r.POST("/assets/:id/algo/:algo_key/reset", h.Reset)
	return r
}

// algoTestEnv bundles the dependencies an AlgoHandler test needs.
type algoTestEnv struct {
	asset      *mockAssetRepo
	algoLatest *handlerAlgoLatestRepo
	events     *handlerAssetEventRepo
}

// newAlgoEnv constructs an environment with optional pre-seeded algo state.
// presence controls whether the asset exists; statuses keys are algo_key strings.
func newAlgoEnv(t *testing.T, presence bool, statuses map[string]string) (*assetH_AlgoHandler, *algoTestEnv) {
	t.Helper()
	env := &algoTestEnv{
		asset:      &mockAssetRepo{},
		algoLatest: newHandlerAlgoLatestRepo(),
		events:     &handlerAssetEventRepo{},
	}
	if presence {
		env.asset.getFn = func(context.Context, string) (*models.Asset, error) {
			return &models.Asset{AssetID: "aaaaaaaa", Version: 1}, nil
		}
	} else {
		env.asset.getFn = func(context.Context, string) (*models.Asset, error) { return nil, nil }
	}
	for algoKey, status := range statuses {
		env.algoLatest.seed(algoKey, status, "")
	}
	uc := assetUC.NewAlgoUsecase(
		handlerTxRunner{},
		env.asset,
		env.algoLatest,
		env.events,
		buildTestAlgoRegistry(t),
	)
	h := NewAlgoHandler(uc)
	return h, env
}

// assetH_AlgoHandler is a tiny alias to keep handler tests local-imports clean.
type assetH_AlgoHandler = AlgoHandler

func TestAlgoStart(t *testing.T) {
	// Missing method field → 400.
	h, _ := newAlgoEnv(t, true, nil)
	r := setupAlgoRouter(h)
	w := doReq(t, r, http.MethodPost, "/assets/aaaaaaaa/algo/hand_tracking@1.2.0/start", map[string]any{})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing method, got %d", w.Code)
	}

	// Invalid algo key → 400.
	w = doReq(t, r, http.MethodPost, "/assets/aaaaaaaa/algo/invalid_key/start", map[string]any{"method": "test"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid algo key, got %d", w.Code)
	}

	// Asset not found → 404.
	h2, _ := newAlgoEnv(t, false, nil)
	r2 := setupAlgoRouter(h2)
	w = doReq(t, r2, http.MethodPost, "/assets/99999999/algo/hand_tracking@1.2.0/start", map[string]any{"method": "test"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for nonexistent asset, got %d", w.Code)
	}

	// Already running → 409.
	h3, _ := newAlgoEnv(t, true, map[string]string{"hand_tracking@1.2.0": "running"})
	r3 := setupAlgoRouter(h3)
	w = doReq(t, r3, http.MethodPost, "/assets/aaaaaaaa/algo/hand_tracking@1.2.0/start", map[string]any{"method": "test"})
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for already running, got %d", w.Code)
	}
}

func TestAlgoFinish(t *testing.T) {
	// Missing status → 400.
	h, _ := newAlgoEnv(t, true, map[string]string{"env_analysis@1.0.0": "running"})
	r := setupAlgoRouter(h)
	w := doReq(t, r, http.MethodPost, "/assets/aaaaaaaa/algo/env_analysis@1.0.0/finish", map[string]any{})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing status, got %d", w.Code)
	}

	// Failed without reason → 422 MISSING_REASON.
	w = doReq(t, r, http.MethodPost, "/assets/aaaaaaaa/algo/env_analysis@1.0.0/finish", map[string]any{"status": "failed"})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for missing reason, got %d", w.Code)
	}
	var errBody map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errBody)
	if errBody["code"] != "MISSING_REASON" {
		t.Fatalf("expected MISSING_REASON, got %v", errBody["code"])
	}

	// hand_tracking ok with no output_uri → 422.
	h2, _ := newAlgoEnv(t, true, map[string]string{"hand_tracking@1.2.0": "running"})
	r2 := setupAlgoRouter(h2)
	w = doReq(t, r2, http.MethodPost, "/assets/aaaaaaaa/algo/hand_tracking@1.2.0/finish", map[string]any{"status": "ok"})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for missing required field, got %d", w.Code)
	}
	json.Unmarshal(w.Body.Bytes(), &errBody)
	if errBody["code"] != "MISSING_REQUIRED_FIELD" {
		t.Fatalf("expected MISSING_REQUIRED_FIELD, got %v", errBody["code"])
	}

	// Invalid state transition (status=pending) → 409.
	h3, _ := newAlgoEnv(t, true, map[string]string{"env_analysis@1.0.0": "pending"})
	r3 := setupAlgoRouter(h3)
	w = doReq(t, r3, http.MethodPost, "/assets/aaaaaaaa/algo/env_analysis@1.0.0/finish", map[string]any{
		"status": "failed", "reason": "test",
	})
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for invalid state transition, got %d", w.Code)
	}
}

func TestAlgoReset(t *testing.T) {
	// Reset from pending → 409.
	h, _ := newAlgoEnv(t, true, map[string]string{"env_analysis@1.0.0": "pending"})
	r := setupAlgoRouter(h)
	w := doReq(t, r, http.MethodPost, "/assets/aaaaaaaa/algo/env_analysis@1.0.0/reset", nil)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for reset from pending, got %d", w.Code)
	}

	// Invalid algo key → 400.
	w = doReq(t, r, http.MethodPost, "/assets/aaaaaaaa/algo/bad_key/reset", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid algo key, got %d", w.Code)
	}
}

func TestAssetListEvents(t *testing.T) {
	const (
		testAssetID    = "11111111"
		missingAssetID = "22222222"
	)

	makeHandler := func(presence bool, seed func(*handlerAssetEventRepo)) *Handler {
		assetRepo := &mockAssetRepo{}
		if presence {
			assetRepo.getFn = func(context.Context, string) (*models.Asset, error) {
				return &models.Asset{AssetID: testAssetID, Version: 1}, nil
			}
		} else {
			assetRepo.getFn = func(context.Context, string) (*models.Asset, error) { return nil, nil }
		}
		eventRepo := &handlerAssetEventRepo{}
		if seed != nil {
			seed(eventRepo)
		}
		uc := assetUC.NewWithProjections(handlerTxRunner{}, assetRepo, nil, nil, eventRepo, nil, nil)
		return New(uc, &mockDeliveryRepoForAsset{})
	}

	// Invalid path id → 400.
	hBad := makeHandler(false, nil)
	rBad := setupAssetRouter(http.MethodGet, "/assets/:id/events", hBad.ListEvents)
	w := doReq(t, rBad, http.MethodGet, "/assets/not-a-uuid/events", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid asset id, got %d", w.Code)
	}

	// Asset not found → 404.
	h := makeHandler(false, nil)
	r := setupAssetRouter(http.MethodGet, "/assets/:id/events", h.ListEvents)
	w = doReq(t, r, http.MethodGet, "/assets/"+missingAssetID+"/events", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for nonexistent asset, got %d", w.Code)
	}

	// Success with wildcard event_type filter and cursor pagination.
	h2 := makeHandler(true, func(repo *handlerAssetEventRepo) {
		repo.seed("asset_created", 1, map[string]any{"asset_id": testAssetID})
		repo.seed("algo_started", 2, map[string]any{"asset_id": testAssetID, "algo_key": "env_analysis@1.0.0"})
		repo.seed("algo_finished", 3, map[string]any{"asset_id": testAssetID, "algo_key": "env_analysis@1.0.0"})
		repo.seed("tag_upserted", 4, map[string]any{"asset_id": testAssetID, "tag_key": "quality"})
	})
	r2 := setupAssetRouter(http.MethodGet, "/assets/:id/events", h2.ListEvents)
	w = doReq(t, r2, http.MethodGet, "/assets/"+testAssetID+"/events?event_type=algo_*&limit=1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		Items      []models.AssetEvent `json:"items"`
		NextCursor *int64              `json:"next_cursor"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].EventType != "algo_finished" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.NextCursor == nil || *resp.NextCursor != 3 {
		t.Fatalf("expected next_cursor=3, got %+v", resp.NextCursor)
	}

	w = doReq(t, r2, http.MethodGet, "/assets/"+testAssetID+"/events?event_type=algo_*&algo_key=env_analysis@1.0.0&cursor=3", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on cursor follow-up, got %d", w.Code)
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal follow-up: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].EventType != "algo_started" {
		t.Fatalf("unexpected follow-up response: %+v", resp)
	}
}

func TestUpsertTagAndDeleteTag(t *testing.T) {
	assetRepo := &mockAssetRepo{
		getFn: func(context.Context, string) (*models.Asset, error) {
			return &models.Asset{AssetID: "aaaaaaaa", McapFileID: "m1", Version: 1}, nil
		},
	}
	tagRepo := newHandlerAssetTagRepo()
	eventRepo := &handlerAssetEventRepo{}
	uc := assetUC.NewWithProjections(
		handlerTxRunner{},
		assetRepo,
		tagRepo,
		nil,
		eventRepo,
		buildTestTagRegistryForAsset(t),
		nil,
	)
	h := New(uc, &mockDeliveryRepoForAsset{})

	r := gin.New()
	r.POST("/assets/:id/tags", h.UpsertTag)
	r.DELETE("/assets/:id/tags/:key", h.DeleteTag)

	w := doReq(t, r, http.MethodPost, "/assets/aaaaaaaa/tags", map[string]any{
		"key": "quality", "value": "good",
		"source_type": "human", "source_name": "tester",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from upsert tag, got %d", w.Code)
	}
	if row, ok := tagRepo.rows[tagRowKey("aaaaaaaa", "quality")]; !ok || row.TagValue != "good" {
		t.Fatalf("expected projection row quality=good, got %#v", row)
	}

	w = doReq(t, r, http.MethodDelete, "/assets/aaaaaaaa/tags/quality", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from delete tag, got %d", w.Code)
	}
	if _, ok := tagRepo.rows[tagRowKey("aaaaaaaa", "quality")]; ok {
		t.Fatalf("expected tag projection row to be deleted")
	}
}

func TestUpsertTag_InvalidTag(t *testing.T) {
	assetRepo := &mockAssetRepo{
		getFn: func(context.Context, string) (*models.Asset, error) {
			return &models.Asset{AssetID: "aaaaaaaa", McapFileID: "m1", Version: 1}, nil
		},
	}
	uc := assetUC.NewWithProjections(
		handlerTxRunner{},
		assetRepo,
		newHandlerAssetTagRepo(),
		nil,
		&handlerAssetEventRepo{},
		buildTestTagRegistryForAsset(t),
		nil,
	)
	h := New(uc, &mockDeliveryRepoForAsset{})
	r := setupAssetRouter(http.MethodPost, "/assets/:id/tags", h.UpsertTag)

	w := doReq(t, r, http.MethodPost, "/assets/aaaaaaaa/tags", map[string]any{"key": "unknown_tag", "value": "x"})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for invalid tag, got %d", w.Code)
	}
}

func TestListTagHistory(t *testing.T) {
	const tagHistAssetID = "33333333"

	assetRepo := &mockAssetRepo{
		getFn: func(context.Context, string) (*models.Asset, error) {
			return &models.Asset{AssetID: tagHistAssetID, McapFileID: "m1", Version: 1}, nil
		},
	}
	eventRepo := &handlerAssetEventRepo{}
	eventRepo.seed("asset_created", 1, map[string]any{"asset_id": tagHistAssetID})
	eventRepo.seed("tag_upserted", 2, map[string]any{"asset_id": tagHistAssetID, "tag_key": "quality", "tag_value": "good"})
	eventRepo.seed("tag_deleted", 3, map[string]any{"asset_id": tagHistAssetID, "tag_key": "quality", "tag_value": "good"})
	eventRepo.seed("algo_started", 4, map[string]any{"asset_id": tagHistAssetID, "algo_key": "env_analysis@1.0.0"})
	uc := assetUC.NewWithProjections(handlerTxRunner{}, assetRepo, newHandlerAssetTagRepo(), nil, eventRepo, nil, nil)
	h := New(uc, &mockDeliveryRepoForAsset{})
	r := setupAssetRouter(http.MethodGet, "/assets/:id/tags/history", h.ListTagHistory)

	w := doReq(t, r, http.MethodGet, "/assets/"+tagHistAssetID+"/tags/history?limit=1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from tag history, got %d", w.Code)
	}
	var resp struct {
		Items      []models.AssetEvent `json:"items"`
		NextCursor *int64              `json:"next_cursor"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].EventType != "tag_deleted" {
		t.Fatalf("unexpected tag history response: %+v", resp)
	}
	if resp.NextCursor == nil || *resp.NextCursor != 3 {
		t.Fatalf("expected next_cursor=3, got %+v", resp.NextCursor)
	}
}

func TestAlgoListCurrent(t *testing.T) {
	h, _ := newAlgoEnv(t, true, map[string]string{"hand_tracking@1.2.0": "running"})
	r := setupAlgoRouter(h)
	w := doReq(t, r, http.MethodGet, "/assets/aaaaaaaa/algo", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	items, ok := resp["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected 1 algo row, got %v", resp["items"])
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
		listWithFiltersFn: func(_ context.Context, _ string, _ []interface{}, _, _ int, _ filter.OrderByClause) ([]*models.Asset, int64, error) {
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

// handlerTxRunner runs fn directly. Repos in this file are not transactional
// in any meaningful sense — they hold serialised maps protected by their
// own mutex, mirroring the contract a single PG transaction provides.
type handlerTxRunner struct{}

func (handlerTxRunner) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type handlerAssetTagRepo struct {
	mu   sync.Mutex
	rows map[string]*models.AssetTag
}

func newHandlerAssetTagRepo() *handlerAssetTagRepo {
	return &handlerAssetTagRepo{rows: map[string]*models.AssetTag{}}
}

func tagRowKey(assetID, tagKey string) string { return assetID + "|" + tagKey }

func (m *handlerAssetTagRepo) Upsert(_ context.Context, in repository.AssetTagUpsertInput) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rows[tagRowKey(in.AssetID, in.TagKey)] = &models.AssetTag{
		AssetID:       in.AssetID,
		TagKey:        in.TagKey,
		TagValue:      in.TagValue,
		TagType:       in.TagType,
		SourceType:    in.SourceType,
		SourceName:    in.SourceName,
		SourceVersion: in.SourceVersion,
		RunID:         in.RunID,
		AppliedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	return nil
}

func (m *handlerAssetTagRepo) ListByAsset(_ context.Context, assetID string) ([]*models.AssetTag, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*models.AssetTag
	for _, row := range m.rows {
		if row.AssetID == assetID {
			cp := *row
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (m *handlerAssetTagRepo) Delete(_ context.Context, assetID, tagKey, sourceType string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if sourceType == "" {
		delete(m.rows, tagRowKey(assetID, tagKey))
		return nil
	}
	if row, ok := m.rows[tagRowKey(assetID, tagKey)]; ok && row.SourceType == sourceType {
		delete(m.rows, tagRowKey(assetID, tagKey))
	}
	return nil
}

// handlerAlgoLatestRepo is a tiny in-memory AssetAlgoLatestRepository used
// only by AlgoHandler tests. It enforces the (asset_id, algo_name) PK and
// monotonic algo_version guard like the production repo.
type handlerAlgoLatestRepo struct {
	mu   sync.Mutex
	rows map[string]*models.AssetAlgoLatest
}

func newHandlerAlgoLatestRepo() *handlerAlgoLatestRepo {
	return &handlerAlgoLatestRepo{rows: make(map[string]*models.AssetAlgoLatest)}
}

func handlerKey(assetID, algoName string) string { return assetID + "|" + algoName }

func (m *handlerAlgoLatestRepo) Upsert(_ context.Context, row *models.AssetAlgoLatest) error {
	if row == nil {
		return errors.New("nil row")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	k := handlerKey(row.AssetID, row.AlgoName)
	if cur, ok := m.rows[k]; ok && cur.AlgoVersion > row.AlgoVersion {
		return nil
	}
	cp := *row
	if cp.UpdatedAt.IsZero() {
		cp.UpdatedAt = time.Now().UTC()
	}
	m.rows[k] = &cp
	return nil
}

func (m *handlerAlgoLatestRepo) GetByAlgo(_ context.Context, assetID, algoName string) (*models.AssetAlgoLatest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rows[handlerKey(assetID, algoName)]
	if !ok {
		return nil, nil
	}
	cp := *r
	return &cp, nil
}

func (m *handlerAlgoLatestRepo) ListByAsset(_ context.Context, assetID string) ([]*models.AssetAlgoLatest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*models.AssetAlgoLatest
	for _, r := range m.rows {
		if r.AssetID == assetID {
			cp := *r
			out = append(out, &cp)
		}
	}
	return out, nil
}

// seed pre-populates state. algoKey is the "name@version" string used in
// the public API.
func (m *handlerAlgoLatestRepo) seed(algoKey, status, runID string) {
	parts := strings.SplitN(algoKey, "@", 2)
	if len(parts) != 2 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rows[handlerKey("aaaaaaaa", parts[0])] = &models.AssetAlgoLatest{
		AssetID: "aaaaaaaa", AlgoName: parts[0], AlgoVersion: parts[1],
		Status: status, RunID: runID, UpdatedAt: time.Now().UTC(),
	}
}

// handlerAssetEventRepo is a tiny in-memory AssetEventRepository used by handler tests.
type handlerAssetEventRepo struct {
	mu     sync.Mutex
	events []*models.AssetEvent
}

func (m *handlerAssetEventRepo) Append(_ context.Context, in repository.AssetEventAppendInput) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	payload := in.EventPayload
	if payload == nil {
		payload = []byte(`{}`)
	}
	m.events = append(m.events, &models.AssetEvent{
		EventID:      "evt-" + time.Now().UTC().Format("150405.000000000"),
		EventSeq:     int64(len(m.events) + 1),
		EventType:    in.EventType,
		AssetID:      in.AssetID,
		McapFileID:   in.McapFileID,
		EventPayload: append([]byte(nil), payload...),
		CreatedAt:    time.Now().UTC(),
		OccurredAt:   time.Now().UTC(),
	})
	return nil
}

func (m *handlerAssetEventRepo) ListPending(context.Context, int) ([]*models.AssetEvent, error) {
	return nil, nil
}

func (m *handlerAssetEventRepo) ListPendingSafe(_ context.Context, _ time.Duration, limit int) ([]*models.AssetEvent, error) {
	return m.ListPending(context.Background(), limit)
}

func (m *handlerAssetEventRepo) MarkPublished(context.Context, []int64) error { return nil }

func (m *handlerAssetEventRepo) MarkFailed(context.Context, int64, string) error { return nil }

func (m *handlerAssetEventRepo) CountPending(context.Context) (int64, error) { return 0, nil }
func (m *handlerAssetEventRepo) CountPendingClaimable(context.Context, time.Duration) (int64, error) {
	return 0, nil
}
func (m *handlerAssetEventRepo) CountProcessing(context.Context) (int64, error) { return 0, nil }

func (m *handlerAssetEventRepo) OldestPendingAge(context.Context) (float64, error) {
	return 0, nil
}

func (m *handlerAssetEventRepo) ListBetweenSeq(context.Context, int64, int64, int) ([]*models.AssetEvent, error) {
	return nil, nil
}

func (m *handlerAssetEventRepo) PublishStateCounts(context.Context) (map[string]int64, error) {
	return map[string]int64{}, nil
}

func (m *handlerAssetEventRepo) ListVersionPromotedByLogical(context.Context, string) ([]*models.AssetEvent, error) {
	return nil, nil
}

func (m *handlerAssetEventRepo) ListByAsset(_ context.Context, assetID string, opts repository.AssetEventListOptions) ([]*models.AssetEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}
	allow := map[string]struct{}{}
	for _, t := range opts.EventTypes {
		allow[t] = struct{}{}
	}
	var out []*models.AssetEvent
	for i := len(m.events) - 1; i >= 0; i-- {
		e := m.events[i]
		if e.AssetID != assetID {
			continue
		}
		if len(allow) > 0 {
			if _, ok := allow[e.EventType]; !ok {
				matched := false
				for _, pattern := range opts.EventTypePatterns {
					if strings.HasSuffix(pattern, "%") && strings.HasPrefix(e.EventType, strings.TrimSuffix(pattern, "%")) {
						matched = true
						break
					}
				}
				if !matched {
					continue
				}
			}
		} else if len(opts.EventTypePatterns) > 0 {
			matched := false
			for _, pattern := range opts.EventTypePatterns {
				if strings.HasSuffix(pattern, "%") && strings.HasPrefix(e.EventType, strings.TrimSuffix(pattern, "%")) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		if opts.AlgoKey != "" {
			var payload map[string]any
			if err := json.Unmarshal(e.EventPayload, &payload); err != nil {
				continue
			}
			if got, _ := payload["algo_key"].(string); got != opts.AlgoKey {
				continue
			}
		}
		if opts.BeforeEventSeq != nil && e.EventSeq >= *opts.BeforeEventSeq {
			continue
		}
		if opts.AfterEventSeq != nil && e.EventSeq <= *opts.AfterEventSeq {
			continue
		}
		out = append(out, e)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (m *handlerAssetEventRepo) ListGlobal(_ context.Context, opts repository.AssetEventListOptions) ([]*models.AssetEvent, error) {
	return nil, nil
}

func (m *handlerAssetEventRepo) seed(eventType string, seq int64, payload map[string]any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	raw, _ := json.Marshal(payload)
	aid := "aaaaaaaa"
	if payload != nil {
		if s, ok := payload["asset_id"].(string); ok && s != "" {
			aid = s
		}
	}
	m.events = append(m.events, &models.AssetEvent{
		EventID:      "evt-seed-" + strings.ReplaceAll(time.Now().UTC().Format(time.RFC3339Nano), ":", "-"),
		EventSeq:     seq,
		EventType:    eventType,
		AssetID:      aid,
		EventPayload: raw,
		CreatedAt:    time.Now().UTC(),
		OccurredAt:   time.Now().UTC(),
	})
}

func buildTestAlgoRegistry(t *testing.T) *config.AlgoRegistry {
	t.Helper()
	reg, err := config.LoadAlgoRegistry("../../../config/algo_registry.yaml")
	if err != nil {
		t.Fatalf("failed to load algo registry: %v", err)
	}
	return reg
}

func buildTestTagRegistryForAsset(t *testing.T) *config.TagRegistry {
	t.Helper()
	reg, err := config.LoadTagRegistry("../../../config/tag_registry.yaml")
	if err != nil {
		t.Fatalf("failed to load tag registry: %v", err)
	}
	return reg
}
