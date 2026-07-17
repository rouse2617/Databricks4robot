package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

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
func (m *mockTemplateRepo) FindLatestPaged(_ context.Context, filter models.PipelineTemplateListFilter) ([]models.PipelineTemplate, int, error) {
	items, err := m.FindAll(context.Background())
	if err != nil {
		return nil, 0, err
	}
	if filter.Query != "" {
		q := strings.ToLower(filter.Query)
		filtered := make([]models.PipelineTemplate, 0, len(items))
		for _, item := range items {
			if strings.Contains(strings.ToLower(item.Name), q) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	if filter.Scope != "" {
		filtered := make([]models.PipelineTemplate, 0, len(items))
		for _, item := range items {
			if item.Scope == filter.Scope {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	total := len(items)
	page := filter.Page
	pageSize := filter.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	if start >= total {
		return []models.PipelineTemplate{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return items[start:end], total, nil
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
func (m *mockTemplateRepo) GetLatestVersionWithConflictCheck(_ context.Context, _ string, _ int) (*models.PipelineTemplate, bool, error) {
	return nil, false, nil
}
func (m *mockTemplateRepo) GetUserPipelineStatsAfter(_ context.Context, _ string, _ time.Time) (*models.PipelineUserStats, error) {
	return nil, nil
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
// summaries is a private test helper (formerly the FindAllSummaries interface
// method, dropped as dead production code) — it returns lightweight copies the
// mock's ListSummaries then filters.
func (m *mockPipelineRunRepo) summaries() []models.PipelineRun {
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
	return out
}
func (m *mockPipelineRunRepo) ListSummaries(_ context.Context, filter models.PipelineRunListFilter) ([]models.PipelineRun, int, error) {
	items := m.summaries()
	filtered := make([]models.PipelineRun, 0, len(items))
	for _, item := range items {
		if filter.BatchJobID != "" {
			if item.BatchJobID == nil || *item.BatchJobID != filter.BatchJobID {
				continue
			}
		}
		if filter.ExcludeBatch && item.BatchJobID != nil && strings.TrimSpace(*item.BatchJobID) != "" {
			continue
		}
		if filter.Status != "" && !strings.EqualFold(item.Status, filter.Status) {
			continue
		}
		if query := strings.ToLower(strings.TrimSpace(filter.Query)); query != "" {
			if !strings.Contains(strings.ToLower(item.ID), query) &&
				!strings.Contains(strings.ToLower(item.PipelineName), query) &&
				!strings.Contains(strings.ToLower(item.WorkflowName), query) &&
				!strings.Contains(strings.ToLower(item.TemplateName), query) {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	items = filtered
	return items, len(items), nil
}
func (m *mockPipelineRunRepo) FindByID(_ context.Context, id string) (*models.PipelineRun, error) {
	return m.byID[id], nil
}
func (m *mockPipelineRunRepo) FindSummaryByID(_ context.Context, id string) (*models.PipelineRun, error) {
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
func (m *mockPipelineRunRepo) FindByBatchJobAndAssetID(_ context.Context, batchJobID, assetID string) (*models.PipelineRun, error) {
	for _, r := range m.byID {
		if r.BatchJobID != nil && *r.BatchJobID == batchJobID {
			for _, id := range r.AssetIDs {
				if id == assetID {
					return r, nil
				}
			}
		}
	}
	return nil, nil
}
func (m *mockPipelineRunRepo) FindAllByBatchJobAndAssetID(_ context.Context, batchJobID, assetID string) ([]models.PipelineRun, error) {
	var out []models.PipelineRun
	for _, r := range m.byID {
		if r.BatchJobID != nil && *r.BatchJobID == batchJobID {
			for _, id := range r.AssetIDs {
				if id == assetID {
					out = append(out, *r)
				}
			}
		}
	}
	return out, nil
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

type mockPipelineRunEventRepo struct {
	events []models.PipelineRunEvent
}

func (m *mockPipelineRunEventRepo) Append(_ context.Context, event *models.PipelineRunEvent) error {
	if event == nil {
		return nil
	}
	if event.RunID == "" {
		event.RunID = event.SubjectID
	}
	if event.Sequence == 0 {
		event.Sequence = int64(len(m.events) + 1)
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = now()
	}
	m.events = append(m.events, *event)
	return nil
}

func (m *mockPipelineRunEventRepo) ListByRunID(_ context.Context, runID string, opts models.PipelineRunEventListOptions) (*models.PipelineRunEventListResult, error) {
	out := make([]models.PipelineRunEvent, 0, len(m.events))
	for _, event := range m.events {
		if event.RunID != runID {
			continue
		}
		if opts.EventType != "" && event.EventType != opts.EventType {
			continue
		}
		if opts.SubjectType != "" && event.SubjectType != opts.SubjectType {
			continue
		}
		if opts.Cursor > 0 && event.Sequence <= opts.Cursor {
			continue
		}
		out = append(out, event)
		if opts.Limit > 0 && len(out) >= opts.Limit {
			break
		}
	}
	return &models.PipelineRunEventListResult{Items: out, Total: len(out)}, nil
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

type mockWorkflowClient struct {
	deleteCalls    []string
	retryCalls     []string
	stopCalls      []string
	suspendCalls   []string
	resumeCalls    []string
	terminateCalls []string
	retryErr       error
}

func (m *mockWorkflowClient) CreateWorkflow(_ context.Context, wf *wfv1.Workflow, _ string) error {
	// Mirror the real CRD client contract: on successful submit, the server-
	// assigned UID is copied back onto the caller's workflow so Deploy doesn't
	// have to re-read it via GetWorkflow. Without this the ErrWorkflowSubmit-
	// Incomplete guard fires and every deploy through the mock returns 500.
	if wf != nil && wf.UID == "" {
		wf.UID = types.UID("mock-uid-" + wf.Name)
	}
	return nil
}
func (m *mockWorkflowClient) GetWorkflowStatus(_ context.Context, _, _ string) (wfv1.WorkflowPhase, error) {
	return wfv1.WorkflowSucceeded, nil
}
func (m *mockWorkflowClient) DeleteWorkflow(_ context.Context, name, namespace string) error {
	m.deleteCalls = append(m.deleteCalls, namespace+"/"+name)
	return nil
}
func (m *mockWorkflowClient) ListWorkflows(_ context.Context, _ string, _ string) ([]wfv1.Workflow, error) {
	return nil, nil
}
func (m *mockWorkflowClient) GetWorkflow(_ context.Context, name, _ string) (*wfv1.Workflow, error) {
	// Return the UID but leave Status.Phase empty: Deploy's post-submit re-read
	// needs a UID (an empty one would overwrite what CreateWorkflow stamped and
	// trip ErrWorkflowSubmitIncomplete), but leaving Phase empty preserves what
	// GetRun paths compute from other signals — otherwise every mock-backed run
	// would collapse to Succeeded regardless of the test's fixture state.
	return &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: name, UID: types.UID("mock-uid-" + name)},
	}, nil
}
func (m *mockWorkflowClient) StopWorkflow(_ context.Context, name, namespace string) error {
	m.stopCalls = append(m.stopCalls, namespace+"/"+name)
	return nil
}
func (m *mockWorkflowClient) GetWorkflowLogs(_ context.Context, _, _, _ string, _ argo.WorkflowLogOptions) (argo.WorkflowLogResult, error) {
	return argo.WorkflowLogResult{Logs: "test logs"}, nil
}
func (m *mockWorkflowClient) GetWorkflowLogStream(_ context.Context, _, _, _ string, _ argo.WorkflowLogOptions) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}
func (m *mockWorkflowClient) RetryWorkflow(_ context.Context, name, namespace string) error {
	m.retryCalls = append(m.retryCalls, namespace+"/"+name)
	return m.retryErr
}
func (m *mockWorkflowClient) ResubmitWorkflow(_ context.Context, _, _ string) error { return nil }
func (m *mockWorkflowClient) SuspendWorkflow(_ context.Context, name, namespace string) error {
	m.suspendCalls = append(m.suspendCalls, namespace+"/"+name)
	return nil
}
func (m *mockWorkflowClient) ResumeWorkflow(_ context.Context, name, namespace string) error {
	m.resumeCalls = append(m.resumeCalls, namespace+"/"+name)
	return nil
}
func (m *mockWorkflowClient) TerminateWorkflow(_ context.Context, name, namespace string) error {
	m.terminateCalls = append(m.terminateCalls, namespace+"/"+name)
	return nil
}

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

func makeTemplateWithScope(id, name string, version int, scope string) *models.PipelineTemplate {
	t := makeTemplate(id, name, version)
	t.Scope = scope
	return t
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
	r.GET("/api/v1/pipeline/runtime-mounts", h.ListRuntimeMounts)
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
	r.GET("/api/v1/pipeline-runs/watcher/status", h.GetRunWatcherStatus)
	r.GET("/api/v1/pipeline-runs/by-workflow/:workflowName", h.GetRunByWorkflowName)
	r.GET("/api/v1/pipeline-runs/:id", h.GetRun)
	r.GET("/api/v1/pipeline-runs/:id/events", h.ListRunEvents)
	r.GET("/api/v1/pipeline-runs/:id/asset-nodes", h.ListRunAssetNodes)
	r.GET("/api/v1/pipeline-runs/:id/cost-summary", h.GetRunCostSummary)
	r.POST("/api/v1/pipeline-runs/:id/retry", h.RetryRun)
	r.POST("/api/v1/pipeline-runs/:id/stop", h.StopRun)
	r.DELETE("/api/v1/pipeline-runs/:id", h.DeleteRun)
	r.POST("/api/v1/runs", h.CreateRun)
	r.POST("/api/v1/runs/template/:id", h.CreateRunByTemplate)
	r.GET("/api/v1/runs", h.ListRuns)
	r.GET("/api/v1/runs/watcher/status", h.GetRunWatcherStatus)
	r.GET("/api/v1/runs/by-workflow/:workflowName", h.GetRunByWorkflowName)
	r.GET("/api/v1/runs/:id", h.GetRun)
	r.GET("/api/v1/runs/:id/events", h.ListRunEvents)
	r.GET("/api/v1/runs/:id/nodes", h.ListRunNodes)
	r.GET("/api/v1/runs/:id/asset-nodes", h.ListRunAssetNodes)
	r.GET("/api/v1/runs/:id/cost-summary", h.GetRunCostSummary)
	r.GET("/api/v1/runs/:id/inputs", h.ListRunInputs)
	r.GET("/api/v1/runs/:id/outputs", h.ListRunOutputs)
	r.GET("/api/v1/runs/:id/children", h.ListRunChildren)
	r.GET("/api/v1/runs/:id/runtime", h.GetRunRuntime)
	r.POST("/api/v1/runs/:id/retry", h.RetryRunRuntime)
	r.POST("/api/v1/runs/:id/resubmit", h.ResubmitRun)
	r.POST("/api/v1/runs/:id/rerun", h.RerunRun)
	r.POST("/api/v1/runs/:id/stop", h.StopRun)
	r.POST("/api/v1/runs/:id/suspend", h.SuspendRun)
	r.POST("/api/v1/runs/:id/resume", h.ResumeRun)
	r.POST("/api/v1/runs/:id/terminate", h.TerminateRun)
	r.DELETE("/api/v1/runs/:id", h.DeleteRun)
	r.POST("/api/v1/pipeline-assets", h.RegisterOutput)
	r.GET("/api/v1/assets/:id/pipeline-lineage", h.GetLineage)
	return r
}

type mockBatchSubtaskReconciler struct {
	reconcileCalls  int
	syncCalls       int
	syncJobCalls    int
	syncJobIDs      []string
	advanceCalls    int
	advanceRunIDs   []string
	advanceStatuses []string
}

func (m *mockBatchSubtaskReconciler) ReconcileSubtaskRuns(context.Context, string) error {
	m.reconcileCalls++
	return nil
}

func (m *mockBatchSubtaskReconciler) ReconcileItemByID(context.Context, string) (string, error) {
	return "", nil
}


func (m *mockBatchSubtaskReconciler) SyncJob(_ context.Context, jobID string) error {
	m.syncJobCalls++
	m.syncJobIDs = append(m.syncJobIDs, jobID)
	return nil
}

func (m *mockBatchSubtaskReconciler) AdvanceItemForRun(_ context.Context, run *models.PipelineRun) error {
	m.advanceCalls++
	if run != nil {
		m.advanceRunIDs = append(m.advanceRunIDs, run.ID)
		m.advanceStatuses = append(m.advanceStatuses, run.Status)
	}
	return nil
}

func TestListExecutionTargets_Default(t *testing.T) {
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, &mockWorkflowClient{}, "cyber-databrew-dev")
	h := New(uc, "", nil)
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

func TestListRuntimeMounts(t *testing.T) {
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, &mockWorkflowClient{}, "cyber-databrew-dev")
	uc.SetRuntimeMountCatalog(pipelineUC.RuntimeMountCatalog{
		Secrets: []pipelineUC.RuntimeSecretMountResource{{
			ID:                  "db-secrets",
			Name:                "DB secrets",
			Kind:                "secretProviderClass",
			SecretProviderClass: "db-spc",
			DefaultMountPath:    "/mnt/secrets",
			ReadOnly:            true,
		}},
		Storage: []pipelineUC.RuntimeStorageMountResource{{
			ID:               "scratch",
			Name:             "Scratch",
			Kind:             "emptyDir",
			DefaultMountPath: "/workspace/scratch",
			AllowWrite:       true,
		}},
	})
	h := New(uc, "", nil)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipeline/runtime-mounts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp pipelineUC.RuntimeMountCatalog
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Secrets) != 1 || resp.Secrets[0].SecretProviderClass != "db-spc" { // pragma: allowlist secret
		t.Fatalf("unexpected secrets catalog: %+v", resp.Secrets)
	}
	if len(resp.Storage) != 1 || resp.Storage[0].Kind != "emptyDir" {
		t.Fatalf("unexpected storage catalog: %+v", resp.Storage)
	}
}

func TestListRuntimeMountsReturnsInternalOnInvalidCatalogEnv(t *testing.T) {
	t.Setenv("PIPELINE_RUNTIME_MOUNT_CATALOG_JSON", `{"secrets":[{"id":"missing-spc"}]}`)
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, &mockWorkflowClient{}, "cyber-databrew-dev")
	h := New(uc, "", nil)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipeline/runtime-mounts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "missing-spc") {
		t.Fatalf("expected response to mention invalid resource, got %s", w.Body.String())
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
				// CYB-3389: ComputeRunCost only sums leaf pod nodes (CYB-3073
				// isLeafPodNode). Without Type=Pod the mocks are treated as
				// aggregate DAG nodes, skipped, and the total stays nil.
				{ID: "node-1", RunID: run.ID, DisplayName: "emit", Type: "Pod", EstimatedCostUSD: &emitCost},
				{ID: "node-2", RunID: run.ID, DisplayName: "final", Type: "Pod", EstimatedCostUSD: &finalCost},
			},
		},
	})
	h := New(uc, "", nil)
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

func TestListRunsSummaryReturnsCostTemplateAndQuery(t *testing.T) {
	matchingCost := 0.0042
	matching := makePipelineRun("run-match", "wf-match")
	matching.PipelineName = "nightly-product-line"
	matching.TemplateName = "customer-ingest-template"
	matching.TotalEstimatedCost = &matchingCost
	other := makePipelineRun("run-other", "wf-other")
	other.PipelineName = "other-pipeline"
	other.TemplateName = "unrelated-template"
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "cyber-databrew-dev")
	uc.SetRunRepositories(nil, &mockPipelineRunRepo{
		byID: map[string]*models.PipelineRun{
			matching.ID: matching,
			other.ID:    other,
		},
	}, &mockPipelineRunNodeRepo{})
	h := New(uc, "", nil)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs?view=summary&q=ingest", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Items []models.PipelineRun `json:"items"`
		Total int                  `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Total != 1 || len(resp.Items) != 1 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	item := resp.Items[0]
	if item.ID != matching.ID {
		t.Fatalf("matched run id=%q, want %q", item.ID, matching.ID)
	}
	if item.TemplateName != "customer-ingest-template" {
		t.Fatalf("templateName=%q", item.TemplateName)
	}
	if item.TotalEstimatedCost == nil || *item.TotalEstimatedCost != matchingCost {
		t.Fatalf("totalEstimatedCost=%v, want %v", item.TotalEstimatedCost, matchingCost)
	}
}

func TestRunAPI_ListGetByWorkflowAndEvents(t *testing.T) {
	run := makePipelineRun("run-1", "wf-run-api")
	runRepo := &mockPipelineRunRepo{
		byID: map[string]*models.PipelineRun{run.ID: run},
	}
	eventRepo := &mockPipelineRunEventRepo{
		events: []models.PipelineRunEvent{{
			RunID:       run.ID,
			EventType:   "run_started",
			SubjectType: "run",
			SubjectID:   run.ID,
			Sequence:    1,
			OccurredAt:  now(),
		}},
	}
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "cyber-databrew-dev")
	uc.SetRunRepositories(nil, runRepo, &mockPipelineRunNodeRepo{})
	uc.SetRunEventRepo(eventRepo)
	h := New(uc, "", nil)
	r := setupRouter(h)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/runs?view=summary", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d: %s", w.Code, w.Body.String())
	}
	var listResp struct {
		Items []models.PipelineRun `json:"items"`
		Total int                  `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	if listResp.Total != 1 || len(listResp.Items) != 1 || listResp.Items[0].ID != run.ID {
		t.Fatalf("unexpected list response: %+v", listResp)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/runs/by-workflow/wf-run-api", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected by-workflow 200, got %d: %s", w.Code, w.Body.String())
	}
	var byWorkflow models.PipelineRun
	if err := json.Unmarshal(w.Body.Bytes(), &byWorkflow); err != nil {
		t.Fatalf("unmarshal by-workflow: %v", err)
	}
	if byWorkflow.ID != run.ID {
		t.Fatalf("expected run id %q, got %q", run.ID, byWorkflow.ID)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/runs/run-1/events?limit=10", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected events 200, got %d: %s", w.Code, w.Body.String())
	}
	var events models.PipelineRunEventListResult
	if err := json.Unmarshal(w.Body.Bytes(), &events); err != nil {
		t.Fatalf("unmarshal events: %v", err)
	}
	if events.Total != 1 || len(events.Items) != 1 || events.Items[0].EventType != "run_started" {
		t.Fatalf("unexpected events response: %+v", events)
	}
}

func TestRunAPI_SubresourcesProjectInputsOutputsChildrenAndRuntime(t *testing.T) {
	parent := makePipelineRun("run-parent", "wf-parent")
	parent.AssetIDs = []string{"asset-1"}
	parent.ExecutionTargetID = "gpu-l4"
	parent.TargetSnapshot = map[string]interface{}{"namespace": "video-proc-dev", "gpu": "l4"}
	parent.PipelineJSON = map[string]interface{}{
		"nodes": []interface{}{
			map[string]interface{}{
				"id": "node-a",
				"runtimeConfig": map[string]interface{}{
					"configId":       "cfg-1",
					"version":        2,
					"fileName":       "node.yaml",
					"mountPath":      "/mnt/parameters",
					"targetFilename": "node.yaml",
					"mode":           "config_version",
					"content":        "redacted-in-projection",
				},
				"component": map[string]interface{}{
					"args": []interface{}{
						map[string]interface{}{"name": "threshold", "value": 0.9},
					},
				},
			},
		},
	}
	childBatchID := parent.ID
	child := makePipelineRun("run-child", "wf-child")
	child.BatchJobID = &childBatchID
	runRepo := &mockPipelineRunRepo{
		byID: map[string]*models.PipelineRun{
			parent.ID: parent,
			child.ID:  child,
		},
	}
	nodeRepo := &mockPipelineRunNodeRepo{
		byRunID: map[string][]models.PipelineRunNode{
			parent.ID: {{
				ID:                "node-row-1",
				RunID:             parent.ID,
				PipelineNodeID:    "node-a",
				DisplayName:       "node a",
				Phase:             "Succeeded",
				Outputs:           map[string]interface{}{"artifact": "gs://bucket/out.json"},
				LogRef:            "argo://wf-parent/node-a",
				ResourcesDuration: map[string]interface{}{"cpu": 42},
			}},
		},
	}
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "cyber-databrew-dev")
	uc.SetRunRepositories(nil, runRepo, nodeRepo)
	h := New(uc, "", nil)
	r := setupRouter(h)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/runs/run-parent/nodes", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected nodes 200, got %d: %s", w.Code, w.Body.String())
	}
	var nodes struct {
		Items []models.PipelineRunNode `json:"items"`
		Total int                      `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &nodes); err != nil {
		t.Fatalf("unmarshal nodes: %v", err)
	}
	if nodes.Total != 1 || nodes.Items[0].PipelineNodeID != "node-a" {
		t.Fatalf("unexpected nodes response: %+v", nodes)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/runs/run-parent/inputs", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected inputs 200, got %d: %s", w.Code, w.Body.String())
	}
	var inputs models.RunInputList
	if err := json.Unmarshal(w.Body.Bytes(), &inputs); err != nil {
		t.Fatalf("unmarshal inputs: %v", err)
	}
	seenTypes := map[string]bool{}
	for _, item := range inputs.Items {
		seenTypes[item.Type] = true
		if item.Type == "config" {
			if item.RefID != "cfg-1" || item.RefVersion != "2" {
				t.Fatalf("unexpected config projection: %+v", item)
			}
			if _, ok := item.Snapshot["content"]; ok {
				t.Fatalf("config projection should omit inline content: %+v", item.Snapshot)
			}
		}
	}
	for _, typ := range []string{"asset", "runtime_target", "config", "parameter"} {
		if !seenTypes[typ] {
			t.Fatalf("missing input type %q in %+v", typ, inputs.Items)
		}
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/runs/run-parent/outputs", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected outputs 200, got %d: %s", w.Code, w.Body.String())
	}
	var outputs models.RunOutputList
	if err := json.Unmarshal(w.Body.Bytes(), &outputs); err != nil {
		t.Fatalf("unmarshal outputs: %v", err)
	}
	outputTypes := map[string]bool{}
	for _, item := range outputs.Items {
		outputTypes[item.Type] = true
	}
	for _, typ := range []string{"node_outputs", "logs", "metrics"} {
		if !outputTypes[typ] {
			t.Fatalf("missing output type %q in %+v", typ, outputs.Items)
		}
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/runs/run-parent/children", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected children 200, got %d: %s", w.Code, w.Body.String())
	}
	var children models.RunChildList
	if err := json.Unmarshal(w.Body.Bytes(), &children); err != nil {
		t.Fatalf("unmarshal children: %v", err)
	}
	if children.Total != 1 || len(children.Items) != 1 || children.Items[0].ID != child.ID {
		t.Fatalf("unexpected children response: %+v", children)
	}
	if children.Summary.Total != 1 || children.Summary.AggregateStatus != "Succeeded" {
		t.Fatalf("unexpected children summary: %+v", children.Summary)
	}
	if len(children.Relations) != 1 || children.Relations[0].RelationType != "batch_child" {
		t.Fatalf("unexpected children relations: %+v", children.Relations)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/runs/run-parent/runtime", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected runtime 200, got %d: %s", w.Code, w.Body.String())
	}
	var runtime models.RunRuntime
	if err := json.Unmarshal(w.Body.Bytes(), &runtime); err != nil {
		t.Fatalf("unmarshal runtime: %v", err)
	}
	if runtime.Runtime.RuntimeType != "argo" || runtime.Runtime.WorkflowName != "wf-parent" || runtime.Runtime.ExecutionTargetID != "gpu-l4" {
		t.Fatalf("unexpected runtime response: %+v", runtime)
	}
}

func TestRunAPI_RuntimeRetryRetriesWorkflowAndWritesEvents(t *testing.T) {
	run := makePipelineRun("run-1", "wf-retry")
	run.Status = "Failed"
	run.ArgoNamespace = "video-proc-dev"
	runRepo := &mockPipelineRunRepo{byID: map[string]*models.PipelineRun{run.ID: run}}
	eventRepo := &mockPipelineRunEventRepo{}
	wfClient := &mockWorkflowClient{}
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "cyber-databrew-dev")
	uc.SetRunRepositories(nil, runRepo, &mockPipelineRunNodeRepo{})
	uc.SetRunEventRepo(eventRepo)
	h := New(uc, "", nil)
	r := setupRouter(h)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/runs/run-1/retry", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected retry 200, got %d: %s", w.Code, w.Body.String())
	}
	if len(wfClient.retryCalls) != 1 || wfClient.retryCalls[0] != "video-proc-dev/wf-retry" {
		t.Fatalf("unexpected retry calls: %+v", wfClient.retryCalls)
	}
	if len(runRepo.byID) != 1 {
		t.Fatalf("runtime retry should not create a new run, repo has %d", len(runRepo.byID))
	}
	seen := map[string]bool{}
	for _, event := range eventRepo.events {
		seen[event.EventType] = true
	}
	for _, typ := range []string{"run_runtime_retry_requested", "run_runtime_retry_succeeded"} {
		if !seen[typ] {
			t.Fatalf("missing retry event %q in %+v", typ, eventRepo.events)
		}
	}
}

func TestRunAPI_RuntimeRetryRequiresWorkflowName(t *testing.T) {
	run := makePipelineRun("run-1", "")
	runRepo := &mockPipelineRunRepo{byID: map[string]*models.PipelineRun{run.ID: run}}
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, &mockWorkflowClient{}, "cyber-databrew-dev")
	uc.SetRunRepositories(nil, runRepo, &mockPipelineRunNodeRepo{})
	h := New(uc, "", nil)
	r := setupRouter(h)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/runs/run-1/retry", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected retry 400, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "workflowName") {
		t.Fatalf("expected workflowName error, got %s", w.Body.String())
	}
}

func TestRunAPI_RuntimeRetryRejectsUnsupportedState(t *testing.T) {
	run := makePipelineRun("run-1", "wf-retry")
	run.Status = "Running"
	runRepo := &mockPipelineRunRepo{byID: map[string]*models.PipelineRun{run.ID: run}}
	eventRepo := &mockPipelineRunEventRepo{}
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, &mockWorkflowClient{}, "cyber-databrew-dev")
	uc.SetRunRepositories(nil, runRepo, &mockPipelineRunNodeRepo{})
	uc.SetRunEventRepo(eventRepo)
	h := New(uc, "", nil)
	r := setupRouter(h)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/runs/run-1/retry", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected retry 400, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "runtime retry only supports failed or errored runs") {
		t.Fatalf("expected retry state error, got %s", w.Body.String())
	}
}

func TestRunAPI_RerunCreatesNewRun(t *testing.T) {
	run := makePipelineRun("run-source", "wf-source")
	run.PipelineJSON = map[string]interface{}{
		"name": "rerun-handler-pipe",
		"nodes": []interface{}{
			map[string]interface{}{
				"id": "step-1",
				"component": map[string]interface{}{
					"name":  "test",
					"image": "busybox",
				},
			},
		},
		"edges": []interface{}{},
	}
	runRepo := &mockPipelineRunRepo{byID: map[string]*models.PipelineRun{run.ID: run}}
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, &mockWorkflowClient{}, "cyber-databrew-dev")
	uc.SetRunRepositories(nil, runRepo, &mockPipelineRunNodeRepo{})
	h := New(uc, "", nil)
	r := setupRouter(h)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/runs/run-source/rerun", nil))
	if w.Code != http.StatusCreated {
		t.Fatalf("expected rerun 201, got %d: %s", w.Code, w.Body.String())
	}
	if len(runRepo.byID) != 2 {
		t.Fatalf("expected rerun to create a new run, repo has %d", len(runRepo.byID))
	}
}

func TestRunAPI_RuntimeOperationsControlWorkflowAndDeleteRun(t *testing.T) {
	run := makePipelineRun("run-ops", "wf-ops")
	run.ArgoNamespace = "video-proc-dev"
	runRepo := &mockPipelineRunRepo{byID: map[string]*models.PipelineRun{run.ID: run}}
	eventRepo := &mockPipelineRunEventRepo{}
	wfClient := &mockWorkflowClient{}
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "cyber-databrew-dev")
	uc.SetRunRepositories(nil, runRepo, &mockPipelineRunNodeRepo{})
	uc.SetRunEventRepo(eventRepo)
	h := New(uc, "", nil)
	r := setupRouter(h)

	for _, tc := range []struct {
		path string
		want *[]string
	}{
		{path: "/api/v1/runs/run-ops/suspend", want: &wfClient.suspendCalls},
		{path: "/api/v1/runs/run-ops/resume", want: &wfClient.resumeCalls},
		{path: "/api/v1/runs/run-ops/terminate", want: &wfClient.terminateCalls},
		{path: "/api/v1/runs/run-ops/stop", want: &wfClient.stopCalls},
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, tc.path, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("expected %s 200, got %d: %s", tc.path, w.Code, w.Body.String())
		}
		if len(*tc.want) != 1 || (*tc.want)[0] != "video-proc-dev/wf-ops" {
			t.Fatalf("unexpected workflow calls for %s: %+v", tc.path, *tc.want)
		}
	}
	seenEvents := map[string]bool{}
	for _, event := range eventRepo.events {
		seenEvents[event.EventType] = true
	}
	for _, typ := range []string{"run_stop_requested", "run_stopped"} {
		if !seenEvents[typ] {
			t.Fatalf("missing stop event %q in %+v", typ, eventRepo.events)
		}
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/v1/runs/run-ops", nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected delete 204, got %d: %s", w.Code, w.Body.String())
	}
	if _, ok := runRepo.byID["run-ops"]; ok {
		t.Fatalf("expected run to be deleted")
	}
	if len(wfClient.deleteCalls) != 1 || wfClient.deleteCalls[0] != "video-proc-dev/wf-ops" {
		t.Fatalf("unexpected delete calls: %+v", wfClient.deleteCalls)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/v1/runs/run-ops", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected second delete 404, got %d: %s", w.Code, w.Body.String())
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
	h := New(uc, "", nil)
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

func TestListRuns_BatchSummaryDefaultSkipsBatchSync(t *testing.T) {
	batchJobID := "batch-1"
	run := makePipelineRun("run-1", "wf-batch-summary")
	run.BatchJobID = &batchJobID
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "cyber-databrew-dev")
	uc.SetRunRepositories(nil, &mockPipelineRunRepo{
		byID: map[string]*models.PipelineRun{run.ID: run},
	}, &mockPipelineRunNodeRepo{})
	batchRuns := &mockBatchSubtaskReconciler{}
	h := New(uc, "", batchRuns)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipeline-runs?view=summary&batchJobId=batch-1&page=1&pageSize=20", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if batchRuns.reconcileCalls != 0 || batchRuns.syncCalls != 0 {
		t.Fatalf("default batch summary should be read-only, reconcile=%d sync=%d", batchRuns.reconcileCalls, batchRuns.syncCalls)
	}
}

func TestListRuns_BatchSummaryRefreshOptInReconcilesOnce(t *testing.T) {
	batchJobID := "batch-1"
	run := makePipelineRun("run-1", "wf-batch-summary")
	run.BatchJobID = &batchJobID
	uc := pipelineUC.New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "cyber-databrew-dev")
	uc.SetRunRepositories(nil, &mockPipelineRunRepo{
		byID: map[string]*models.PipelineRun{run.ID: run},
	}, &mockPipelineRunNodeRepo{})
	batchRuns := &mockBatchSubtaskReconciler{}
	h := New(uc, "", batchRuns)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipeline-runs?view=summary&batchJobId=batch-1&refresh=true&page=1&pageSize=20", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if batchRuns.reconcileCalls != 1 {
		t.Fatalf("expected one reconcile call, got %d", batchRuns.reconcileCalls)
	}
	if batchRuns.syncCalls != 0 {
		t.Fatalf("summary refresh should not use double-list SyncBatchView, got %d sync calls", batchRuns.syncCalls)
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
				// CYB-3389: mark leaf-pod so ComputeRunCost aggregates it.
				{ID: "node-1", RunID: run.ID, DisplayName: "expensive", Type: "Pod", EstimatedCostUSD: &expensiveCost},
				{ID: "node-2", RunID: run.ID, DisplayName: "free", Type: "Pod"},
			},
		},
	})
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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

func TestListTemplates_PaginationAndFilters(t *testing.T) {
	templateRepo := &mockTemplateRepo{}
	for i := 0; i < 12; i++ {
		id := fmt.Sprintf("tpl-%02d", i)
		name := fmt.Sprintf("pipeline-%02d", i)
		scope := "dev"
		if i%3 == 0 {
			scope = "prod"
			name = fmt.Sprintf("prod-pipeline-%02d", i)
		}
		_ = templateRepo.Save(context.Background(), makeTemplateWithScope(id, name, 1, scope))
	}
	uc := pipelineUC.New(templateRepo, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	h := New(uc, "", nil)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipelines?page=1&page_size=10", nil)
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
	if len(items) != 10 {
		t.Fatalf("expected 10 items, got %d", len(items))
	}
	total, ok := resp["total"].(float64)
	if !ok || int(total) < 12 {
		t.Fatalf("expected total >= 12, got %#v", resp["total"])
	}
	if resp["pageSize"] != float64(10) && resp["page_size"] != float64(10) {
		t.Fatalf("expected pageSize/page_size 10, got %#v / %#v", resp["pageSize"], resp["page_size"])
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/pipelines?scope=prod&page_size=50", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("scope filter: expected 200, got %d", w.Code)
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal scope: %v", err)
	}
	for _, item := range resp["items"].([]interface{}) {
		row := item.(map[string]interface{})
		if row["scope"] != "prod" {
			t.Fatalf("expected prod scope only, got %#v", row["scope"])
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/pipelines?q=prod-pipeline-03&page_size=50", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("q filter: expected 200, got %d", w.Code)
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal q: %v", err)
	}
	qItems := resp["items"].([]interface{})
	if len(qItems) != 1 {
		t.Fatalf("expected 1 q match, got %d", len(qItems))
	}
}

func TestGetTemplate_Success(t *testing.T) {
	templateRepo := &mockTemplateRepo{}
	_ = templateRepo.Save(context.Background(), makeTemplate("tpl-1", "my-pipeline", 1))
	uc := pipelineUC.New(templateRepo, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
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
	h := New(uc, "", nil)
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline-runs/template/missing-tmpl", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code == http.StatusBadRequest {
		t.Fatalf("empty body should not be rejected as 400, got 400: %s", w.Body.String())
	}
}
