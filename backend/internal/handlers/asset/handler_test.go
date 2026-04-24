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

	"data-platform/internal/models"
	assetUC "data-platform/internal/usecase/asset"
)

type mockAssetRepo struct {
	getFn            func(ctx context.Context, assetID string) (*models.Asset, error)
	setFn            func(ctx context.Context, a *models.Asset) error
	softDeleteFn     func(ctx context.Context, assetID string) error
	listByMcapFileFn func(ctx context.Context, mcapFileID string) ([]*models.Asset, error)
	writeSegIndexFn  func(ctx context.Context, a *models.Asset) error
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
	h := New(assetUC.New(repo))
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
	}
	h := New(assetUC.New(repo))
	r := setupAssetRouter(http.MethodGet, "/assets", h.List)

	w := doReq(t, r, http.MethodGet, "/assets", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	w = doReq(t, r, http.MethodGet, "/assets?mcap_file_id=m1&page=1&page_size=1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestCreate(t *testing.T) {
	repo := &mockAssetRepo{}
	h := New(assetUC.New(repo))
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
	h := New(assetUC.New(repo))

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
	h := New(assetUC.New(&mockAssetRepo{}))
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
