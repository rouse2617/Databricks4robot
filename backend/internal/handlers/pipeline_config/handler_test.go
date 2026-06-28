package pipeline_config

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/middleware"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	uc "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline_config"
)

type mockConfigRepo struct {
	byID map[string]*models.PipelineConfig
}

func (m *mockConfigRepo) Create(_ context.Context, config *models.PipelineConfig, version *models.PipelineConfigVersion) error {
	if m.byID == nil {
		m.byID = map[string]*models.PipelineConfig{}
	}
	now := time.Now().UTC()
	config.CreatedAt = now
	config.UpdatedAt = now
	version.ConfigID = config.ID
	version.Version = config.CurrentVersion
	version.CreatedAt = now
	copied := *config
	copied.VersionCount = 1
	copied.Versions = []models.PipelineConfigVersion{versionSummary(*version)}
	m.byID[config.ID] = &copied
	return nil
}

func (m *mockConfigRepo) FindAll(_ context.Context, filter *repository.PipelineConfigFilter) ([]models.PipelineConfig, error) {
	out := []models.PipelineConfig{}
	for _, cfg := range m.byID {
		if filter != nil {
			if filter.Owner != "" && cfg.Owner != filter.Owner {
				continue
			}
			if filter.Scope != "" && cfg.Scope != filter.Scope {
				continue
			}
			if filter.Lifecycle != "" && cfg.Lifecycle != filter.Lifecycle {
				continue
			}
			if filter.Query != "" {
				q := strings.ToLower(filter.Query)
				haystack := strings.ToLower(cfg.Name + " " + cfg.Description + " " + strings.Join(cfg.Tags, " "))
				if !strings.Contains(haystack, q) {
					continue
				}
			}
		}
		copied := *cfg
		copied.Versions = nil
		out = append(out, copied)
	}
	return out, nil
}

func (m *mockConfigRepo) FindByID(_ context.Context, id string) (*models.PipelineConfig, error) {
	if cfg, ok := m.byID[id]; ok {
		copied := *cfg
		return &copied, nil
	}
	return nil, nil
}

func (m *mockConfigRepo) UpdateMetadata(_ context.Context, config *models.PipelineConfig) error {
	cfg, ok := m.byID[config.ID]
	if !ok {
		return repository.ErrPipelineConfigNotFound
	}
	cfg.Name = config.Name
	cfg.Description = config.Description
	cfg.Tags = config.Tags
	cfg.FileType = config.FileType
	cfg.Lifecycle = config.Lifecycle
	cfg.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *mockConfigRepo) CreateVersion(_ context.Context, configID string, version *models.PipelineConfigVersion) error {
	cfg, ok := m.byID[configID]
	if !ok {
		return repository.ErrPipelineConfigNotFound
	}
	version.ConfigID = configID
	version.Version = len(cfg.Versions) + 1
	version.CreatedAt = time.Now().UTC()
	cfg.CurrentVersion = version.Version
	cfg.Lifecycle = version.Status
	cfg.VersionCount = len(cfg.Versions) + 1
	cfg.Versions = append([]models.PipelineConfigVersion{versionSummary(*version)}, cfg.Versions...)
	return nil
}

func (m *mockConfigRepo) FindVersion(_ context.Context, configID string, version int) (*models.PipelineConfigVersion, error) {
	cfg, ok := m.byID[configID]
	if !ok {
		return nil, nil
	}
	for _, item := range cfg.Versions {
		if item.Version == version {
			full := item
			if full.Content == "" {
				full.Content = "threshold: 0.9\n"
			}
			return &full, nil
		}
	}
	return nil, nil
}

func (m *mockConfigRepo) FindVersions(_ context.Context, configID string) ([]models.PipelineConfigVersion, error) {
	if cfg, ok := m.byID[configID]; ok {
		return cfg.Versions, nil
	}
	return nil, nil
}

func (m *mockConfigRepo) Deprecate(_ context.Context, id string) error {
	cfg, ok := m.byID[id]
	if !ok {
		return repository.ErrPipelineConfigNotFound
	}
	cfg.Lifecycle = "deprecated"
	for i := range cfg.Versions {
		if cfg.Versions[i].Version == cfg.CurrentVersion {
			cfg.Versions[i].Status = "deprecated"
		}
	}
	return nil
}

func (m *mockConfigRepo) UpdateVersionStatus(_ context.Context, configID string, version int, status string) (*models.PipelineConfigVersion, error) {
	cfg, ok := m.byID[configID]
	if !ok {
		return nil, repository.ErrPipelineConfigNotFound
	}
	for i := range cfg.Versions {
		if cfg.Versions[i].Version == version {
			cfg.Versions[i].Status = status
			updated := cfg.Versions[i]
			return &updated, nil
		}
	}
	return nil, nil
}

func (m *mockConfigRepo) UpdateLifecycle(_ context.Context, configID string, lifecycle string) error {
	cfg, ok := m.byID[configID]
	if !ok {
		return repository.ErrPipelineConfigNotFound
	}
	cfg.Lifecycle = lifecycle
	return nil
}

func (m *mockConfigRepo) UpdateVersionContent(_ context.Context, configID string, version int, content string, summary string) (*models.PipelineConfigVersion, error) {
	cfg, ok := m.byID[configID]
	if !ok {
		return nil, repository.ErrPipelineConfigNotFound
	}
	for i := range cfg.Versions {
		if cfg.Versions[i].Version == version {
			cfg.Versions[i].Content = content
			cfg.Versions[i].Summary = summary
			updated := cfg.Versions[i]
			return &updated, nil
		}
	}
	return nil, nil
}

func versionSummary(version models.PipelineConfigVersion) models.PipelineConfigVersion {
	version.Content = ""
	return version
}

func setupConfigRouter(h *Handler, email, role string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxKeyEmail, email)
		c.Set(middleware.CtxKeyRole, role)
		c.Next()
	})
	r.GET("/api/v1/pipeline-configs", h.List)
	r.POST("/api/v1/pipeline-configs", h.Create)
	r.GET("/api/v1/pipeline-configs/:id", h.Get)
	r.PUT("/api/v1/pipeline-configs/:id", h.Update)
	r.POST("/api/v1/pipeline-configs/:id/versions", h.CreateVersion)
	r.GET("/api/v1/pipeline-configs/:id/versions/:version", h.GetVersion)
	r.POST("/api/v1/pipeline-configs/:id/deprecate", h.Deprecate)
	return r
}

func TestPipelineConfigCreateListAndVersion(t *testing.T) {
	repo := &mockConfigRepo{}
	h := New(uc.New(repo))
	r := setupConfigRouter(h, "alice@example.com", "user")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline-configs", strings.NewReader(`{
		"name":"detector.yaml",
		"description":"Detector thresholds",
		"tags":["vision","prod"],
		"lifecycle":"ready",
		"content":"threshold: 0.8\n",
		"summary":"initial"
	}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var created models.PipelineConfig
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal create: %v", err)
	}
	if created.Owner != "alice@example.com" || created.CurrentVersion != 1 {
		t.Fatalf("unexpected created config: %#v", created)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/pipeline-configs?q=vision", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var list struct {
		Items []models.PipelineConfig `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("expected one owned config, got %d", len(list.Items))
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/pipeline-configs/"+created.ID+"/versions", strings.NewReader(`{"status":"ready","content":"threshold: 0.9\n","summary":"raise threshold"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 version, got %d: %s", w.Code, w.Body.String())
	}
	var version models.PipelineConfigVersion
	if err := json.Unmarshal(w.Body.Bytes(), &version); err != nil {
		t.Fatalf("unmarshal version: %v", err)
	}
	if version.Version != 2 || version.Content != "" {
		t.Fatalf("expected v2 summary without content, got %#v", version)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/pipeline-configs/"+created.ID+"/versions/2", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 version detail, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "threshold") {
		t.Fatalf("expected version content in detail, got %s", w.Body.String())
	}
}

func TestPipelineConfigOwnerIsolation(t *testing.T) {
	repo := &mockConfigRepo{byID: map[string]*models.PipelineConfig{
		"cfg-bob": {
			ID:             "cfg-bob",
			Name:           "private.yaml",
			Owner:          "bob@example.com",
			Scope:          "dev",
			Tags:           []string{},
			FileType:       "yaml",
			Lifecycle:      "ready",
			CurrentVersion: 1,
			Versions:       []models.PipelineConfigVersion{{Version: 1, Status: "ready"}},
		},
	}}
	h := New(uc.New(repo))
	r := setupConfigRouter(h, "alice@example.com", "user")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipeline-configs/cfg-bob", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected cross-owner 404, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/pipeline-configs", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d: %s", w.Code, w.Body.String())
	}
	var list struct {
		Items []models.PipelineConfig `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	if len(list.Items) != 0 {
		t.Fatalf("expected no cross-owner list items, got %#v", list.Items)
	}
}

func TestPipelineConfigInvalidLifecycle(t *testing.T) {
	repo := &mockConfigRepo{}
	h := New(uc.New(repo))
	r := setupConfigRouter(h, "alice@example.com", "user")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline-configs", strings.NewReader(`{"name":"bad.yaml","lifecycle":"active","content":"x: 1\n"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
