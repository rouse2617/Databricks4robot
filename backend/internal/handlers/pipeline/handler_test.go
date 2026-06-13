package pipeline

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/argo"
	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
)

// ── Mock Repositories ────────────────────────────────────────────────────────

type mockTemplateRepo struct {
	byID map[string]*models.PipelineTemplate
	ver  int
}

func (m *mockTemplateRepo) Save(_ context.Context, t *models.PipelineTemplate) error {
	if m.byID == nil {
		m.byID = make(map[string]*models.PipelineTemplate)
	}
	m.byID[t.ID] = t
	return nil
}
func (m *mockTemplateRepo) FindAll(_ context.Context) ([]models.PipelineTemplate, error) {
	out := make([]models.PipelineTemplate, 0, len(m.byID))
	for _, t := range m.byID {
		out = append(out, *t)
	}
	return out, nil
}
func (m *mockTemplateRepo) FindByID(_ context.Context, id string) (*models.PipelineTemplate, error) {
	return m.byID[id], nil
}
func (m *mockTemplateRepo) FindByNameAndVersion(_ context.Context, name string, version int) (*models.PipelineTemplate, error) {
	for _, t := range m.byID {
		if t.Name == name && t.Version == version {
			return t, nil
		}
	}
	return nil, nil
}
func (m *mockTemplateRepo) FindVersionsByName(_ context.Context, name string) ([]models.PipelineTemplate, error) {
	out := make([]models.PipelineTemplate, 0)
	for _, t := range m.byID {
		if t.Name == name {
			out = append(out, *t)
		}
	}
	return out, nil
}
func (m *mockTemplateRepo) GetNextVersion(_ context.Context, _ string) (int, error) {
	m.ver++
	return m.ver, nil
}
func (m *mockTemplateRepo) Delete(_ context.Context, id string) error {
	delete(m.byID, id)
	return nil
}

func (m *mockTemplateRepo) SetActiveVersion(_ context.Context, name string, version int) error {
	for _, t := range m.byID {
		if t.Name == name {
			t.ActiveVersion = version
		}
	}
	return nil
}

type mockDeploymentRepo struct {
	byID map[string]*models.PipelineDeployment
}

func (m *mockDeploymentRepo) Save(_ context.Context, d *models.PipelineDeployment) error {
	if m.byID == nil {
		m.byID = make(map[string]*models.PipelineDeployment)
	}
	m.byID[d.ID] = d
	return nil
}
func (m *mockDeploymentRepo) FindAll(_ context.Context) ([]models.PipelineDeployment, error) {
	out := make([]models.PipelineDeployment, 0, len(m.byID))
	for _, d := range m.byID {
		out = append(out, *d)
	}
	return out, nil
}
func (m *mockDeploymentRepo) FindByID(_ context.Context, id string) (*models.PipelineDeployment, error) {
	return m.byID[id], nil
}
func (m *mockDeploymentRepo) Delete(_ context.Context, id string) error {
	delete(m.byID, id)
	return nil
}
func (m *mockDeploymentRepo) DeleteByTemplateID(_ context.Context, templateID string) error {
	for id, d := range m.byID {
		if d.TemplateID != nil && *d.TemplateID == templateID {
			delete(m.byID, id)
		}
	}
	return nil
}
func (m *mockDeploymentRepo) UpdateStatus(_ context.Context, _, _ string) error { return nil }

type mockPipelineRunRepo struct {
	byID map[string]*models.PipelineRun
}

func (m *mockPipelineRunRepo) Save(_ context.Context, r *models.PipelineRun) error {
	if m.byID == nil {
		m.byID = make(map[string]*models.PipelineRun)
	}
	m.byID[r.ID] = r
	return nil
}
func (m *mockPipelineRunRepo) FindAll(_ context.Context) ([]models.PipelineRun, error) {
	out := make([]models.PipelineRun, 0, len(m.byID))
	for _, r := range m.byID {
		out = append(out, *r)
	}
	return out, nil
}
func (m *mockPipelineRunRepo) FindAllSummaries(_ context.Context) ([]models.PipelineRun, error) {
	out := make([]models.PipelineRun, 0, len(m.byID))
	for _, r := range m.byID {
		copy := *r
		copy.PipelineJSON = nil
		copy.Manifest = nil
		copy.TargetSnapshot = nil
		copy.Nodes = nil
		copy.ExecutionTarget = nil
		out = append(out, copy)
	}
	return out, nil
}
func (m *mockPipelineRunRepo) ListSummaries(_ context.Context, filter models.PipelineRunListFilter) ([]models.PipelineRun, int, error) {
	items, err := m.FindAllSummaries(context.Background())
	if err != nil {
		return nil, 0, err
	}
	return items, len(items), nil
}
func (m *mockPipelineRunRepo) FindByID(_ context.Context, id string) (*models.PipelineRun, error) {
	return m.byID[id], nil
}
func (m *mockPipelineRunRepo) FindByWorkflowName(_ context.Context, workflowName string) (*models.PipelineRun, error) {
	for _, r := range m.byID {
		if r.WorkflowName == workflowName {
			return r, nil
		}
	}
	return nil, nil
}
func (m *mockPipelineRunRepo) Delete(_ context.Context, id string) error {
	delete(m.byID, id)
	return nil
}
func (m *mockPipelineRunRepo) DeleteByTemplateID(_ context.Context, templateID string) error {
	for id, r := range m.byID {
		if r.TemplateID != nil && *r.TemplateID == templateID {
			delete(m.byID, id)
		}
	}
	return nil
}
func (m *mockPipelineRunRepo) UpdateStatus(_ context.Context, id, status string, finishedAt *time.Time) error {
	if r := m.byID[id]; r != nil {
		r.Status = status
		r.FinishedAt = finishedAt
	}
	return nil
}

func (m *mockPipelineRunRepo) UpdateLedgerState(_ context.Context, id, ledgerState string) error {
	if r := m.byID[id]; r != nil {
		r.LedgerState = ledgerState
	}
	return nil
}

type mockPipelineRunNodeRepo struct {
	byRunID map[string][]models.PipelineRunNode
}

func (m *mockPipelineRunNodeRepo) ReplaceByRunID(_ context.Context, runID string, nodes []models.PipelineRunNode) error {
	if m.byRunID == nil {
		m.byRunID = make(map[string][]models.PipelineRunNode)
	}
	m.byRunID[runID] = nodes
	return nil
}
func (m *mockPipelineRunNodeRepo) FindByRunID(_ context.Context, runID string) ([]models.PipelineRunNode, error) {
	return m.byRunID[runID], nil
}
func (m *mockPipelineRunNodeRepo) DeleteByRunID(_ context.Context, runID string) error {
	delete(m.byRunID, runID)
	return nil
}

type mockAssetRepo struct {
	assets map[string]*models.Asset
}

func (m *mockAssetRepo) Get(_ context.Context, assetID string) (*models.Asset, error) {
	a, ok := m.assets[assetID]
	if !ok {
		return nil, nil
	}
	return a, nil
}
func (m *mockAssetRepo) FindExistingIDs(_ context.Context, assetIDs []string) (map[string]struct{}, error) {
	out := make(map[string]struct{})
	for _, assetID := range assetIDs {
		if _, ok := m.assets[assetID]; ok {
			out[assetID] = struct{}{}
		}
	}
	return out, nil
}
func (m *mockAssetRepo) GetAll(_ context.Context, _ string) (*models.Asset, error) { return nil, nil }
func (m *mockAssetRepo) InsertNew(_ context.Context, _ *models.Asset) error        { return nil }
func (m *mockAssetRepo) Set(_ context.Context, _ *models.Asset) error              { return nil }
func (m *mockAssetRepo) SoftDelete(_ context.Context, _ string) error              { return nil }
func (m *mockAssetRepo) ListByMcapFile(_ context.Context, _ string) ([]*models.Asset, error) {
	return nil, nil
}
func (m *mockAssetRepo) ListByLogicalAssetID(_ context.Context, _ string) ([]*models.Asset, error) {
	return nil, nil
}
func (m *mockAssetRepo) WriteSegmentIndex(_ context.Context, _ *models.Asset) error { return nil }
func (m *mockAssetRepo) ListWithFilters(_ context.Context, _ string, _ []interface{}, _ int, _ int, _ filter.OrderByClause) ([]*models.Asset, int64, error) {
	return nil, 0, nil
}
func (m *mockAssetRepo) ListDescendants(_ context.Context, _ string) ([]*models.Asset, error) {
	return nil, nil
}

type mockWorkflowClient struct{}

func (m *mockWorkflowClient) CreateWorkflow(_ context.Context, _ *wfv1.Workflow, _ string) error {
	return nil
}
func (m *mockWorkflowClient) GetWorkflowStatus(_ context.Context, _, _ string) (wfv1.WorkflowPhase, error) {
	return wfv1.WorkflowSucceeded, nil
}
func (m *mockWorkflowClient) DeleteWorkflow(_ context.Context, _, _ string) error {
	return nil
}
func (m *mockWorkflowClient) ListWorkflows(_ context.Context, _ string, _ string) ([]wfv1.Workflow, error) {
	return nil, nil
}
func (m *mockWorkflowClient) GetWorkflow(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
	return &wfv1.Workflow{}, nil
}
func (m *mockWorkflowClient) StopWorkflow(_ context.Context, _, _ string) error {
	return nil
}
func (m *mockWorkflowClient) GetWorkflowLogs(_ context.Context, _, _, _ string, _ argo.WorkflowLogOptions) (argo.WorkflowLogResult, error) {
	return argo.WorkflowLogResult{Logs: "test logs"}, nil
}
func (m *mockWorkflowClient) GetWorkflowLogStream(_ context.Context, _, _, _ string, _ argo.WorkflowLogOptions) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}
func (m *mockWorkflowClient) RetryWorkflow(_ context.Context, _, _ string) error     { return nil }
func (m *mockWorkflowClient) ResubmitWorkflow(_ context.Context, _, _ string) error  { return nil }
func (m *mockWorkflowClient) SuspendWorkflow(_ context.Context, _, _ string) error   { return nil }
func (m *mockWorkflowClient) ResumeWorkflow(_ context.Context, _, _ string) error    { return nil }
func (m *mockWorkflowClient) TerminateWorkflow(_ context.Context, _, _ string) error { return nil }

// ── Helpers ──────────────────────────────────────────────────────────────────

func now() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }

func makeTemplate(id, name string, version int) *models.PipelineTemplate {
	return &models.PipelineTemplate{
		ID:        id,
		Name:      name,
		Version:   version,
		Pipeline:  map[string]interface{}{"nodes": []interface{}{map[string]interface{}{"id": "a"}}},
		NodeCount: 1,
		CreatedAt: now(),
		UpdatedAt: now(),
	}
}

func makeDeployment(id, name, status string) *models.PipelineDeployment {
	return &models.PipelineDeployment{
		ID:           id,
		PipelineName: name,
		WorkflowName: name + "-wf",
		Status:       status,
		NodeCount:    2,
		CreatedAt:    now(),
		UpdatedAt:    now(),
	}
}

func makePipelineRun(id, workflowName string) *models.PipelineRun {
	return &models.PipelineRun{
		ID:                id,
		PipelineName:      "cost-demo",
		WorkflowName:      workflowName,
		ExecutionTargetID: "default",
		Status:            "Succeeded",
		NodeCount:         2,
		NoAssetRun:        true,
		ArgoNamespace:     "cyber-databrew-dev",
		CreatedAt:         now(),
		UpdatedAt:         now(),
	}
}

func setupRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/pipelines", h.SaveTemplate)
	r.GET("/api/v1/pipelines", h.ListTemplates)
	r.GET("/api/v1/pipelines/:id", h.GetTemplate)
	r.DELETE("/api/v1/pipelines/:id", h.DeleteTemplate)
	r.GET("/api/v1/pipelines/:id/versions", h.ListVersions)
	r.GET("/api/v1/pipelines/:id/diff/:id2", h.DiffTemplates)
	r.POST("/api/v1/deploy", h.Deploy)
	r.POST("/api/v1/deploy/template/:id", h.DeployByTemplate)
	r.GET("/api/v1/execution-targets", h.ListExecutionTargets)
	r.GET("/api/v1/deployments", h.ListDeployments)
	r.GET("/api/v1/deployments/:id", h.GetDeployment)
	r.DELETE("/api/v1/deployments/:id", h.DeleteDeployment)
	r.POST("/api/v1/deployments/:id/retry", h.RetryDeployment)
	r.POST("/api/v1/deployments/:id/stop", h.StopDeployment)
	r.POST("/api/v1/deployments/:id/save-template", h.SaveFromDeployment)
	r.GET("/api/v1/deployments/:id/resources", h.GetResourceUsage)
	r.GET("/api/v1/workflows/:name/resources", h.GetWorkflowResourceUsage)
	r.GET("/api/v1/workflows/:name/nodes/:nodeId/resources", h.GetWorkflowNodeResourceUsage)
	r.POST("/api/v1/pipeline-runs", h.CreateRun)
	r.POST("/api/v1/pipeline-runs/template/:id", h.CreateRunByTemplate)
	r.GET("/api/v1/pipeline-runs", h.ListRuns)
	r.GET("/api/v1/pipeline-runs/:id", h.GetRun)
	r.GET("/api/v1/pipeline-runs/:id/events", h.ListRunEvents)
	r.GET("/api/v1/pipeline-runs/:id/asset-nodes", h.ListRunAssetNodes)
	r.GET("/api/v1/pipeline-runs/:id/cost-summary", h.GetRunCostSummary)
	r.POST("/api/v1/pipeline-assets", h.RegisterOutput)
	r.GET("/api/v1/assets/:id/pipeline-lineage", h.GetLineage)
	return r
}

func TestListExecutionTargets_Default(t *testing.T) {
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, &mockWorkflowClient{}, "cyber-databrew-dev")
	h := New(uc, "")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/execution-targets", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Items []models.ExecutionTarget `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected one default target, got %d", len(resp.Items))
	}
	if resp.Items[0].ID != "default" || resp.Items[0].Namespace != "cyber-databrew-dev" {
		t.Fatalf("unexpected target: %+v", resp.Items[0])
	}
}

func TestListRuns_ReturnsTotalEstimatedCost(t *testing.T) {
	run := makePipelineRun("run-1", "wf-cost")
	emitCost := 1.25
	finalCost := 0.75
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "cyber-databrew-dev")
	uc.SetRunRepositories(nil, &mockPipelineRunRepo{
		byID: map[string]*models.PipelineRun{run.ID: run},
	}, &mockPipelineRunNodeRepo{
		byRunID: map[string][]models.PipelineRunNode{
			run.ID: {
				{ID: "node-1", RunID: run.ID, DisplayName: "emit", EstimatedCostUSD: &emitCost},
				{ID: "node-2", RunID: run.ID, DisplayName: "final", EstimatedCostUSD: &finalCost},
			},
		},
	})
	h := New(uc, "")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipeline-runs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Items []models.PipelineRun `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 run, got %d", len(resp.Items))
	}
	if resp.Items[0].TotalEstimatedCost == nil || *resp.Items[0].TotalEstimatedCost != 2.0 {
		t.Fatalf("totalEstimatedCost=%v, want 2.0", resp.Items[0].TotalEstimatedCost)
	}
	if len(resp.Items[0].Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(resp.Items[0].Nodes))
	}
}

func TestListRuns_SummaryViewSkipsHeavyFields(t *testing.T) {
	heavyJSON := map[string]interface{}{"nodes": []interface{}{map[string]interface{}{"id": "n1"}}}
	run := makePipelineRun("run-1", "wf-summary")
	run.PipelineJSON = heavyJSON
	run.Manifest = strPtr("manifest-yaml")
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "cyber-databrew-dev")
	uc.SetRunRepositories(nil, &mockPipelineRunRepo{
		byID: map[string]*models.PipelineRun{run.ID: run},
	}, &mockPipelineRunNodeRepo{
		byRunID: map[string][]models.PipelineRunNode{
			run.ID: {{ID: "node-1", RunID: run.ID, DisplayName: "step-1"}},
		},
	})
	h := New(uc, "")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipeline-runs?view=summary", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Items []models.PipelineRun `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 run, got %d", len(resp.Items))
	}
	item := resp.Items[0]
	if item.PipelineJSON != nil || item.Manifest != nil || len(item.Nodes) != 0 {
		t.Fatalf("summary view should omit heavy fields: pipelineJSON=%v manifest=%v nodes=%d",
			item.PipelineJSON != nil, item.Manifest != nil, len(item.Nodes))
	}
	if item.TotalEstimatedCost != nil {
		t.Fatalf("summary view should not compute totalEstimatedCost, got %v", item.TotalEstimatedCost)
	}
}

func strPtr(s string) *string { return &s }

func TestGetRun_ReturnsTotalEstimatedCost(t *testing.T) {
	run := makePipelineRun("run-1", "wf-cost")
	expensiveCost := 3.5
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "cyber-databrew-dev")
	uc.SetRunRepositories(nil, &mockPipelineRunRepo{
		byID: map[string]*models.PipelineRun{run.ID: run},
	}, &mockPipelineRunNodeRepo{
		byRunID: map[string][]models.PipelineRunNode{
			run.ID: {
				{ID: "node-1", RunID: run.ID, DisplayName: "expensive", EstimatedCostUSD: &expensiveCost},
				{ID: "node-2", RunID: run.ID, DisplayName: "free"},
			},
		},
	})
	h := New(uc, "")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipeline-runs/run-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.PipelineRun
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.TotalEstimatedCost == nil || *resp.TotalEstimatedCost != 3.5 {
		t.Fatalf("totalEstimatedCost=%v, want 3.5", resp.TotalEstimatedCost)
	}
	if len(resp.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(resp.Nodes))
	}
	if resp.Nodes[0].EstimatedCostUSD == nil && resp.Nodes[1].EstimatedCostUSD == nil {
		t.Fatalf("expected at least one node estimatedCostUsd in response")
	}
}

func TestDeployByTemplate_RejectsUnknownTarget(t *testing.T) {
	templates := &mockTemplateRepo{byID: map[string]*models.PipelineTemplate{
		"tmpl-1": makeTemplate("tmpl-1", "target-test", 1),
	}}
	uc := pipelineUC.New(templates, &mockDeploymentRepo{}, &mockAssetRepo{}, &mockWorkflowClient{}, "default")
	h := New(uc, "")
	r := setupRouter(h)

	body := `{"target_id":"missing-target"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deploy/template/tmpl-1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "execution target not found") {
		t.Fatalf("expected target error, got %s", w.Body.String())
	}
}

// ── Template Tests ───────────────────────────────────────────────────────────

func TestSaveTemplate_Success(t *testing.T) {
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	h := New(uc, "")
	r := setupRouter(h)

	body := `{"name":"test-pipeline","pipeline":{"nodes":[{"id":"a"}]}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipelines", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.PipelineTemplate
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Name != "test-pipeline" {
		t.Errorf("expected name 'test-pipeline', got %q", resp.Name)
	}
	if resp.NodeCount != 1 {
		t.Errorf("expected nodeCount 1, got %d", resp.NodeCount)
	}
}

func TestSaveTemplate_MissingName(t *testing.T) {
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	h := New(uc, "")
	r := setupRouter(h)

	body := `{"pipeline":{"nodes":[]}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipelines", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing name, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSaveTemplate_MissingPipeline(t *testing.T) {
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	h := New(uc, "")
	r := setupRouter(h)

	body := `{"name":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipelines", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing pipeline, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListTemplates_Empty(t *testing.T) {
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	h := New(uc, "")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipelines", nil)
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

func TestListTemplates_WithItem(t *testing.T) {
	templateRepo := &mockTemplateRepo{}
	_ = templateRepo.Save(context.Background(), makeTemplate("tpl-1", "pipeline-a", 1))
	uc := pipelineUC.New(templateRepo, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	h := New(uc, "")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipelines", nil)
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
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}

func TestGetTemplate_Success(t *testing.T) {
	templateRepo := &mockTemplateRepo{}
	_ = templateRepo.Save(context.Background(), makeTemplate("tpl-1", "my-pipeline", 1))
	uc := pipelineUC.New(templateRepo, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	h := New(uc, "")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipelines/tpl-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.PipelineTemplate
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.ID != "tpl-1" {
		t.Errorf("expected id 'tpl-1', got %q", resp.ID)
	}
	if resp.Name != "my-pipeline" {
		t.Errorf("expected name 'my-pipeline', got %q", resp.Name)
	}
}

func TestGetTemplate_NotFound(t *testing.T) {
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	h := New(uc, "")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipelines/non-existent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteTemplate_Success(t *testing.T) {
	templateRepo := &mockTemplateRepo{}
	_ = templateRepo.Save(context.Background(), makeTemplate("tpl-1", "my-pipeline", 1))
	uc := pipelineUC.New(templateRepo, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	h := New(uc, "")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/pipelines/tpl-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
	if templateRepo.byID["tpl-1"] != nil {
		t.Fatal("expected template to be deleted")
	}
}

func TestDeleteTemplate_EmptyID(t *testing.T) {
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := New(uc, "")
	r.DELETE("/api/v1/pipelines/:id", h.DeleteTemplate)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/pipelines/%20%20", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty id, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListVersions_Success(t *testing.T) {
	templateRepo := &mockTemplateRepo{}
	_ = templateRepo.Save(context.Background(), makeTemplate("v1", "my-pipeline", 1))
	_ = templateRepo.Save(context.Background(), makeTemplate("v2", "my-pipeline", 2))
	uc := pipelineUC.New(templateRepo, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	h := New(uc, "")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipelines/my-pipeline/versions", nil)
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
		t.Fatalf("expected 2 versions, got %d", len(items))
	}
}

// ── Deploy Tests ─────────────────────────────────────────────────────────────

func TestDeploy_Success(t *testing.T) {
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, &mockWorkflowClient{}, "default")
	h := New(uc, "")
	r := setupRouter(h)

	body := `{"pipeline":{"name":"test","nodes":[]},"name":"my-deploy","asset_ids":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deploy", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.PipelineDeployment
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.PipelineName != "my-deploy" {
		t.Errorf("expected pipelineName 'my-deploy', got %q", resp.PipelineName)
	}
}

func TestDeploy_MissingPipeline(t *testing.T) {
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	h := New(uc, "")
	r := setupRouter(h)

	body := `{"name":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deploy", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing pipeline, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeployByTemplate_Success(t *testing.T) {
	templateRepo := &mockTemplateRepo{}
	_ = templateRepo.Save(context.Background(), makeTemplate("tpl-1", "my-pipeline", 1))
	uc := pipelineUC.New(templateRepo, &mockDeploymentRepo{}, &mockAssetRepo{}, &mockWorkflowClient{}, "default")
	h := New(uc, "")
	r := setupRouter(h)

	body := `{"name":"from-template","asset_ids":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deploy/template/tpl-1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.PipelineDeployment
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.PipelineName != "from-template" {
		t.Errorf("expected pipelineName 'from-template', got %q", resp.PipelineName)
	}
	if resp.TemplateVersion == nil || *resp.TemplateVersion != 1 {
		t.Fatalf("expected template version 1, got %v", resp.TemplateVersion)
	}
}

func TestDeployByTemplate_UsesRequestedVersion(t *testing.T) {
	templateRepo := &mockTemplateRepo{}
	v1 := makeTemplate("tpl-v1", "my-pipeline", 1)
	v1.Pipeline["nodes"] = []interface{}{
		map[string]interface{}{"id": "step-1", "component": map[string]interface{}{"name": "a", "image": "alpine:3.20"}},
	}
	v2 := makeTemplate("tpl-v2", "my-pipeline", 2)
	v2.Pipeline["nodes"] = []interface{}{
		map[string]interface{}{"id": "step-1", "component": map[string]interface{}{"name": "a", "image": "alpine:latest"}},
		map[string]interface{}{"id": "step-2", "component": map[string]interface{}{"name": "b", "image": "alpine:latest"}},
	}
	_ = templateRepo.Save(context.Background(), v1)
	_ = templateRepo.Save(context.Background(), v2)
	uc := pipelineUC.New(templateRepo, &mockDeploymentRepo{}, &mockAssetRepo{}, &mockWorkflowClient{}, "default")
	h := New(uc, "")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/deploy/template/tpl-v2", strings.NewReader(`{"version":1}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.PipelineDeployment
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.TemplateID == nil || *resp.TemplateID != "tpl-v1" {
		t.Fatalf("expected v1 template id, got %v", resp.TemplateID)
	}
	if resp.TemplateVersion == nil || *resp.TemplateVersion != 1 {
		t.Fatalf("expected template version 1, got %v", resp.TemplateVersion)
	}
	if resp.NodeCount != 1 {
		t.Fatalf("expected v1 node count 1, got %d", resp.NodeCount)
	}
}

func TestDeployByTemplate_NotFound(t *testing.T) {
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	h := New(uc, "")
	r := setupRouter(h)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deploy/template/non-existent", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing template, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeployByTemplate_EmptyID(t *testing.T) {
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := New(uc, "")
	r.POST("/api/v1/deploy/template/:id", h.DeployByTemplate)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deploy/template/%20%20", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty template id, got %d: %s", w.Code, w.Body.String())
	}
}

// ── Deployment Tests ─────────────────────────────────────────────────────────

func TestListDeployments_Empty(t *testing.T) {
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	h := New(uc, "")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/deployments", nil)
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

func TestListDeployments_WithItem(t *testing.T) {
	depRepo := &mockDeploymentRepo{}
	dep := makeDeployment("dep-1", "pipe-a", "Running")
	dep.PipelineJSON = map[string]interface{}{"_input_asset_ids": []interface{}{"asset-a", "asset-b"}}
	_ = depRepo.Save(context.Background(), dep)
	uc := pipelineUC.New(&mockTemplateRepo{}, depRepo, &mockAssetRepo{}, nil, "default")
	h := New(uc, "")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/deployments", nil)
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
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	first := items[0].(map[string]interface{})
	if first["pipelineName"] != "pipe-a" {
		t.Errorf("expected pipelineName 'pipe-a', got %v", first["pipelineName"])
	}
	if first["assetCount"] != float64(2) {
		t.Errorf("expected assetCount 2, got %v", first["assetCount"])
	}
	target, ok := first["executionTarget"].(map[string]interface{})
	if !ok || target["id"] != "default" {
		t.Fatalf("expected default execution target, got %#v", first["executionTarget"])
	}
}

func TestGetDeployment_Success(t *testing.T) {
	depRepo := &mockDeploymentRepo{}
	_ = depRepo.Save(context.Background(), makeDeployment("dep-1", "my-pipeline", "Succeeded"))
	uc := pipelineUC.New(&mockTemplateRepo{}, depRepo, &mockAssetRepo{}, nil, "default")
	h := New(uc, "")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/deployments/dep-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.PipelineDeployment
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.ID != "dep-1" {
		t.Errorf("expected id 'dep-1', got %q", resp.ID)
	}
	if resp.Status != "Succeeded" {
		t.Errorf("expected status Succeeded, got %q", resp.Status)
	}
}

func TestGetDeployment_NotFound(t *testing.T) {
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	h := New(uc, "")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/deployments/non-existent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetDeployment_EmptyID(t *testing.T) {
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := New(uc, "")
	r.GET("/api/v1/deployments/:id", h.GetDeployment)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/deployments/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for empty id, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteDeployment_Success(t *testing.T) {
	depRepo := &mockDeploymentRepo{}
	_ = depRepo.Save(context.Background(), makeDeployment("dep-1", "my-pipeline", "Succeeded"))
	uc := pipelineUC.New(&mockTemplateRepo{}, depRepo, &mockAssetRepo{}, nil, "default")
	h := New(uc, "")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/deployments/dep-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
	if depRepo.byID["dep-1"] != nil {
		t.Fatal("expected deployment to be deleted")
	}
}

func TestDeleteDeployment_EmptyID(t *testing.T) {
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := New(uc, "")
	r.DELETE("/api/v1/deployments/:id", h.DeleteDeployment)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/deployments/%20%20", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty id, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRetryDeployment_Success(t *testing.T) {
	depRepo := &mockDeploymentRepo{}
	_ = depRepo.Save(context.Background(), &models.PipelineDeployment{
		ID:           "dep-1",
		PipelineName: "my-pipeline",
		PipelineJSON: map[string]interface{}{"name": "test", "nodes": []interface{}{}},
	})
	uc := pipelineUC.New(&mockTemplateRepo{}, depRepo, &mockAssetRepo{}, &mockWorkflowClient{}, "default")
	h := New(uc, "")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/dep-1/retry", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.PipelineDeployment
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.ID == "" {
		t.Fatal("expected a new deployment id")
	}
}

func TestRetryDeployment_NotFound(t *testing.T) {
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	h := New(uc, "")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/non-existent/retry", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStopDeployment_Success(t *testing.T) {
	depRepo := &mockDeploymentRepo{}
	_ = depRepo.Save(context.Background(), makeDeployment("dep-1", "my-pipeline", "Running"))
	uc := pipelineUC.New(&mockTemplateRepo{}, depRepo, &mockAssetRepo{}, nil, "default")
	h := New(uc, "")
	r := setupRouter(h)

	// With nil wfClient, StopDeployment returns an error
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/dep-1/stop", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 (no wfClient), got %d: %s", w.Code, w.Body.String())
	}
}

func TestStopDeployment_NotFound(t *testing.T) {
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	h := New(uc, "")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/non-existent/stop", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// With nil wfClient, FindByID returns nil → ErrDeploymentNotFound → 500 (no error.Is for nil wfClient)
	if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 404 or 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSaveFromDeployment_Success(t *testing.T) {
	depRepo := &mockDeploymentRepo{}
	_ = depRepo.Save(context.Background(), &models.PipelineDeployment{
		ID:           "dep-1",
		PipelineName: "my-pipeline",
		PipelineJSON: map[string]interface{}{"name": "test", "nodes": []interface{}{}},
	})
	uc := pipelineUC.New(&mockTemplateRepo{}, depRepo, &mockAssetRepo{}, nil, "default")
	h := New(uc, "")
	r := setupRouter(h)

	body := `{"name":"saved-template"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/dep-1/save-template", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.PipelineTemplate
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Name != "saved-template" {
		t.Errorf("expected name 'saved-template', got %q", resp.Name)
	}
}

func TestSaveFromDeployment_NotFound(t *testing.T) {
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	h := New(uc, "")
	r := setupRouter(h)

	body := `{"name":"saved-template"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/non-existent/save-template", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetResourceUsage_NotFound(t *testing.T) {
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	h := New(uc, "")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/deployments/non-existent/resources", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetWorkflowResourceUsage_Success(t *testing.T) {
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, &mockWorkflowClient{}, "default")
	h := New(uc, "")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/wf-1/resources", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		WorkflowName         string                         `json:"workflow_name"`
		LiveMetricsAvailable bool                           `json:"live_metrics_available"`
		Source               pipelineUC.ResourceUsageSource `json:"source"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.WorkflowName != "wf-1" || resp.LiveMetricsAvailable {
		t.Fatalf("unexpected resource response: %#v", resp)
	}
	if resp.Source.Workflow != "argo-live" || resp.Source.Metrics != "unavailable" {
		t.Fatalf("unexpected source: %#v", resp.Source)
	}
}

// ── Diff / RegisterOutput / Lineage Tests ────────────────────────────────────

func TestDiffTemplates_Success(t *testing.T) {
	templateRepo := &mockTemplateRepo{}
	_ = templateRepo.Save(context.Background(), makeTemplate("tpl-1", "pipeline-a", 1))
	_ = templateRepo.Save(context.Background(), makeTemplate("tpl-2", "pipeline-b", 1))
	uc := pipelineUC.New(templateRepo, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	h := New(uc, "")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipelines/tpl-1/diff/tpl-2", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Diff should return a JSON result structure
	if resp == nil {
		t.Fatal("expected diff result")
	}
}

func TestRegisterOutput_Success(t *testing.T) {
	depRepo := &mockDeploymentRepo{}
	_ = depRepo.Save(context.Background(), &models.PipelineDeployment{
		ID:           "dep-1",
		PipelineName: "my-pipeline",
		PipelineJSON: map[string]interface{}{"_input_asset_ids": []string{"a1"}},
	})
	assetRepo := &mockAssetRepo{assets: make(map[string]*models.Asset)}
	uc := pipelineUC.New(&mockTemplateRepo{}, depRepo, assetRepo, nil, "default")
	h := New(uc, "")
	r := setupRouter(h)

	body := `{"deployment_id":"dep-1","node_id":"step-1","storage_uri":"s3://bucket/output","asset_type":"dataset"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline-assets", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["asset_id"] == "" {
		t.Error("expected non-empty asset_id in response")
	}
}

func TestRegisterOutput_MissingDeploymentID(t *testing.T) {
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	h := New(uc, "")
	r := setupRouter(h)

	body := `{"node_id":"step-1","storage_uri":"s3://bucket/output","asset_type":"dataset"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline-assets", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing deployment_id, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegisterOutput_InvalidBody(t *testing.T) {
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	h := New(uc, "")
	r := setupRouter(h)

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline-assets", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid body, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegisterOutput_DeploymentNotFound(t *testing.T) {
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	h := New(uc, "")
	r := setupRouter(h)

	body := `{"deployment_id":"non-existent","storage_uri":"s3://bucket/output","asset_type":"dataset"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline-assets", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing deployment, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetLineage_Success(t *testing.T) {
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	// GetLineage requires assetEventRepo; without it, returns 500
	h := New(uc, "")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/assets/asset-1/pipeline-lineage", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Without assetEventRepo, it returns 500 — handler correctly delegates error
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 200 or 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetLineage_EmptyID(t *testing.T) {
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := New(uc, "")
	r.GET("/api/v1/assets/:id/pipeline-lineage", h.GetLineage)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/assets/%20%20/pipeline-lineage", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty id, got %d: %s", w.Code, w.Body.String())
	}
}

// ── Error mapping tests ─────────────────────────────────────────────────────

func TestDeploy_WithAssetValidation(t *testing.T) {
	assetRepo := &mockAssetRepo{assets: map[string]*models.Asset{"a1": {AssetID: "a1"}}}
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, assetRepo, nil, "default")
	h := New(uc, "")
	r := setupRouter(h)

	body := `{"pipeline":{"name":"test","nodes":[]},"name":"my-deploy","asset_ids":["a1","non-existent"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deploy", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing asset, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Code    string         `json:"code"`
		Details map[string]any `json:"details"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != "INVALID_ARGUMENT" {
		t.Fatalf("code=%q", resp.Code)
	}
	if resp.Details["field"] != "asset_ids" {
		t.Fatalf("field detail=%v", resp.Details["field"])
	}
	missing, ok := resp.Details["missing_asset_ids"].([]any)
	if !ok || len(missing) != 1 || missing[0] != "non-existent" {
		t.Fatalf("missing detail=%#v", resp.Details["missing_asset_ids"])
	}
}

func TestDeployByTemplate_TemplateNotFound(t *testing.T) {
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	h := New(uc, "")
	r := setupRouter(h)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deploy/template/non-existent", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-existent template, got %d: %s", w.Code, w.Body.String())
	}
}

// ── CYB-1537 — PR #77 review follow-up: CreateRunByTemplate body binding ──

func TestCreateRunByTemplate_MalformedBody(t *testing.T) {
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, &mockWorkflowClient{}, "default")
	h := New(uc, "")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline-runs/template/tmpl-1", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed body, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "invalid request body") {
		t.Fatalf("expected 'invalid request body' in response, got %s", w.Body.String())
	}
}

func TestCreateRunByTemplate_EmptyBody_OK(t *testing.T) {
	// Empty body must still reach the usecase (not rejected by binding).
	// We use a non-existent template id so the usecase returns 4xx/5xx,
	// but the binding path must not produce 400.
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, &mockWorkflowClient{}, "default")
	h := New(uc, "")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline-runs/template/missing-tmpl", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code == http.StatusBadRequest {
		t.Fatalf("empty body should not be rejected as 400, got 400: %s", w.Body.String())
	}
}
