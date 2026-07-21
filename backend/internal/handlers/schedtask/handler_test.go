package schedtask

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/schedtask"
)

// ─── stub repo ─────────────────────────────────────────────────

type stubRepo struct {
	mu           sync.Mutex
	rules        map[string]schedtask.Rule
	listErr      error
	runNowCalled []string
	setEnabled   []struct {
		id string
		on bool
	}
}

func newStubRepo(rules ...schedtask.Rule) *stubRepo {
	s := &stubRepo{rules: make(map[string]schedtask.Rule)}
	for _, r := range rules {
		s.rules[r.ID] = r
	}
	return s
}

func (s *stubRepo) List(_ context.Context) ([]schedtask.Rule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listErr != nil {
		return nil, s.listErr
	}
	out := make([]schedtask.Rule, 0, len(s.rules))
	for _, r := range s.rules {
		out = append(out, r)
	}
	return out, nil
}
func (s *stubRepo) Get(_ context.Context, id string) (*schedtask.Rule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.rules[id]; ok {
		return &r, nil
	}
	return nil, nil
}
func (s *stubRepo) Create(_ context.Context, r *schedtask.Rule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, dup := s.rules[r.ID]; dup {
		return errors.New("dup")
	}
	s.rules[r.ID] = *r
	return nil
}
func (s *stubRepo) Update(_ context.Context, r *schedtask.Rule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rules[r.ID]; !ok {
		return errors.New("not found")
	}
	s.rules[r.ID] = *r
	return nil
}
func (s *stubRepo) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.rules, id)
	return nil
}
func (s *stubRepo) SetEnabled(_ context.Context, id string, on bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.rules[id]
	if !ok {
		return errors.New("not found")
	}
	r.Enabled = on
	s.rules[id] = r
	s.setEnabled = append(s.setEnabled, struct {
		id string
		on bool
	}{id, on})
	return nil
}
func (s *stubRepo) RequestRunNow(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.rules[id]
	if !ok {
		return errors.New("not found")
	}
	now := time.Now().UTC()
	r.RunNowRequestedAt = &now
	s.rules[id] = r
	s.runNowCalled = append(s.runNowCalled, id)
	return nil
}

// ─── router setup ─────────────────────────────────────────────

func setupRouter(repo Repo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := New(repo)
	g := r.Group("/api/v1/scheduled-tasks")
	{
		g.GET("", h.List)
		g.GET("/:id", h.Get)
		g.POST("", h.Create)
		g.PUT("/:id", h.Update)
		g.DELETE("/:id", h.Delete)
		g.POST("/:id/pause", h.Pause)
		g.POST("/:id/resume", h.Resume)
		g.POST("/:id/run-now", h.RunNow)
	}
	return r
}

func do(t *testing.T, r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf *bytes.Buffer
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		buf = bytes.NewBuffer(b)
	} else {
		buf = &bytes.Buffer{}
	}
	req := httptest.NewRequest(method, path, buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ─── tests ────────────────────────────────────────────────────

func TestList_FiltersAndPaginates(t *testing.T) {
	repo := newStubRepo(
		schedtask.Rule{ID: "a", Name: "alpha", Enabled: true, SourceType: "rest"},
		schedtask.Rule{ID: "b", Name: "beta", Enabled: false, SourceType: "rest"},
		schedtask.Rule{ID: "c", Name: "gamma-grace", Enabled: true, SourceType: "rest"},
	)
	r := setupRouter(repo)

	// enabled=true excludes beta.
	w := do(t, r, "GET", "/api/v1/scheduled-tasks?enabled=true", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Items []schedtask.Rule `json:"items"`
		Total int              `json:"total"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 2 {
		t.Fatalf("enabled=true total=%d want 2 (got %+v)", resp.Total, resp.Items)
	}
	// q=grace matches only gamma-grace.
	w = do(t, r, "GET", "/api/v1/scheduled-tasks?q=grace", nil)
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 1 || resp.Items[0].ID != "c" {
		t.Fatalf("q=grace wrong: %+v", resp)
	}
	// pageSize=1 gives 1 item back, total still 3.
	w = do(t, r, "GET", "/api/v1/scheduled-tasks?pageSize=1&page=2", nil)
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 3 || len(resp.Items) != 1 {
		t.Fatalf("paging wrong: %+v", resp)
	}
}

func TestCreate_ValidatesAndAssignsID(t *testing.T) {
	repo := newStubRepo()
	r := setupRouter(repo)

	// Missing required fields → 400.
	w := do(t, r, "POST", "/api/v1/scheduled-tasks", map[string]any{"name": "x"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
	// Bad triggerMode → 400.
	w = do(t, r, "POST", "/api/v1/scheduled-tasks", map[string]any{
		"name": "n", "templateId": "tpl", "targetId": "tgt", "sourceType": "rest", "triggerMode": "cron",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("bad triggerMode expected 400, got %d", w.Code)
	}
	// Happy path.
	w = do(t, r, "POST", "/api/v1/scheduled-tasks", map[string]any{
		"name": "grace-1h", "templateId": "tpl", "targetId": "tgt",
		"sourceType": "rest", "triggerMode": "incremental",
		"triggerConfig": map[string]any{"intervalSeconds": 3600},
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create expected 201, got %d body=%s", w.Code, w.Body.String())
	}
	var got schedtask.Rule
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.ID == "" || !strings.HasPrefix(got.ID, "sched_") {
		t.Fatalf("expected sched_ prefixed id, got %q", got.ID)
	}
	if !got.Enabled {
		t.Fatal("default enabled should be true")
	}
}

func TestUpdate_PreservesCreateMeta(t *testing.T) {
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := newStubRepo(schedtask.Rule{
		ID: "sched_1", Name: "old", Enabled: true,
		TemplateID: "tpl", TargetID: "tgt",
		SourceType: "rest", TriggerMode: "incremental",
		CreatedBy: "alice@example.com", CreatedAt: created,
	})
	r := setupRouter(repo)
	w := do(t, r, "PUT", "/api/v1/scheduled-tasks/sched_1", map[string]any{
		"name": "new", "templateId": "tpl", "targetId": "tgt",
		"sourceType": "rest", "triggerMode": "rolling",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("update code=%d body=%s", w.Code, w.Body.String())
	}
	var got schedtask.Rule
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.Name != "new" || got.TriggerMode != "rolling" {
		t.Fatalf("update didn't apply: %+v", got)
	}
	if got.CreatedBy != "alice@example.com" || !got.CreatedAt.Equal(created) {
		t.Fatalf("create meta not preserved: createdBy=%q createdAt=%v", got.CreatedBy, got.CreatedAt)
	}
}

// TestUpdate_DoesNotReenablePaused covers the gemini-review point: an Update
// payload without `enabled` must not silently re-enable a paused rule.
func TestUpdate_DoesNotReenablePaused(t *testing.T) {
	repo := newStubRepo(schedtask.Rule{
		ID: "sched_paused", Name: "p", Enabled: false,
		TemplateID: "tpl", TargetID: "tgt",
		SourceType: "rest", TriggerMode: "incremental",
	})
	r := setupRouter(repo)
	// Payload omits `enabled` entirely — must keep enabled=false.
	w := do(t, r, "PUT", "/api/v1/scheduled-tasks/sched_paused", map[string]any{
		"name": "still-paused", "templateId": "tpl", "targetId": "tgt",
		"sourceType": "rest", "triggerMode": "incremental",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("update code=%d body=%s", w.Code, w.Body.String())
	}
	if repo.rules["sched_paused"].Enabled {
		t.Fatal("PUT without enabled must not resume a paused rule")
	}
	// Explicit enabled:true still wins.
	w = do(t, r, "PUT", "/api/v1/scheduled-tasks/sched_paused", map[string]any{
		"name": "now-on", "templateId": "tpl", "targetId": "tgt",
		"sourceType": "rest", "triggerMode": "incremental", "enabled": true,
	})
	if w.Code != http.StatusOK || !repo.rules["sched_paused"].Enabled {
		t.Fatalf("explicit enabled:true should resume; code=%d enabled=%v", w.Code, repo.rules["sched_paused"].Enabled)
	}
}

func TestPauseResume_TogglesEnabled(t *testing.T) {
	repo := newStubRepo(schedtask.Rule{ID: "sched_1", Enabled: true, SourceType: "rest", TriggerMode: "incremental"})
	r := setupRouter(repo)
	w := do(t, r, "POST", "/api/v1/scheduled-tasks/sched_1/pause", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("pause code=%d", w.Code)
	}
	if repo.rules["sched_1"].Enabled {
		t.Fatal("pause should set enabled=false")
	}
	w = do(t, r, "POST", "/api/v1/scheduled-tasks/sched_1/resume", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("resume code=%d", w.Code)
	}
	if !repo.rules["sched_1"].Enabled {
		t.Fatal("resume should set enabled=true")
	}
}

func TestPause_NotFound(t *testing.T) {
	repo := newStubRepo()
	r := setupRouter(repo)
	w := do(t, r, "POST", "/api/v1/scheduled-tasks/nope/pause", nil)
	// SetEnabled returns error, handler surfaces as 500 (matches dispatcher style).
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on unknown id, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestRunNow_RequestsMarkerAndReturns202(t *testing.T) {
	repo := newStubRepo(schedtask.Rule{ID: "sched_1", Enabled: true, SourceType: "rest", TriggerMode: "incremental"})
	r := setupRouter(repo)
	// No body allowed.
	w := do(t, r, "POST", "/api/v1/scheduled-tasks/sched_1/run-now", nil)
	if w.Code != http.StatusAccepted {
		t.Fatalf("run-now expected 202, got %d body=%s", w.Code, w.Body.String())
	}
	if len(repo.runNowCalled) != 1 || repo.runNowCalled[0] != "sched_1" {
		t.Fatalf("run-now not persisted: %+v", repo.runNowCalled)
	}
	// Body with unsupported mode → 400.
	w = do(t, r, "POST", "/api/v1/scheduled-tasks/sched_1/run-now", map[string]any{"mode": "cron"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("unsupported mode expected 400, got %d", w.Code)
	}
}

func TestDelete_204(t *testing.T) {
	repo := newStubRepo(schedtask.Rule{ID: "sched_1"})
	r := setupRouter(repo)
	w := do(t, r, "DELETE", "/api/v1/scheduled-tasks/sched_1", nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete expected 204, got %d", w.Code)
	}
	if _, ok := repo.rules["sched_1"]; ok {
		t.Fatal("delete didn't remove row")
	}
}

// Quick guard against JSON shape drift on List (frontend depends on
// {items, total, page, pageSize}).
func TestList_ResponseShape(t *testing.T) {
	repo := newStubRepo(schedtask.Rule{ID: "a", Name: "n", SourceType: "rest"})
	r := setupRouter(repo)
	w := do(t, r, "GET", "/api/v1/scheduled-tasks", nil)
	if w.Code != http.StatusOK {
		t.Fatal("list not 200")
	}
	var raw map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &raw)
	for _, k := range []string{"items", "total", "page", "pageSize"} {
		if _, ok := raw[k]; !ok {
			t.Errorf("missing key %q in response: %s", k, w.Body.String())
		}
	}
}

// silence unused imports on trimmed tests
var _ = fmt.Sprintf
