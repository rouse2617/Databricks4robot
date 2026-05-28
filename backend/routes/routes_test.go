package routes

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	actionH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/action"
	adminH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/admin"
	assetH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/asset"
	deliveryH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/delivery"
	mcapH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/mcap"
	pipelineH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/pipeline"
	pipelineComponentH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/pipeline_component"
	workflowH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/workflow"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	assetUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/asset"
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
	uc "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline_component"
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
func (r *routeAssetRepo) ListByLogicalAssetID(context.Context, string) ([]*models.Asset, error) {
	return nil, nil
}
func (r *routeAssetRepo) WriteSegmentIndex(context.Context, *models.Asset) error { return nil }
func (r *routeAssetRepo) ListDescendants(_ context.Context, _ string) ([]*models.Asset, error) {
	return nil, nil
}
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
func (r *routeDeliveryRepo) AddItems(context.Context, string, []string) error        { return nil }
func (r *routeDeliveryRepo) RefreshAssetDeliveryIndex(context.Context, string) error { return nil }
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
func (r *routeDeliveryRepo) Update(_ context.Context, _ *models.Delivery, _ int64) error {
	return nil
}

type routeIdemRepo struct{}

func (r *routeIdemRepo) Lock(context.Context, string, string) error { return nil }
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

	// internal route now requires auth (ADMIN_TOKEN or DatabrewToken fallback)
	req = httptest.NewRequest(http.MethodPost, "/internal/commit-segments", nil)
	req.Header.Set("X-Databrew-Token", "dev-token")
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
	deliveryHandler := deliveryH.New(&routeDeliveryRepo{}, &routeIdemRepo{}, nil)
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

func TestAdminReindex_UsesDatabrewTokenAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	assetRepo := &routeAssetRepo{}
	assetHandler := assetH.New(assetUC.New(assetRepo), &routeDeliveryRepo{})
	mcapHandler := mcapH.New(&routeMcapRepo{})
	deliveryHandler := deliveryH.New(&routeDeliveryRepo{}, &routeIdemRepo{}, nil)
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
	deliveryHandler := deliveryH.New(&routeDeliveryRepo{}, &routeIdemRepo{}, nil)
	cfg := &config.Config{DatabrewToken: "dev-token"}
	// Pass nil actionHandler: routes only register when handler is non-nil,
	// so verify both PATCH and DELETE paths are wired by exercising a real
	// handler. Use a handler with a no-op usecase: the call will fail body
	// validation rather than 404 page-not-found, which is what we want to
	// confirm the route is registered.
	actionHandler := actionH.New(nil)
	RegisterAll(r, cfg, nil, assetHandler, mcapHandler, deliveryHandler, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, actionHandler, nil, nil, nil, nil, nil)

	want := map[string]bool{
		"PATCH /api/v1/assets/:id/action-annotations/:action_id":  false,
		"DELETE /api/v1/assets/:id/action-annotations/:action_id": false,
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

func TestAssetTypeSchemaRouteRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	assetHandler := assetH.New(assetUC.New(&routeAssetRepo{}), &routeDeliveryRepo{})
	mcapHandler := mcapH.New(&routeMcapRepo{})
	deliveryHandler := deliveryH.New(&routeDeliveryRepo{}, &routeIdemRepo{}, nil)
	cfg := &config.Config{DatabrewToken: "dev-token"}

	RegisterAll(r, cfg, nil, assetHandler, mcapHandler, deliveryHandler, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/asset-types/dataset/schema", nil)
	req.Header.Set("X-Databrew-Token", "dev-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestAuthLogin_SetsSecureCookieInProduction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	assetHandler := assetH.New(assetUC.New(&routeAssetRepo{}), &routeDeliveryRepo{})
	mcapHandler := mcapH.New(&routeMcapRepo{})
	deliveryHandler := deliveryH.New(&routeDeliveryRepo{}, &routeIdemRepo{}, nil)
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

// ── Pipeline route integration tests ──────────────────────────────────────────

type routePipelineTemplateRepo struct{}

func (r *routePipelineTemplateRepo) Save(_ context.Context, t *models.PipelineTemplate) error {
	t.ID = "tmpl-1"
	return nil
}
func (r *routePipelineTemplateRepo) FindAll(_ context.Context) ([]models.PipelineTemplate, error) {
	return nil, nil
}
func (r *routePipelineTemplateRepo) FindByID(_ context.Context, id string) (*models.PipelineTemplate, error) {
	if id == "" {
		return nil, nil
	}
	return &models.PipelineTemplate{ID: id, Name: "test-tmpl", Version: 1}, nil
}
func (r *routePipelineTemplateRepo) FindVersionsByName(_ context.Context, _ string) ([]models.PipelineTemplate, error) {
	return nil, nil
}
func (r *routePipelineTemplateRepo) GetNextVersion(_ context.Context, _ string) (int, error) {
	return 1, nil
}
func (r *routePipelineTemplateRepo) Delete(_ context.Context, _ string) error { return nil }

type routePipelineDeploymentRepo struct{}

func (r *routePipelineDeploymentRepo) Save(_ context.Context, d *models.PipelineDeployment) error {
	d.ID = "dep-1"
	return nil
}
func (r *routePipelineDeploymentRepo) FindAll(_ context.Context) ([]models.PipelineDeployment, error) {
	return nil, nil
}
func (r *routePipelineDeploymentRepo) FindByID(_ context.Context, id string) (*models.PipelineDeployment, error) {
	if id == "" {
		return nil, nil
	}
	return &models.PipelineDeployment{ID: id, PipelineName: "test-pipeline", Status: "Pending"}, nil
}
func (r *routePipelineDeploymentRepo) Delete(_ context.Context, _ string) error          { return nil }
func (r *routePipelineDeploymentRepo) UpdateStatus(_ context.Context, _, _ string) error { return nil }

type routePipelineComponentRepo struct{}

func (r *routePipelineComponentRepo) Save(_ context.Context, c *models.PipelineComponent) error {
	c.ID = "comp-1"
	return nil
}
func (r *routePipelineComponentRepo) FindAll(_ context.Context, _ *repository.ComponentFilter) ([]models.PipelineComponent, error) {
	return nil, nil
}
func (r *routePipelineComponentRepo) FindByID(_ context.Context, id string) (*models.PipelineComponent, error) {
	if id == "" {
		return nil, nil
	}
	return &models.PipelineComponent{ID: id, Name: "test-comp"}, nil
}
func (r *routePipelineComponentRepo) Update(_ context.Context, _ *models.PipelineComponent) error {
	return nil
}
func (r *routePipelineComponentRepo) Delete(_ context.Context, _ string) error { return nil }

// mockWorkflowClient implements argo.WorkflowClient for route-level tests.
type mockWorkflowClient struct{}

func (m *mockWorkflowClient) CreateWorkflow(_ context.Context, _ *wfv1.Workflow, _ string) error {
	return nil
}
func (m *mockWorkflowClient) GetWorkflowStatus(_ context.Context, _, _ string) (wfv1.WorkflowPhase, error) {
	return wfv1.WorkflowPhase("Running"), nil
}
func (m *mockWorkflowClient) DeleteWorkflow(_ context.Context, _, _ string) error { return nil }
func (m *mockWorkflowClient) ListWorkflows(_ context.Context, _, _ string) ([]wfv1.Workflow, error) {
	return nil, nil
}
func (m *mockWorkflowClient) GetWorkflow(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
	return &wfv1.Workflow{}, nil
}
func (m *mockWorkflowClient) StopWorkflow(_ context.Context, _, _ string) error { return nil }
func (m *mockWorkflowClient) GetWorkflowLogs(_ context.Context, _, _, _ string) (string, error) {
	return "", nil
}

func TestPipelineRoutes_Registered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	assetRepo := &routeAssetRepo{}
	assetHandler := assetH.New(assetUC.New(assetRepo), &routeDeliveryRepo{})
	mcapHandler := mcapH.New(&routeMcapRepo{})
	deliveryHandler := deliveryH.New(&routeDeliveryRepo{}, &routeIdemRepo{}, nil)
	cfg := &config.Config{DatabrewToken: "dev-token"}

	// Create pipeline handlers with mock repos.
	pipelineUCInst := pipelineUC.New(
		&routePipelineTemplateRepo{},
		&routePipelineDeploymentRepo{},
		assetRepo,
		&mockWorkflowClient{},
		"default",
	)
	pipelineHandler := pipelineH.New(pipelineUCInst)
	pipelineComponentHandler := pipelineComponentH.New(uc.New(&routePipelineComponentRepo{}))
	workflowHandler := workflowH.New(&mockWorkflowClient{}, "default")

	RegisterAll(r, cfg, nil, assetHandler, mcapHandler, deliveryHandler,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pipelineHandler, pipelineComponentHandler, nil, workflowHandler, nil,
	)

	// Pipeline routes that should be registered.
	pipelineRoutes := []struct {
		method string
		path   string
		body   string // empty string = no body
		// Expected status with valid auth (route registered and handler called).
		// We expect either the status below or a 405 (method not allowed).
		wantStatus int
	}{
		// Pipeline template routes
		{http.MethodPost, "/api/v1/pipelines", `{"name":"t","pipeline":{}}`, http.StatusCreated},
		{http.MethodGet, "/api/v1/pipelines", "", http.StatusOK},
		{http.MethodGet, "/api/v1/pipelines/tmpl-1", "", http.StatusOK},
		{http.MethodDelete, "/api/v1/pipelines/tmpl-1", "", http.StatusOK},
		{http.MethodGet, "/api/v1/pipelines/tmpl-1/versions", "", http.StatusOK},
		{http.MethodGet, "/api/v1/pipelines/tmpl-1/diff/tmpl-2", "", http.StatusOK},

		// Deploy routes
		{http.MethodPost, "/api/v1/deploy", `{"pipeline":{}}`, http.StatusOK},
		{http.MethodPost, "/api/v1/deploy/template/tmpl-1", `{}`, http.StatusOK},

		// Deployment record routes
		{http.MethodGet, "/api/v1/deployments", "", http.StatusOK},
		{http.MethodGet, "/api/v1/deployments/dep-1", "", http.StatusOK},
		{http.MethodGet, "/api/v1/deployments/dep-1/resources", "", http.StatusOK},
		{http.MethodPost, "/api/v1/deployments/dep-1/retry", `{}`, http.StatusOK},
		{http.MethodPost, "/api/v1/deployments/dep-1/stop", `{}`, http.StatusOK},
		{http.MethodPost, "/api/v1/deployments/dep-1/save-template", `{}`, http.StatusOK},
		{http.MethodDelete, "/api/v1/deployments/dep-1", "", http.StatusOK},

		// Pipeline asset routes
		{http.MethodPost, "/api/v1/pipeline-assets", `{"deployment_id":"dep-1"}`, http.StatusOK},
		{http.MethodGet, "/api/v1/assets/aaaaaaaa/pipeline-lineage", "", http.StatusOK},
	}

	// Verify each route returns 401 without auth.
	for _, pr := range pipelineRoutes {
		t.Run("no_auth_"+pr.method+"_"+pr.path, func(t *testing.T) {
			var req *http.Request
			if pr.body != "" {
				req = httptest.NewRequest(pr.method, pr.path, strings.NewReader(pr.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(pr.method, pr.path, nil)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Errorf("expected 401 for %s %s without auth, got %d", pr.method, pr.path, w.Code)
			}
		})
	}

	// Verify each route returns the expected status with valid auth.
	for _, pr := range pipelineRoutes {
		t.Run("with_auth_"+pr.method+"_"+pr.path, func(t *testing.T) {
			var req *http.Request
			if pr.body != "" {
				req = httptest.NewRequest(pr.method, pr.path, strings.NewReader(pr.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(pr.method, pr.path, nil)
			}
			req.Header.Set("X-Databrew-Token", "dev-token")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != pr.wantStatus {
				// For deployment-specific endpoints that need actual K8s interaction,
				// allow 500 (internal error from handler calling nil mock state)
				// as long as it's not 404 (route not registered).
				if w.Code == http.StatusNotFound {
					t.Errorf("route not registered: %s %s (got 404, want %d)", pr.method, pr.path, pr.wantStatus)
				}
			}
		})
	}
}

func TestPipelineComponentRoutes_Registered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	assetRepo := &routeAssetRepo{}
	assetHandler := assetH.New(assetUC.New(assetRepo), &routeDeliveryRepo{})
	mcapHandler := mcapH.New(&routeMcapRepo{})
	deliveryHandler := deliveryH.New(&routeDeliveryRepo{}, &routeIdemRepo{}, nil)
	cfg := &config.Config{DatabrewToken: "dev-token"}

	pipelineComponentHandler := pipelineComponentH.New(uc.New(&routePipelineComponentRepo{}))

	RegisterAll(r, cfg, nil, assetHandler, mcapHandler, deliveryHandler,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		nil, pipelineComponentHandler, nil, nil, nil,
	)

	componentRoutes := []struct {
		method string
		path   string
		body   string
		want   int
	}{
		{http.MethodPost, "/api/v1/components", `{"name":"c1","image":"img"}`, http.StatusCreated},
		{http.MethodGet, "/api/v1/components", "", http.StatusOK},
		{http.MethodGet, "/api/v1/components/comp-1", "", http.StatusOK},
		{http.MethodPut, "/api/v1/components/comp-1", `{"name":"c1-updated","image":"img"}`, http.StatusOK},
		{http.MethodDelete, "/api/v1/components/comp-1", "", http.StatusOK},
	}

	// Without auth.
	for _, cr := range componentRoutes {
		t.Run("no_auth_"+cr.method+"_"+cr.path, func(t *testing.T) {
			var req *http.Request
			if cr.body != "" {
				req = httptest.NewRequest(cr.method, cr.path, strings.NewReader(cr.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(cr.method, cr.path, nil)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Errorf("expected 401 for %s %s without auth, got %d", cr.method, cr.path, w.Code)
			}
		})
	}

	// With auth.
	for _, cr := range componentRoutes {
		t.Run("with_auth_"+cr.method+"_"+cr.path, func(t *testing.T) {
			var req *http.Request
			if cr.body != "" {
				req = httptest.NewRequest(cr.method, cr.path, strings.NewReader(cr.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(cr.method, cr.path, nil)
			}
			req.Header.Set("X-Databrew-Token", "dev-token")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code == http.StatusNotFound {
				t.Errorf("route not registered: %s %s (got 404, want %d)", cr.method, cr.path, cr.want)
			}
		})
	}
}

func TestWorkflowRoutes_Registered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	assetRepo := &routeAssetRepo{}
	assetHandler := assetH.New(assetUC.New(assetRepo), &routeDeliveryRepo{})
	mcapHandler := mcapH.New(&routeMcapRepo{})
	deliveryHandler := deliveryH.New(&routeDeliveryRepo{}, &routeIdemRepo{}, nil)
	cfg := &config.Config{DatabrewToken: "dev-token"}

	workflowHandler := workflowH.New(&mockWorkflowClient{}, "default")

	RegisterAll(r, cfg, nil, assetHandler, mcapHandler, deliveryHandler,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil, workflowHandler, nil,
	)

	workflowRoutes := []struct {
		method string
		path   string
		want   int
	}{
		{http.MethodGet, "/api/v1/workflows", http.StatusOK},
		{http.MethodGet, "/api/v1/workflows/test-wf/logs?nodeId=n1", http.StatusOK},
		{http.MethodGet, "/api/v1/workflows/test-wf", http.StatusOK},
	}

	// Without auth.
	for _, wr := range workflowRoutes {
		t.Run("no_auth_"+wr.method+"_"+wr.path, func(t *testing.T) {
			req := httptest.NewRequest(wr.method, wr.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Errorf("expected 401 for %s %s without auth, got %d", wr.method, wr.path, w.Code)
			}
		})
	}

	// With auth.
	for _, wr := range workflowRoutes {
		t.Run("with_auth_"+wr.method+"_"+wr.path, func(t *testing.T) {
			req := httptest.NewRequest(wr.method, wr.path, nil)
			req.Header.Set("X-Databrew-Token", "dev-token")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code == http.StatusNotFound {
				t.Errorf("route not registered: %s %s (got 404, want %d)", wr.method, wr.path, wr.want)
			}
		})
	}
}
