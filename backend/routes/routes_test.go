package routes

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	actionH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/action"
	adminH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/admin"
	assetH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/asset"
	deliveryH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/delivery"
	mcapH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/mcap"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	assetUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/asset"
)

type routeAssetRepo struct{}

func (r *routeAssetRepo) Get(ctx context.Context, id string) (*models.Asset, error) {
	return r.GetAll(ctx, id)
}
func (r *routeAssetRepo) GetAll(context.Context, string) (*models.Asset, error) {
	return &models.Asset{AssetID: "aaaaaaaa", Tags: map[string]string{}}, nil
}
func (r *routeAssetRepo) InsertNew(context.Context, *models.Asset) error { return nil }
func (r *routeAssetRepo) Set(context.Context, *models.Asset) error       { return nil }
func (r *routeAssetRepo) SoftDelete(context.Context, string) error       { return nil }
func (r *routeAssetRepo) ListByMcapFile(context.Context, string) ([]*models.Asset, error) {
	return []*models.Asset{}, nil
}
func (r *routeAssetRepo) WriteSegmentIndex(context.Context, *models.Asset) error { return nil }
func (r *routeAssetRepo) ListWithFilters(context.Context, string, []interface{}, int, int, filter.OrderByClause) ([]*models.Asset, int64, error) {
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
func (r *routeDeliveryRepo) List(context.Context, int, int, string, string) ([]*models.Delivery, int64, error) {
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
	deliveryHandler := deliveryH.New(&routeDeliveryRepo{}, &routeIdemRepo{}, nil)
	cfg := &config.Config{GraceToken: "dev-token"}

	RegisterAll(r, cfg, nil, assetHandler, mcapHandler, deliveryHandler, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

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

	// auth login trims blank token as bad request
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", http.NoBody)
	req.Header.Set("Content-Type", "application/json")
	req.Body = io.NopCloser(strings.NewReader(`{"token":"   "}`))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("blank token expected 400, got %d", w.Code)
	}

	// internal route now requires auth (ADMIN_TOKEN or GraceToken fallback)
	req = httptest.NewRequest(http.MethodPost, "/internal/commit-segments", nil)
	req.Header.Set("X-Grace-Token", "dev-token")
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
	deliveryHandler := deliveryH.New(&routeDeliveryRepo{}, &routeIdemRepo{}, nil)
	cfg := &config.Config{GraceToken: "dev-token"}

	RegisterAll(r, cfg, nil, assetHandler, mcapHandler, deliveryHandler, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/healthz/outbox", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("/healthz/outbox expected 404 after removal, got %d", w.Code)
	}
}

func TestAdminRoutes_DisabledInProductionWithoutAdminToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	assetHandler := assetH.New(assetUC.New(&routeAssetRepo{}), &routeDeliveryRepo{})
	mcapHandler := mcapH.New(&routeMcapRepo{})
	deliveryHandler := deliveryH.New(&routeDeliveryRepo{}, &routeIdemRepo{}, nil)
	adminHandler := adminH.New(&routeAssetRepo{}, nil, nil, &routeMcapRepo{}, nil, nil, nil, nil, nil)
	cfg := &config.Config{GraceToken: "dev-token", Env: "production"}

	RegisterAll(r, cfg, nil, assetHandler, mcapHandler, deliveryHandler, nil, nil, nil, nil, nil, adminHandler, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/search/reindex", nil)
	req.Header.Set("X-Grace-Token", "dev-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unmounted admin route in production, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/internal/commit-segments", nil)
	req.Header.Set("X-Grace-Token", "dev-token")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unmounted internal route in production, got %d", w.Code)
	}
}

func TestAdminReindex_UsesGraceTokenAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	assetRepo := &routeAssetRepo{}
	assetHandler := assetH.New(assetUC.New(assetRepo), &routeDeliveryRepo{})
	mcapHandler := mcapH.New(&routeMcapRepo{})
	deliveryHandler := deliveryH.New(&routeDeliveryRepo{}, &routeIdemRepo{}, nil)
	adminHandler := adminH.New(assetRepo, nil, nil, &routeMcapRepo{}, nil, nil, nil, nil, nil)
	cfg := &config.Config{GraceToken: "dev-token"}

	RegisterAll(r, cfg, nil, assetHandler, mcapHandler, deliveryHandler, nil, nil, nil, nil, nil, adminHandler, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/search/reindex", nil)
	req.Header.Set("X-Grace-Token", "dev-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 from handler with nil ES, got %d", w.Code)
	}
}

func TestActionRoutes_PatchAndDeleteRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	assetHandler := assetH.New(assetUC.New(&routeAssetRepo{}), &routeDeliveryRepo{})
	mcapHandler := mcapH.New(&routeMcapRepo{})
	deliveryHandler := deliveryH.New(&routeDeliveryRepo{}, &routeIdemRepo{}, nil)
	cfg := &config.Config{GraceToken: "dev-token"}
	// Pass nil actionHandler: routes only register when handler is non-nil,
	// so verify both PATCH and DELETE paths are wired by exercising a real
	// handler. Use a handler with a no-op usecase: the call will fail body
	// validation rather than 404 page-not-found, which is what we want to
	// confirm the route is registered.
	actionHandler := actionH.New(nil)
	RegisterAll(r, cfg, nil, assetHandler, mcapHandler, deliveryHandler, nil, nil, nil, nil, nil, nil, nil, nil, actionHandler, nil)

	want := map[string]bool{
		"PATCH /api/v1/assets/:id/actions/:action_id":  false,
		"DELETE /api/v1/assets/:id/actions/:action_id": false,
	}
	for _, ri := range r.Routes() {
		key := ri.Method + " " + ri.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for k, ok := range want {
		if !ok {
			t.Fatalf("route not registered: %s", k)
		}
	}
}

func TestAuthLogin_SetsSecureCookieInProduction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	assetHandler := assetH.New(assetUC.New(&routeAssetRepo{}), &routeDeliveryRepo{})
	mcapHandler := mcapH.New(&routeMcapRepo{})
	deliveryHandler := deliveryH.New(&routeDeliveryRepo{}, &routeIdemRepo{}, nil)
	cfg := &config.Config{GraceToken: "dev-token", Env: "production"}

	RegisterAll(r, cfg, nil, assetHandler, mcapHandler, deliveryHandler, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"token":"dev-token"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login expected 200, got %d", w.Code)
	}
	cookie := w.Header().Get("Set-Cookie")
	if cookie == "" {
		t.Fatalf("expected session cookie")
	}
	if !strings.Contains(cookie, "Secure") {
		t.Fatalf("expected Secure cookie in production, got %q", cookie)
	}
}
