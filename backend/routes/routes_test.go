package routes

import (
	"context"
	"encoding/json"
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
	auditH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/audit"
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
func (r *routeAssetRepo) ListByLogicalAssetID(context.Context, string) ([]*models.Asset, error) {
	return nil, nil
}
func (r *routeAssetRepo) ListDescendants(context.Context, string) ([]*models.Asset, error) {
	return nil, nil
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
func (r *routeDeliveryRepo) AddItems(ctx context.Context, deliveryID string, assetIDs []string) error {
	return nil
}
func (r *routeDeliveryRepo) RefreshAssetDeliveryIndex(ctx context.Context, assetID string) error {
	return nil
}
func (r *routeDeliveryRepo) ListByCustomer(context.Context, string) ([]string, error) {
	return []string{"d1"}, nil
}
func (r *routeDeliveryRepo) ListByAsset(context.Context, string) ([]string, error) {
	return []string{"d1"}, nil
}
func (r *routeDeliveryRepo) ListItems(context.Context, string) ([]*models.DeliveryItem, error) {
	return []*models.DeliveryItem{{DeliveryID: "d1", AssetID: "aaaaaaaa"}}, nil
}
func (r *routeDeliveryRepo) List(ctx context.Context, page, pageSize int, status, customerID string) ([]*models.Delivery, int64, error) {
	return []*models.Delivery{}, 0, nil
}
func (r *routeDeliveryRepo) Update(ctx context.Context, d *models.Delivery, expectedRowVersion int64) error {
	return nil
}

type routeIdemRepo struct{}

func (r *routeIdemRepo) Lock(context.Context, string, string) error { return nil }
func (r *routeIdemRepo) Get(context.Context, string, string) (*repository.IdempotencyRecord, error) {
	return nil, nil
}
func (r *routeIdemRepo) Save(context.Context, *repository.IdempotencyRecord) error { return nil }

type routeCustomerRepo struct{}

func (r *routeCustomerRepo) Insert(context.Context, *models.Customer) error { return nil }
func (r *routeCustomerRepo) Get(context.Context, string) (*models.Customer, error) {
	return &models.Customer{CustomerID: "c1"}, nil
}
func (r *routeCustomerRepo) Update(context.Context, *models.Customer) error { return nil }
func (r *routeCustomerRepo) Exists(context.Context, string) (bool, error)   { return true, nil }
func (r *routeCustomerRepo) List(ctx context.Context, status, slaTier, region string, limit int, cursor string) ([]*models.Customer, error) {
	return nil, nil
}

func TestRegisterAll(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	assetHandler := assetH.New(assetUC.New(&routeAssetRepo{}), &routeDeliveryRepo{})
	mcapHandler := mcapH.New(&routeMcapRepo{})
	deliveryHandler := deliveryH.New(&routeDeliveryRepo{}, &routeIdemRepo{}, &routeCustomerRepo{})
	cfg := &config.Config{DatabrewToken: "dev-token"}

	RegisterAll(r, cfg, nil, assetHandler, mcapHandler, deliveryHandler, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

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

	// protected route with X-Databrew-Token
	req = httptest.NewRequest(http.MethodGet, "/api/v1/mcap-files", nil)
	req.Header.Set("X-Databrew-Token", "dev-token")
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
	assertStandardError(t, w, "INVALID_ARGUMENT")

	// internal route now requires auth (ADMIN_TOKEN or GraceToken fallback)
	req = httptest.NewRequest(http.MethodPost, "/internal/commit-segments", nil)
	req.Header.Set("X-Databrew-Token", "dev-token")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 (bind error), got %d", w.Code)
	}

	assertRoutesRegistered(t, r, []string{
		"GET /api/v1/assets/:id/events/stream",
		"GET /api/v1/events/stream",
		"POST /api/v1/deliveries/draft",
		"POST /api/v1/deliveries/:id/items",
		"POST /api/v1/deliveries/:id/commit",
		"POST /api/v1/deliveries/:id/cancel",
		"POST /api/v1/deliveries/:id/retry",
		"POST /api/v1/deliveries/:id/ack",
	})
}

func TestAuditSearchRouteRegisteredWhenHandlerProvided(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	assetHandler := assetH.New(assetUC.New(&routeAssetRepo{}), &routeDeliveryRepo{})
	mcapHandler := mcapH.New(&routeMcapRepo{})
	deliveryHandler := deliveryH.New(&routeDeliveryRepo{}, &routeIdemRepo{}, &routeCustomerRepo{})
	cfg := &config.Config{DatabrewToken: "dev-token"}

	RegisterAll(r, cfg, nil, assetHandler, mcapHandler, deliveryHandler, nil, nil, nil, nil, auditH.New(nil), nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	assertRoutesRegistered(t, r, []string{"GET /api/v1/audit/search"})
}

func TestAuthRoutes_ReturnStandardErrorEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	assetHandler := assetH.New(assetUC.New(&routeAssetRepo{}), &routeDeliveryRepo{})
	mcapHandler := mcapH.New(&routeMcapRepo{})
	deliveryHandler := deliveryH.New(&routeDeliveryRepo{}, &routeIdemRepo{}, &routeCustomerRepo{})
	cfg := &config.Config{DatabrewToken: "dev-token", AllowedDomain: "example.com"}

	RegisterAll(r, cfg, nil, assetHandler, mcapHandler, deliveryHandler, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	cases := []struct {
		name string
		path string
		body string
		code int
		err  string
	}{
		{
			name: "email login invalid json",
			path: "/api/v1/auth/email-login",
			body: `{`,
			code: http.StatusBadRequest,
			err:  "INVALID_ARGUMENT",
		},
		{
			name: "email domain rejected",
			path: "/api/v1/auth/email-login",
			body: `{"email":"user@other.test"}`,
			code: http.StatusUnauthorized,
			err:  "UNAUTHORIZED",
		},
		{
			name: "legacy login blank token",
			path: "/api/v1/auth/login",
			body: `{"token":"   "}`,
			code: http.StatusBadRequest,
			err:  "INVALID_ARGUMENT",
		},
		{
			name: "legacy login invalid token",
			path: "/api/v1/auth/login",
			body: `{"token":"wrong"}`,
			code: http.StatusUnauthorized,
			err:  "UNAUTHORIZED",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.code {
				t.Fatalf("expected %d, got %d: %s", tc.code, w.Code, w.Body.String())
			}
			assertStandardError(t, w, tc.err)
		})
	}
}

func assertStandardError(t *testing.T, w *httptest.ResponseRecorder, wantCode string) {
	t.Helper()
	var body struct {
		Code      string         `json:"code"`
		Message   string         `json:"message"`
		RequestID string         `json:"request_id"`
		Details   map[string]any `json:"details"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("error body is not JSON: %v; body=%s", err, w.Body.String())
	}
	if body.Code != wantCode {
		t.Fatalf("expected error code %q, got %q; body=%s", wantCode, body.Code, w.Body.String())
	}
	if body.Message == "" {
		t.Fatalf("expected error message; body=%s", w.Body.String())
	}
	if body.RequestID == "" {
		t.Fatalf("expected request_id; body=%s", w.Body.String())
	}
}

func assertRoutesRegistered(t *testing.T, r *gin.Engine, want []string) {
	t.Helper()
	seen := map[string]bool{}
	for _, ri := range r.Routes() {
		seen[ri.Method+" "+ri.Path] = true
	}
	for _, route := range want {
		if !seen[route] {
			t.Fatalf("route not registered: %s", route)
		}
	}
}

func TestRemovedHealthzOutboxRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	assetHandler := assetH.New(assetUC.New(&routeAssetRepo{}), &routeDeliveryRepo{})
	mcapHandler := mcapH.New(&routeMcapRepo{})
	deliveryHandler := deliveryH.New(&routeDeliveryRepo{}, &routeIdemRepo{}, &routeCustomerRepo{})
	cfg := &config.Config{DatabrewToken: "dev-token"}

	RegisterAll(r, cfg, nil, assetHandler, mcapHandler, deliveryHandler, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

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
	deliveryHandler := deliveryH.New(&routeDeliveryRepo{}, &routeIdemRepo{}, &routeCustomerRepo{})
	adminHandler := adminH.New(&routeAssetRepo{}, nil, nil, &routeMcapRepo{}, nil, nil, nil, nil, nil)
	cfg := &config.Config{DatabrewToken: "dev-token", Env: "production"}

	RegisterAll(r, cfg, nil, assetHandler, mcapHandler, deliveryHandler, nil, nil, nil, nil, nil, nil, nil, nil, adminHandler, nil, nil, nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/search/reindex", nil)
	req.Header.Set("X-Databrew-Token", "dev-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unmounted admin route in production, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/internal/commit-segments", nil)
	req.Header.Set("X-Databrew-Token", "dev-token")
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
	deliveryHandler := deliveryH.New(&routeDeliveryRepo{}, &routeIdemRepo{}, &routeCustomerRepo{})
	adminHandler := adminH.New(assetRepo, nil, nil, &routeMcapRepo{}, nil, nil, nil, nil, nil)
	cfg := &config.Config{DatabrewToken: "dev-token"}

	RegisterAll(r, cfg, nil, assetHandler, mcapHandler, deliveryHandler, nil, nil, nil, nil, nil, nil, nil, nil, adminHandler, nil, nil, nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/search/reindex", nil)
	req.Header.Set("X-Databrew-Token", "dev-token")
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
	deliveryHandler := deliveryH.New(&routeDeliveryRepo{}, &routeIdemRepo{}, &routeCustomerRepo{})
	cfg := &config.Config{DatabrewToken: "dev-token"}
	actionHandler := actionH.New(nil)
	RegisterAll(r, cfg, nil, assetHandler, mcapHandler, deliveryHandler, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, actionHandler, nil, nil, nil, nil, nil)

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
	deliveryHandler := deliveryH.New(&routeDeliveryRepo{}, &routeIdemRepo{}, &routeCustomerRepo{})
	cfg := &config.Config{DatabrewToken: "dev-token", Env: "production"}

	RegisterAll(r, cfg, nil, assetHandler, mcapHandler, deliveryHandler, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

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
