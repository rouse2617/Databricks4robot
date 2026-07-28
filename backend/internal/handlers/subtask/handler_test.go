package subtask

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	st "github.com/CyberOrigin2077/cyber-databrew/internal/subtask"
)

// These tests pin the subscription-tasks CRUD HTTP contract (CYB-3778): request
// validation, default-filling, the enabled-merge on update, the paging clamp,
// and the status codes the handler returns. The handler had no Go unit tests —
// only an end-to-end dev smoke exercised it. See CYB-4266.

func init() { gin.SetMode(gin.TestMode) }

func iptr(i int) *int   { return &i }
func bptr(b bool) *bool { return &b }

func validReq() taskRequest {
	return taskRequest{
		Name:             "n",
		ProjectID:        "p",
		SubscriptionID:   "s",
		PipelineBindings: []pipelineBindingRequest{{TemplateID: "t", TargetID: "g"}},
	}
}

// ---- fake Repo ----

type fakeRepo struct {
	tasks     []st.Task
	getResult *st.Task
	getErr    error
	listErr   error
	createErr error
	updateErr error
	deleteErr error
	setEnErr  error

	created   *st.Task
	updated   *st.Task
	deletedID string
	setEnID   string
	setEnVal  bool
}

func (f *fakeRepo) List(context.Context) ([]st.Task, error)       { return f.tasks, f.listErr }
func (f *fakeRepo) Get(context.Context, string) (*st.Task, error) { return f.getResult, f.getErr }
func (f *fakeRepo) Create(_ context.Context, t *st.Task) error    { f.created = t; return f.createErr }
func (f *fakeRepo) Update(_ context.Context, t *st.Task) error    { f.updated = t; return f.updateErr }
func (f *fakeRepo) Delete(_ context.Context, id string) error     { f.deletedID = id; return f.deleteErr }
func (f *fakeRepo) SetEnabled(_ context.Context, id string, enabled bool) error {
	f.setEnID, f.setEnVal = id, enabled
	return f.setEnErr
}

func newRouter(h *Handler) *gin.Engine {
	r := gin.New()
	r.GET("/subscription-tasks", h.List)
	r.POST("/subscription-tasks", h.Create)
	r.GET("/subscription-tasks/:id", h.Get)
	r.PUT("/subscription-tasks/:id", h.Update)
	r.DELETE("/subscription-tasks/:id", h.Delete)
	r.POST("/subscription-tasks/:id/pause", h.Pause)
	r.POST("/subscription-tasks/:id/resume", h.Resume)
	return r
}

func do(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ---- pure-function tests ----

func TestValidateRequest(t *testing.T) {
	noName := validReq()
	noName.Name = "  "
	noProject := validReq()
	noProject.ProjectID = ""
	noSub := validReq()
	noSub.SubscriptionID = ""
	noBindings := validReq()
	noBindings.PipelineBindings = nil
	noTemplate := validReq()
	noTemplate.PipelineBindings = []pipelineBindingRequest{{TemplateID: " ", TargetID: "g"}}
	noTarget := validReq()
	noTarget.PipelineBindings = []pipelineBindingRequest{{TemplateID: "t", TargetID: ""}}
	badInterval := validReq()
	badInterval.PullIntervalSec = iptr(0)
	maxZero := validReq()
	maxZero.MaxMessagesPerPull = iptr(0)
	maxHuge := validReq()
	maxHuge.MaxMessagesPerPull = iptr(10001)
	maxEdge := validReq()
	maxEdge.MaxMessagesPerPull = iptr(10000)

	tests := []struct {
		name      string
		req       taskRequest
		forCreate bool
		wantErr   bool
	}{
		{"valid create", validReq(), true, false},
		{"valid update", validReq(), false, false},
		{"empty name allowed on update", noName, false, false},
		{"empty name rejected on create", noName, true, true},
		{"missing projectId", noProject, true, true},
		{"missing subscriptionId", noSub, true, true},
		{"no bindings", noBindings, true, true},
		{"binding missing templateId", noTemplate, true, true},
		{"binding missing targetId", noTarget, true, true},
		{"non-positive pull interval", badInterval, true, true},
		{"maxMessages zero", maxZero, true, true},
		{"maxMessages over 10000", maxHuge, true, true},
		{"maxMessages at 10000 is ok", maxEdge, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRequest(tt.req, tt.forCreate)
			if tt.wantErr != (err != nil) {
				t.Fatalf("validateRequest err=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestRequestToTask_DefaultsAndTrim(t *testing.T) {
	req := taskRequest{
		Name:           "  n  ",
		ProjectID:      " p ",
		SubscriptionID: " s ",
		PipelineBindings: []pipelineBindingRequest{
			{TemplateID: " t ", TargetID: " g ", TemplateVersion: iptr(2)},
		},
	}
	got := requestToTask(req, "id-1")
	if got.ID != "id-1" {
		t.Errorf("ID = %q, want id-1", got.ID)
	}
	if got.Name != "n" || got.ProjectID != "p" || got.SubscriptionID != "s" {
		t.Errorf("fields not trimmed: %+v", got)
	}
	if !got.Enabled {
		t.Errorf("Enabled default should be true")
	}
	if got.PullIntervalSec != 10 {
		t.Errorf("PullIntervalSec default = %d, want 10", got.PullIntervalSec)
	}
	if got.MaxMessagesPerPull != 1000 {
		t.Errorf("MaxMessagesPerPull default = %d, want 1000", got.MaxMessagesPerPull)
	}
	if len(got.PipelineBindings) != 1 || got.PipelineBindings[0].TemplateID != "t" || got.PipelineBindings[0].TargetID != "g" {
		t.Errorf("binding not trimmed: %+v", got.PipelineBindings)
	}
	if got.PipelineBindings[0].TemplateVersion == nil || *got.PipelineBindings[0].TemplateVersion != 2 {
		t.Errorf("templateVersion not carried through: %+v", got.PipelineBindings[0])
	}
}

func TestRequestToTask_HonorsExplicitValues(t *testing.T) {
	req := validReq()
	req.Enabled = bptr(false)
	req.PullIntervalSec = iptr(30)
	req.MaxMessagesPerPull = iptr(5)
	got := requestToTask(req, "")
	if got.Enabled {
		t.Errorf("Enabled should be false")
	}
	if got.PullIntervalSec != 30 || got.MaxMessagesPerPull != 5 {
		t.Errorf("explicit values not honored: %+v", got)
	}
}

func TestParsePaging(t *testing.T) {
	ctx := func(qs string) *gin.Context {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/?"+qs, nil)
		return c
	}
	tests := []struct {
		name             string
		qs               string
		wantPage, wantSz int
	}{
		{"defaults", "", 1, 50},
		{"explicit", "page=3&pageSize=10", 3, 10},
		{"pageSize clamped to 500", "pageSize=999", 1, 500},
		{"invalid page falls back to 1", "page=abc", 1, 50},
		{"zero page falls back to 1", "page=0", 1, 50},
		{"invalid pageSize falls back to 50", "pageSize=xyz", 1, 50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, size := parsePaging(ctx(tt.qs))
			if page != tt.wantPage || size != tt.wantSz {
				t.Errorf("parsePaging(%q) = %d/%d, want %d/%d", tt.qs, page, size, tt.wantPage, tt.wantSz)
			}
		})
	}
}

// ---- handler status-code + side-effect tests ----

const validBody = `{"name":"n","projectId":"p","subscriptionId":"s","pipelineBindings":[{"templateId":"t","targetId":"g"}]}`

func TestCreate(t *testing.T) {
	t.Run("201 and persists with generated id + creator", func(t *testing.T) {
		repo := &fakeRepo{}
		w := do(newRouter(New(repo)), http.MethodPost, "/subscription-tasks", validBody)
		if w.Code != http.StatusCreated {
			t.Fatalf("code = %d, want 201; body=%s", w.Code, w.Body.String())
		}
		if repo.created == nil {
			t.Fatal("repo.Create not called")
		}
		if len(repo.created.ID) == 0 || repo.created.ID[:4] != "sub_" {
			t.Errorf("generated id = %q, want sub_ prefix", repo.created.ID)
		}
		if repo.created.CreatedBy == "" {
			t.Errorf("CreatedBy should be set")
		}
	})
	t.Run("400 on malformed json", func(t *testing.T) {
		w := do(newRouter(New(&fakeRepo{})), http.MethodPost, "/subscription-tasks", "{")
		if w.Code != http.StatusBadRequest {
			t.Fatalf("code = %d, want 400", w.Code)
		}
	})
	t.Run("400 on validation failure", func(t *testing.T) {
		body := `{"name":"n","subscriptionId":"s","pipelineBindings":[{"templateId":"t","targetId":"g"}]}` // no projectId
		w := do(newRouter(New(&fakeRepo{})), http.MethodPost, "/subscription-tasks", body)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("code = %d, want 400", w.Code)
		}
	})
	t.Run("500 on repo error", func(t *testing.T) {
		w := do(newRouter(New(&fakeRepo{createErr: errors.New("boom")})), http.MethodPost, "/subscription-tasks", validBody)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("code = %d, want 500", w.Code)
		}
	})
}

func TestGet(t *testing.T) {
	t.Run("200 when found", func(t *testing.T) {
		w := do(newRouter(New(&fakeRepo{getResult: &st.Task{ID: "sub_1"}})), http.MethodGet, "/subscription-tasks/sub_1", "")
		if w.Code != http.StatusOK {
			t.Fatalf("code = %d, want 200", w.Code)
		}
	})
	t.Run("404 when nil", func(t *testing.T) {
		w := do(newRouter(New(&fakeRepo{getResult: nil})), http.MethodGet, "/subscription-tasks/missing", "")
		if w.Code != http.StatusNotFound {
			t.Fatalf("code = %d, want 404", w.Code)
		}
	})
	t.Run("500 on repo error", func(t *testing.T) {
		w := do(newRouter(New(&fakeRepo{getErr: errors.New("boom")})), http.MethodGet, "/subscription-tasks/x", "")
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("code = %d, want 500", w.Code)
		}
	})
}

func TestUpdate(t *testing.T) {
	t.Run("404 when task does not exist", func(t *testing.T) {
		w := do(newRouter(New(&fakeRepo{getResult: nil})), http.MethodPut, "/subscription-tasks/missing", validBody)
		if w.Code != http.StatusNotFound {
			t.Fatalf("code = %d, want 404", w.Code)
		}
	})
	t.Run("enabled kept from existing when omitted", func(t *testing.T) {
		// existing is disabled; body omits "enabled" → update must preserve false,
		// not fall back to the create-time default of true.
		repo := &fakeRepo{getResult: &st.Task{ID: "sub_1", Enabled: false}}
		w := do(newRouter(New(repo)), http.MethodPut, "/subscription-tasks/sub_1", validBody)
		if w.Code != http.StatusOK {
			t.Fatalf("code = %d, want 200; body=%s", w.Code, w.Body.String())
		}
		if repo.updated == nil {
			t.Fatal("repo.Update not called")
		}
		if repo.updated.Enabled {
			t.Errorf("Enabled should have merged to false from existing, got true")
		}
	})
	t.Run("enabled honored when provided", func(t *testing.T) {
		repo := &fakeRepo{getResult: &st.Task{ID: "sub_1", Enabled: false}}
		body := `{"name":"n","enabled":true,"projectId":"p","subscriptionId":"s","pipelineBindings":[{"templateId":"t","targetId":"g"}]}`
		_ = do(newRouter(New(repo)), http.MethodPut, "/subscription-tasks/sub_1", body)
		if repo.updated == nil || !repo.updated.Enabled {
			t.Errorf("Enabled should be true from body, got %+v", repo.updated)
		}
	})
	t.Run("400 on validation failure", func(t *testing.T) {
		repo := &fakeRepo{getResult: &st.Task{ID: "sub_1"}}
		body := `{"name":"n","subscriptionId":"s","pipelineBindings":[{"templateId":"t","targetId":"g"}]}` // no projectId
		w := do(newRouter(New(repo)), http.MethodPut, "/subscription-tasks/sub_1", body)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("code = %d, want 400", w.Code)
		}
	})
}

func TestDelete(t *testing.T) {
	t.Run("204 no content", func(t *testing.T) {
		repo := &fakeRepo{}
		w := do(newRouter(New(repo)), http.MethodDelete, "/subscription-tasks/sub_1", "")
		if w.Code != http.StatusNoContent {
			t.Fatalf("code = %d, want 204", w.Code)
		}
		if repo.deletedID != "sub_1" {
			t.Errorf("deletedID = %q, want sub_1", repo.deletedID)
		}
	})
	t.Run("500 on repo error", func(t *testing.T) {
		w := do(newRouter(New(&fakeRepo{deleteErr: errors.New("boom")})), http.MethodDelete, "/subscription-tasks/x", "")
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("code = %d, want 500", w.Code)
		}
	})
}

func TestPauseResume(t *testing.T) {
	t.Run("pause sets enabled false and returns 200", func(t *testing.T) {
		repo := &fakeRepo{getResult: &st.Task{ID: "sub_1"}}
		w := do(newRouter(New(repo)), http.MethodPost, "/subscription-tasks/sub_1/pause", "")
		if w.Code != http.StatusOK {
			t.Fatalf("code = %d, want 200", w.Code)
		}
		if repo.setEnVal {
			t.Errorf("pause should set enabled=false")
		}
	})
	t.Run("resume sets enabled true and returns 200", func(t *testing.T) {
		repo := &fakeRepo{getResult: &st.Task{ID: "sub_1"}}
		w := do(newRouter(New(repo)), http.MethodPost, "/subscription-tasks/sub_1/resume", "")
		if w.Code != http.StatusOK {
			t.Fatalf("code = %d, want 200", w.Code)
		}
		if !repo.setEnVal {
			t.Errorf("resume should set enabled=true")
		}
	})
	t.Run("404 when task missing after toggle", func(t *testing.T) {
		w := do(newRouter(New(&fakeRepo{getResult: nil})), http.MethodPost, "/subscription-tasks/x/pause", "")
		if w.Code != http.StatusNotFound {
			t.Fatalf("code = %d, want 404", w.Code)
		}
	})
}

func TestList_FilterAndPaging(t *testing.T) {
	repo := &fakeRepo{tasks: []st.Task{
		{ID: "sub_1", Name: "alpha", Enabled: true},
		{ID: "sub_2", Name: "beta", Enabled: false},
		{ID: "sub_3", Name: "alphabet", Enabled: true},
	}}
	r := newRouter(New(repo))

	// q=alpha matches alpha + alphabet (case-insensitive substring), enabled=true keeps both.
	w := do(r, http.MethodGet, "/subscription-tasks?q=ALPHA&enabled=true", "")
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", w.Code)
	}
	var resp struct {
		Items    []st.Task `json:"items"`
		Total    int       `json:"total"`
		Page     int       `json:"page"`
		PageSize int       `json:"pageSize"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Total != 2 || len(resp.Items) != 2 {
		t.Errorf("q=alpha&enabled=true → total=%d items=%d, want 2/2", resp.Total, len(resp.Items))
	}

	// enabled=false keeps only beta.
	w = do(r, http.MethodGet, "/subscription-tasks?enabled=false", "")
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 1 || (len(resp.Items) == 1 && resp.Items[0].ID != "sub_2") {
		t.Errorf("enabled=false → total=%d items=%v, want 1 (sub_2)", resp.Total, resp.Items)
	}

	// paging: pageSize=1 returns 1 item but total reflects the full filtered set.
	w = do(r, http.MethodGet, "/subscription-tasks?pageSize=1", "")
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 3 || len(resp.Items) != 1 || resp.PageSize != 1 {
		t.Errorf("pageSize=1 → total=%d items=%d pageSize=%d, want 3/1/1", resp.Total, len(resp.Items), resp.PageSize)
	}
}
