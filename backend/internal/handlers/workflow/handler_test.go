package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/CyberOrigin2077/cyber-databrew/internal/argo"
	"github.com/CyberOrigin2077/cyber-databrew/internal/k8s"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// ── Mock WorkflowClient ─────────────────────────────────────────────────────

type mockWorkflowClient struct {
	listFn         func(ctx context.Context, namespace, labelSelector string) ([]wfv1.Workflow, error)
	getFn          func(ctx context.Context, name, namespace string) (*wfv1.Workflow, error)
	logsFn         func(ctx context.Context, workflowName, nodeId, namespace string, opts argo.WorkflowLogOptions) (argo.WorkflowLogResult, error)
	streamFn       func(ctx context.Context, workflowName, podName, namespace string, opts argo.WorkflowLogOptions) (io.ReadCloser, error)
	operation      string
	opErr          error
	namespace      string
	resubmitResult *wfv1.Workflow
	lastLogNodeID  string
	lastStreamPod  string
	lastLogOpts    argo.WorkflowLogOptions
	lastStreamOpts argo.WorkflowLogOptions
}

type mockPodClient struct {
	diagFn func(ctx context.Context, namespace, podName string) (*k8s.PodDiagnostics, error)
}

func (m *mockPodClient) GetPodDiagnostics(ctx context.Context, namespace, podName string) (*k8s.PodDiagnostics, error) {
	if m.diagFn != nil {
		return m.diagFn(ctx, namespace, podName)
	}
	return nil, errors.New("not implemented")
}

type mockExecClient struct {
	reqs []k8s.PodExecRequest
	err  error
}

func (m *mockExecClient) ExecPod(_ context.Context, req k8s.PodExecRequest, stdout, _ io.Writer) error {
	m.reqs = append(m.reqs, req)
	if m.err != nil {
		return m.err
	}
	_, _ = stdout.Write([]byte("ok\n"))
	return nil
}

type mockRunRepo struct {
	run        *models.PipelineRun
	byWorkflow map[string]*models.PipelineRun
	saved      []*models.PipelineRun
}

func (m *mockRunRepo) Save(_ context.Context, run *models.PipelineRun) error {
	m.saved = append(m.saved, run)
	if m.byWorkflow == nil {
		m.byWorkflow = map[string]*models.PipelineRun{}
	}
	m.byWorkflow[run.WorkflowName] = run
	return nil
}
func (m *mockRunRepo) FindAll(context.Context) ([]models.PipelineRun, error) { return nil, nil }
func (m *mockRunRepo) RecentDispatchStatsByTarget(context.Context, time.Duration) (map[string]models.TargetDispatchStats, error) {
	return nil, nil
}

func (m *mockRunRepo) FindActiveRunSummaries(context.Context, int) ([]models.PipelineRun, error) {
	return nil, nil
}
func (m *mockRunRepo) FindActiveRunSummariesAfter(context.Context, time.Time, string, int) ([]models.PipelineRun, error) {
	return nil, nil
}

func (m *mockRunRepo) ListSummaries(context.Context, models.PipelineRunListFilter) ([]models.PipelineRun, int, error) {
	return nil, 0, nil
}
func (m *mockRunRepo) FindByID(context.Context, string) (*models.PipelineRun, error) { return nil, nil }
func (m *mockRunRepo) FindSummaryByID(context.Context, string) (*models.PipelineRun, error) {
	return nil, nil
}
func (m *mockRunRepo) FindByWorkflowName(_ context.Context, workflowName string) (*models.PipelineRun, error) {
	if m.byWorkflow != nil {
		return m.byWorkflow[workflowName], nil
	}
	return m.run, nil
}
func (m *mockRunRepo) FindByBatchJobAndAssetID(context.Context, string, string) (*models.PipelineRun, error) {
	return nil, nil
}
func (m *mockRunRepo) FindAllByBatchJobAndAssetID(context.Context, string, string) ([]models.PipelineRun, error) {
	return nil, nil
}
func (m *mockRunRepo) Delete(context.Context, string) error                           { return nil }
func (m *mockRunRepo) DeleteByTemplateID(context.Context, string) error               { return nil }
func (m *mockRunRepo) UpdateStatus(context.Context, string, string, *time.Time) error { return nil }
func (m *mockRunRepo) UpdateLedgerState(context.Context, string, string) error        { return nil }

type mockRunEventRepo struct {
	events []models.PipelineRunEvent
}

func (m *mockRunEventRepo) Append(_ context.Context, event *models.PipelineRunEvent) error {
	m.events = append(m.events, *event)
	return nil
}
func (m *mockRunEventRepo) ListByRunID(context.Context, string, models.PipelineRunEventListOptions) (*models.PipelineRunEventListResult, error) {
	return &models.PipelineRunEventListResult{}, nil
}

func (m *mockWorkflowClient) CreateWorkflow(_ context.Context, _ *wfv1.Workflow, _ string) error {
	return nil
}
func (m *mockWorkflowClient) GetWorkflowStatus(_ context.Context, _, _ string) (wfv1.WorkflowPhase, error) {
	return "", nil
}
func (m *mockWorkflowClient) DeleteWorkflow(_ context.Context, _, _ string) error {
	m.operation = "delete"
	if m.opErr != nil {
		return m.opErr
	}
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
func (m *mockWorkflowClient) ResubmitWorkflowWithResult(_ context.Context, name, namespace string) (*wfv1.Workflow, error) {
	m.operation = "resubmit"
	m.namespace = namespace
	if m.opErr != nil {
		return nil, m.opErr
	}
	if m.resubmitResult != nil {
		return m.resubmitResult, nil
	}
	return makeWorkflow(name+"-abcde", "Running", 1), nil
}
func (m *mockWorkflowClient) SuspendWorkflow(_ context.Context, _, namespace string) error {
	m.operation = "suspend"
	m.namespace = namespace
	if m.opErr != nil {
		return m.opErr
	}
	return nil
}
func (m *mockWorkflowClient) ResumeWorkflow(_ context.Context, _, namespace string) error {
	m.operation = "resume"
	m.namespace = namespace
	if m.opErr != nil {
		return m.opErr
	}
	return nil
}
func (m *mockWorkflowClient) TerminateWorkflow(_ context.Context, _, namespace string) error {
	m.operation = "terminate"
	m.namespace = namespace
	if m.opErr != nil {
		return m.opErr
	}
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
func (m *mockWorkflowClient) GetWorkflowLogs(ctx context.Context, workflowName, nodeId, namespace string, opts argo.WorkflowLogOptions) (argo.WorkflowLogResult, error) {
	m.lastLogNodeID = nodeId
	m.lastLogOpts = opts
	if m.logsFn != nil {
		return m.logsFn(ctx, workflowName, nodeId, namespace, opts)
	}
	return argo.WorkflowLogResult{}, nil
}
func (m *mockWorkflowClient) GetWorkflowLogStream(ctx context.Context, workflowName, podName, namespace string, opts argo.WorkflowLogOptions) (io.ReadCloser, error) {
	m.lastStreamPod = podName
	m.lastStreamOpts = opts
	if m.streamFn != nil {
		return m.streamFn(ctx, workflowName, podName, namespace, opts)
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
	r.GET("/workflows/:name/logs/stream", h.StreamWorkflowLogs)
	r.GET("/workflows/:name/log/stream", h.StreamWorkflowLogs)
	r.GET("/workflows/:name/nodes/:nodeId/pod", h.GetNodePodDiagnostics)
	r.POST("/workflows/:name/nodes/:nodeId/terminal-sessions", h.CreateTerminalSession)
	r.GET("/pod-terminal/sessions/:id", h.GetTerminalSession)
	r.POST("/pod-terminal/sessions/:id/terminate", h.TerminateTerminalSession)
	r.GET("/pod-terminal/sessions/:id/attach", h.AttachTerminalSession)
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

func stringPtr(v string) *string {
	return &v
}

func intPtr(v int) *int {
	return &v
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

func TestGetWorkflow_UsesPipelineRunArgoNamespace(t *testing.T) {
	var namespaceSeen string
	h := New(&mockWorkflowClient{
		getFn: func(_ context.Context, name, namespace string) (*wfv1.Workflow, error) {
			namespaceSeen = namespace
			return makeWorkflow(name, "Succeeded", 1), nil
		},
	}, "default")
	h.SetRunRepositories(&mockRunRepo{byWorkflow: map[string]*models.PipelineRun{
		"wf-video": {
			ID:            "run-1",
			WorkflowName:  "wf-video",
			ArgoNamespace: "video-proc-dev",
		},
	}}, nil, nil)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows/wf-video", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if namespaceSeen != "video-proc-dev" {
		t.Fatalf("expected video-proc-dev namespace, got %q", namespaceSeen)
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

func TestGetWorkflow_IncludesStaticPendingDAGTasks(t *testing.T) {
	localTZ := time.FixedZone("CST", 8*60*60)
	wf := &wfv1.Workflow{}
	wf.CreationTimestamp = metav1.Now()
	wf.Name = "cyb1613-full-dag"
	wf.Status.Phase = wfv1.WorkflowRunning
	wf.Spec.Entrypoint = "dag"
	wf.Spec.Templates = []wfv1.Template{
		{
			Name: "dag",
			DAG: &wfv1.DAGTemplate{
				Tasks: []wfv1.DAGTask{
					{Name: "step-prepare", Template: "step-prepare"},
					{
						Name:         "step-checksum",
						Template:     "step-checksum",
						Dependencies: []string{"step-prepare"},
					},
					{
						Name:         "step-validate",
						Template:     "step-validate",
						Dependencies: []string{"step-checksum"},
					},
					{
						Name:         "step-store",
						Template:     "step-store",
						Dependencies: []string{"step-validate"},
					},
				},
			},
		},
		{Name: "step-prepare", Container: &corev1.Container{Image: "alpine:3.20"}},
		{Name: "step-checksum", Container: &corev1.Container{Image: "alpine:3.20"}},
		{Name: "step-validate", Container: &corev1.Container{Image: "alpine:3.20"}},
		{Name: "step-store", Container: &corev1.Container{Image: "alpine:3.20"}},
	}
	wf.Status.Nodes = wfv1.Nodes{
		"root": {
			ID:          "root",
			Name:        "cyb1613-full-dag",
			DisplayName: "cyb1613-full-dag",
			Type:        wfv1.NodeTypeDAG,
			Phase:       wfv1.NodeRunning,
			Children:    []string{"prepare", "checksum"},
		},
		"prepare": {
			ID:           "prepare",
			Name:         "cyb1613-full-dag.step-prepare",
			DisplayName:  "step-prepare",
			Type:         wfv1.NodeTypePod,
			TemplateName: "step-prepare",
			Phase:        wfv1.NodeSucceeded,
			BoundaryID:   "root",
			StartedAt:    metav1.NewTime(time.Date(2026, 6, 3, 14, 38, 19, 0, localTZ)),
		},
		"checksum": {
			ID:           "checksum",
			Name:         "cyb1613-full-dag.step-checksum",
			DisplayName:  "step-checksum",
			Type:         wfv1.NodeTypePod,
			TemplateName: "step-checksum",
			Phase:        wfv1.NodeRunning,
			BoundaryID:   "root",
		},
	}

	h := New(&mockWorkflowClient{
		getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
			return wf, nil
		},
	}, "default")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows/cyb1613-full-dag", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Nodes []struct {
			ID           string `json:"id"`
			DisplayName  string `json:"displayName"`
			Type         string `json:"type"`
			TemplateName string `json:"templateName"`
			Phase        string `json:"phase"`
			PodName      string `json:"podName"`
			StartedAt    string `json:"startedAt"`
		} `json:"nodes"`
		Edges []workflowDagEdge `json:"edges"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	type workflowNodeTestItem struct {
		ID           string
		DisplayName  string
		Type         string
		TemplateName string
		Phase        string
		PodName      string
		StartedAt    string
	}
	nodesByDisplayName := map[string]workflowNodeTestItem{}
	for _, node := range resp.Nodes {
		nodesByDisplayName[node.DisplayName] = workflowNodeTestItem{
			ID:           node.ID,
			DisplayName:  node.DisplayName,
			Type:         node.Type,
			TemplateName: node.TemplateName,
			Phase:        node.Phase,
			PodName:      node.PodName,
			StartedAt:    node.StartedAt,
		}
	}
	for _, name := range []string{"step-prepare", "step-checksum", "step-validate", "step-store"} {
		if _, ok := nodesByDisplayName[name]; !ok {
			t.Fatalf("expected node %q in response, got %#v", name, resp.Nodes)
		}
	}
	if nodesByDisplayName["step-prepare"].Phase != string(wfv1.NodeSucceeded) {
		t.Fatalf("runtime phase was not preserved: %#v", nodesByDisplayName["step-prepare"])
	}
	if nodesByDisplayName["step-prepare"].StartedAt != "2026-06-03T06:38:19Z" {
		t.Fatalf("expected runtime time to be UTC RFC3339, got %#v", nodesByDisplayName["step-prepare"])
	}
	if nodesByDisplayName["step-validate"].Phase != string(wfv1.NodePending) {
		t.Fatalf("expected static task pending, got %#v", nodesByDisplayName["step-validate"])
	}
	if nodesByDisplayName["step-store"].Type != "Pod" {
		t.Fatalf("expected static task to resolve template type Pod, got %#v", nodesByDisplayName["step-store"])
	}
	if nodesByDisplayName["step-store"].PodName != "" {
		t.Fatalf("static pending task should not have a pod name: %#v", nodesByDisplayName["step-store"])
	}

	edgeSet := map[string]bool{}
	for _, edge := range resp.Edges {
		edgeSet[edge.Source+"->"+edge.Target] = true
	}
	if !edgeSet["checksum->step-validate"] {
		t.Fatalf("expected edge checksum->step-validate, got %#v", resp.Edges)
	}
	if !edgeSet["step-validate->step-store"] {
		t.Fatalf("expected edge step-validate->step-store, got %#v", resp.Edges)
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
		logsFn: func(_ context.Context, _, nodeId, _ string, opts argo.WorkflowLogOptions) (argo.WorkflowLogResult, error) {
			return argo.WorkflowLogResult{
				Logs:       "log output for node " + nodeId,
				LineCount:  1,
				Truncated:  false,
				LimitBytes: *opts.LimitBytes,
			}, nil
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
	if resp["workflowName"] != "test-wf" || resp["podName"] != "test-wf-step-emit-123" {
		t.Fatalf("unexpected log metadata: %#v", resp)
	}
	pagination, ok := resp["pagination"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected pagination metadata, got %#v", resp["pagination"])
	}
	if pagination["available"] != false || pagination["nextCursor"] != nil {
		t.Fatalf("unexpected pagination metadata: %#v", pagination)
	}
	window, ok := resp["window"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected window metadata, got %#v", resp["window"])
	}
	if window["mode"] != "tail" || window["scope"] != "bounded-live-window" {
		t.Fatalf("unexpected window metadata: %#v", window)
	}
	if client.lastLogNodeID != "test-wf-step-emit-123" {
		t.Fatalf("expected resolved pod name, got %q", client.lastLogNodeID)
	}
	if client.lastLogOpts.TailLines == nil || *client.lastLogOpts.TailLines != defaultWorkflowLogTailLines {
		t.Fatalf("expected default tailLines %d, got %#v", defaultWorkflowLogTailLines, client.lastLogOpts.TailLines)
	}
	if client.lastLogOpts.LimitBytes == nil || *client.lastLogOpts.LimitBytes != defaultWorkflowLogLimitBytes {
		t.Fatalf("expected default limitBytes %d, got %#v", defaultWorkflowLogLimitBytes, client.lastLogOpts.LimitBytes)
	}
}

func TestGetWorkflowLogs_ClampsBounds(t *testing.T) {
	wf := makeWorkflow("test-wf", "Succeeded", 1)
	client := &mockWorkflowClient{
		getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
			return wf, nil
		},
		logsFn: func(_ context.Context, _, _, _ string, opts argo.WorkflowLogOptions) (argo.WorkflowLogResult, error) {
			return argo.WorkflowLogResult{Logs: "ok\n", LineCount: 1, Truncated: true, LimitBytes: *opts.LimitBytes}, nil
		},
	}
	h := New(client, "default")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows/test-wf/logs?nodeId=a&tailLines=999999&limitBytes=99999999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if client.lastLogOpts.TailLines == nil || *client.lastLogOpts.TailLines != maxWorkflowLogTailLines {
		t.Fatalf("tailLines was not clamped: %#v", client.lastLogOpts.TailLines)
	}
	if client.lastLogOpts.LimitBytes == nil || *client.lastLogOpts.LimitBytes != maxWorkflowLogLimitBytes {
		t.Fatalf("limitBytes was not clamped: %#v", client.lastLogOpts.LimitBytes)
	}
	var resp struct {
		Truncation struct {
			TailLinesClamped  bool `json:"tailLinesClamped"`
			LimitBytesClamped bool `json:"limitBytesClamped"`
		} `json:"truncation"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Truncation.TailLinesClamped || !resp.Truncation.LimitBytesClamped {
		t.Fatalf("expected clamp metadata, got %#v", resp.Truncation)
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

func TestGetWorkflowLogs_RejectsCursor(t *testing.T) {
	wf := makeWorkflow("test-wf", "Succeeded", 1)
	client := &mockWorkflowClient{
		getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
			return wf, nil
		},
	}
	h := New(client, "default")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows/test-wf/logs?nodeId=a&cursor=older", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for cursor, got %d: %s", w.Code, w.Body.String())
	}
	if client.lastLogNodeID != "" {
		t.Fatalf("cursor request should not fetch logs, got node %q", client.lastLogNodeID)
	}
}

func TestStreamWorkflowLogs_StructuredEvents(t *testing.T) {
	wf := makeWorkflow("test-wf", "Running", 1)
	client := &mockWorkflowClient{
		getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
			return wf, nil
		},
		streamFn: func(_ context.Context, _, _, _ string, _ argo.WorkflowLogOptions) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader(`{"result":{"podName":"a","content":"hello\n"}}` + "\n")), nil
		},
	}
	h := New(client, "default")
	r := setupRouter(h)
	server := httptest.NewServer(r)
	defer server.Close()

	resp, err := http.Get(server.URL + "/workflows/test-wf/logs/stream?nodeId=a")
	if err != nil {
		t.Fatalf("stream request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("expected event-stream content type, got %q", ct)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read stream body: %v", err)
	}
	body := string(raw)
	if !strings.Contains(body, "event: heartbeat") || !strings.Contains(body, "event: log") {
		t.Fatalf("expected structured SSE events, got %q", body)
	}
	if !strings.Contains(body, `"podName":"a"`) || !strings.Contains(body, `"container":"main"`) {
		t.Fatalf("expected log event metadata, got %q", body)
	}
	if client.lastStreamOpts.TailLines == nil || *client.lastStreamOpts.TailLines != defaultWorkflowLogTailLines {
		t.Fatalf("expected default stream tailLines, got %#v", client.lastStreamOpts.TailLines)
	}
	// Running node → live-tail (CYB-3483).
	if !client.lastStreamOpts.Follow {
		t.Fatalf("running node must be followed for live tail")
	}
}

// A finished node must NOT be followed: following a completed pod never EOFs
// through Argo's follow API, hanging the SSE until the Cloud Run timeout → 504
// and triggering client reconnect storms. (CYB-3483)
func TestStreamWorkflowLogs_DoesNotFollowFinishedNode(t *testing.T) {
	wf := makeWorkflow("test-wf-done", "Succeeded", 1)
	wf.Status.Nodes["a"] = wfv1.NodeStatus{
		ID:           "a",
		Name:         "step-a",
		DisplayName:  "step-a",
		Type:         wfv1.NodeTypePod,
		TemplateName: "template-a",
		Phase:        wfv1.NodeSucceeded,
	}
	client := &mockWorkflowClient{
		getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) { return wf, nil },
		streamFn: func(_ context.Context, _, _, _ string, _ argo.WorkflowLogOptions) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader(`{"result":{"podName":"a","content":"done\n"}}` + "\n")), nil
		},
	}
	h := New(client, "default")
	r := setupRouter(h)
	server := httptest.NewServer(r)
	defer server.Close()

	resp, err := http.Get(server.URL + "/workflows/test-wf-done/logs/stream?nodeId=a")
	if err != nil {
		t.Fatalf("stream request: %v", err)
	}
	defer resp.Body.Close()
	if _, err := io.ReadAll(resp.Body); err != nil {
		t.Fatalf("read stream body: %v", err)
	}
	if client.lastStreamOpts.Follow {
		t.Fatalf("finished node must NOT be followed (Follow=true hangs until request timeout)")
	}
}

// When a node has no live logs to stream — workflow TTL'd/GC'd, pod recycled, or
// pod not yet created — the SSE handler must end gracefully with an "end" event
// (200, text/event-stream) rather than a 4xx/5xx. A non-2xx fires
// EventSource.onerror → exponential-backoff reconnect storm; an "end" event
// closes the client cleanly. Mirrors the non-stream GetWorkflowLogs graceful
// branches for the SSE variant (CYB-3579).
func TestStreamWorkflowLogs_GracefulEndForMissingLogs(t *testing.T) {
	notFound := fmt.Errorf("%w: pods not found", argo.ErrNotFound)

	tests := []struct {
		name       string
		wfName     string
		getFn      func(context.Context, string, string) (*wfv1.Workflow, error)
		streamFn   func(context.Context, string, string, string, argo.WorkflowLogOptions) (io.ReadCloser, error)
		wantReason string
	}{
		{
			name:   "pod recycled: stream returns ErrNotFound",
			wfName: "wf-recycled",
			getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
				return makeWorkflow("wf-recycled", "Running", 1), nil
			},
			streamFn: func(_ context.Context, _, _, _ string, _ argo.WorkflowLogOptions) (io.ReadCloser, error) {
				return nil, notFound
			},
			wantReason: "pod-recycled",
		},
		{
			name:   "workflow gone: get returns ErrNotFound",
			wfName: "wf-gone",
			getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
				return nil, notFound
			},
			wantReason: "workflow-gone",
		},
		{
			name:   "pending workflow: no pod yet",
			wfName: "wf-pending",
			getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
				return makeWorkflow("wf-pending", "Pending", 0), nil
			},
			wantReason: "pod-not-created",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockWorkflowClient{getFn: tt.getFn, streamFn: tt.streamFn}
			h := New(client, "default")
			server := httptest.NewServer(setupRouter(h))
			defer server.Close()

			resp, err := http.Get(server.URL + "/workflows/" + tt.wfName + "/logs/stream?nodeId=a")
			if err != nil {
				t.Fatalf("stream request: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected 200 graceful end, got %d", resp.StatusCode)
			}
			if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
				t.Fatalf("expected event-stream content type, got %q", ct)
			}
			raw, _ := io.ReadAll(resp.Body)
			body := string(raw)
			if !strings.Contains(body, "event: end") {
				t.Fatalf("expected terminal end event, got %q", body)
			}
			if !strings.Contains(body, `"reason":"`+tt.wantReason+`"`) {
				t.Fatalf("expected end reason %q, got %q", tt.wantReason, body)
			}
			// The stream never opened, so no heartbeat/log frames must be emitted.
			if strings.Contains(body, "event: heartbeat") || strings.Contains(body, "event: log") {
				t.Fatalf("graceful end must not emit stream frames, got %q", body)
			}
		})
	}
}

// A genuinely unknown node id on a live (non-pending) workflow is a real client
// error and still 400s — the graceful-end path must not swallow it.
func TestStreamWorkflowLogs_UnknownNodeStill400(t *testing.T) {
	client := &mockWorkflowClient{
		getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
			return makeWorkflow("wf-unknown", "Running", 1), nil // only node "a" exists
		},
	}
	h := New(client, "default")
	server := httptest.NewServer(setupRouter(h))
	defer server.Close()

	resp, err := http.Get(server.URL + "/workflows/wf-unknown/logs/stream?nodeId=zzz")
	if err != nil {
		t.Fatalf("stream request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown node on live workflow, got %d", resp.StatusCode)
	}
}

// A genuine (non-not-found) stream fault still surfaces as 500 so the client
// retries transient argo/RBAC failures and #462 logs them server-side.
func TestStreamWorkflowLogs_GenuineFaultStill500(t *testing.T) {
	client := &mockWorkflowClient{
		getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
			return makeWorkflow("wf-fault", "Running", 1), nil
		},
		streamFn: func(_ context.Context, _, _, _ string, _ argo.WorkflowLogOptions) (io.ReadCloser, error) {
			return nil, fmt.Errorf("argo-server connection refused")
		},
	}
	h := New(client, "default")
	server := httptest.NewServer(setupRouter(h))
	defer server.Close()

	resp, err := http.Get(server.URL + "/workflows/wf-fault/logs/stream?nodeId=a")
	if err != nil {
		t.Fatalf("stream request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500 for genuine fault, got %d", resp.StatusCode)
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
			client := &mockWorkflowClient{
				getFn: func(_ context.Context, name, namespace string) (*wfv1.Workflow, error) {
					if tt.wantOp != "retry" {
						return nil, nil
					}
					wf := makeWorkflow(name, "Failed", 1)
					wf.Status.Nodes["a"] = wfv1.NodeStatus{
						ID:           "a",
						Name:         "step-a",
						DisplayName:  "step-a",
						Type:         wfv1.NodeTypePod,
						TemplateName: "template-a",
						Phase:        wfv1.NodeFailed,
					}
					return wf, nil
				},
			}
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

// TestWorkflowOperationsMissingWorkflowReturn404 (CYB-4280) locks the fix that
// control ops on a missing workflow return 404 WORKFLOW_NOT_FOUND — the same as
// retry — instead of 500. retry gets not-found from the GetWorkflow precheck
// (getFn); the sibling ops get it from the client method itself (opErr), which
// mirrors the real crd client wrapping argo.ErrNotFound via translateK8sErr.
func TestWorkflowOperationsMissingWorkflowReturn404(t *testing.T) {
	notFound := fmt.Errorf("%w: workflows.argoproj.io \"__missing__\" not found", argo.ErrNotFound)
	tests := []struct {
		op     string
		method string
		path   string
	}{
		{"retry", http.MethodPost, "/workflows/__missing__/retry"},
		{"resubmit", http.MethodPost, "/workflows/__missing__/resubmit"},
		{"suspend", http.MethodPost, "/workflows/__missing__/suspend"},
		{"resume", http.MethodPost, "/workflows/__missing__/resume"},
		{"terminate", http.MethodPost, "/workflows/__missing__/terminate"},
		{"delete", http.MethodDelete, "/workflows/__missing__"},
	}
	for _, tt := range tests {
		t.Run(tt.op, func(t *testing.T) {
			client := &mockWorkflowClient{
				getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
					return nil, notFound
				},
				opErr: notFound,
			}
			h := New(client, "fallback")
			r := setupRouter(h)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusNotFound {
				t.Fatalf("%s: expected 404, got %d: %s", tt.op, w.Code, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), "WORKFLOW_NOT_FOUND") {
				t.Fatalf("%s: expected WORKFLOW_NOT_FOUND in body, got %s", tt.op, w.Body.String())
			}
		})
	}
}

func TestResubmitWorkflow_CreatesPipelineRunLedger(t *testing.T) {
	createdAt := time.Date(2026, 6, 17, 9, 3, 2, 0, time.UTC)
	finishedAt := createdAt.Add(10 * time.Second)
	resubmittedWorkflow := makeWorkflowWithMeta("source-wf-gthtk", "Succeeded", 2, createdAt, &finishedAt, map[string]string{
		"workflows.argoproj.io/resubmitted-from-workflow": "source-wf",
	})
	resubmittedWorkflow.Status.StartedAt = metav1.NewTime(createdAt.Add(1 * time.Second))
	resubmittedWorkflow.Status.Message = "done"
	sourceRun := &models.PipelineRun{
		ID:                "run-source",
		TemplateID:        stringPtr("template-1"),
		PipelineName:      "pipeline-1",
		TemplateVersion:   intPtr(3),
		WorkflowName:      "source-wf",
		ExecutionTargetID: "target-1",
		TargetSnapshot:    map[string]interface{}{"namespace": "argo"},
		Status:            "Succeeded",
		NodeCount:         1,
		AssetIDs:          []string{"asset-1"},
		AssetCount:        1,
		Manifest:          stringPtr("manifest"),
		PipelineJSON:      map[string]interface{}{"nodes": []interface{}{}},
		ArgoNamespace:     "argo",
		Scope:             "dev",
		Owner:             "tester",
		BatchJobID:        stringPtr("batch-1"),
		LedgerState:       "active",
	}
	runRepo := &mockRunRepo{byWorkflow: map[string]*models.PipelineRun{
		"source-wf": sourceRun,
	}}
	eventRepo := &mockRunEventRepo{}
	h := New(&mockWorkflowClient{resubmitResult: resubmittedWorkflow}, "argo")
	h.SetRunRepositories(runRepo, eventRepo, nil)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/workflows/source-wf/resubmit", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["workflowName"] != "source-wf-gthtk" {
		t.Fatalf("expected new workflow name, got %#v", resp)
	}
	if len(runRepo.saved) != 1 {
		t.Fatalf("expected one saved run, got %d", len(runRepo.saved))
	}
	saved := runRepo.saved[0]
	if saved.ID == "" || saved.ID == sourceRun.ID {
		t.Fatalf("expected new run id, got %q", saved.ID)
	}
	if saved.WorkflowName != "source-wf-gthtk" || saved.Status != "Succeeded" {
		t.Fatalf("unexpected saved run workflow/status: %#v", saved)
	}
	if saved.TemplateID == nil || *saved.TemplateID != "template-1" {
		t.Fatalf("expected template copied, got %#v", saved.TemplateID)
	}
	if saved.BatchJobID == nil || *saved.BatchJobID != "batch-1" {
		t.Fatalf("expected batch copied, got %#v", saved.BatchJobID)
	}
	if saved.StartedAt == nil || saved.FinishedAt == nil {
		t.Fatalf("expected workflow times copied, got started=%v finished=%v", saved.StartedAt, saved.FinishedAt)
	}
	if len(eventRepo.events) != 1 {
		t.Fatalf("expected one run event, got %d", len(eventRepo.events))
	}
	event := eventRepo.events[0]
	if event.EventType != "run_resubmitted" || event.RunID != saved.ID || event.WorkflowName != saved.WorkflowName {
		t.Fatalf("unexpected event: %#v", event)
	}
	if event.Payload["sourceWorkflowName"] != "source-wf" {
		t.Fatalf("expected source workflow payload, got %#v", event.Payload)
	}
}

func TestResubmitWorkflow_DoesNotDuplicateExistingLedger(t *testing.T) {
	resubmittedWorkflow := makeWorkflow("source-wf-gthtk", "Running", 1)
	existingRun := &models.PipelineRun{ID: "run-existing", WorkflowName: "source-wf-gthtk"}
	runRepo := &mockRunRepo{byWorkflow: map[string]*models.PipelineRun{
		"source-wf":       {ID: "run-source", WorkflowName: "source-wf"},
		"source-wf-gthtk": existingRun,
	}}
	h := New(&mockWorkflowClient{resubmitResult: resubmittedWorkflow}, "argo")
	h.SetRunRepositories(runRepo, &mockRunEventRepo{}, nil)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/workflows/source-wf/resubmit", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if len(runRepo.saved) != 0 {
		t.Fatalf("expected no duplicate save, got %d", len(runRepo.saved))
	}
}

func TestStopWorkflow_SyncsPipelineRunLedger(t *testing.T) {
	run := &models.PipelineRun{
		ID:           "run-1",
		WorkflowName: "wf-1",
		Status:       "Running",
	}
	runRepo := &mockRunRepo{byWorkflow: map[string]*models.PipelineRun{"wf-1": run}}
	eventRepo := &mockRunEventRepo{}
	client := &mockWorkflowClient{}
	h := New(client, "argo")
	h.SetRunRepositories(runRepo, eventRepo, nil)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/workflows/wf-1/stop", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if client.operation != "stop" || client.namespace != "argo" {
		t.Fatalf("expected stop operation in argo namespace, got op=%q namespace=%q", client.operation, client.namespace)
	}
	if run.Status != string(wfv1.WorkflowFailed) {
		t.Fatalf("expected run failed, got %q", run.Status)
	}
	if run.Message != "workflow shutdown with strategy: Stop" {
		t.Fatalf("unexpected run message %q", run.Message)
	}
	if run.FinishedAt == nil {
		t.Fatal("expected finished_at to be set")
	}
	if len(runRepo.saved) != 1 {
		t.Fatalf("expected saved run once, got %d", len(runRepo.saved))
	}
	if len(eventRepo.events) != 1 || eventRepo.events[0].EventType != "run_failed" {
		t.Fatalf("expected run_failed event, got %#v", eventRepo.events)
	}
}

func TestStopWorkflow_UsesPipelineRunArgoNamespace(t *testing.T) {
	run := &models.PipelineRun{
		ID:            "run-1",
		WorkflowName:  "wf-video",
		Status:        "Running",
		ArgoNamespace: "video-proc-dev",
	}
	runRepo := &mockRunRepo{byWorkflow: map[string]*models.PipelineRun{"wf-video": run}}
	client := &mockWorkflowClient{}
	h := New(client, "default")
	h.SetRunRepositories(runRepo, &mockRunEventRepo{}, nil)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/workflows/wf-video/stop", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if client.operation != "stop" || client.namespace != "video-proc-dev" {
		t.Fatalf("expected stop operation in video-proc-dev namespace, got op=%q namespace=%q", client.operation, client.namespace)
	}
}

func TestRetryWorkflow_NotRetryable(t *testing.T) {
	client := &mockWorkflowClient{
		getFn: func(_ context.Context, name, namespace string) (*wfv1.Workflow, error) {
			wf := makeWorkflow(name, "Failed", 1)
			wf.Status.Message = "Stopped"
			wf.Status.Nodes["a"] = wfv1.NodeStatus{
				ID:    "a",
				Name:  "step-a",
				Type:  wfv1.NodeTypePod,
				Phase: wfv1.NodePending,
			}
			return wf, nil
		},
	}
	h := New(client, "default")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/workflows/test-wf/retry", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
	if client.operation != "" {
		t.Fatalf("expected no retry call, got %q", client.operation)
	}
}

func TestGetNodePodDiagnostics_Success(t *testing.T) {
	h := New(&mockWorkflowClient{
		getFn: func(_ context.Context, name, namespace string) (*wfv1.Workflow, error) {
			if name != "test-wf" || namespace != "default" {
				t.Fatalf("unexpected workflow lookup %s/%s", namespace, name)
			}
			return makeWorkflow("test-wf", "Running", 2), nil
		},
	}, "default")
	h.SetPodClient(&mockPodClient{
		diagFn: func(_ context.Context, namespace, podName string) (*k8s.PodDiagnostics, error) {
			if namespace != "default" {
				t.Fatalf("unexpected namespace %s", namespace)
			}
			if podName != "a" {
				t.Fatalf("unexpected pod name %s", podName)
			}
			return &k8s.PodDiagnostics{
				Cluster:            "dev-gke",
				Namespace:          namespace,
				PodName:            podName,
				PodIP:              "10.1.2.3",
				ServiceAccountName: "workflow-sa",
				RestartCount:       1,
				Containers: []k8s.ContainerInfo{
					{Name: "main", Image: "alpine:3.20", Ready: true, RestartCount: 1, State: "Running"},
				},
				Conditions: []k8s.PodConditionInfo{{Type: "Ready", Status: "True"}},
				Events:     []k8s.EventInfo{{Type: "Normal", Reason: "Pulled", Message: "pulled"}},
			}, nil
		},
	})
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows/test-wf/nodes/a/pod", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["podName"] != "a" {
		t.Fatalf("expected podName a, got %v", resp["podName"])
	}
	if resp["cluster"] != "dev-gke" {
		t.Fatalf("expected cluster dev-gke, got %v", resp["cluster"])
	}
}

func TestGetNodePodDiagnostics_Unconfigured(t *testing.T) {
	h := New(&mockWorkflowClient{
		getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
			return makeWorkflow("test-wf", "Running", 1), nil
		},
	}, "default")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows/test-wf/nodes/a/pod", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
	assertErrorCode(t, w.Body.Bytes(), "K8S_UNAVAILABLE")
}

func TestGetNodePodDiagnostics_WorkflowNotFoundBeforeKubernetesAvailability(t *testing.T) {
	h := New(&mockWorkflowClient{
		getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
			return nil, argo.ErrNotFound
		},
	}, "default")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows/missing-wf/nodes/a/pod", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
	assertErrorCode(t, w.Body.Bytes(), "WORKFLOW_NOT_FOUND")
}

func TestGetNodePodDiagnostics_UsesTemplateNameForParameterizedPod(t *testing.T) {
	wf := &wfv1.Workflow{}
	wf.Name = "fanout-demo"
	wf.Status.Nodes = wfv1.Nodes{
		"fanout-demo-3231058135": {
			ID:           "fanout-demo-3231058135",
			Name:         "fanout-demo.process-asset(0:seg_001)",
			DisplayName:  "process-asset(0:seg_001)",
			Type:         wfv1.NodeTypePod,
			TemplateName: "process-one",
			Phase:        wfv1.NodeSucceeded,
		},
	}
	h := New(&mockWorkflowClient{
		getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
			return wf, nil
		},
	}, "default")
	h.SetPodClient(&mockPodClient{
		diagFn: func(_ context.Context, _, podName string) (*k8s.PodDiagnostics, error) {
			if podName != "fanout-demo-process-one-3231058135" {
				t.Fatalf("unexpected pod name %s", podName)
			}
			return &k8s.PodDiagnostics{
				Namespace:    "default",
				PodName:      podName,
				RestartCount: 0,
				Containers:   []k8s.ContainerInfo{},
				Conditions:   []k8s.PodConditionInfo{},
				Events:       []k8s.EventInfo{},
			}, nil
		},
	})
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows/fanout-demo/nodes/fanout-demo-3231058135/pod", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetNodePodDiagnostics_NodeNotFound(t *testing.T) {
	h := New(&mockWorkflowClient{
		getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
			return makeWorkflow("test-wf", "Running", 1), nil
		},
	}, "default")
	h.SetPodClient(&mockPodClient{})
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows/test-wf/nodes/missing/pod", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
	assertErrorCode(t, w.Body.Bytes(), "NODE_NOT_FOUND")
}

func TestGetNodePodDiagnostics_PodNotFound_ReturnsStub(t *testing.T) {
	h := New(&mockWorkflowClient{
		getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
			return makeWorkflow("test-wf", "Running", 1), nil
		},
	}, "default")
	h.SetPodClient(&mockPodClient{
		diagFn: func(_ context.Context, _, _ string) (*k8s.PodDiagnostics, error) {
			return nil, apierrors.NewNotFound(schema.GroupResource{Resource: "pods"}, "pod-a")
		},
	})
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows/test-wf/nodes/a/pod", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var diag k8s.PodDiagnostics
	if err := json.Unmarshal(w.Body.Bytes(), &diag); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !diag.GarbageCollected {
		t.Error("expected garbageCollected=true")
	}
	if diag.PodName == "" {
		t.Error("expected non-empty podName in stub")
	}
}

func TestGetNodePodDiagnostics_KubernetesErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code int
		want string
	}{
		{
			name: "forbidden",
			err:  apierrors.NewForbidden(schema.GroupResource{Resource: "pods"}, "pod-a", errors.New("no rbac")),
			code: http.StatusForbidden,
			want: "K8S_FORBIDDEN",
		},
		{
			name: "unavailable",
			err:  k8s.ErrUnavailable,
			code: http.StatusServiceUnavailable,
			want: "K8S_UNAVAILABLE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := New(&mockWorkflowClient{
				getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
					return makeWorkflow("test-wf", "Running", 1), nil
				},
			}, "default")
			h.SetPodClient(&mockPodClient{
				diagFn: func(_ context.Context, _, _ string) (*k8s.PodDiagnostics, error) {
					return nil, tt.err
				},
			})
			r := setupRouter(h)

			req := httptest.NewRequest(http.MethodGet, "/workflows/test-wf/nodes/a/pod", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.code {
				t.Fatalf("expected %d, got %d: %s", tt.code, w.Code, w.Body.String())
			}
			assertErrorCode(t, w.Body.Bytes(), tt.want)
		})
	}
}

func TestCreateTerminalSession_DisabledByDefault(t *testing.T) {
	h := New(&mockWorkflowClient{
		getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
			return makeWorkflow("test-wf", "Running", 1), nil
		},
	}, "default")
	h.SetRunRepositories(&mockRunRepo{run: &models.PipelineRun{
		ID:             "run-1",
		WorkflowName:   "test-wf",
		TargetSnapshot: map[string]interface{}{},
	}}, &mockRunEventRepo{}, nil)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/workflows/test-wf/nodes/a/terminal-sessions", strings.NewReader(`{"command":"sh"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	assertErrorCode(t, w.Body.Bytes(), "POD_EXEC_FORBIDDEN")
}

func TestCreateTerminalSession_AllowedPolicyCreatesSession(t *testing.T) {
	eventRepo := &mockRunEventRepo{}
	h := New(&mockWorkflowClient{
		getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
			return makeWorkflow("test-wf", "Running", 1), nil
		},
	}, "default")
	h.SetRunRepositories(&mockRunRepo{run: &models.PipelineRun{
		ID:                "run-1",
		WorkflowName:      "test-wf",
		ExecutionTargetID: "target-1",
		TargetSnapshot: map[string]interface{}{
			"cluster":   "gke-dev",
			"namespace": "default",
			"terminal": map[string]interface{}{
				"enabled":         true,
				"allowedCommands": []interface{}{"sh", "pwd"},
			},
		},
	}}, eventRepo, nil)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/workflows/test-wf/nodes/a/terminal-sessions", strings.NewReader(`{"command":"pwd"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["id"] == "" || resp["attachUrl"] == "" {
		t.Fatalf("expected id and attachUrl, got %#v", resp)
	}
	if resp["podName"] == "" || resp["command"] != "pwd" {
		t.Fatalf("unexpected terminal response %#v", resp)
	}
	if resp["containerName"] != "main" {
		t.Fatalf("expected default containerName main, got %#v", resp["containerName"])
	}
	if len(eventRepo.events) != 1 || eventRepo.events[0].EventType != "pod_terminal_session_created" {
		t.Fatalf("expected created run event, got %#v", eventRepo.events)
	}
}

func TestCreateTerminalSession_ResourceDefaultsPolicyCreatesSession(t *testing.T) {
	h := New(&mockWorkflowClient{
		getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
			return makeWorkflow("test-wf", "Running", 1), nil
		},
	}, "default")
	h.SetRunRepositories(&mockRunRepo{run: &models.PipelineRun{
		ID:                "run-1",
		WorkflowName:      "test-wf",
		ExecutionTargetID: "target-1",
		TargetSnapshot: map[string]interface{}{
			"resourceDefaults": map[string]interface{}{
				"terminal": map[string]interface{}{
					"enabled":         true,
					"allowedCommands": []interface{}{"sh", "pwd"},
				},
			},
		},
	}}, &mockRunEventRepo{}, nil)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/workflows/test-wf/nodes/a/terminal-sessions", strings.NewReader(`{"command":"pwd"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateTerminalSession_InvalidJSON(t *testing.T) {
	h := New(&mockWorkflowClient{
		getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
			return makeWorkflow("test-wf", "Running", 1), nil
		},
	}, "default")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/workflows/test-wf/nodes/a/terminal-sessions", strings.NewReader(`{"command":`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	assertErrorCode(t, w.Body.Bytes(), "INVALID_ARGUMENT")
}

func TestTerminateTerminalSession(t *testing.T) {
	h := New(&mockWorkflowClient{
		getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
			return makeWorkflow("test-wf", "Running", 1), nil
		},
	}, "default")
	h.SetRunRepositories(&mockRunRepo{run: &models.PipelineRun{
		ID:                "run-1",
		WorkflowName:      "test-wf",
		ExecutionTargetID: "target-1",
		TargetSnapshot: map[string]interface{}{
			"terminal": map[string]interface{}{"enabled": true},
		},
	}}, &mockRunEventRepo{}, nil)
	r := setupRouter(h)

	createReq := httptest.NewRequest(http.MethodPost, "/workflows/test-wf/nodes/a/terminal-sessions", strings.NewReader(`{"command":"sh"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)
	if createW.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d: %s", createW.Code, createW.Body.String())
	}
	var created map[string]any
	_ = json.Unmarshal(createW.Body.Bytes(), &created)
	id, _ := created["id"].(string)

	req := httptest.NewRequest(http.MethodPost, "/pod-terminal/sessions/"+id+"/terminate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "terminated" {
		t.Fatalf("expected terminated status, got %#v", resp)
	}
}

func TestTerminalCommandArgs(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{name: "single", in: "pwd", want: []string{"pwd"}},
		{name: "shell", in: "ls -lah /tmp", want: []string{"sh", "-lc", "ls -lah /tmp"}},
		{name: "empty", in: "  ", want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := terminalCommandArgs(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d: %#v", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got %#v, want %#v", got, tt.want)
				}
			}
		})
	}
}

func assertErrorCode(t *testing.T, body []byte, want string) {
	t.Helper()
	var resp map[string]any
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal error body: %v", err)
	}
	if resp["code"] != want {
		t.Fatalf("expected error code %s, got %v", want, resp["code"])
	}
}

// CYB-3568: a TTL'd/GC'd workflow is gone, not a server fault — the logs
// endpoint must return 404, not a bare 500.
func TestGetWorkflowLogs_WorkflowNotFoundReturns404(t *testing.T) {
	h := New(&mockWorkflowClient{
		getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
			return nil, argo.ErrNotFound
		},
	}, "default")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows/wf-gone/logs?nodeId=n1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for a gone workflow, got %d: %s", w.Code, w.Body.String())
	}
}

// CYB-3568: when the workflow survives but the pod was recycled, live log fetch
// returns ErrNotFound — surface an empty log window (200), not a 500.
func TestGetWorkflowLogs_PodRecycledReturnsEmpty(t *testing.T) {
	globalPodNameCache.set("wf-recycled/n1", "wf-recycled-n1-pod")
	h := New(&mockWorkflowClient{
		getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
			return &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "wf-recycled"}}, nil
		},
		logsFn: func(_ context.Context, _, _, _ string, _ argo.WorkflowLogOptions) (argo.WorkflowLogResult, error) {
			return argo.WorkflowLogResult{}, argo.ErrNotFound
		},
	}, "default")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows/wf-recycled/logs?nodeId=n1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with empty logs for a recycled pod, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["source"] != "unavailable" || resp["logs"] != "" {
		t.Fatalf("expected graceful empty logs (source=unavailable), got %v", resp)
	}
}

// CYB-3575: a Pending / not-yet-scheduled workflow has no pods yet, so node
// logs must return an empty window (200), not a 400 client error.
func TestGetWorkflowLogs_PendingWorkflowReturnsEmpty(t *testing.T) {
	h := New(&mockWorkflowClient{
		getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
			return &wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{Name: "wf-pending"},
				Status:     wfv1.WorkflowStatus{Phase: wfv1.WorkflowPending},
			}, nil
		},
	}, "default")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows/wf-pending/logs?nodeId=step-not-started", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 empty logs for a pending workflow, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["source"] != "pending" || resp["logs"] != "" {
		t.Fatalf("expected source=pending + empty logs, got %v", resp)
	}
}

// A genuinely unknown node id on a live (Running) workflow is still a 400.
func TestGetWorkflowLogs_UnknownNodeOnRunningWorkflowReturns400(t *testing.T) {
	h := New(&mockWorkflowClient{
		getFn: func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
			return &wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{Name: "wf-running"},
				Status:     wfv1.WorkflowStatus{Phase: wfv1.WorkflowRunning},
			}, nil
		},
	}, "default")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows/wf-running/logs?nodeId=bogus-node-xyz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for an unknown node on a running workflow, got %d: %s", w.Code, w.Body.String())
	}
}
