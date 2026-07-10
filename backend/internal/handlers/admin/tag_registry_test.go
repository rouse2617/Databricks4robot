package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

type inMemoryTagRegistryRepo struct {
	mu      sync.Mutex
	entries map[string]*models.TagRegistryEntry
}

func newInMemoryTagRegistryRepo() *inMemoryTagRegistryRepo {
	return &inMemoryTagRegistryRepo{entries: map[string]*models.TagRegistryEntry{}}
}

func (r *inMemoryTagRegistryRepo) Count(context.Context) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.entries), nil
}

func (r *inMemoryTagRegistryRepo) List(context.Context) ([]*models.TagRegistryEntry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*models.TagRegistryEntry, 0, len(r.entries))
	for _, e := range r.entries {
		cp := *e
		out = append(out, &cp)
	}
	return out, nil
}

func (r *inMemoryTagRegistryRepo) Get(_ context.Context, key string) (*models.TagRegistryEntry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.entries[key]
	if !ok {
		return nil, nil
	}
	cp := *e
	return &cp, nil
}

func (r *inMemoryTagRegistryRepo) Create(_ context.Context, e *models.TagRegistryEntry) (*models.TagRegistryEntry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.entries[e.Key]; ok {
		return nil, repository.ErrDuplicateTagKey
	}
	cp := *e
	r.entries[e.Key] = &cp
	return &cp, nil
}

func (r *inMemoryTagRegistryRepo) Update(_ context.Context, e *models.TagRegistryEntry) (*models.TagRegistryEntry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.entries[e.Key]; !ok {
		return nil, nil
	}
	cp := *e
	r.entries[e.Key] = &cp
	return &cp, nil
}

func (r *inMemoryTagRegistryRepo) Delete(_ context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.entries, key)
	return nil
}

func newEmptyRegistry(t *testing.T) *config.TagRegistry {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "tag_registry.yaml")
	if err := os.WriteFile(p, []byte("tags: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	reg, err := config.LoadTagRegistry(p)
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	return reg
}

func setupTagRegistryRouter(h *TagRegistryHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/tag-registry", h.List)
	r.POST("/tag-registry", h.Create)
	r.PATCH("/tag-registry/:key", h.Update)
	r.DELETE("/tag-registry/:key", h.Delete)
	return r
}

func postJSON(r *gin.Engine, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func TestTagRegistryCreateRefreshesRegistryThenLists(t *testing.T) {
	repo := newInMemoryTagRegistryRepo()
	reg := newEmptyRegistry(t)
	r := setupTagRegistryRouter(NewTagRegistryHandler(repo, reg))

	w := postJSON(r, "/tag-registry", `{"key":"severity","type":"enum","values":["critical","high","low"]}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d body=%s", w.Code, w.Body.String())
	}

	// The in-memory validation map must be refreshed without a restart.
	if err := reg.Validate("severity", "high"); err != nil {
		t.Fatalf("expected severity=high valid after create: %v", err)
	}
	if err := reg.Validate("severity", "fatal"); err == nil {
		t.Fatal("expected severity=fatal rejected after create")
	}

	lw := httptest.NewRecorder()
	r.ServeHTTP(lw, httptest.NewRequest(http.MethodGet, "/tag-registry", nil))
	if lw.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", lw.Code)
	}
	var resp struct {
		Items []*models.TagRegistryEntry `json:"items"`
	}
	if err := json.Unmarshal(lw.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Key != "severity" {
		t.Fatalf("unexpected list: %+v", resp.Items)
	}
}

func TestTagRegistryCreateDuplicateConflict(t *testing.T) {
	repo := newInMemoryTagRegistryRepo()
	r := setupTagRegistryRouter(NewTagRegistryHandler(repo, newEmptyRegistry(t)))
	body := `{"key":"severity","type":"enum","values":["high"]}`
	if w := postJSON(r, "/tag-registry", body); w.Code != http.StatusCreated {
		t.Fatalf("seed create: expected 201, got %d", w.Code)
	}
	if w := postJSON(r, "/tag-registry", body); w.Code != http.StatusConflict {
		t.Fatalf("duplicate: expected 409, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTagRegistryCreateValidation(t *testing.T) {
	repo := newInMemoryTagRegistryRepo()
	r := setupTagRegistryRouter(NewTagRegistryHandler(repo, newEmptyRegistry(t)))
	// enum with no values → 422
	if w := postJSON(r, "/tag-registry", `{"key":"x","type":"enum"}`); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("enum-no-values: expected 422, got %d", w.Code)
	}
	// unknown type → 422
	if w := postJSON(r, "/tag-registry", `{"key":"x","type":"blob"}`); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("bad-type: expected 422, got %d", w.Code)
	}
	// missing key → 422
	if w := postJSON(r, "/tag-registry", `{"type":"string"}`); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("no-key: expected 422, got %d", w.Code)
	}
}

func TestTagRegistryUpdateNotFound(t *testing.T) {
	repo := newInMemoryTagRegistryRepo()
	r := setupTagRegistryRouter(NewTagRegistryHandler(repo, newEmptyRegistry(t)))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/tag-registry/ghost", strings.NewReader(`{"type":"string"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTagRegistryDeleteRefreshesRegistry(t *testing.T) {
	repo := newInMemoryTagRegistryRepo()
	reg := newEmptyRegistry(t)
	r := setupTagRegistryRouter(NewTagRegistryHandler(repo, reg))
	if w := postJSON(r, "/tag-registry", `{"key":"severity","type":"enum","values":["high"]}`); w.Code != http.StatusCreated {
		t.Fatalf("seed: expected 201, got %d", w.Code)
	}
	dw := httptest.NewRecorder()
	r.ServeHTTP(dw, httptest.NewRequest(http.MethodDelete, "/tag-registry/severity", nil))
	if dw.Code != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d", dw.Code)
	}
	// After delete, severity is unregistered again → open-vocabulary: any value accepted.
	if err := reg.Validate("severity", "anything"); err != nil {
		t.Fatalf("expected severity open-vocabulary after delete, got: %v", err)
	}
}
