package routes

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"data-platform/internal/config"
	adminH "data-platform/internal/handlers/admin"
	assetH "data-platform/internal/handlers/asset"
	deliveryH "data-platform/internal/handlers/delivery"
	mcapH "data-platform/internal/handlers/mcap"
	"data-platform/internal/models"
	"data-platform/internal/repository"
	assetUC "data-platform/internal/usecase/asset"
)

type routeAssetRepo struct{}

func (r *routeAssetRepo) Get(context.Context, string) (*models.Asset, error) {
	return &models.Asset{AssetID: "aaaaaaaa", Tags: map[string]string{}}, nil
}
func (r *routeAssetRepo) InsertNew(context.Context, *models.Asset) error { return nil }
func (r *routeAssetRepo) Set(context.Context, *models.Asset) error       { return nil }
func (r *routeAssetRepo) SoftDelete(context.Context, string) error       { return nil }
func (r *routeAssetRepo) ListByMcapFile(context.Context, string) ([]*models.Asset, error) {
	return []*models.Asset{}, nil
}
func (r *routeAssetRepo) WriteSegmentIndex(context.Context, *models.Asset) error { return nil }
func (r *routeAssetRepo) ListWithFilters(context.Context, string, []interface{}, int, int, string) ([]*models.Asset, int64, error) {
	return nil, 0, nil
}
func (r *routeAssetRepo) MergeCfAlgo(context.Context, string, int64, map[string]interface{}, map[string]interface{}) (int64, error) {
	return 0, nil
}

type routeMcapRepo struct{}

func (r *routeMcapRepo) Get(context.Context, string) (*models.McapFile, error) {
	return &models.McapFile{McapFileID: "m1"}, nil
}
func (r *routeMcapRepo) Set(context.Context, *models.McapFile) error { return nil }
func (r *routeMcapRepo) UpdateIngestState(context.Context, string, models.IngestState) error {
	return nil
}
func (r *routeMcapRepo) List(context.Context, int, int, string, string) ([]*models.McapFile, int64, error) {
	return []*models.McapFile{}, 0, nil
}

type routeDeliveryRepo struct{}

func (r *routeDeliveryRepo) Set(context.Context, *models.Delivery) error { return nil }
func (r *routeDeliveryRepo) Get(context.Context, string) (*models.Delivery, error) {
	return &models.Delivery{DeliveryID: "d1"}, nil
}
func (r *routeDeliveryRepo) WriteIndexes(context.Context, string, *models.Delivery) error { return nil }
func (r *routeDeliveryRepo) ListByCustomer(context.Context, string) ([]string, error) {
	return []string{"d1"}, nil
}
func (r *routeDeliveryRepo) ListByAsset(context.Context, string) ([]string, error) {
	return []string{"d1"}, nil
}
func (r *routeDeliveryRepo) ListItems(context.Context, string) ([]*models.DeliveryItem, error) {
	return []*models.DeliveryItem{{DeliveryID: "d1", AssetID: "aaaaaaaa"}}, nil
}
func (r *routeDeliveryRepo) List(context.Context, int, int, string) ([]*models.Delivery, int64, error) {
	return []*models.Delivery{}, 0, nil
}

type routeIdemRepo struct{}

func (r *routeIdemRepo) Get(context.Context, string, string) (*repository.IdempotencyRecord, error) {
	return nil, nil
}
func (r *routeIdemRepo) Save(context.Context, *repository.IdempotencyRecord) error { return nil }

func TestRegisterAll(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	assetHandler := assetH.New(assetUC.New(&routeAssetRepo{}), &routeDeliveryRepo{})
	mcapHandler := mcapH.New(&routeMcapRepo{})
	deliveryHandler := deliveryH.New(&routeDeliveryRepo{}, &routeIdemRepo{})
	cfg := &config.Config{GraceToken: "dev-token"}

	RegisterAll(r, cfg, assetHandler, mcapHandler, deliveryHandler, nil, nil, nil, nil, nil, nil, nil, nil)

	// healthz: no auth
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("healthz expected 200, got %d", w.Code)
	}
	if w.Header().Get("X-Request-ID") == "" {
		t.Fatalf("expected request id header")
	}

	// protected route without token
	req = httptest.NewRequest(http.MethodGet, "/api/v1/mcap-files", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	// protected route with X-Grace-Token
	req = httptest.NewRequest(http.MethodGet, "/api/v1/mcap-files", nil)
	req.Header.Set("X-Grace-Token", "dev-token")
	req.Header.Set("X-Request-ID", "rid-1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Header().Get("X-Request-ID"); got != "rid-1" {
		t.Fatalf("expected request-id echo, got %q", got)
	}

	// protected route with Authorization bearer
	req = httptest.NewRequest(http.MethodGet, "/api/v1/mcap-files/m1", nil)
	req.Header.Set("Authorization", "Bearer dev-token")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// auth login + me via cookie session
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", http.NoBody)
	req.Header.Set("Content-Type", "application/json")
	req.Body = io.NopCloser(strings.NewReader(`{"token":"dev-token"}`))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login expected 200, got %d", w.Code)
	}
	cookie := w.Header().Get("Set-Cookie")
	if cookie == "" {
		t.Fatalf("expected session cookie")
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Cookie", cookie)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("me expected 200, got %d", w.Code)
	}

	// internal route bypasses auth
	req = httptest.NewRequest(http.MethodPost, "/internal/commit-segments", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 (bind error), got %d", w.Code)
	}
}

func TestRemovedHealthzOutboxRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	assetHandler := assetH.New(assetUC.New(&routeAssetRepo{}), &routeDeliveryRepo{})
	mcapHandler := mcapH.New(&routeMcapRepo{})
	deliveryHandler := deliveryH.New(&routeDeliveryRepo{}, &routeIdemRepo{})
	cfg := &config.Config{GraceToken: "dev-token"}

	RegisterAll(r, cfg, assetHandler, mcapHandler, deliveryHandler, nil, nil, nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/healthz/outbox", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("/healthz/outbox expected 404 after removal, got %d", w.Code)
	}
}

func TestAdminReindex_UsesGraceTokenAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	assetRepo := &routeAssetRepo{}
	assetHandler := assetH.New(assetUC.New(assetRepo), &routeDeliveryRepo{})
	mcapHandler := mcapH.New(&routeMcapRepo{})
	deliveryHandler := deliveryH.New(&routeDeliveryRepo{}, &routeIdemRepo{})
	adminHandler := adminH.New(assetRepo, nil, nil, &routeMcapRepo{}, nil)
	cfg := &config.Config{GraceToken: "dev-token"}

	RegisterAll(r, cfg, assetHandler, mcapHandler, deliveryHandler, nil, nil, nil, nil, adminHandler, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/search/reindex", nil)
	req.Header.Set("X-Grace-Token", "dev-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 from handler with nil ES, got %d", w.Code)
	}
}
