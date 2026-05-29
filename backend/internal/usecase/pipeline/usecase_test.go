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

	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
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
func (m *mockTemplateRepo) FindVersionsByName(_ context.Context, _ string) ([]models.PipelineTemplate, error) {
	return nil, nil
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
func (m *mockDeploymentRepo) UpdateStatus(_ context.Context, _, _ string) error { return nil }

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
	getWorkflowFn func(ctx context.Context, name, namespace string) (*wfv1.Workflow, error)
}

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
func (m *mockWorkflowClient) GetWorkflow(ctx context.Context, name, namespace string) (*wfv1.Workflow, error) {
	if m.getWorkflowFn != nil {
		return m.getWorkflowFn(ctx, name, namespace)
	}
	return &wfv1.Workflow{}, nil
}
func (m *mockWorkflowClient) StopWorkflow(_ context.Context, _, _ string) error {
	return nil
}
func (m *mockWorkflowClient) GetWorkflowLogs(_ context.Context, _, _, _ string) (string, error) {
	return "", nil
}
func (m *mockWorkflowClient) GetWorkflowLogStream(_ context.Context, _, _, _, _ string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}
func (m *mockWorkflowClient) RetryWorkflow(_ context.Context, _, _ string) error { return nil }
func (m *mockWorkflowClient) ResubmitWorkflow(_ context.Context, _, _ string) error { return nil }
func (m *mockWorkflowClient) SuspendWorkflow(_ context.Context, _, _ string) error { return nil }
func (m *mockWorkflowClient) ResumeWorkflow(_ context.Context, _, _ string) error { return nil }
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
