package pipeline_component

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	uc "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline_component"
)

// ── Mock Repository ──────────────────────────────────────────────────────────

type mockComponentRepo struct {
	byID     map[string]*models.PipelineComponent
	releases map[string]*models.PipelineComponentRelease
}

func (m *mockComponentRepo) Save(_ context.Context, c *models.PipelineComponent) error {
	if m.byID == nil {
		m.byID = make(map[string]*models.PipelineComponent)
	}
	m.byID[c.ID] = c
	return nil
}

func (m *mockComponentRepo) FindAll(_ context.Context, filter *repository.ComponentFilter) ([]models.PipelineComponent, error) {
	out := make([]models.PipelineComponent, 0)
	for _, c := range m.byID {
		if filter != nil {
			if filter.Query != "" && !strings.Contains(strings.ToLower(c.Name), strings.ToLower(filter.Query)) {
				continue
			}
			if filter.Source != "" && c.Source != filter.Source {
				continue
			}
		}
		out = append(out, *c)
	}
	return out, nil
}

func (m *mockComponentRepo) FindByID(_ context.Context, id string) (*models.PipelineComponent, error) {
	return m.byID[id], nil
}

func (m *mockComponentRepo) Update(_ context.Context, c *models.PipelineComponent) error {
	if m.byID == nil {
		return nil
	}
	existing, ok := m.byID[c.ID]
	if !ok {
		return nil
	}
	existing.Name = c.Name
	existing.Description = c.Description
	existing.Image = c.Image
	existing.Tag = c.Tag
	existing.Source = c.Source
	existing.Type = c.Type
	existing.Command = c.Command
	existing.Args = c.Args
	existing.Env = c.Env
	existing.Resources = c.Resources
	existing.EnvVars = c.EnvVars
	existing.InputPorts = c.InputPorts
	existing.OutputPorts = c.OutputPorts
	existing.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *mockComponentRepo) Delete(_ context.Context, id string) error {
	delete(m.byID, id)
	return nil
}

func (m *mockComponentRepo) UpsertRelease(_ context.Context, release *models.PipelineComponentRelease) error {
	if m.releases == nil {
		m.releases = make(map[string]*models.PipelineComponentRelease)
	}
	for id, existing := range m.releases {
		if existing.ComponentID == release.ComponentID && existing.ReleaseLabel == release.ReleaseLabel {
			release.ID = existing.ID
			release.CreatedAt = existing.CreatedAt
			copied := *release
			m.releases[id] = &copied
			return nil
		}
	}
	copied := *release
	m.releases[release.ID] = &copied
	return nil
}

func (m *mockComponentRepo) FindReleases(_ context.Context, filter *repository.ComponentReleaseFilter) ([]models.PipelineComponentRelease, error) {
	out := make([]models.PipelineComponentRelease, 0)
	for _, release := range m.releases {
		if filter != nil {
			if filter.Query != "" {
				q := strings.ToLower(filter.Query)
				haystack := strings.ToLower(release.ComponentID + " " + release.TaskName + " " + release.DisplayName + " " + release.ReleaseLabel)
				if !strings.Contains(haystack, q) {
					continue
				}
			}
			if filter.ComponentID != "" && release.ComponentID != filter.ComponentID {
				continue
			}
			if filter.TaskName != "" && release.TaskName != filter.TaskName {
				continue
			}
			if filter.Status != "" && release.Status != filter.Status {
				continue
			}
			if filter.Channel != "" && release.Channel != filter.Channel {
				continue
			}
			if filter.Selectable != nil && release.Selectable != *filter.Selectable {
				continue
			}
		}
		out = append(out, *release)
	}
	return out, nil
}

func (m *mockComponentRepo) FindReleaseByID(_ context.Context, id string) (*models.PipelineComponentRelease, error) {
	if m.releases == nil {
		return nil, nil
	}
	return m.releases[id], nil
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func makeComponent(id, name, source string) *models.PipelineComponent {
	return &models.PipelineComponent{
		ID:          id,
		Name:        name,
		Type:        "container",
		Description: "test component",
		Image:       "docker.io/test/" + name,
		Tag:         "latest",
		Source:      source,
		InputPorts:  []models.PortDef{},
		OutputPorts: []models.PortDef{},
		CreatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func setupRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/components", h.CreateComponent)
	r.GET("/api/v1/components", h.ListComponents)
	r.GET("/api/v1/components/:id", h.GetComponent)
	r.PUT("/api/v1/components/:id", h.UpdateComponent)
	r.DELETE("/api/v1/components/:id", h.DeleteComponent)
	r.POST("/api/v1/pipeline-components", h.CreateComponent)
	r.GET("/api/v1/pipeline-components", h.ListComponents)
	r.GET("/api/v1/pipeline-components/:id", h.GetComponent)
	r.PUT("/api/v1/pipeline-components/:id", h.UpdateComponent)
	r.DELETE("/api/v1/pipeline-components/:id", h.DeleteComponent)
	r.GET("/api/v1/pipeline-component-releases", h.ListReleases)
	r.POST("/api/v1/pipeline-component-releases/sync", h.SyncReleases)
	r.GET("/api/v1/pipeline-component-releases/:id", h.GetRelease)
	return r
}

// ── Tests ────────────────────────────────────────────────────────────────────

func TestCreateComponent_Success(t *testing.T) {
	repo := &mockComponentRepo{}
	usecase := uc.New(repo)
	h := New(usecase)
	r := setupRouter(h)

	body := `{"name":"my-component","type":"container","description":"my desc","image":"docker.io/test/my-component","tag":"v1","command":["python"],"args":["main.py"],"env":{"MODE":"test"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline-components", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.PipelineComponent
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Name != "my-component" {
		t.Errorf("expected name 'my-component', got %q", resp.Name)
	}
	if resp.ID == "" {
		t.Error("expected non-empty ID")
	}
	if resp.Type != "container" {
		t.Errorf("expected type container, got %q", resp.Type)
	}
	if len(resp.Command) != 1 || resp.Command[0] != "python" {
		t.Fatalf("expected command to round trip, got %#v", resp.Command)
	}
}

func TestCreateComponent_InvalidBody(t *testing.T) {
	repo := &mockComponentRepo{}
	usecase := uc.New(repo)
	h := New(usecase)
	r := setupRouter(h)

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/components", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid body, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateComponent_MissingRequiredFields(t *testing.T) {
	repo := &mockComponentRepo{}
	usecase := uc.New(repo)
	h := New(usecase)
	r := setupRouter(h)

	body := `{"name":"missing-image","type":"container"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline-components", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing image, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListComponents_Empty(t *testing.T) {
	repo := &mockComponentRepo{}
	usecase := uc.New(repo)
	h := New(usecase)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/components", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	items, ok := resp["items"].([]interface{})
	if !ok {
		t.Fatal("expected items array")
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}

func TestListComponents_WithItems(t *testing.T) {
	repo := &mockComponentRepo{}
	// Pre-populate by saving through usecase
	usecase := uc.New(repo)
	_, _ = usecase.Create(context.Background(), &models.PipelineComponent{Name: "comp-a", Type: "container", Image: "img/a"})
	_, _ = usecase.Create(context.Background(), &models.PipelineComponent{Name: "comp-b", Type: "container", Image: "img/b"})
	h := New(usecase)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/components", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	items := resp["items"].([]interface{})
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestListComponents_WithQueryAndSource(t *testing.T) {
	repo := &mockComponentRepo{}
	usecase := uc.New(repo)
	// Pre-populate with components of different sources
	_, _ = usecase.Create(context.Background(), &models.PipelineComponent{Name: "alpha", Type: "container", Image: "img/a", Source: "system"})
	_, _ = usecase.Create(context.Background(), &models.PipelineComponent{Name: "beta", Type: "container", Image: "img/b", Source: "custom"})
	_, _ = usecase.Create(context.Background(), &models.PipelineComponent{Name: "gamma", Type: "container", Image: "img/c", Source: "system"})
	h := New(usecase)
	r := setupRouter(h)

	// Filter by source=system
	req := httptest.NewRequest(http.MethodGet, "/api/v1/components?source=system", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	items := resp["items"].([]interface{})
	if len(items) != 2 {
		t.Fatalf("expected 2 system components, got %d", len(items))
	}
}

func TestGetComponent_Success(t *testing.T) {
	repo := &mockComponentRepo{}
	usecase := uc.New(repo)
	created, _ := usecase.Create(context.Background(), &models.PipelineComponent{Name: "my-component", Type: "container", Image: "img/c", Description: "test"})
	h := New(usecase)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/components/"+created.ID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.PipelineComponent
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.ID != created.ID {
		t.Errorf("expected id %q, got %q", created.ID, resp.ID)
	}
	if resp.Name != "my-component" {
		t.Errorf("expected name 'my-component', got %q", resp.Name)
	}
}

func TestGetComponent_NotFound(t *testing.T) {
	repo := &mockComponentRepo{}
	usecase := uc.New(repo)
	h := New(usecase)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/components/non-existent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetComponent_EmptyID(t *testing.T) {
	repo := &mockComponentRepo{}
	usecase := uc.New(repo)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := New(usecase)
	r.GET("/api/v1/components/:id", h.GetComponent)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/components/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest {
		t.Fatalf("expected 404 or 400 for empty id, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateComponent_Success(t *testing.T) {
	repo := &mockComponentRepo{}
	usecase := uc.New(repo)
	created, _ := usecase.Create(context.Background(), &models.PipelineComponent{Name: "old-name", Type: "container", Image: "img/old"})
	h := New(usecase)
	r := setupRouter(h)

	body := `{"name":"updated-component","type":"script","description":"updated","image":"docker.io/test/updated","tag":"v2","command":["bash"],"args":["run.sh"]}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/pipeline-components/"+created.ID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.PipelineComponent
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.ID != created.ID {
		t.Errorf("expected id %q, got %q", created.ID, resp.ID)
	}
	if resp.Name != "updated-component" {
		t.Errorf("expected name 'updated-component', got %q", resp.Name)
	}
	if resp.Type != "script" {
		t.Errorf("expected type script, got %q", resp.Type)
	}
}

func TestUpdateComponent_InvalidBody(t *testing.T) {
	repo := &mockComponentRepo{}
	usecase := uc.New(repo)
	h := New(usecase)
	r := setupRouter(h)

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/components/comp-1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid body, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateComponent_EmptyID(t *testing.T) {
	repo := &mockComponentRepo{}
	usecase := uc.New(repo)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := New(usecase)
	r.PUT("/api/v1/components/:id", h.UpdateComponent)

	body := `{"name":"test"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/components/%20%20", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty id, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteComponent_Success(t *testing.T) {
	repo := &mockComponentRepo{}
	usecase := uc.New(repo)
	created, _ := usecase.Create(context.Background(), &models.PipelineComponent{Name: "to-delete", Type: "container", Image: "img/del"})
	h := New(usecase)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/components/"+created.ID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
	// Verify it's gone
	if _, err := usecase.Get(context.Background(), created.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := usecase.Get(context.Background(), created.ID)
	if got != nil {
		t.Fatal("expected component to be deleted")
	}
}

func TestDeleteComponent_EmptyID(t *testing.T) {
	repo := &mockComponentRepo{}
	usecase := uc.New(repo)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := New(usecase)
	r.DELETE("/api/v1/components/:id", h.DeleteComponent)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/components/%20%20", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty id, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSyncReleases_ValidDigestPinnedRelease(t *testing.T) {
	repo := &mockComponentRepo{}
	usecase := uc.New(repo)
	h := New(usecase)
	r := setupRouter(h)

	body := `{"items":[{
		"componentId":"hand-detect-yolov26m",
		"taskName":"hand-detect-yolov26m",
		"taskPath":"tasks/hand_detect_yolov26m",
		"releaseLabel":"main-abcdef1",
		"runtimeImage":"us-central1-docker.pkg.dev/proj/video-proc-images/hand-detect-yolov26m@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"runtimeSnapshot":{
			"command":["python","src/main.py"],
			"inputPorts":[{"name":"input","type":"asset"}],
			"outputPorts":[{"name":"output","type":"asset"}],
			"resources":{"cpu":"14000m","memory":"55Gi","gpu":"1"}
		}
	}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline-component-releases/sync", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Items []models.PipelineComponentRelease `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected one release, got %d", len(resp.Items))
	}
	got := resp.Items[0]
	if got.ID == "" {
		t.Fatal("expected generated stable release id")
	}
	if got.Channel != "candidate" {
		t.Fatalf("expected candidate channel, got %q", got.Channel)
	}
	if !got.Selectable || got.ValidationStatus != "passed" {
		t.Fatalf("expected selectable passed release, got selectable=%v validation=%q errors=%v", got.Selectable, got.ValidationStatus, got.ValidationErrors)
	}
	if got.Owner != "platform" {
		t.Fatalf("expected platform owner default, got %q", got.Owner)
	}
}

func TestSyncReleases_MissingDigestIsNotSelectable(t *testing.T) {
	repo := &mockComponentRepo{}
	usecase := uc.New(repo)
	h := New(usecase)
	r := setupRouter(h)

	body := `{"items":[{
		"componentId":"hand-detect-yolov26m",
		"taskName":"hand-detect-yolov26m",
		"releaseLabel":"pr-128-abcdef1",
		"runtimeImage":"us-central1-docker.pkg.dev/proj/video-proc-images/hand-detect-yolov26m:abcdef1",
		"runtimeSnapshot":{
			"command":["python","src/main.py"],
			"resources":{"cpu":"14000m","memory":"55Gi"}
		}
	}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline-component-releases/sync", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Items []models.PipelineComponentRelease `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected one release, got %d", len(resp.Items))
	}
	got := resp.Items[0]
	if got.Selectable {
		t.Fatal("expected release without digest to be unselectable")
	}
	if got.ValidationStatus != "failed" {
		t.Fatalf("expected failed validation, got %q", got.ValidationStatus)
	}
	if len(got.ValidationErrors) == 0 {
		t.Fatal("expected validation errors")
	}
}

func TestListAndGetReleases(t *testing.T) {
	repo := &mockComponentRepo{}
	usecase := uc.New(repo)
	_, err := usecase.SyncReleases(context.Background(), []models.PipelineComponentRelease{{
		ComponentID:  "head-track-stereo",
		TaskName:     "head-track-stereo",
		ReleaseLabel: "v0.4.0",
		RuntimeImage: "repo/head-track-stereo@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		RuntimeSnapshot: models.ComponentReleaseRuntimeSnapshot{
			Command:     []string{"python", "src/main.py"},
			InputPorts:  []models.PortDef{{Name: "input", Type: "asset"}},
			OutputPorts: []models.PortDef{{Name: "output", Type: "asset"}},
			Resources:   map[string]interface{}{"cpu": "4000m", "memory": "16Gi"},
		},
	}})
	if err != nil {
		t.Fatalf("sync release: %v", err)
	}
	h := New(usecase)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipeline-component-releases?selectable=true&q=head", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var listResp struct {
		Items []models.PipelineComponentRelease `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	if len(listResp.Items) != 1 {
		t.Fatalf("expected one release, got %d", len(listResp.Items))
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/pipeline-component-releases/"+listResp.Items[0].ID, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var getResp models.PipelineComponentRelease
	if err := json.Unmarshal(w.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("unmarshal get: %v", err)
	}
	if getResp.ReleaseLabel != "v0.4.0" {
		t.Fatalf("expected release label v0.4.0, got %q", getResp.ReleaseLabel)
	}
}
