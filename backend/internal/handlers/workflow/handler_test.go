package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/CyberOrigin2077/cyber-databrew/internal/argo"
)

// ── Mock WorkflowClient ─────────────────────────────────────────────────────

type mockWorkflowClient struct {
	listFn        func(ctx context.Context, namespace, labelSelector string) ([]wfv1.Workflow, error)
	getFn         func(ctx context.Context, name, namespace string) (*wfv1.Workflow, error)
	logsFn        func(ctx context.Context, workflowName, nodeId, namespace string) (string, error)
	streamFn      func(ctx context.Context, workflowName, podName, container, namespace string) (io.ReadCloser, error)
	operation     string
	namespace     string
	lastLogNodeID string
	lastStreamPod string
}

func (m *mockWorkflowClient) CreateWorkflow(_ context.Context, _ *wfv1.Workflow, _ string) error {
	return nil
}
func (m *mockWorkflowClient) GetWorkflowStatus(_ context.Context, _, _ string) (wfv1.WorkflowPhase, error) {
	return "", nil
}
func (m *mockWorkflowClient) DeleteWorkflow(_ context.Context, _, _ string) error {
	m.operation = "delete"
	return nil
}
func (m *mockWorkflowClient) StopWorkflow(_ context.Context, _, namespace string) error {
	m.operation = "stop"
	m.namespace = namespace
	return nil
}
func (m *mockWorkflowClient) RetryWorkflow(_ context.Context, _, namespace string) error {
	m.operation = "retry"
	m.namespace = namespace
	return nil
}
func (m *mockWorkflowClient) ResubmitWorkflow(_ context.Context, _, namespace string) error {
	m.operation = "resubmit"
	m.namespace = namespace
	return nil
}
func (m *mockWorkflowClient) SuspendWorkflow(_ context.Context, _, namespace string) error {
	m.operation = "suspend"
	m.namespace = namespace
	return nil
}
func (m *mockWorkflowClient) ResumeWorkflow(_ context.Context, _, namespace string) error {
	m.operation = "resume"
	m.namespace = namespace
	return nil
}
func (m *mockWorkflowClient) TerminateWorkflow(_ context.Context, _, namespace string) error {
	m.operation = "terminate"
	m.namespace = namespace
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
	m.lastLogNodeID = nodeId
	if m.logsFn != nil {
		return m.logsFn(ctx, workflowName, nodeId, namespace)
	}
	return "", nil
}
func (m *mockWorkflowClient) GetWorkflowLogStream(ctx context.Context, workflowName, podName, container, namespace string) (io.ReadCloser, error) {
	m.lastStreamPod = podName
	if m.streamFn != nil {
		return m.streamFn(ctx, workflowName, podName, container, namespace)
	}
	return io.NopCloser(strings.NewReader("")), nil
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func makeWorkflow(name, phase string, nodeCount int) *wfv1.Workflow {
	return makeWorkflowWithMeta(name, phase, nodeCount, metav1.Now().Time, nil, nil)
}

func makeWorkflowWithMeta(
	name, phase string,
	nodeCount int,
	createdAt time.Time,
	finishedAt *time.Time,
	labels map[string]string,
) *wfv1.Workflow {
	whf := &wfv1.Workflow{}
	whf.CreationTimestamp = metav1.NewTime(createdAt)
	whf.Name = name
	whf.Status.Phase = wfv1.WorkflowPhase(phase)
	whf.Status.Nodes = make(wfv1.Nodes)
	if finishedAt != nil {
		whf.Status.FinishedAt = metav1.NewTime(*finishedAt)
	}
	if labels != nil {
		whf.Labels = labels
	}
	for i := 0; i < nodeCount; i++ {
		id := string(rune('a' + i))
		whf.Status.Nodes[id] = wfv1.NodeStatus{
			ID:                id,
			Name:              "step-" + id,
			DisplayName:       "step-" + id,
			Type:              wfv1.NodeTypePod,
			TemplateName:      "template-" + id,
			Phase:             wfv1.NodePhase("Running"),
			HostNodeName:      "node-" + id,
			Progress:          wfv1.Progress("1/2"),
			EstimatedDuration: wfv1.EstimatedDuration(12),
			Children:          []string{"child-" + id},
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
	r.POST("/workflows/:name/retry", h.RetryWorkflow)
	r.POST("/workflows/:name/resubmit", h.ResubmitWorkflow)
	r.POST("/workflows/:name/suspend", h.SuspendWorkflow)
	r.POST("/workflows/:name/stop", h.StopWorkflow)
	r.POST("/workflows/:name/resume", h.ResumeWorkflow)
	r.POST("/workflows/:name/terminate", h.TerminateWorkflow)
	r.DELETE("/workflows/:name", h.DeleteWorkflow)
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

func TestListWorkflows_FilterByName(t *testing.T) {
	base := time.Date(2026, 5, 28, 10, 0, 0, 0, time.UTC)
	h := New(&mockWorkflowClient{
		listFn: func(_ context.Context, _, _ string) ([]wfv1.Workflow, error) {
			return []wfv1.Workflow{
				*makeWorkflowWithMeta("AlphaRun", "Running", 1, base.Add(-2*time.Hour), nil, nil),
				*makeWorkflowWithMeta("beta-run", "Succeeded", 1, base.Add(-1*time.Hour), nil, nil),
				*makeWorkflowWithMeta("delta-ALPHA", "Failed", 1, base.Add(-30*time.Minute), nil, nil),
			}, nil
		},
	}, "default")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows?name=alpha", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	items := resp["items"].([]interface{})
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestListWorkflows_FilterByStatus(t *testing.T) {
	base := time.Date(2026, 5, 28, 10, 0, 0, 0, time.UTC)
	h := New(&mockWorkflowClient{
		listFn: func(_ context.Context, _, _ string) ([]wfv1.Workflow, error) {
			return []wfv1.Workflow{
				*makeWorkflowWithMeta("wf-running", "Running", 1, base, nil, nil),
				*makeWorkflowWithMeta("wf-succeeded", "Succeeded", 1, base, nil, nil),
			}, nil
		},
	}, "default")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows?status=Succeeded", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	items := resp["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].(map[string]interface{})["name"] != "wf-succeeded" {
		t.Errorf("expected wf-succeeded, got %v", items[0].(map[string]interface{})["name"])
	}
}

func TestListWorkflows_FilterByLabels(t *testing.T) {
	base := time.Date(2026, 5, 28, 10, 0, 0, 0, time.UTC)
	h := New(&mockWorkflowClient{
		listFn: func(_ context.Context, _, _ string) ([]wfv1.Workflow, error) {
			return []wfv1.Workflow{
				*makeWorkflowWithMeta(
					"wf-a",
					"Running",
					1,
					base.Add(-2*time.Hour),
					nil,
					map[string]string{"team": "ml", "env": "prod"},
				),
				*makeWorkflowWithMeta(
					"wf-b",
					"Running",
					1,
					base.Add(-1*time.Hour),
					nil,
					map[string]string{"team": "ml", "env": "staging"},
				),
				*makeWorkflowWithMeta(
					"wf-c",
					"Running",
					1,
					base.Add(-30*time.Minute),
					nil,
					map[string]string{"team": "ops", "env": "prod"},
				),
			}, nil
		},
	}, "default")
	r := setupRouter(h)

	req := httptest.NewRequest(
		http.MethodGet,
		"/workflows?label=team=ml&label=env=prod",
		nil,
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	items := resp["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].(map[string]interface{})["name"] != "wf-a" {
		t.Errorf("expected wf-a, got %v", items[0].(map[string]interface{})["name"])
	}
}

func TestListWorkflows_InvalidCreatedAfter(t *testing.T) {
	h := New(&mockWorkflowClient{}, "default")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows?createdAfter=not-a-date", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListWorkflows_InvalidFinishedBefore(t *testing.T) {
	h := New(&mockWorkflowClient{}, "default")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows?finishedBefore=bad", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListWorkflows_FilterByCreatedAfter(t *testing.T) {
	base := time.Date(2026, 5, 28, 10, 0, 0, 0, time.UTC)
	oldTime := base.Add(-2 * time.Hour)
	newTime := base
	h := New(&mockWorkflowClient{
		listFn: func(_ context.Context, _, _ string) ([]wfv1.Workflow, error) {
			return []wfv1.Workflow{
				*makeWorkflowWithMeta("wf-old", "Running", 1, oldTime, nil, nil),
				*makeWorkflowWithMeta("wf-new", "Running", 1, newTime, nil, nil),
			}, nil
		},
	}, "default")
	r := setupRouter(h)

	req := httptest.NewRequest(
		http.MethodGet,
		"/workflows?createdAfter="+base.Format(time.RFC3339),
		nil,
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	items := resp["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].(map[string]interface{})["name"] != "wf-new" {
		t.Errorf("expected wf-new, got %v", items[0].(map[string]interface{})["name"])
	}
}

func TestListWorkflows_FilterByFinishedBefore(t *testing.T) {
	base := time.Date(2026, 5, 28, 10, 0, 0, 0, time.UTC)
	oldFinished := base.Add(-2 * time.Hour)
	newFinished := base.Add(2 * time.Hour)
	h := New(&mockWorkflowClient{
		listFn: func(_ context.Context, _, _ string) ([]wfv1.Workflow, error) {
			return []wfv1.Workflow{
				*makeWorkflowWithMeta(
					"wf-early-finish",
					"Succeeded",
					1,
					base.Add(-3*time.Hour),
					&oldFinished,
					nil,
				),
				*makeWorkflowWithMeta(
					"wf-late-finish",
					"Succeeded",
					1,
					base.Add(-2*time.Hour),
					&newFinished,
					nil,
				),
				*makeWorkflowWithMeta("wf-running", "Running", 1, base.Add(-time.Hour), nil, nil),
			}, nil
		},
	}, "default")
	r := setupRouter(h)

	req := httptest.NewRequest(
		http.MethodGet,
		"/workflows?finishedBefore="+base.Format(time.RFC3339),
		nil,
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	items := resp["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].(map[string]interface{})["name"] != "wf-early-finish" {
		t.Errorf("expected wf-early-finish, got %v", items[0].(map[string]interface{})["name"])
	}
}

func TestGetWorkflow_NotFound(t *testing.T) {
	h := New(&mockWorkflowClient{
		getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
			return nil, argo.ErrNotFound
		},
	}, "default")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows/missing-wf", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetWorkflow_InternalError(t *testing.T) {
	h := New(&mockWorkflowClient{
		getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
			return nil, errors.New("argo unavailable")
		},
	}, "default")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows/test-wf", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
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
	firstNode := nodes[0].(map[string]interface{})
	if firstNode["type"] != "Pod" {
		t.Errorf("expected node type Pod, got %v", firstNode["type"])
	}
	if firstNode["templateName"] == "" {
		t.Errorf("expected templateName to be included")
	}
	if firstNode["hostNodeName"] == "" {
		t.Errorf("expected hostNodeName to be included")
	}
	if _, ok := firstNode["children"].([]interface{}); !ok {
		t.Errorf("expected children to be included")
	}
}

func TestGetWorkflow_NormalizedEdgesForOmittedDAGStep(t *testing.T) {
	wf := &wfv1.Workflow{}
	wf.CreationTimestamp = metav1.Now()
	wf.Name = "e2e-two-step-auto"
	wf.Status.Phase = wfv1.WorkflowPhase("Failed")
	wf.Spec.Entrypoint = "dag"
	wf.Spec.Templates = []wfv1.Template{
		{
			Name: "dag",
			DAG: &wfv1.DAGTemplate{
				Tasks: []wfv1.DAGTask{
					{Name: "step-step-1", Template: "step-step-1"},
					{
						Name:         "step-step-2",
						Template:     "step-step-2",
						Dependencies: []string{"step-step-1"},
					},
				},
			},
		},
	}
	wf.Status.Nodes = wfv1.Nodes{
		"root": {
			ID:          "root",
			Name:        "e2e-two-step-auto",
			DisplayName: "e2e-two-step-auto",
			Type:        wfv1.NodeTypeDAG,
			Phase:       wfv1.NodeFailed,
			Children:    []string{"step-1", "step-2"},
		},
		"step-1": {
			ID:           "step-1",
			Name:         "e2e-two-step-auto.step-step-1",
			DisplayName:  "step-step-1",
			Type:         wfv1.NodeTypePod,
			TemplateName: "step-step-1",
			Phase:        wfv1.NodeFailed,
			BoundaryID:   "root",
		},
		"step-2": {
			ID:           "step-2",
			Name:         "e2e-two-step-auto.step-step-2",
			DisplayName:  "step-step-2",
			TemplateName: "step-step-2",
			Phase:        wfv1.NodePhase("Omitted"),
			BoundaryID:   "root",
		},
	}

	h := New(&mockWorkflowClient{
		getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
			return wf, nil
		},
	}, "default")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows/e2e-two-step-auto", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	edges := resp["edges"].([]interface{})
	if len(edges) != 1 {
		t.Fatalf("expected 1 normalized edge, got %d: %#v", len(edges), edges)
	}
	edge := edges[0].(map[string]interface{})
	if edge["source"] != "step-1" || edge["target"] != "step-2" {
		t.Fatalf("unexpected edge endpoints: %#v", edge)
	}
	if edge["kind"] != "dag" {
		t.Fatalf("expected dag edge, got %#v", edge)
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
	wf := makeWorkflow("test-wf", "Succeeded", 0)
	wf.Status.Nodes["test-wf-123"] = wfv1.NodeStatus{
		ID:          "test-wf-123",
		Name:        "test-wf.step-emit",
		DisplayName: "step-emit",
		Type:        wfv1.NodeTypePod,
		Phase:       wfv1.NodeSucceeded,
	}
	client := &mockWorkflowClient{
		getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
			return wf, nil
		},
		logsFn: func(_ context.Context, _, nodeId, _ string) (string, error) {
			return "log output for node " + nodeId, nil
		},
	}
	h := New(client, "default")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows/test-wf/logs?nodeId=test-wf-123", nil)
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
	if !ok || logs != "log output for node test-wf-step-emit-123" {
		t.Errorf("unexpected logs: %v", resp["logs"])
	}
	if client.lastLogNodeID != "test-wf-step-emit-123" {
		t.Fatalf("expected resolved pod name, got %q", client.lastLogNodeID)
	}
}

func TestGetWorkflow_IncludesResolvedPodName(t *testing.T) {
	wf := makeWorkflow("test-wf", "Succeeded", 0)
	wf.Status.Nodes["test-wf-123"] = wfv1.NodeStatus{
		ID:          "test-wf-123",
		Name:        "test-wf.step-emit",
		DisplayName: "step-emit",
		Type:        wfv1.NodeTypePod,
		Phase:       wfv1.NodeSucceeded,
	}
	h := New(&mockWorkflowClient{
		getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
			return wf, nil
		},
	}, "default")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows/test-wf", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Nodes []struct {
			ID      string `json:"id"`
			PodName string `json:"podName"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(resp.Nodes))
	}
	if resp.Nodes[0].PodName != "test-wf-step-emit-123" {
		t.Fatalf("unexpected podName: %q", resp.Nodes[0].PodName)
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

func TestWorkflowOperations(t *testing.T) {
	tests := []struct {
		method string
		path   string
		wantOp string
	}{
		{http.MethodPost, "/workflows/test-wf/retry", "retry"},
		{http.MethodPost, "/workflows/test-wf/resubmit", "resubmit"},
		{http.MethodPost, "/workflows/test-wf/suspend", "suspend"},
		{http.MethodPost, "/workflows/test-wf/stop", "stop"},
		{http.MethodPost, "/workflows/test-wf/resume", "resume"},
		{http.MethodPost, "/workflows/test-wf/terminate", "terminate"},
		{http.MethodDelete, "/workflows/test-wf", "delete"},
	}

	for _, tt := range tests {
		t.Run(tt.wantOp, func(t *testing.T) {
			client := &mockWorkflowClient{}
			h := New(client, "fallback")
			r := setupRouter(h)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
			}
			if client.operation != tt.wantOp {
				t.Fatalf("expected operation %s, got %s", tt.wantOp, client.operation)
			}
			var resp map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if resp["message"] != "ok" {
				t.Fatalf("expected ok message, got %q", resp["message"])
			}
		})
	}
}
