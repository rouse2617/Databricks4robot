package pipeline

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/CyberOrigin2077/cyber-databrew/internal/argo"
	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/usecase/assetvalidation"
)

// ── Mocks ─────────────────────────────────────────────────────────────────

type mockTemplateRepo struct {
	saved  []*models.PipelineTemplate
	byID   map[string]*models.PipelineTemplate
	byName map[string][]models.PipelineTemplate
	ver    int
}

func (m *mockTemplateRepo) Save(_ context.Context, t *models.PipelineTemplate) error {
	m.saved = append(m.saved, t)
	if m.byID == nil {
		m.byID = make(map[string]*models.PipelineTemplate)
	}
	m.byID[t.ID] = t
	if m.byName == nil {
		m.byName = make(map[string][]models.PipelineTemplate)
	}
	m.byName[t.Name] = append(m.byName[t.Name], *t)
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
	for _, t := range m.byName[name] {
		if t.Version == version {
			copy := t
			return &copy, nil
		}
	}
	for _, t := range m.byID {
		if t.Name == name && t.Version == version {
			return t, nil
		}
	}
	return nil, nil
}
func (m *mockTemplateRepo) FindVersionsByName(_ context.Context, name string) ([]models.PipelineTemplate, error) {
	out := append([]models.PipelineTemplate(nil), m.byName[name]...)
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

type mockDeploymentRepo struct {
	saved []*models.PipelineDeployment
	byID  map[string]*models.PipelineDeployment
}

func (m *mockDeploymentRepo) Save(_ context.Context, d *models.PipelineDeployment) error {
	m.saved = append(m.saved, d)
	if m.byID == nil {
		m.byID = make(map[string]*models.PipelineDeployment)
	}
	m.byID[d.ID] = d
	return nil
}
func (m *mockDeploymentRepo) FindAll(_ context.Context) ([]models.PipelineDeployment, error) {
	out := make([]models.PipelineDeployment, len(m.saved))
	for i, d := range m.saved {
		out[i] = *d
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
	if m.byID != nil {
		for id, d := range m.byID {
			if d.TemplateID != nil && *d.TemplateID == templateID {
				delete(m.byID, id)
			}
		}
	}
	nextSaved := make([]*models.PipelineDeployment, 0, len(m.saved))
	for _, d := range m.saved {
		if d.TemplateID != nil && *d.TemplateID == templateID {
			continue
		}
		nextSaved = append(nextSaved, d)
	}
	m.saved = nextSaved
	return nil
}
func (m *mockDeploymentRepo) UpdateStatus(_ context.Context, id, status string) error {
	if d := m.byID[id]; d != nil {
		d.Status = status
	}
	for _, d := range m.saved {
		if d.ID == id {
			d.Status = status
		}
	}
	return nil
}

type mockAssetRepo struct {
	assets    map[string]*models.Asset
	findCalls int
	lastFind  []string
}

func (m *mockAssetRepo) Get(_ context.Context, assetID string) (*models.Asset, error) {
	a, ok := m.assets[assetID]
	if !ok {
		return nil, nil
	}
	return a, nil
}
func (m *mockAssetRepo) FindExistingIDs(_ context.Context, assetIDs []string) (map[string]struct{}, error) {
	m.findCalls++
	m.lastFind = append([]string(nil), assetIDs...)
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
	getWorkflowFn       func(ctx context.Context, name, namespace string) (*wfv1.Workflow, error)
	getWorkflowStatusFn func(ctx context.Context, name, namespace string) (wfv1.WorkflowPhase, error)
}

func (m *mockWorkflowClient) CreateWorkflow(_ context.Context, _ *wfv1.Workflow, _ string) error {
	return nil
}
func (m *mockWorkflowClient) GetWorkflowStatus(ctx context.Context, name, namespace string) (wfv1.WorkflowPhase, error) {
	if m.getWorkflowStatusFn != nil {
		return m.getWorkflowStatusFn(ctx, name, namespace)
	}
	return wfv1.WorkflowSucceeded, nil
}
func (m *mockWorkflowClient) DeleteWorkflow(_ context.Context, _, _ string) error {
	return nil
}
func (m *mockWorkflowClient) ListWorkflows(_ context.Context, _ string, _ string) ([]wfv1.Workflow, error) {
	return nil, nil
}
func (m *mockWorkflowClient) GetWorkflow(ctx context.Context, name, namespace string) (*wfv1.Workflow, error) {
	if m.getWorkflowFn != nil {
		return m.getWorkflowFn(ctx, name, namespace)
	}
	return &wfv1.Workflow{}, nil
}
func (m *mockWorkflowClient) StopWorkflow(_ context.Context, _, _ string) error {
	return nil
}
func (m *mockWorkflowClient) GetWorkflowLogs(_ context.Context, _, _, _ string, _ argo.WorkflowLogOptions) (argo.WorkflowLogResult, error) {
	return argo.WorkflowLogResult{}, nil
}
func (m *mockWorkflowClient) GetWorkflowLogStream(_ context.Context, _, _, _ string, _ argo.WorkflowLogOptions) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}
func (m *mockWorkflowClient) RetryWorkflow(_ context.Context, _, _ string) error     { return nil }
func (m *mockWorkflowClient) ResubmitWorkflow(_ context.Context, _, _ string) error  { return nil }
func (m *mockWorkflowClient) SuspendWorkflow(_ context.Context, _, _ string) error   { return nil }
func (m *mockWorkflowClient) ResumeWorkflow(_ context.Context, _, _ string) error    { return nil }
func (m *mockWorkflowClient) TerminateWorkflow(_ context.Context, _, _ string) error { return nil }

func newMockAssetRepo() *mockAssetRepo {
	return &mockAssetRepo{assets: make(map[string]*models.Asset)}
}

func newUsecase(assetRepo *mockAssetRepo) *Usecase {
	return &Usecase{
		templateRepo:   &mockTemplateRepo{byID: make(map[string]*models.PipelineTemplate)},
		deploymentRepo: &mockDeploymentRepo{},
		assetRepo:      assetRepo,
		wfClient:       &mockWorkflowClient{},
	}
}

// ── Tests ─────────────────────────────────────────────────────────────────

func TestDeploy_ValidatesAssetExistence(t *testing.T) {
	ctx := context.Background()
	pipe := map[string]interface{}{
		"name": "test-pipe",
		"nodes": []interface{}{
			map[string]interface{}{
				"id": "step-1",
				"component": map[string]interface{}{
					"name": "test", "image": "busybox",
				},
			},
		},
		"edges": []interface{}{},
	}

	t.Run("missing asset returns ErrAssetNotFound", func(t *testing.T) {
		repo := newMockAssetRepo()
		uc := newUsecase(repo)
		// No assets in repo — asset "nonexistent" will not be found.
		_, err := uc.Deploy(ctx, pipe, "", []string{"nonexistent"})
		if err == nil {
			t.Fatal("expected error for missing asset, got nil")
		}
		if !errors.Is(err, ErrAssetNotFound) {
			t.Fatalf("expected ErrAssetNotFound, got: %v", err)
		}
	})

	t.Run("valid asset passes validation", func(t *testing.T) {
		repo := newMockAssetRepo()
		repo.assets["asset-1"] = &models.Asset{
			AssetID:    "asset-1",
			AssetType:  "dataset",
			StorageURI: "gs://bucket/data",
		}
		repo.assets["asset-2"] = &models.Asset{
			AssetID:    "asset-2",
			AssetType:  "model",
			StorageURI: "gs://bucket/model",
		}
		uc := newUsecase(repo)

		dep, err := uc.Deploy(ctx, pipe, "", []string{"asset-1", "asset-2"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dep == nil {
			t.Fatal("expected deployment, got nil")
		}
		if dep.PipelineName != "test-pipe" {
			t.Fatalf("expected pipeline name 'test-pipe', got %q", dep.PipelineName)
		}
		if repo.findCalls != 1 {
			t.Fatalf("expected one batch asset lookup, got %d", repo.findCalls)
		}
	})

	t.Run("no asset IDs skips validation", func(t *testing.T) {
		repo := newMockAssetRepo()
		uc := newUsecase(repo)

		dep, err := uc.Deploy(ctx, pipe, "", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dep == nil {
			t.Fatal("expected deployment, got nil")
		}
	})

	t.Run("first invalid asset among many triggers error", func(t *testing.T) {
		repo := newMockAssetRepo()
		repo.assets["valid-1"] = &models.Asset{AssetID: "valid-1"}
		uc := newUsecase(repo)

		_, err := uc.Deploy(ctx, pipe, "", []string{"valid-1", "missing-2"})
		if err == nil {
			t.Fatal("expected error for missing asset, got nil")
		}
		if !errors.Is(err, ErrAssetNotFound) {
			t.Fatalf("expected ErrAssetNotFound, got: %v", err)
		}
		var validationErr *assetvalidation.ValidationError
		if !errors.As(err, &validationErr) {
			t.Fatalf("expected validation error details, got: %v", err)
		}
		if len(validationErr.MissingIDs) != 1 || validationErr.MissingIDs[0] != "missing-2" {
			t.Fatalf("missing IDs = %v", validationErr.MissingIDs)
		}
		if repo.findCalls != 1 || strings.Join(repo.lastFind, ",") != "valid-1,missing-2" {
			t.Fatalf("expected one batch lookup for both IDs, calls=%d ids=%v", repo.findCalls, repo.lastFind)
		}
	})

	t.Run("duplicate asset ID returns validation details", func(t *testing.T) {
		repo := newMockAssetRepo()
		repo.assets["asset-1"] = &models.Asset{AssetID: "asset-1"}
		uc := newUsecase(repo)

		_, err := uc.Deploy(ctx, pipe, "", []string{"asset-1", "asset-1"})
		if err == nil {
			t.Fatal("expected error for duplicate asset, got nil")
		}
		var validationErr *assetvalidation.ValidationError
		if !errors.As(err, &validationErr) {
			t.Fatalf("expected validation error details, got: %v", err)
		}
		if len(validationErr.DuplicateIDs) != 1 || validationErr.DuplicateIDs[0] != "asset-1" {
			t.Fatalf("duplicate IDs = %v", validationErr.DuplicateIDs)
		}
	})
}

func TestAssetIDsFromPipelineJSON(t *testing.T) {
	t.Run("string slice", func(t *testing.T) {
		got := assetIDsFromPipelineJSON(map[string]interface{}{
			"_input_asset_ids": []string{"a1", "a2"},
		})
		if len(got) != 2 || got[0] != "a1" || got[1] != "a2" {
			t.Fatalf("unexpected: %#v", got)
		}
	})

	t.Run("interface slice", func(t *testing.T) {
		got := assetIDsFromPipelineJSON(map[string]interface{}{
			"_input_asset_ids": []interface{}{"a1", "a2"},
		})
		if len(got) != 2 {
			t.Fatalf("unexpected: %#v", got)
		}
	})
}

func TestGetResourceUsage_ReturnsPods(t *testing.T) {
	ctx := context.Background()
	manifest := `apiVersion: argoproj.io/v1alpha1
kind: Workflow
spec:
  templates:
  - name: main
    container:
      resources:
        requests:
          cpu: "100m"
`
	depRepo := &mockDeploymentRepo{}
	depRepo.byID = map[string]*models.PipelineDeployment{
		"dep-1": {
			ID:           "dep-1",
			WorkflowName: "wf-1",
			Status:       "Running",
			Manifest:     &manifest,
		},
	}
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, name, _ string) (*wfv1.Workflow, error) {
		if name != "wf-1" {
			t.Fatalf("unexpected workflow name %q", name)
		}
		wf := &wfv1.Workflow{}
		wf.Status.Phase = wfv1.WorkflowRunning
		wf.Status.Nodes = wfv1.Nodes{
			"pod-1": {
				ID:           "pod-1",
				Type:         wfv1.NodeTypePod,
				TemplateName: "main",
				HostNodeName: "node-a",
				ResourcesDuration: wfv1.ResourcesDuration{
					corev1.ResourceCPU: wfv1.NewResourceDuration(10 * time.Second),
				},
			},
		}
		return wf, nil
	}

	uc := New(&mockTemplateRepo{}, depRepo, &mockAssetRepo{}, wfClient, "default")
	report, err := uc.GetResourceUsage(ctx, "dep-1")
	if err != nil {
		t.Fatalf("GetResourceUsage: %v", err)
	}
	if len(report.Pods) != 1 {
		t.Fatalf("expected 1 pod, got %d", len(report.Pods))
	}
	if report.Pods[0].PodName != "pod-1" || report.Pods[0].CPURequest != "100m" {
		t.Fatalf("unexpected pod report: %#v", report.Pods[0])
	}
	if report.Status != string(wfv1.WorkflowRunning) {
		t.Fatalf("expected Running status, got %q", report.Status)
	}
	if report.Source.Workflow != "argo-live" || report.Source.Metrics != "unavailable" || report.Source.Spec != "stored-manifest" {
		t.Fatalf("unexpected source metadata: %#v", report.Source)
	}
	if report.LiveMetricsAvailable {
		t.Fatal("expected live metrics unavailable")
	}
	if report.ObservedAt == "" {
		t.Fatal("expected observed_at")
	}
}

func TestGetWorkflowNodeResourceUsage_ReturnsOneNode(t *testing.T) {
	ctx := context.Background()
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, name, _ string) (*wfv1.Workflow, error) {
		if name != "wf-1" {
			t.Fatalf("unexpected workflow name %q", name)
		}
		wf := &wfv1.Workflow{}
		wf.Status.Phase = wfv1.WorkflowRunning
		wf.Status.Nodes = wfv1.Nodes{
			"pod-1": {ID: "pod-1", Type: wfv1.NodeTypePod},
			"pod-2": {ID: "pod-2", Type: wfv1.NodeTypePod},
		}
		return wf, nil
	}

	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	report, err := uc.GetWorkflowNodeResourceUsage(ctx, "wf-1", "pod-2")
	if err != nil {
		t.Fatalf("GetWorkflowNodeResourceUsage: %v", err)
	}
	if len(report.Pods) != 1 || report.Pods[0].PodName != "pod-2" {
		t.Fatalf("unexpected report pods: %#v", report.Pods)
	}
	if report.Source.Spec != "unavailable" {
		t.Fatalf("expected unavailable spec source, got %#v", report.Source)
	}
}

func TestDeploymentStatusRefresh_MarksMissingWorkflowExpired(t *testing.T) {
	ctx := context.Background()
	depRepo := &mockDeploymentRepo{}
	dep := &models.PipelineDeployment{
		ID:           "dep-1",
		PipelineName: "pipe-a",
		WorkflowName: "expired-wf",
		Status:       "Running",
	}
	if err := depRepo.Save(ctx, dep); err != nil {
		t.Fatalf("save deployment: %v", err)
	}
	wfClient := &mockWorkflowClient{
		getWorkflowStatusFn: func(_ context.Context, name, _ string) (wfv1.WorkflowPhase, error) {
			if name != "expired-wf" {
				t.Fatalf("unexpected workflow name %q", name)
			}
			return wfv1.WorkflowUnknown, argo.ErrNotFound
		},
	}
	uc := New(&mockTemplateRepo{}, depRepo, &mockAssetRepo{}, wfClient, "default")

	got, err := uc.GetDeployment(ctx, "dep-1")
	if err != nil {
		t.Fatalf("GetDeployment: %v", err)
	}
	if got.Status != "Expired" {
		t.Fatalf("expected Expired status, got %q", got.Status)
	}
	if depRepo.byID["dep-1"].Status != "Expired" {
		t.Fatalf("expected repo status Expired, got %q", depRepo.byID["dep-1"].Status)
	}
}

func TestListDeployments_MarksMissingWorkflowExpired(t *testing.T) {
	ctx := context.Background()
	depRepo := &mockDeploymentRepo{}
	if err := depRepo.Save(ctx, &models.PipelineDeployment{
		ID:           "dep-1",
		PipelineName: "pipe-a",
		WorkflowName: "expired-wf",
		Status:       "Running",
	}); err != nil {
		t.Fatalf("save deployment: %v", err)
	}
	wfClient := &mockWorkflowClient{
		getWorkflowStatusFn: func(_ context.Context, _ string, _ string) (wfv1.WorkflowPhase, error) {
			return wfv1.WorkflowUnknown, argo.ErrNotFound
		},
	}
	uc := New(&mockTemplateRepo{}, depRepo, &mockAssetRepo{}, wfClient, "default")

	list, err := uc.ListDeployments(ctx)
	if err != nil {
		t.Fatalf("ListDeployments: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected one deployment, got %d", len(list))
	}
	if list[0].Status != "Expired" {
		t.Fatalf("expected Expired status, got %q", list[0].Status)
	}
}

func TestRetryDeployment_PreservesInputAssetIDs(t *testing.T) {
	ctx := context.Background()
	depRepo := &mockDeploymentRepo{}
	depRepo.byID = map[string]*models.PipelineDeployment{
		"dep-1": {
			ID:           "dep-1",
			PipelineName: "pipe-a",
			PipelineJSON: map[string]interface{}{
				"name":             "pipe-a",
				"nodes":            []interface{}{},
				"_input_asset_ids": []interface{}{"asset-1", "asset-2"},
			},
		},
	}
	assetRepo := newMockAssetRepo()
	assetRepo.assets["asset-1"] = &models.Asset{AssetID: "asset-1"}
	assetRepo.assets["asset-2"] = &models.Asset{AssetID: "asset-2"}

	uc := New(&mockTemplateRepo{}, depRepo, assetRepo, &mockWorkflowClient{}, "default")
	dep, err := uc.RetryDeployment(ctx, "dep-1")
	if err != nil {
		t.Fatalf("RetryDeployment: %v", err)
	}
	raw, ok := dep.PipelineJSON["_input_asset_ids"].([]string)
	if !ok {
		t.Fatalf("expected []string _input_asset_ids, got %T", dep.PipelineJSON["_input_asset_ids"])
	}
	if len(raw) != 2 || raw[0] != "asset-1" || raw[1] != "asset-2" {
		t.Fatalf("unexpected asset ids: %#v", raw)
	}
}

// ── CYB-1537 — PR #77 review follow-up: runRepo primary lookup ─────────────

type mockRunRepo struct {
	byID         map[string]*models.PipelineRun
	byWf         map[string]*models.PipelineRun
	findAllErr   error
	findAllCalls int
}

func (m *mockRunRepo) Save(_ context.Context, r *models.PipelineRun) error {
	if m.byID == nil {
		m.byID = map[string]*models.PipelineRun{}
	}
	if m.byWf == nil {
		m.byWf = map[string]*models.PipelineRun{}
	}
	m.byID[r.ID] = r
	m.byWf[r.WorkflowName] = r
	return nil
}
func (m *mockRunRepo) FindAll(_ context.Context) ([]models.PipelineRun, error) {
	m.findAllCalls++
	if m.findAllErr != nil {
		return nil, m.findAllErr
	}
	out := make([]models.PipelineRun, 0, len(m.byID))
	for _, r := range m.byID {
		out = append(out, *r)
	}
	return out, nil
}
func (m *mockRunRepo) FindByID(_ context.Context, id string) (*models.PipelineRun, error) {
	if m.byID == nil {
		return nil, nil
	}
	return m.byID[id], nil
}
func (m *mockRunRepo) FindByWorkflowName(_ context.Context, name string) (*models.PipelineRun, error) {
	if m.byWf == nil {
		return nil, nil
	}
	return m.byWf[name], nil
}
func (m *mockRunRepo) Delete(_ context.Context, id string) error {
	if r, ok := m.byID[id]; ok {
		delete(m.byID, id)
		delete(m.byWf, r.WorkflowName)
	}
	return nil
}
func (m *mockRunRepo) DeleteByTemplateID(_ context.Context, _ string) error {
	return nil
}
func (m *mockRunRepo) UpdateStatus(_ context.Context, id, status string, finishedAt *time.Time) error {
	if r, ok := m.byID[id]; ok {
		r.Status = status
		r.FinishedAt = finishedAt
	}
	return nil
}

// trackingDeploymentRepo wraps mockDeploymentRepo to count FindAll calls so
// the CYB-1537 tests can assert whether the legacy scan ran.
type trackingDeploymentRepo struct {
	*mockDeploymentRepo
	findAllCalls int
}

func (t *trackingDeploymentRepo) FindAll(_ context.Context) ([]models.PipelineDeployment, error) {
	t.findAllCalls++
	return t.mockDeploymentRepo.FindAll(nil)
}

func TestGetWorkflowResourceUsage_UsesRunRepo(t *testing.T) {
	ctx := context.Background()
	manifest := "kind: Workflow\nspec: {}\n"
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:           "run-1",
				WorkflowName: "wf-1",
				Status:       "Running",
				Manifest:     &manifest,
			},
		},
		byWf: map[string]*models.PipelineRun{
			"wf-1": {
				ID:           "run-1",
				WorkflowName: "wf-1",
				Status:       "Running",
				Manifest:     &manifest,
			},
		},
	}
	depRepo := &trackingDeploymentRepo{mockDeploymentRepo: &mockDeploymentRepo{}}
	// Intentionally also seed the legacy table; if the fix is wrong, the
	// fallback would return this row.
	dep := &models.PipelineDeployment{ID: "dep-legacy", WorkflowName: "wf-1", Status: "Running"}
	if err := depRepo.Save(ctx, dep); err != nil {
		t.Fatalf("save legacy deployment: %v", err)
	}
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, name, _ string) (*wfv1.Workflow, error) {
		wf := &wfv1.Workflow{}
		wf.Status.Phase = wfv1.WorkflowRunning
		return wf, nil
	}

	uc := New(&mockTemplateRepo{}, depRepo, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})

	report, err := uc.GetWorkflowResourceUsage(ctx, "wf-1")
	if err != nil {
		t.Fatalf("GetWorkflowResourceUsage: %v", err)
	}
	if report.DeploymentID != "run-1" {
		t.Fatalf("expected deploymentID from runRepo (run-1), got %q", report.DeploymentID)
	}
	if depRepo.findAllCalls != 0 {
		t.Fatalf("expected deploymentRepo.FindAll NOT to be called when runRepo has the row, got %d calls", depRepo.findAllCalls)
	}
}

func TestGetWorkflowResourceUsage_FallsBackToDeploymentRepo(t *testing.T) {
	ctx := context.Background()
	manifest := "kind: Workflow\nspec: {}\n"
	runRepo := &mockRunRepo{} // empty
	depRepo := &trackingDeploymentRepo{mockDeploymentRepo: &mockDeploymentRepo{}}
	dep := &models.PipelineDeployment{
		ID:           "dep-legacy",
		WorkflowName: "wf-legacy",
		Status:       "Running",
		Manifest:     &manifest,
	}
	if err := depRepo.Save(ctx, dep); err != nil {
		t.Fatalf("save legacy deployment: %v", err)
	}
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, name, _ string) (*wfv1.Workflow, error) {
		wf := &wfv1.Workflow{}
		wf.Status.Phase = wfv1.WorkflowRunning
		return wf, nil
	}

	uc := New(&mockTemplateRepo{}, depRepo, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})

	report, err := uc.GetWorkflowResourceUsage(ctx, "wf-legacy")
	if err != nil {
		t.Fatalf("GetWorkflowResourceUsage: %v", err)
	}
	if report.DeploymentID != "dep-legacy" {
		t.Fatalf("expected fallback deploymentID dep-legacy, got %q", report.DeploymentID)
	}
	if depRepo.findAllCalls == 0 {
		t.Fatal("expected deploymentRepo.FindAll to be called as fallback, got 0 calls")
	}
}

// No-op mocks for SetRunRepositories arguments. The tests in this block do
// not exercise target or node persistence.
type mockTargetRepo struct{}

func (mockTargetRepo) Save(_ context.Context, _ *models.ExecutionTarget) error { return nil }
func (mockTargetRepo) FindAll(_ context.Context) ([]models.ExecutionTarget, error) {
	return nil, nil
}
func (mockTargetRepo) FindByID(_ context.Context, _ string) (*models.ExecutionTarget, error) {
	return nil, nil
}
func (mockTargetRepo) FindDefault(_ context.Context) (*models.ExecutionTarget, error) {
	return nil, nil
}

type mockRunNodeRepo struct{}

func (mockRunNodeRepo) ReplaceByRunID(_ context.Context, _ string, _ []models.PipelineRunNode) error {
	return nil
}
func (mockRunNodeRepo) FindByRunID(_ context.Context, _ string) ([]models.PipelineRunNode, error) {
	return nil, nil
}
func (mockRunNodeRepo) DeleteByRunID(_ context.Context, _ string) error { return nil }

type mockRunEventRepo struct {
	events []models.PipelineRunEvent
}

func (m *mockRunEventRepo) Append(_ context.Context, event *models.PipelineRunEvent) error {
	if event == nil {
		return nil
	}
	m.events = append(m.events, *event)
	return nil
}

func (m *mockRunEventRepo) ListByRunID(_ context.Context, runID string, _ models.PipelineRunEventListOptions) (*models.PipelineRunEventListResult, error) {
	out := []models.PipelineRunEvent{}
	for _, event := range m.events {
		if event.RunID == runID {
			out = append(out, event)
		}
	}
	return &models.PipelineRunEventListResult{Items: out, Total: len(out)}, nil
}

type mockWatcherStateRepo struct {
	state *models.PipelineRunWatcherState
}

func (m *mockWatcherStateRepo) Save(_ context.Context, state *models.PipelineRunWatcherState) error {
	if state == nil {
		return nil
	}
	copy := *state
	m.state = &copy
	return nil
}

func (m *mockWatcherStateRepo) FindByID(_ context.Context, _ string) (*models.PipelineRunWatcherState, error) {
	if m.state == nil {
		return nil, nil
	}
	copy := *m.state
	return &copy, nil
}

func TestListRunEvents_ReturnsStoredEvents(t *testing.T) {
	ctx := context.Background()
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {ID: "run-1", WorkflowName: "wf-1", Status: "Succeeded"},
		},
	}
	eventRepo := &mockRunEventRepo{
		events: []models.PipelineRunEvent{
			{ID: "evt-1", RunID: "run-1", EventType: runEventSubmitted, SubjectType: "run", SubjectID: "run-1"},
		},
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetRunEventRepo(eventRepo)

	result, err := uc.ListRunEvents(ctx, "run-1", models.PipelineRunEventListOptions{Limit: 20})
	if err != nil {
		t.Fatalf("ListRunEvents: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].ID != "evt-1" {
		t.Fatalf("unexpected events: %#v", result.Items)
	}
}

func TestGetRun_AppendsWorkflowAndNodeEvents(t *testing.T) {
	ctx := context.Background()
	started := time.Date(2026, 6, 3, 1, 2, 3, 0, time.UTC)
	finished := started.Add(10 * time.Second)
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {ID: "run-1", WorkflowName: "wf-1", Status: "Running"},
		},
	}
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, name, namespace string) (*wfv1.Workflow, error) {
		if name != "wf-1" || namespace != "default" {
			t.Fatalf("unexpected workflow lookup name=%q namespace=%q", name, namespace)
		}
		return &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "wf-1", UID: "uid-1"},
			Status: wfv1.WorkflowStatus{
				Phase:      wfv1.WorkflowSucceeded,
				FinishedAt: metav1.Time{Time: finished},
				Nodes: map[string]wfv1.NodeStatus{
					"node-1": {
						ID:           "node-1",
						Name:         "wf-1-step",
						DisplayName:  "step",
						Type:         wfv1.NodeTypePod,
						Phase:        wfv1.NodeSucceeded,
						StartedAt:    metav1.Time{Time: started},
						FinishedAt:   metav1.Time{Time: finished},
						TemplateName: "step",
					},
				},
			},
		}, nil
	}
	eventRepo := &mockRunEventRepo{}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetRunEventRepo(eventRepo)

	if _, err := uc.GetRun(ctx, "run-1"); err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	seen := map[string]bool{}
	for _, event := range eventRepo.events {
		seen[event.EventType] = true
	}
	for _, eventType := range []string{runEventWorkflowObserved, runEventWorkflowPhaseChanged, runEventCompleted, runEventNodeStarted, runEventNodeSucceeded, runEventPodCreated, runEventPodPhaseChanged} {
		if !seen[eventType] {
			t.Fatalf("expected event type %s in %#v", eventType, eventRepo.events)
		}
	}
}

func TestSavePipelineRun_AppendsScheduledAndWorkflowCreatedEvents(t *testing.T) {
	ctx := context.Background()
	runRepo := &mockRunRepo{}
	eventRepo := &mockRunEventRepo{}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetRunEventRepo(eventRepo)

	templateVersion := 1
	dep := &models.PipelineDeployment{
		ID:              "run-1",
		TemplateVersion: &templateVersion,
		PipelineName:    "pipe",
		WorkflowName:    "wf-1",
		Status:          "Pending",
		NodeCount:       1,
		CreatedAt:       time.Now().UTC(),
		ExecutionTarget: &models.ExecutionTarget{ID: "default", Namespace: "default"},
	}
	if err := uc.savePipelineRun(ctx, dep, templateVersion, "uid-1"); err != nil {
		t.Fatalf("savePipelineRun: %v", err)
	}
	seen := map[string]bool{}
	for _, event := range eventRepo.events {
		seen[event.EventType] = true
	}
	for _, eventType := range []string{runEventSubmitted, runEventScheduled, runEventWorkflowCreated} {
		if !seen[eventType] {
			t.Fatalf("expected event type %s in %#v", eventType, eventRepo.events)
		}
	}
}

func TestSyncActiveRunEvents_SavesWatcherHealth(t *testing.T) {
	ctx := context.Background()
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {ID: "run-1", WorkflowName: "wf-1", Status: "Running"},
		},
	}
	watcherRepo := &mockWatcherStateRepo{}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, &mockWorkflowClient{}, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetRunEventRepo(&mockRunEventRepo{})
	uc.SetObservabilityRepositories(nil, nil, watcherRepo)

	synced, err := uc.SyncActiveRunEvents(ctx, 25)
	if err != nil {
		t.Fatalf("SyncActiveRunEvents: %v", err)
	}
	if synced != 1 {
		t.Fatalf("expected synced=1, got %d", synced)
	}
	if watcherRepo.state == nil {
		t.Fatal("expected watcher state to be saved")
	}
	if watcherRepo.state.LastSuccessAt == nil || watcherRepo.state.LastScanStartedAt == nil || watcherRepo.state.LastScanFinishedAt == nil {
		t.Fatalf("expected watcher timestamps, got %#v", watcherRepo.state)
	}
	if watcherRepo.state.LastSyncedRunCount != 1 || watcherRepo.state.ConsecutiveFailures != 0 || watcherRepo.state.TotalScans != 1 {
		t.Fatalf("unexpected watcher counters: %#v", watcherRepo.state)
	}
	status, err := uc.GetRunWatcherStatus(ctx)
	if err != nil {
		t.Fatalf("GetRunWatcherStatus: %v", err)
	}
	if !status.Healthy || status.Stale {
		t.Fatalf("expected healthy non-stale watcher, got %#v", status)
	}
}
