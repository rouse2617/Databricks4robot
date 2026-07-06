package workflow

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	sigsyaml "sigs.k8s.io/yaml"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

type mockPipelineRunNodeRepo struct {
	byRunID map[string][]models.PipelineRunNode
}

func (m *mockPipelineRunNodeRepo) ReplaceByRunID(context.Context, string, []models.PipelineRunNode) error {
	return nil
}
func (m *mockPipelineRunNodeRepo) FindByRunID(_ context.Context, runID string) ([]models.PipelineRunNode, error) {
	return m.byRunID[runID], nil
}
func (m *mockPipelineRunNodeRepo) DeleteByRunID(context.Context, string) error { return nil }

var _ repository.PipelineRunNodeRepository = (*mockPipelineRunNodeRepo)(nil)

// submissionTimeManifest marshals a minimal workflow spec the same way
// production code does (yaml.v3, no Status set), so the test exercises the
// real (de)serialization path instead of a hand-written YAML fixture.
func submissionTimeManifest(t *testing.T, workflowName string) string {
	t.Helper()
	wf := wfv1.Workflow{}
	wf.Name = workflowName
	wf.Spec.Entrypoint = "step-a"
	wf.Spec.Templates = []wfv1.Template{{
		Name:      "step-a",
		Container: &corev1.Container{Image: "busybox"},
	}}
	b, err := sigsyaml.Marshal(&wf)
	if err != nil {
		t.Fatalf("marshal manifest fixture: %v", err)
	}
	return string(b)
}

// Covers spec scenario: "已终态且底层对象仍存在的 run".
func TestGetWorkflow_TerminalRunUsesDBReconstruction(t *testing.T) {
	manifest := submissionTimeManifest(t, "wf-terminal")
	argoCalled := false
	client := &mockWorkflowClient{
		getFn: func(context.Context, string, string) (*wfv1.Workflow, error) {
			argoCalled = true
			return nil, errors.New("should not be called for a terminal run")
		},
	}
	h := New(client, "default")
	started := time.Now().Add(-time.Minute)
	finished := time.Now()
	h.SetRunRepositories(&mockRunRepo{byWorkflow: map[string]*models.PipelineRun{
		"wf-terminal": {
			ID:           "run-1",
			WorkflowName: "wf-terminal",
			Status:       "Succeeded",
			Manifest:     &manifest,
			StartedAt:    &started,
			FinishedAt:   &finished,
		},
	}}, nil, &mockPipelineRunNodeRepo{
		byRunID: map[string][]models.PipelineRunNode{
			"run-1": {
				{ArgoNodeID: "node-1", ArgoNodeName: "wf-terminal", DisplayName: "step-a", TemplateName: "step-a", Type: "Pod", Phase: "Succeeded"},
			},
		},
	})
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows/wf-terminal", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if argoCalled {
		t.Fatal("expected no call to wfClient.GetWorkflow for a terminal run with a reconstructible manifest")
	}
}

// Covers spec scenario: "活跃(未终态)run 保持原有行为".
func TestGetWorkflow_ActiveRunStillCallsArgo(t *testing.T) {
	manifest := submissionTimeManifest(t, "wf-active")
	argoCalled := false
	client := &mockWorkflowClient{
		getFn: func(_ context.Context, name, _ string) (*wfv1.Workflow, error) {
			argoCalled = true
			wf := &wfv1.Workflow{}
			wf.Name = name
			wf.Status.Phase = wfv1.WorkflowRunning
			return wf, nil
		},
	}
	h := New(client, "default")
	h.SetRunRepositories(&mockRunRepo{byWorkflow: map[string]*models.PipelineRun{
		"wf-active": {
			ID:           "run-2",
			WorkflowName: "wf-active",
			Status:       "Running",
			Manifest:     &manifest,
		},
	}}, nil, &mockPipelineRunNodeRepo{})
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows/wf-active", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !argoCalled {
		t.Fatal("expected an active run to still query Argo directly")
	}
}

// Covers spec scenario: "构造降级响应所需的历史数据缺失".
func TestGetWorkflow_ManifestMissingFallsBackToArgo(t *testing.T) {
	argoCalled := false
	client := &mockWorkflowClient{
		getFn: func(_ context.Context, name, _ string) (*wfv1.Workflow, error) {
			argoCalled = true
			wf := &wfv1.Workflow{}
			wf.Name = name
			wf.Status.Phase = wfv1.WorkflowSucceeded
			return wf, nil
		},
	}
	h := New(client, "default")
	h.SetRunRepositories(&mockRunRepo{byWorkflow: map[string]*models.PipelineRun{
		"wf-no-manifest": {
			ID:           "run-3",
			WorkflowName: "wf-no-manifest",
			Status:       "Succeeded",
			Manifest:     nil,
		},
	}}, nil, &mockPipelineRunNodeRepo{})
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/workflows/wf-no-manifest", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !argoCalled {
		t.Fatal("expected a fallback to Argo when the run has no reconstructible manifest")
	}
}
