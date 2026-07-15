package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/auth"
	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
	actionH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/action"
	assetH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/asset"
	deliveryH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/delivery"
	evalH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/eval"
	mcapH "github.com/CyberOrigin2077/cyber-databrew/internal/handlers/mcap"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	assetUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/asset"
)

// fakeReadOnlyKeyRepo authenticates exactly one API key and grants it only the
// assets:read scope, so requests reach RequireScope as a read-only principal.
type fakeReadOnlyKeyRepo struct{ prefix, secretHash string }

func (f *fakeReadOnlyKeyRepo) Create(context.Context, *models.APIKey) error { return nil }
func (f *fakeReadOnlyKeyRepo) FindByPrefix(_ context.Context, prefix string) (*models.APIKey, error) {
	if prefix != f.prefix {
		return nil, nil
	}
	return &models.APIKey{
		ID:         "k-readonly",
		KeyPrefix:  f.prefix,
		SecretHash: f.secretHash,
		Status:     "active",
		Scopes:     []string{"assets:read"},
	}, nil
}
func (f *fakeReadOnlyKeyRepo) List(context.Context) ([]models.APIKey, error) { return nil, nil }
func (f *fakeReadOnlyKeyRepo) Revoke(context.Context, string) error         { return nil }
func (f *fakeReadOnlyKeyRepo) TouchLastUsed(context.Context, string) error  { return nil }

// CYB-3296 regression guard: every mutating asset route must reject a read-only
// principal with 403. Before the fix, POST /assets/:id/actions, PATCH/DELETE
// .../actions/:action_id and POST /assets/:id/eval-results were missing
// RequireScope("assets:write"), so a read-only API key could mutate data
// (confirmed exploitable on dev). This test builds the real router with a
// read-only key and asserts 403 (not 4xx-from-handler / 200) on each.
func TestMutatingAssetRoutesRequireWriteScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	assetHandler := assetH.New(assetUC.New(&routeAssetRepo{}), &routeDeliveryRepo{})
	mcapHandler := mcapH.New(&routeMcapRepo{})
	deliveryHandler := deliveryH.New(&routeDeliveryRepo{}, &routeIdemRepo{}, &routeCustomerRepo{})
	// Non-nil so the routes register; RequireScope runs before the handler so the
	// handlers themselves are never invoked in this test.
	actionHandler := actionH.New(nil)
	evalHandler := evalH.New(nil, nil, nil)

	full, prefix, secretHash, err := auth.GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey: %v", err)
	}
	roRepo := &fakeReadOnlyKeyRepo{prefix: prefix, secretHash: secretHash}

	cfg := &config.Config{DatabrewToken: "dev-token", JWTSecret: "test-secret"}
	RegisterAll(
		r, cfg, nil,
		assetHandler, mcapHandler, deliveryHandler,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		evalHandler, actionHandler,
		nil, nil, nil, nil, nil, nil, nil,
		roRepo, nil, nil, nil,
	)

	cases := []struct {
		name         string
		method, path string
	}{
		{"create action", http.MethodPost, "/api/v1/assets/aaaaaaaa/actions"},
		{"update action", http.MethodPatch, "/api/v1/assets/aaaaaaaa/actions/bbbbbbbb"},
		{"delete action", http.MethodDelete, "/api/v1/assets/aaaaaaaa/actions/bbbbbbbb"},
		{"report eval", http.MethodPost, "/api/v1/assets/aaaaaaaa/eval-results"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, http.NoBody)
			req.Header.Set("X-Databrew-Token", full)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusForbidden {
				t.Errorf("%s %s with read-only key = %d, want 403", tc.method, tc.path, w.Code)
			}
		})
	}
}
