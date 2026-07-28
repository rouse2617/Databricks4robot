package apikey

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/middleware"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// stubAPIKeyRepo is a minimal APIKeyRepository stub. Only Create is used by
// the tests below; the other methods are needed to satisfy the interface.
type stubAPIKeyRepo struct {
	createCalled bool
	created      *models.APIKey
}

func (s *stubAPIKeyRepo) Create(_ context.Context, k *models.APIKey) error {
	s.createCalled = true
	s.created = k
	return nil
}
func (s *stubAPIKeyRepo) FindByPrefix(context.Context, string) (*models.APIKey, error) {
	return nil, nil
}
func (s *stubAPIKeyRepo) List(context.Context) ([]models.APIKey, error) { return nil, nil }
func (s *stubAPIKeyRepo) Revoke(context.Context, string) error          { return nil }
func (s *stubAPIKeyRepo) TouchLastUsed(context.Context, string) error   { return nil }

// setupCreateRouter mounts POST /api-keys with a middleware that stamps a
// Principal on the gin context, mimicking what auth middleware would do in
// production for the given AuthMethod.
func setupCreateRouter(authMethod string, h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api-keys", func(c *gin.Context) {
		c.Set(middleware.CtxKeyPrincipal, middleware.Principal{
			Subject:    "test",
			Role:       "admin",
			AuthMethod: authMethod,
		})
		c.Next()
	}, h.Create)
	return r
}

func doCreate(t *testing.T, r *gin.Engine, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	buf, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api-keys", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// SECURITY: an admin JWT (obtained via email-login, no proof of email
// ownership) must NOT be able to mint a wildcard-scope api_key that would
// outlive the 24h JWT session.
func TestCreate_RejectsWildcardScopeFromJWT(t *testing.T) {
	repo := &stubAPIKeyRepo{}
	r := setupCreateRouter(middleware.AuthMethodJWT, New(repo))

	w := doCreate(t, r, map[string]any{
		"name":   "attacker-key",
		"owner":  "attacker",
		"scopes": []string{"*"},
	})
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for wildcard scope from JWT, got %d body=%s", w.Code, w.Body.String())
	}
	if repo.createCalled {
		t.Fatalf("repo.Create must NOT be called when scope check fails")
	}
}

// Same restriction for apikeys:manage — a JWT-minted manager key could
// perpetuate itself by issuing further keys, bypassing session TTL.
func TestCreate_RejectsApiKeysManageScopeFromJWT(t *testing.T) {
	repo := &stubAPIKeyRepo{}
	r := setupCreateRouter(middleware.AuthMethodJWT, New(repo))

	w := doCreate(t, r, map[string]any{
		"name":   "self-perpetuating-key",
		"scopes": []string{"apikeys:manage"},
	})
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for apikeys:manage from JWT, got %d body=%s", w.Code, w.Body.String())
	}
	if repo.createCalled {
		t.Fatalf("repo.Create must NOT be called when scope check fails")
	}
}

// Static-admin-token callers have proof-of-possession and may mint
// privileged keys (bootstrap path).
func TestCreate_AllowsWildcardScopeFromStaticToken(t *testing.T) {
	repo := &stubAPIKeyRepo{}
	r := setupCreateRouter(middleware.AuthMethodStaticToken, New(repo))

	w := doCreate(t, r, map[string]any{
		"name":   "bootstrap-key",
		"scopes": []string{"*"},
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for wildcard scope from static token, got %d body=%s", w.Code, w.Body.String())
	}
	if !repo.createCalled {
		t.Fatalf("repo.Create must be called on happy path")
	}
}

// Non-privileged scopes pass through regardless of AuthMethod.
func TestCreate_AllowsNormalScopesFromJWT(t *testing.T) {
	repo := &stubAPIKeyRepo{}
	r := setupCreateRouter(middleware.AuthMethodJWT, New(repo))

	w := doCreate(t, r, map[string]any{
		"name":   "reader-key",
		"scopes": []string{"assets:read", "assets:write"},
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for normal scopes from JWT, got %d body=%s", w.Code, w.Body.String())
	}
	if !repo.createCalled {
		t.Fatalf("repo.Create must be called on happy path")
	}
}

// Empty scope entry must be rejected before it reaches the repository.
func TestCreate_RejectsEmptyScopeEntry(t *testing.T) {
	repo := &stubAPIKeyRepo{}
	r := setupCreateRouter(middleware.AuthMethodJWT, New(repo))

	w := doCreate(t, r, map[string]any{
		"name":   "bad",
		"scopes": []string{"assets:read", "   "},
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for whitespace-only scope entry, got %d", w.Code)
	}
	if repo.createCalled {
		t.Fatalf("repo.Create must NOT be called when scope validation fails")
	}
}
