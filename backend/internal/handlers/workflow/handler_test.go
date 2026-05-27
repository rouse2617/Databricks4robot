package workflow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ── Mock WorkflowClient ─────────────────────────────────────────────────────

type mockWorkflowClient struct {
	listFn    func(ctx context.Context, namespace, labelSelector string) ([]wfv1.Workflow, error)
	getFn     func(ctx context.Context, name, namespace string) (*wfv1.Workflow, error)
	logsFn    func(ctx context.Context, workflowName, nodeId, namespace string) (string, error)
}

func (m *mockWorkflowClient) CreateWorkflow(_ context.Context, _ *wfv1.Workflow, _ string) error {
	return nil
}
func (m *mockWorkflowClient) GetWorkflowStatus(_ context.Context, _, _ string) (wfv1.WorkflowPhase, error) {
	return "", nil
}
func (m *mockWorkflowClient) DeleteWorkflow(_ context.Context, _, _ string) error {
	return nil
}
func (m *mockWorkflowClient) StopWorkflow(_ context.Context, _, _ string) error {
	return nil
}
func (m *mockWorkflowClient) ListWorkflows(ctx context.Context, namespace, labelSelector string) ([]wfv1.Workflow, error) {
	if m.listFn != nil {
		return m.listFn(ctx, namespace, labelSelector)
	}
	return nil, nil
}
func (m *mockWorkflowClient) GetWorkflow(ctx context.Context, name, namespace string) (*wfv1.Workflow, error) {
	if m.getFn != nil {
		return m.getFn(ctx, name, namespace)
	}
	return nil, nil
}
func (m *mockWorkflowClient) GetWorkflowLogs(ctx context.Context, workflowName, nodeId, namespace string) (string, error) {
	if m.logsFn != nil {
		return m.logsFn(ctx, workflowName, nodeId, namespace)
	}
	return "", nil
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func makeWorkflow(name, phase string, nodeCount int) *wfv1.Workflow {
	whf := &wfv1.Workflow{}
	fi := metav1.Now()
	whf.CreationTimestamp = fi
	whf.Name = name
	whf.Status.Phase = wfv1.WorkflowPhase(phase)
	whf.Status.Nodes = make(wfv1.Nodes)
	for i := 0; i < nodeCount; i++ {
		id := string(rune('a' + i))
		whf.Status.Nodes[id] = wfv1.NodeStatus{
			ID:     id,
			Name:   "step-" + id,
			Phase:  wfv1.NodePhase("Running"),
		}
	}
	return whf
}

func setupRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/workflows", h.ListWorkflows)
	r.GET("/workflows/:name/logs", h.GetWorkflowLogs)
	r.GET("/workflows/:name", h.GetWorkflow)
	return r
}

// ── Tests ───────────────────────────────────────────────────────────────────

func TestListWorkflows_Empty(t *testing.T) {
	h := New(&mockWorkflowClient{}, "default")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows", nil)
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

func TestListWorkflows_WithItems(t *testing.T) {
	h := New(&mockWorkflowClient{
		listFn: func(_ context.Context, _, _ string) ([]wfv1.Workflow, error) {
			return []wfv1.Workflow{
				*makeWorkflow("wf-1", "Running", 2),
				*makeWorkflow("wf-2", "Succeeded", 3),
			}, nil
		},
	}, "default")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows", nil)
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
	first := items[0].(map[string]interface{})
	if first["name"] != "wf-1" {
		t.Errorf("expected name wf-1, got %v", first["name"])
	}
	if first["status"] != "Running" {
		t.Errorf("expected status Running, got %v", first["status"])
	}
	second := items[1].(map[string]interface{})
	if second["name"] != "wf-2" {
		t.Errorf("expected name wf-2, got %v", second["name"])
	}
	if second["nodeCount"] != float64(3) {
		t.Errorf("expected nodeCount 3, got %v", second["nodeCount"])
	}
}

func TestGetWorkflow_Success(t *testing.T) {
	h := New(&mockWorkflowClient{
		getFn: func(_ context.Context, name, _ string) (*wfv1.Workflow, error) {
			return makeWorkflow(name, "Running", 2), nil
		},
	}, "default")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows/test-wf", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["name"] != "test-wf" {
		t.Errorf("expected name 'test-wf', got %v", resp["name"])
	}
	if resp["status"] != "Running" {
		t.Errorf("expected status 'Running', got %v", resp["status"])
	}
	nodes := resp["nodes"].([]interface{})
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
}

func TestGetWorkflow_EmptyName(t *testing.T) {
	h := New(&mockWorkflowClient{}, "default")
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/workflows/:name", h.GetWorkflow)

	// Gin /workflows/:name doesn't match /workflows/ — returns 404 before handler
	// Test empty name via accessing a non-existent route instead
	req := httptest.NewRequest(http.MethodGet, "/workflows/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest {
		t.Fatalf("expected 404 or 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetWorkflowLogs_Success(t *testing.T) {
	h := New(&mockWorkflowClient{
		logsFn: func(_ context.Context, _, nodeId, _ string) (string, error) {
			return "log output for node " + nodeId, nil
		},
	}, "default")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows/test-wf/logs?nodeId=step-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	logs, ok := resp["logs"].(string)
	if !ok || logs != "log output for node step-1" {
		t.Errorf("unexpected logs: %v", resp["logs"])
	}
}

func TestGetWorkflowLogs_EmptyNodeId(t *testing.T) {
	h := New(&mockWorkflowClient{}, "default")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows/test-wf/logs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing nodeId, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetWorkflowLogs_EmptyName(t *testing.T) {
	h := New(&mockWorkflowClient{}, "default")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows/%20%20/logs?nodeId=step-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty name, got %d: %s", w.Code, w.Body.String())
	}
}
