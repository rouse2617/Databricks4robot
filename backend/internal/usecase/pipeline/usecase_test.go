package pipeline

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
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

func (m *mockTemplateRepo) SetActiveVersion(_ context.Context, name string, version int) error {
	for _, t := range m.byID {
		if t.Name == name {
			t.ActiveVersion = version
		}
	}
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
	createWorkflowFn    func(ctx context.Context, wf *wfv1.Workflow, namespace string) error
}

func (m *mockWorkflowClient) CreateWorkflow(_ context.Context, _ *wfv1.Workflow, _ string) error {
	if m.createWorkflowFn != nil {
		return m.createWorkflowFn(context.Background(), nil, "")
	}
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
	return &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       "workflow-uid",
		},
	}, nil
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

type mockPipelineConfigRepo struct {
	versions map[string]*models.PipelineConfigVersion
	configs  map[string]*models.PipelineConfig
}

func (m *mockPipelineConfigRepo) Create(context.Context, *models.PipelineConfig, *models.PipelineConfigVersion) error {
	return nil
}
func (m *mockPipelineConfigRepo) FindAll(context.Context, *repository.PipelineConfigFilter) ([]models.PipelineConfig, error) {
	return nil, nil
}
func (m *mockPipelineConfigRepo) FindByID(_ context.Context, id string) (*models.PipelineConfig, error) {
	return m.configs[id], nil
}
func (m *mockPipelineConfigRepo) UpdateMetadata(context.Context, *models.PipelineConfig) error {
	return nil
}
func (m *mockPipelineConfigRepo) CreateVersion(context.Context, string, *models.PipelineConfigVersion) error {
	return nil
}
func (m *mockPipelineConfigRepo) FindVersion(_ context.Context, configID string, version int) (*models.PipelineConfigVersion, error) {
	return m.versions[fmt.Sprintf("%s:%d", configID, version)], nil
}
func (m *mockPipelineConfigRepo) FindVersions(context.Context, string) ([]models.PipelineConfigVersion, error) {
	return nil, nil
}
func (m *mockPipelineConfigRepo) Deprecate(context.Context, string) error {
	return nil
}

type mockRuntimeConfigStore struct {
	lastNamespace    string
	lastDeploymentID string
	lastProjection   RuntimeConfigProjection
	lastOwner        *RuntimeConfigOwnerReference
	volumeName       string
}

func (m *mockRuntimeConfigStore) Create(_ context.Context, namespace, deploymentID string, config RuntimeConfigProjection, owner *RuntimeConfigOwnerReference) (string, error) {
	m.lastNamespace = namespace
	m.lastDeploymentID = deploymentID
	m.lastProjection = config
	m.lastOwner = owner
	if config.VolumeName != "" {
		return config.VolumeName, nil
	}
	if m.volumeName == "" {
		m.volumeName = "runtime-config-test"
	}
	return m.volumeName, nil
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

	t.Run("external video ID passes compatibility and injects VIDEO_ID", func(t *testing.T) {
		const videoID = "019dabf3-5685-769f-8ec3-3992767ebe65"
		repo := newMockAssetRepo()
		uc := newUsecase(repo)

		dep, err := uc.Deploy(ctx, pipe, "", []string{videoID}, DeployOptions{DryRun: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dep == nil || dep.Manifest == nil {
			t.Fatal("expected dry-run manifest")
		}
		if len(dep.AssetIDs) != 1 || dep.AssetIDs[0] != videoID {
			t.Fatalf("expected compatible video ID to be preserved, got %#v", dep.AssetIDs)
		}
		manifest := *dep.Manifest
		for _, want := range []string{
			"name: VIDEO_ID",
			"value: " + videoID,
			"name: ASSET_0_ID",
			"name: REQUEST_ID",
		} {
			if !strings.Contains(manifest, want) {
				t.Fatalf("expected manifest to contain %q, got %s", want, manifest)
			}
		}
		if repo.findCalls != 1 || strings.Join(repo.lastFind, ",") != videoID {
			t.Fatalf("expected one asset lookup for video ID, calls=%d ids=%v", repo.findCalls, repo.lastFind)
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

func TestDeploy_RejectsResourceAboveConfiguredCeiling(t *testing.T) {
	ctx := context.Background()
	pipe := map[string]interface{}{
		"name": "resource-guard-pipe",
		"nodes": []interface{}{
			map[string]interface{}{
				"id": "step-1",
				"component": map[string]interface{}{
					"name":  "oversized",
					"image": "busybox",
					"resources": map[string]interface{}{
						"cpu":    "14",
						"memory": "55Gi",
						"disk":   "50Gi",
					},
				},
			},
		},
		"edges": []interface{}{},
	}
	repo := newMockAssetRepo()
	uc := newUsecase(repo)
	created := false
	uc.wfClient = &mockWorkflowClient{
		createWorkflowFn: func(context.Context, *wfv1.Workflow, string) error {
			created = true
			return nil
		},
	}
	uc.SetResourceGuardConfig(ResourceGuardConfig{
		MaxCPU:    "8",
		MaxMemory: "28Gi",
		MaxDisk:   "250Gi",
		MaxGPU:    "1",
	})

	_, err := uc.Deploy(ctx, pipe, "", nil)
	if err == nil {
		t.Fatal("expected resource guard error, got nil")
	}
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("expected ErrInvalidArgument, got %v", err)
	}
	if !strings.Contains(err.Error(), "cpu=14") || !strings.Contains(err.Error(), "cpu=8") {
		t.Fatalf("expected cpu ceiling in error, got %v", err)
	}
	if created {
		t.Fatal("expected workflow not to be created after resource guard failure")
	}
}

func TestDeploy_IncludesRuntimeConfigMountAndEnv(t *testing.T) {
	ctx := context.Background()
	pipe := map[string]interface{}{
		"name": "test-pipe",
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
	repo := newMockAssetRepo()
	uc := newUsecase(repo)
	configRepo := &mockPipelineConfigRepo{
		configs: map[string]*models.PipelineConfig{
			"cfg-1": {ID: "cfg-1", Name: "detector.yaml"},
		},
		versions: map[string]*models.PipelineConfigVersion{
			"cfg-1:2": {
				ConfigID: "cfg-1",
				Version:  2,
				Content:  "threshold: 0.8\n",
			},
		},
	}
	store := &mockRuntimeConfigStore{volumeName: "runtime-config-test"}
	uc.pipelineConfigRepo = configRepo
	uc.runtimeConfigStore = store

	dep, err := uc.Deploy(ctx, pipe, "", nil, DeployOptions{
		ConfigSelection: &RuntimeConfigSelection{
			Mode:           "saved",
			ConfigID:       "cfg-1",
			Version:        2,
			FileName:       "detector.yaml",
			MountPath:      "/workspace/configs",
			TargetFilename: "effective.yaml",
		},
	})
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if dep == nil || dep.Manifest == nil {
		t.Fatal("expected manifest on deployment")
	}
	if store.lastProjection.FileName != "effective.yaml" {
		t.Fatalf("expected projected filename effective.yaml, got %q", store.lastProjection.FileName)
	}
	if store.lastProjection.Content != "threshold: 0.8\n" {
		t.Fatalf("unexpected projected content %q", store.lastProjection.Content)
	}
	if store.lastProjection.VolumeName == "" {
		t.Fatal("expected runtime config projection volume name")
	}
	if store.lastOwner == nil || store.lastOwner.Kind != "Workflow" || store.lastOwner.UID != "workflow-uid" {
		t.Fatalf("expected workflow owner reference, got %#v", store.lastOwner)
	}
	manifest := *dep.Manifest
	if !strings.Contains(manifest, store.lastProjection.VolumeName) {
		t.Fatalf("expected runtime config volume in manifest, got %s", manifest)
	}
	if !strings.Contains(manifest, "PIPELINE_CONFIG_PATH") || !strings.Contains(manifest, "/workspace/configs/effective.yaml") {
		t.Fatalf("expected config env path in manifest, got %s", manifest)
	}
	if !strings.Contains(manifest, "subpath: effective.yaml") {
		t.Fatalf("expected config subPath mount in manifest, got %s", manifest)
	}
}

func TestDeployByTemplateID_ForwardsRuntimeConfigSelection(t *testing.T) {
	ctx := context.Background()
	pipe := map[string]interface{}{
		"name": "test-pipe",
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
	repo := newMockAssetRepo()
	uc := newUsecase(repo)
	uc.templateRepo = &mockTemplateRepo{
		byID: map[string]*models.PipelineTemplate{
			"tmpl-1": {
				ID:       "tmpl-1",
				Name:     "tmpl",
				Version:  1,
				Pipeline: pipe,
			},
		},
		byName: map[string][]models.PipelineTemplate{
			"tmpl": {{
				ID:       "tmpl-1",
				Name:     "tmpl",
				Version:  1,
				Pipeline: pipe,
			}},
		},
	}
	uc.pipelineConfigRepo = &mockPipelineConfigRepo{
		configs: map[string]*models.PipelineConfig{
			"cfg-1": {ID: "cfg-1", Name: "detector.yaml"},
		},
		versions: map[string]*models.PipelineConfigVersion{
			"cfg-1:1": {
				ConfigID: "cfg-1",
				Version:  1,
				Content:  "threshold: 0.8\n",
			},
		},
	}
	store := &mockRuntimeConfigStore{volumeName: "runtime-config-template"}
	uc.runtimeConfigStore = store

	dep, err := uc.DeployByTemplateID(ctx, "tmpl-1", "", nil, DeployOptions{
		ConfigSelection: &RuntimeConfigSelection{
			Mode:           "saved",
			ConfigID:       "cfg-1",
			Version:        1,
			FileName:       "detector.yaml",
			MountPath:      "/workspace/configs",
			TargetFilename: "effective.yaml",
		},
	})
	if err != nil {
		t.Fatalf("DeployByTemplateID: %v", err)
	}
	if dep == nil || dep.Manifest == nil {
		t.Fatal("expected manifest on deployment")
	}
	if store.lastProjection.FileName != "effective.yaml" {
		t.Fatalf("expected forwarded projected filename effective.yaml, got %q", store.lastProjection.FileName)
	}
	manifest := *dep.Manifest
	if store.lastProjection.VolumeName == "" {
		t.Fatal("expected runtime config projection volume name")
	}
	if store.lastOwner == nil || store.lastOwner.Kind != "Workflow" || store.lastOwner.UID != "workflow-uid" {
		t.Fatalf("expected workflow owner reference, got %#v", store.lastOwner)
	}
	if !strings.Contains(manifest, store.lastProjection.VolumeName) {
		t.Fatalf("expected forwarded runtime config volume in manifest, got %s", manifest)
	}
	if !strings.Contains(manifest, "PIPELINE_CONFIG_PATH") {
		t.Fatalf("expected forwarded runtime config env in manifest, got %s", manifest)
	}
}

func TestDeploy_IncludesNodeRuntimeConfigsAndAssetEnv(t *testing.T) {
	ctx := context.Background()
	pipe := map[string]interface{}{
		"name": "node-config-pipe",
		"nodes": []interface{}{
			map[string]interface{}{
				"id": "step-a",
				"component": map[string]interface{}{
					"name":  "detector",
					"image": "busybox",
				},
				"runtimeConfig": map[string]interface{}{
					"mode":           "saved",
					"configId":       "cfg-a",
					"version":        1,
					"fileName":       "a-source.yaml",
					"mountPath":      "/workspace/configs",
					"targetFilename": "a.yaml",
				},
			},
			map[string]interface{}{
				"id": "step-b",
				"component": map[string]interface{}{
					"name":  "classifier",
					"image": "busybox",
				},
				"runtimeConfig": map[string]interface{}{
					"mode":           "saved",
					"configId":       "cfg-b",
					"version":        2,
					"fileName":       "b-source.yaml",
					"mountPath":      "/workspace/configs",
					"targetFilename": "b.yaml",
				},
			},
			map[string]interface{}{
				"id": "step-c",
				"component": map[string]interface{}{
					"name":  "unconfigured",
					"image": "busybox",
				},
			},
		},
		"edges": []interface{}{},
	}
	repo := newMockAssetRepo()
	repo.assets["asset-a"] = &models.Asset{
		AssetID:    "asset-a",
		StorageURI: "gs://bucket/asset-a",
		AssetType:  "segment",
	}
	uc := newUsecase(repo)
	uc.pipelineConfigRepo = &mockPipelineConfigRepo{
		configs: map[string]*models.PipelineConfig{
			"cfg-a": {ID: "cfg-a", Name: "a-source.yaml", Lifecycle: "ready"},
			"cfg-b": {ID: "cfg-b", Name: "b-source.yaml", Lifecycle: "ready"},
		},
		versions: map[string]*models.PipelineConfigVersion{
			"cfg-a:1": {ConfigID: "cfg-a", Version: 1, Status: "ready", Content: "threshold: 0.8\n"},
			"cfg-b:2": {ConfigID: "cfg-b", Version: 2, Status: "ready", Content: "labels: [car]\n"},
		},
	}
	store := &mockRuntimeConfigStore{volumeName: "runtime-config-nodes"}
	uc.runtimeConfigStore = store

	dep, err := uc.Deploy(ctx, pipe, "", []string{"asset-a"})
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if dep == nil || dep.Manifest == nil {
		t.Fatal("expected manifest on deployment")
	}
	if len(store.lastProjection.Files) != 2 {
		t.Fatalf("expected two projected node config files, got %#v", store.lastProjection.Files)
	}
	if store.lastProjection.VolumeName == "" {
		t.Fatal("expected runtime config projection volume name")
	}
	if store.lastOwner == nil || store.lastOwner.Kind != "Workflow" || store.lastOwner.UID != "workflow-uid" {
		t.Fatalf("expected workflow owner reference, got %#v", store.lastOwner)
	}
	if got := store.lastProjection.Files["01-step-a-a.yaml"]; got != "threshold: 0.8\n" {
		t.Fatalf("unexpected step-a config content %q", got)
	}
	if got := store.lastProjection.Files["02-step-b-b.yaml"]; got != "labels: [car]\n" {
		t.Fatalf("unexpected step-b config content %q", got)
	}
	manifest := *dep.Manifest
	if strings.Count(manifest, "name: PIPELINE_CONFIG_PATH") != 2 {
		t.Fatalf("expected config env only on two configured nodes, got manifest %s", manifest)
	}
	if strings.Count(manifest, "name: ASSET_IDS") != 3 {
		t.Fatalf("expected asset env on all three nodes, got manifest %s", manifest)
	}
	if strings.Count(manifest, "name: REQUEST_ID") != 3 {
		t.Fatalf("expected request id env on all three nodes, got manifest %s", manifest)
	}
	if strings.Count(manifest, "name: VIDEO_ID") != 3 {
		t.Fatalf("expected single-asset video id env on all three nodes, got manifest %s", manifest)
	}
	for _, want := range []string{
		store.lastProjection.VolumeName,
		"subpath: 01-step-a-a.yaml",
		"subpath: 02-step-b-b.yaml",
		"value: cfg-a",
		"value: cfg-b",
		"value: asset-a",
		"value: gs://bucket/asset-a",
	} {
		if !strings.Contains(manifest, want) {
			t.Fatalf("expected manifest to contain %q, got %s", want, manifest)
		}
	}
}

func TestDeploy_IncludesRuntimeSecretAndStorageMounts(t *testing.T) {
	ctx := context.Background()
	pipe := map[string]interface{}{
		"name": "runtime-mount-pipe",
		"nodes": []interface{}{
			map[string]interface{}{
				"id": "step-a",
				"component": map[string]interface{}{
					"name":  "probe",
					"image": "busybox",
				},
				"runtimeSecrets": []interface{}{
					map[string]interface{}{
						"resourceId": "db-secrets",
						"mountPath":  "/mnt/db-secrets",
					},
				},
				"storageMounts": []interface{}{
					map[string]interface{}{
						"resourceId": "model-pvc",
						"mountPath":  "/workspace/models",
						"readOnly":   true,
					},
					map[string]interface{}{
						"resourceId": "scratch",
						"mountPath":  "/workspace/scratch",
						"readOnly":   false,
					},
				},
			},
		},
		"edges": []interface{}{},
	}
	uc := newUsecase(newMockAssetRepo())
	uc.SetRuntimeMountCatalog(RuntimeMountCatalog{
		Secrets: []RuntimeSecretMountResource{{
			ID:                  "db-secrets",
			Name:                "DB secrets",
			Kind:                runtimeSecretKindSecretProviderClass,
			SecretProviderClass: "db-secret-provider",
			DefaultMountPath:    "/mnt/secrets",
			ReadOnly:            true,
		}},
		Storage: []RuntimeStorageMountResource{
			{
				ID:               "model-pvc",
				Name:             "Model PVC",
				Kind:             runtimeStorageKindPVC,
				PVCName:          "shared-models",
				DefaultMountPath: "/workspace/models",
				ReadOnly:         true,
				AllowWrite:       false,
			},
			{
				ID:               "scratch",
				Name:             "Scratch",
				Kind:             runtimeStorageKindEmptyDir,
				DefaultMountPath: "/workspace/scratch",
				ReadOnly:         false,
				AllowWrite:       true,
			},
		},
	})

	dep, err := uc.Deploy(ctx, pipe, "", nil)
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if dep == nil || dep.Manifest == nil {
		t.Fatal("expected manifest on deployment")
	}
	manifest := *dep.Manifest
	for _, want := range []string{
		"db-secret-provider",
		"secrets-store-gke.csi.k8s.io",
		"shared-models",
		"/mnt/db-secrets",
		"/workspace/models",
		"/workspace/scratch",
		"PIPELINE_SECRET_DB_SECRETS_PATH",
		"PIPELINE_STORAGE_MODEL_PVC_PATH",
		"PIPELINE_STORAGE_SCRATCH_PATH",
	} {
		if !strings.Contains(manifest, want) {
			t.Fatalf("expected manifest to contain %q, got %s", want, manifest)
		}
	}
}

func TestDeploy_UsesDefaultExecutionTargetServiceAccountFromEnv(t *testing.T) {
	t.Setenv("PIPELINE_DEFAULT_SERVICE_ACCOUNT", "cyber-databrew-backend-argo")
	ctx := context.Background()
	pipe := map[string]interface{}{
		"name": "service-account-pipe",
		"nodes": []interface{}{
			map[string]interface{}{
				"id": "step-a",
				"component": map[string]interface{}{
					"name":  "probe",
					"image": "busybox",
				},
			},
		},
		"edges": []interface{}{},
	}
	uc := newUsecase(newMockAssetRepo())

	dep, err := uc.Deploy(ctx, pipe, "", nil, DeployOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Deploy dry-run: %v", err)
	}
	if dep == nil || dep.Manifest == nil {
		t.Fatal("expected manifest on dry-run deployment")
	}
	if !strings.Contains(*dep.Manifest, "serviceaccountname: cyber-databrew-backend-argo") {
		t.Fatalf("expected manifest service account, got %s", *dep.Manifest)
	}
	if dep.ExecutionTarget == nil || dep.ExecutionTarget.ServiceAccount != "cyber-databrew-backend-argo" {
		t.Fatalf("expected execution target service account, got %+v", dep.ExecutionTarget)
	}
}

func TestDeploy_AliasesSSDeliveryCyberpipeNodeForGraceCompatibility(t *testing.T) {
	ctx := context.Background()
	pipe := map[string]interface{}{
		"name": "ss-delivery-pipe",
		"nodes": []interface{}{
			map[string]interface{}{
				"id": "ss_delivery_lerobot",
				"component": map[string]interface{}{
					"name":    "ss-delivery-lerobot",
					"image":   "example.com/ss-delivery-lerobot:latest",
					"command": []interface{}{"python", "src/main.py"},
					"env": []interface{}{
						map[string]interface{}{"name": "TASK_NAME", "value": "ss-delivery-lerobot"},
						map[string]interface{}{"name": "CYBERPIPE_NODE", "value": "ss_delivery_lerobot"},
					},
				},
			},
		},
		"edges": []interface{}{},
	}
	uc := newUsecase(newMockAssetRepo())

	dep, err := uc.Deploy(ctx, pipe, "", nil, DeployOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Deploy dry-run: %v", err)
	}
	if dep == nil || dep.Manifest == nil {
		t.Fatal("expected manifest on dry-run deployment")
	}
	manifest := *dep.Manifest
	for _, want := range []string{
		"name: step-ss-delivery-lerobot",
		"name: CYBERPIPE_NODE",
		"value: tony_delivery_lerobot",
	} {
		if !strings.Contains(manifest, want) {
			t.Fatalf("expected manifest to contain %q, got %s", want, manifest)
		}
	}
	if strings.Contains(manifest, "value: ss_delivery_lerobot") {
		t.Fatalf("expected manifest to alias CYBERPIPE_NODE, got %s", manifest)
	}
}

func TestDeploy_RejectsUnknownRuntimeMountResource(t *testing.T) {
	ctx := context.Background()
	pipe := map[string]interface{}{
		"name": "runtime-mount-pipe",
		"nodes": []interface{}{
			map[string]interface{}{
				"id": "step-a",
				"component": map[string]interface{}{
					"name":  "probe",
					"image": "busybox",
				},
				"runtimeSecrets": []interface{}{
					map[string]interface{}{"resourceId": "missing-secret"},
				},
			},
		},
		"edges": []interface{}{},
	}
	uc := newUsecase(newMockAssetRepo())
	uc.SetRuntimeMountCatalog(RuntimeMountCatalog{})

	_, err := uc.Deploy(ctx, pipe, "", nil)
	if err == nil {
		t.Fatal("expected unknown runtime mount error")
	}
	if !errors.Is(err, ErrInvalidArgument) || !strings.Contains(err.Error(), "missing-secret") {
		t.Fatalf("expected missing-secret invalid argument error, got %v", err)
	}
}

func TestDeploy_RejectsRuntimeStorageWriteModeMismatch(t *testing.T) {
	ctx := context.Background()
	pipe := map[string]interface{}{
		"name": "runtime-mount-pipe",
		"nodes": []interface{}{
			map[string]interface{}{
				"id": "step-a",
				"component": map[string]interface{}{
					"name":  "probe",
					"image": "busybox",
				},
				"storageMounts": []interface{}{
					map[string]interface{}{
						"resourceId": "readonly-pvc",
						"readOnly":   false,
					},
				},
			},
		},
		"edges": []interface{}{},
	}
	uc := newUsecase(newMockAssetRepo())
	uc.SetRuntimeMountCatalog(RuntimeMountCatalog{
		Storage: []RuntimeStorageMountResource{{
			ID:               "readonly-pvc",
			Name:             "Read only PVC",
			Kind:             runtimeStorageKindPVC,
			PVCName:          "models",
			DefaultMountPath: "/workspace/models",
			ReadOnly:         true,
			AllowWrite:       false,
		}},
	})

	_, err := uc.Deploy(ctx, pipe, "", nil)
	if err == nil {
		t.Fatal("expected write mode mismatch error")
	}
	if !errors.Is(err, ErrInvalidArgument) || !strings.Contains(err.Error(), "write mode") {
		t.Fatalf("expected write mode invalid argument error, got %v", err)
	}
}

func TestRuntimeMountAllowedOnTargetMatchesIDNameAndNamespace(t *testing.T) {
	target := &models.ExecutionTarget{
		ID:        "target-uuid",
		Name:      "video-proc-dev",
		Namespace: "video-proc-dev",
	}

	for _, allowed := range []string{"target-uuid", "video-proc-dev"} {
		if !runtimeMountAllowedOnTarget(target, []string{allowed}) {
			t.Fatalf("expected target to allow %q", allowed)
		}
	}
	if runtimeMountAllowedOnTarget(target, []string{"default"}) {
		t.Fatal("expected target not to allow default")
	}
}

func TestDefaultRuntimeMountCatalogLoadsGenericJSONSecretTargets(t *testing.T) {
	t.Setenv("PIPELINE_RUNTIME_MOUNT_TARGET_IDS", "default")
	t.Setenv("PIPELINE_RUNTIME_MOUNT_CATALOG_JSON", `{
		"secrets": [{
			"id": "platform-db-secrets",
			"name": "Platform DB secrets",
			"secretProviderClass": "databrew-platform-db-creds",
			"defaultMountPath": "/mnt/secrets",
			"targetIds": ["gpu-pool"]
		}]
	}`)

	catalog, err := defaultRuntimeMountCatalog()
	if err != nil {
		t.Fatalf("defaultRuntimeMountCatalog: %v", err)
	}

	var platformSecret *RuntimeSecretMountResource
	for i := range catalog.Secrets {
		if catalog.Secrets[i].ID == "platform-db-secrets" {
			platformSecret = &catalog.Secrets[i]
			break
		}
	}
	if platformSecret == nil { // pragma: allowlist secret
		t.Fatalf("expected platform secret in catalog: %+v", catalog.Secrets)
	}
	if len(platformSecret.TargetIDs) != 1 || platformSecret.TargetIDs[0] != "gpu-pool" {
		t.Fatalf("expected resource target IDs, got %+v", platformSecret.TargetIDs)
	}
	if len(catalog.Storage) == 0 || len(catalog.Storage[0].TargetIDs) != 1 || catalog.Storage[0].TargetIDs[0] != "default" {
		t.Fatalf("expected storage to keep global target IDs, got %+v", catalog.Storage)
	}
}

func TestDefaultRuntimeMountCatalogRejectsInvalidJSONCatalog(t *testing.T) {
	t.Setenv("PIPELINE_RUNTIME_MOUNT_CATALOG_JSON", `{"secrets":[{"id":"missing-spc"}]}`)

	if _, err := defaultRuntimeMountCatalog(); err == nil {
		t.Fatal("expected invalid catalog error")
	}
}

func TestDeploy_NodeRuntimeConfigRejectsNotReadyConfig(t *testing.T) {
	ctx := context.Background()
	pipe := map[string]interface{}{
		"name": "node-config-pipe",
		"nodes": []interface{}{
			map[string]interface{}{
				"id": "step-a",
				"component": map[string]interface{}{
					"name":  "detector",
					"image": "busybox",
				},
				"runtimeConfig": map[string]interface{}{
					"mode":           "saved",
					"configId":       "cfg-draft",
					"version":        1,
					"mountPath":      "/workspace/configs",
					"targetFilename": "draft.yaml",
				},
			},
		},
		"edges": []interface{}{},
	}
	uc := newUsecase(newMockAssetRepo())
	uc.pipelineConfigRepo = &mockPipelineConfigRepo{
		configs: map[string]*models.PipelineConfig{
			"cfg-draft": {ID: "cfg-draft", Name: "draft.yaml", Lifecycle: "draft"},
		},
		versions: map[string]*models.PipelineConfigVersion{
			"cfg-draft:1": {ConfigID: "cfg-draft", Version: 1, Status: "draft", Content: "draft: true\n"},
		},
	}
	uc.runtimeConfigStore = &mockRuntimeConfigStore{volumeName: "runtime-config-nodes"}

	_, err := uc.Deploy(ctx, pipe, "", nil)
	if err == nil {
		t.Fatal("expected deploy to reject draft node config")
	}
	if !strings.Contains(err.Error(), "node step-a") || !strings.Contains(err.Error(), "not ready") {
		t.Fatalf("expected node-specific not-ready error, got %v", err)
	}
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

	list, err := uc.listDeployments(ctx, true)
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
func (m *mockRunRepo) FindAllSummaries(_ context.Context) ([]models.PipelineRun, error) {
	return m.FindAll(context.Background())
}
func (m *mockRunRepo) ListSummaries(_ context.Context, filter models.PipelineRunListFilter) ([]models.PipelineRun, int, error) {
	items, err := m.FindAll(context.Background())
	if err != nil {
		return nil, 0, err
	}
	return items, len(items), nil
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
func (m *mockRunRepo) FindByBatchJobAndAssetID(_ context.Context, batchJobID, assetID string) (*models.PipelineRun, error) {
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
func (m *mockRunRepo) FindAllByBatchJobAndAssetID(_ context.Context, batchJobID, assetID string) ([]models.PipelineRun, error) {
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

func (m *mockRunRepo) UpdateLedgerState(_ context.Context, id, ledgerState string) error {
	if r, ok := m.byID[id]; ok {
		r.LedgerState = ledgerState
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
				ID:            "run-1",
				WorkflowName:  "wf-1",
				Status:        "Running",
				Manifest:      &manifest,
				ArgoNamespace: "video-proc-dev",
			},
		},
		byWf: map[string]*models.PipelineRun{
			"wf-1": {
				ID:            "run-1",
				WorkflowName:  "wf-1",
				Status:        "Running",
				Manifest:      &manifest,
				ArgoNamespace: "video-proc-dev",
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
	wfClient.getWorkflowFn = func(_ context.Context, name, namespace string) (*wfv1.Workflow, error) {
		if namespace != "video-proc-dev" {
			t.Fatalf("expected workflow lookup in video-proc-dev, got %q", namespace)
		}
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

// No-op by default, with optional in-memory rows for tests that exercise
// execution-target-specific behavior.
type mockTargetRepo struct {
	byID map[string]*models.ExecutionTarget
}

func (m *mockTargetRepo) Save(_ context.Context, target *models.ExecutionTarget) error {
	if target == nil {
		return nil
	}
	if m.byID == nil {
		m.byID = map[string]*models.ExecutionTarget{}
	}
	copy := *target
	m.byID[target.ID] = &copy
	return nil
}
func (m *mockTargetRepo) FindAll(_ context.Context) ([]models.ExecutionTarget, error) {
	out := make([]models.ExecutionTarget, 0, len(m.byID))
	for _, target := range m.byID {
		out = append(out, *target)
	}
	return out, nil
}
func (m *mockTargetRepo) FindByID(_ context.Context, id string) (*models.ExecutionTarget, error) {
	if m.byID == nil {
		return nil, nil
	}
	target := m.byID[id]
	if target == nil {
		return nil, nil
	}
	copy := *target
	return &copy, nil
}
func (m *mockTargetRepo) FindDefault(_ context.Context) (*models.ExecutionTarget, error) {
	for _, target := range m.byID {
		if target.IsDefault {
			copy := *target
			return &copy, nil
		}
	}
	return nil, nil
}

type mockRunNodeRepo struct {
	byRun map[string][]models.PipelineRunNode
}

func (m *mockRunNodeRepo) ReplaceByRunID(_ context.Context, runID string, nodes []models.PipelineRunNode) error {
	if m.byRun == nil {
		m.byRun = map[string][]models.PipelineRunNode{}
	}
	m.byRun[runID] = append([]models.PipelineRunNode(nil), nodes...)
	return nil
}
func (m *mockRunNodeRepo) FindByRunID(_ context.Context, runID string) ([]models.PipelineRunNode, error) {
	if m.byRun == nil {
		return nil, nil
	}
	return append([]models.PipelineRunNode(nil), m.byRun[runID]...), nil
}
func (m *mockRunNodeRepo) DeleteByRunID(_ context.Context, runID string) error {
	if m.byRun != nil {
		delete(m.byRun, runID)
	}
	return nil
}

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

func TestRefreshRunStatus_SkipsBatchPlaceholderNotFound(t *testing.T) {
	ctx := context.Background()
	batchJobID := "batch-1"
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-batch": {
				ID:           "run-batch",
				WorkflowName: "pipe-batch-asset123",
				Status:       "Pending",
				BatchJobID:   &batchJobID,
				CreatedAt:    time.Now().UTC(),
			},
		},
	}
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, name, namespace string) (*wfv1.Workflow, error) {
		return nil, argo.ErrNotFound
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})

	uc.refreshRunStatus(ctx, runRepo.byID["run-batch"])
	if runRepo.byID["run-batch"].Status != "Pending" {
		t.Fatalf("expected Pending placeholder run to stay Pending, got %q", runRepo.byID["run-batch"].Status)
	}
}

func TestRefreshRunStatus_PlaceholderWithStaleTTLMessageStaysPending(t *testing.T) {
	ctx := context.Background()
	batchJobID := "batch-1"
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-batch": {
				ID:           "run-batch",
				WorkflowName: "pipe-batch-asset123",
				Status:       "Running",
				Message:      staleWorkflowTTLCleanupMessage,
				BatchJobID:   &batchJobID,
				CreatedAt:    time.Now().UTC(),
			},
		},
	}
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
		return nil, argo.ErrNotFound
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})

	uc.refreshRunStatus(ctx, runRepo.byID["run-batch"])
	run := runRepo.byID["run-batch"]
	if run.Status != "Running" {
		t.Fatalf("expected Running placeholder run, got %q", run.Status)
	}
	if run.Message != "" {
		t.Fatalf("expected stale TTL message cleared, got %q", run.Message)
	}
}

func TestRefreshRunStatus_KeepsActiveRunWithWorkflowUIDActiveOnNotFound(t *testing.T) {
	ctx := context.Background()
	createdAt := time.Now().UTC().Add(-10 * time.Minute)
	batchJobID := "batch-1"
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:              "run-1",
				WorkflowName:    "wf-video-proc",
				Status:          "Running",
				ArgoNamespace:   "video-proc-dev",
				ArgoWorkflowUID: "uid-1",
				Message:         staleWorkflowTTLCleanupMessage,
				BatchJobID:      &batchJobID,
				CreatedAt:       createdAt,
			},
		},
	}
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, name, namespace string) (*wfv1.Workflow, error) {
		if name != "wf-video-proc" || namespace != "video-proc-dev" {
			t.Fatalf("unexpected workflow lookup name=%q namespace=%q", name, namespace)
		}
		return nil, argo.ErrNotFound
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})

	uc.refreshRunStatus(ctx, runRepo.byID["run-1"])

	run := runRepo.byID["run-1"]
	if run.Status != "Running" {
		t.Fatalf("expected active run to stay Running, got %q", run.Status)
	}
	if run.Message != "" {
		t.Fatalf("expected stale TTL message cleared for active run, got %q", run.Message)
	}
	if run.FinishedAt != nil {
		t.Fatalf("expected active run to remain unfinished, got %v", run.FinishedAt)
	}
}

func TestListRunSummaries_DefaultBatchViewDoesNotRefreshActiveRuns(t *testing.T) {
	ctx := context.Background()
	batchJobID := "batch-1"
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:           "run-1",
				WorkflowName: "wf-1",
				Status:       "Running",
				BatchJobID:   &batchJobID,
				CreatedAt:    time.Now().UTC(),
			},
		},
	}
	getWorkflowCalls := 0
	wfClient := &mockWorkflowClient{
		getWorkflowFn: func(_ context.Context, name, namespace string) (*wfv1.Workflow, error) {
			getWorkflowCalls++
			return &wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
				Status:     wfv1.WorkflowStatus{Phase: wfv1.WorkflowRunning},
			}, nil
		},
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})

	items, total, err := uc.ListRunSummaries(ctx, models.PipelineRunListFilter{BatchJobID: batchJobID})
	if err != nil {
		t.Fatalf("ListRunSummaries: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("expected one summary, total=%d len=%d", total, len(items))
	}
	if getWorkflowCalls != 0 {
		t.Fatalf("default batch summary should not refresh Argo, got %d calls", getWorkflowCalls)
	}
}

func TestListRunSummaries_RefreshActiveOptInRefreshesActiveRuns(t *testing.T) {
	ctx := context.Background()
	batchJobID := "batch-1"
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:           "run-1",
				WorkflowName: "wf-1",
				Status:       "Running",
				BatchJobID:   &batchJobID,
				CreatedAt:    time.Now().UTC(),
			},
		},
	}
	getWorkflowCalls := 0
	wfClient := &mockWorkflowClient{
		getWorkflowFn: func(_ context.Context, name, namespace string) (*wfv1.Workflow, error) {
			getWorkflowCalls++
			return &wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
				Status:     wfv1.WorkflowStatus{Phase: wfv1.WorkflowRunning},
			}, nil
		},
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})

	_, _, err := uc.ListRunSummaries(ctx, models.PipelineRunListFilter{
		BatchJobID:    batchJobID,
		RefreshActive: true,
	})
	if err != nil {
		t.Fatalf("ListRunSummaries: %v", err)
	}
	if getWorkflowCalls == 0 {
		t.Fatal("expected refreshActive batch summary to refresh Argo")
	}
}

func TestRefreshRunForList_ReconcilesMisclassifiedError(t *testing.T) {
	ctx := context.Background()
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:           "run-1",
				WorkflowName: "wf-1",
				Status:       "Error",
				Message:      staleWorkflowTTLCleanupMessage,
			},
		},
	}
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, name, _ string) (*wfv1.Workflow, error) {
		if name != "wf-1" {
			t.Fatalf("unexpected workflow name %q", name)
		}
		return &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "wf-1", UID: "uid-1"},
			Status:     wfv1.WorkflowStatus{Phase: wfv1.WorkflowRunning},
		}, nil
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})

	run := runRepo.byID["run-1"]
	uc.RefreshRunForList(ctx, run)
	if run.Status != "Running" {
		t.Fatalf("expected reconciled Running status, got %q", run.Status)
	}
}

func TestRefreshRunForList_LiveWorkflowWinsOverStaleLedger(t *testing.T) {
	ctx := context.Background()
	staleFinishedAt := time.Date(2026, 6, 19, 9, 37, 26, 0, time.UTC)
	startedAt := time.Date(2026, 6, 19, 9, 30, 50, 0, time.UTC)
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:              "run-1",
				WorkflowName:    "wf-1",
				Status:          "Error",
				Message:         staleWorkflowTTLCleanupMessage,
				ArgoNamespace:   "video-proc-dev",
				ArgoWorkflowUID: "uid-1",
				CreatedAt:       startedAt,
				FinishedAt:      &staleFinishedAt,
			},
		},
	}
	assetNodeRepo := &mockAssetNodeRepo{
		byRun: map[string][]models.PipelineRunAssetNode{
			"run-1": {
				{RunID: "run-1", PipelineNodeID: "step-1", Status: "Error", Message: staleWorkflowTTLCleanupMessage},
			},
		},
	}
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, name, namespace string) (*wfv1.Workflow, error) {
		if name != "wf-1" || namespace != "video-proc-dev" {
			t.Fatalf("unexpected workflow lookup name=%q namespace=%q", name, namespace)
		}
		return &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "wf-1", Namespace: "video-proc-dev", UID: "uid-1"},
			Status: wfv1.WorkflowStatus{
				Phase:     wfv1.WorkflowRunning,
				StartedAt: metav1.Time{Time: startedAt},
				Nodes: map[string]wfv1.NodeStatus{
					"node-1": {
						ID:           "node-1",
						Name:         "wf-1.step-1",
						DisplayName:  "step-1",
						Type:         wfv1.NodeTypePod,
						Phase:        wfv1.NodePending,
						Message:      "Unschedulable: 0/12 nodes are available: 1 Insufficient ephemeral-storage.",
						StartedAt:    metav1.Time{Time: startedAt},
						TemplateName: "step-1",
					},
				},
			},
		}, nil
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetObservabilityRepositories(assetNodeRepo, nil, nil)
	uc.SetResourceGuardConfig(ResourceGuardConfig{UnschedulablePendingThreshold: 15 * time.Minute})
	uc.now = func() time.Time { return startedAt.Add(7 * time.Minute) }

	run := runRepo.byID["run-1"]
	uc.RefreshRunForList(ctx, run)
	if run.Status != string(wfv1.WorkflowRunning) {
		t.Fatalf("expected live workflow to keep run Running, got %q", run.Status)
	}
	if run.Message != "" {
		t.Fatalf("expected stale terminal message cleared, got %q", run.Message)
	}
	if run.FinishedAt != nil {
		t.Fatalf("expected finished_at cleared for live workflow, got %v", run.FinishedAt)
	}
}

func TestRefreshRunForList_RevivesRecentTTLNotFoundMisclassification(t *testing.T) {
	ctx := context.Background()
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:              "run-1",
				WorkflowName:    "wf-1",
				Status:          "Error",
				Message:         staleWorkflowTTLCleanupMessage,
				ArgoNamespace:   "video-proc-dev",
				ArgoWorkflowUID: "uid-1",
				CreatedAt:       time.Now().UTC().Add(-20 * time.Minute),
			},
		},
	}
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, name, namespace string) (*wfv1.Workflow, error) {
		if name != "wf-1" || namespace != "video-proc-dev" {
			t.Fatalf("unexpected workflow lookup name=%q namespace=%q", name, namespace)
		}
		return nil, argo.ErrNotFound
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetObservabilityRepositories(&mockAssetNodeRepo{}, nil, nil)

	run := runRepo.byID["run-1"]
	uc.RefreshRunForList(ctx, run)
	if run.Status != "Running" {
		t.Fatalf("expected revived Running status, got %q", run.Status)
	}
	if run.Message != "" {
		t.Fatalf("expected stale TTL message cleared, got %q", run.Message)
	}
	if run.FinishedAt != nil {
		t.Fatalf("expected revived run to be unfinished, got %v", run.FinishedAt)
	}
}

func TestListBatchAssetRuns_RefreshesMisclassifiedAttempt(t *testing.T) {
	ctx := context.Background()
	batchJobID := "batch-1"
	assetID := "asset-1"
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:           "run-1",
				WorkflowName: "wf-1",
				Status:       "Error",
				Message:      staleWorkflowTTLCleanupMessage,
				BatchJobID:   &batchJobID,
				AssetIDs:     []string{assetID},
			},
		},
	}
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, name, _ string) (*wfv1.Workflow, error) {
		if name != "wf-1" {
			t.Fatalf("unexpected workflow name %q", name)
		}
		return &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "wf-1", UID: "uid-1"},
			Status:     wfv1.WorkflowStatus{Phase: wfv1.WorkflowRunning},
		}, nil
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})

	runs, err := uc.ListBatchAssetRuns(ctx, batchJobID, assetID)
	if err != nil {
		t.Fatalf("ListBatchAssetRuns: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected one run, got %d", len(runs))
	}
	if runs[0].Status != "Running" {
		t.Fatalf("expected returned run status Running, got %q", runs[0].Status)
	}
	if runRepo.byID["run-1"].Status != "Running" {
		t.Fatalf("expected persisted run status Running, got %q", runRepo.byID["run-1"].Status)
	}
}

func TestGetRun_ReconcilesNewBatchPlaceholderErrorToPending(t *testing.T) {
	ctx := context.Background()
	batchJobID := "batch-1"
	finishedAt := time.Now().UTC()
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-batch": {
				ID:           "run-batch",
				WorkflowName: "pipe-batch-asset123-runbatch",
				Status:       "Error",
				Message:      staleWorkflowTTLCleanupMessage,
				BatchJobID:   &batchJobID,
				CreatedAt:    time.Now().UTC(),
				FinishedAt:   &finishedAt,
			},
		},
	}
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
		return nil, argo.ErrNotFound
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})

	run, err := uc.GetRun(ctx, "run-batch")
	if err != nil {
		t.Fatalf("GetRun() error = %v", err)
	}
	if run.Status != "Pending" {
		t.Fatalf("status = %q, want Pending", run.Status)
	}
	if run.FinishedAt != nil {
		t.Fatalf("finishedAt = %v, want nil", run.FinishedAt)
	}
}

func TestGetRun_ReconcilesMisclassifiedError(t *testing.T) {
	ctx := context.Background()
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:           "run-1",
				WorkflowName: "wf-1",
				Status:       "Error",
				Message:      "Argo 工作流已被 TTL 清理",
			},
		},
	}
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, name, namespace string) (*wfv1.Workflow, error) {
		if name != "wf-1" {
			t.Fatalf("unexpected workflow name %q", name)
		}
		return &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "wf-1", UID: "uid-1"},
			Status:     wfv1.WorkflowStatus{Phase: wfv1.WorkflowSucceeded},
		}, nil
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetRunEventRepo(&mockRunEventRepo{})

	run, err := uc.GetRun(ctx, "run-1")
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if run.Status != "Succeeded" {
		t.Fatalf("expected reconciled Succeeded status, got %q", run.Status)
	}
}

type mockAssetNodeRepo struct {
	byRun map[string][]models.PipelineRunAssetNode
}

func (m *mockAssetNodeRepo) ReplaceByRunID(_ context.Context, runID string, rows []models.PipelineRunAssetNode) error {
	if m.byRun == nil {
		m.byRun = map[string][]models.PipelineRunAssetNode{}
	}
	m.byRun[runID] = append([]models.PipelineRunAssetNode(nil), rows...)
	return nil
}

func (m *mockAssetNodeRepo) ListByRunID(_ context.Context, runID string, _ models.PipelineRunAssetNodeListOptions) (*models.PipelineRunAssetNodeListResult, error) {
	items := []models.PipelineRunAssetNode{}
	if m.byRun != nil {
		items = append(items, m.byRun[runID]...)
	}
	return &models.PipelineRunAssetNodeListResult{Items: items, Total: len(items)}, nil
}

func (m *mockAssetNodeRepo) ListByRunIDs(_ context.Context, runIDs []string) ([]models.PipelineRunAssetNode, error) {
	out := make([]models.PipelineRunAssetNode, 0)
	for _, runID := range runIDs {
		if m.byRun != nil {
			out = append(out, m.byRun[runID]...)
		}
	}
	return out, nil
}

func TestReconcileTerminalRunFromLedger_StuckRunningWithSucceededNodes(t *testing.T) {
	ctx := context.Background()
	finishedAt := time.Now().UTC().Add(-time.Hour)
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:           "run-1",
				WorkflowName: "wf-1",
				Status:       "Running",
				FinishedAt:   &finishedAt,
			},
		},
	}
	assetNodeRepo := &mockAssetNodeRepo{
		byRun: map[string][]models.PipelineRunAssetNode{
			"run-1": {
				{RunID: "run-1", AssetID: "a1", PipelineNodeID: "n1", Status: "Succeeded"},
				{RunID: "run-1", AssetID: "a1", PipelineNodeID: "n2", Status: "Succeeded"},
			},
		},
	}
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
		return nil, argo.ErrNotFound
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetObservabilityRepositories(assetNodeRepo, nil, nil)

	run := runRepo.byID["run-1"]
	uc.RefreshRunForList(ctx, run)
	if run.Status != "Succeeded" {
		t.Fatalf("expected Succeeded from ledger, got %q", run.Status)
	}
}

func TestRefreshRunForList_WorkflowMissingUsesDiagnosticAssetNode(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 18, 10, 0, 0, 0, time.UTC)
	createdAt := now.Add(-10 * time.Minute)
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:              "run-1",
				WorkflowName:    "wf-1",
				ArgoWorkflowUID: "uid-1",
				Status:          "Running",
				CreatedAt:       createdAt,
			},
		},
	}
	assetNodeRepo := &mockAssetNodeRepo{
		byRun: map[string][]models.PipelineRunAssetNode{
			"run-1": {
				{
					RunID:          "run-1",
					AssetID:        "a1",
					PipelineNodeID: "step-head-tracking",
					Status:         "Error",
					Message:        "Unschedulable: 0/12 nodes are available: 2 Insufficient ephemeral-storage.",
				},
				{RunID: "run-1", AssetID: "a1", PipelineNodeID: "step-hand-detection", Status: "Pending"},
			},
		},
	}
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
		return nil, argo.ErrNotFound
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetObservabilityRepositories(assetNodeRepo, nil, nil)
	uc.now = func() time.Time { return now }

	run := runRepo.byID["run-1"]
	uc.RefreshRunForList(ctx, run)
	if run.Status != string(wfv1.WorkflowError) {
		t.Fatalf("expected Error from diagnostic asset node, got %q", run.Status)
	}
	if !strings.Contains(run.Message, "Insufficient ephemeral-storage") {
		t.Fatalf("expected scheduler diagnostic message, got %q", run.Message)
	}
	if run.FinishedAt == nil || !run.FinishedAt.Equal(now) {
		t.Fatalf("expected finished_at %v, got %v", now, run.FinishedAt)
	}
}

func TestRefreshRunForList_StaleTerminalMessageUsesDiagnosticAssetNode(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 18, 10, 0, 0, 0, time.UTC)
	finishedAt := now.Add(-time.Minute)
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:           "run-1",
				WorkflowName: "wf-1",
				Status:       string(wfv1.WorkflowError),
				Message:      staleWorkflowTTLCleanupMessage,
				FinishedAt:   &finishedAt,
				CreatedAt:    now.Add(-10 * time.Minute),
			},
		},
	}
	assetNodeRepo := &mockAssetNodeRepo{
		byRun: map[string][]models.PipelineRunAssetNode{
			"run-1": {
				{
					RunID:          "run-1",
					AssetID:        "a1",
					PipelineNodeID: "step-head-tracking",
					Status:         "Error",
					Message:        "Unschedulable: 0/12 nodes are available: 2 Insufficient ephemeral-storage.",
				},
				{RunID: "run-1", AssetID: "a1", PipelineNodeID: "step-hand-detection", Status: "Pending"},
			},
		},
	}
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
		return nil, argo.ErrNotFound
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetObservabilityRepositories(assetNodeRepo, nil, nil)
	uc.now = func() time.Time { return now }

	run := runRepo.byID["run-1"]
	uc.RefreshRunForList(ctx, run)
	if run.Status != string(wfv1.WorkflowError) {
		t.Fatalf("expected Error status to remain, got %q", run.Status)
	}
	if !strings.Contains(run.Message, "Insufficient ephemeral-storage") {
		t.Fatalf("expected stale TTL message replaced with scheduler diagnostic, got %q", run.Message)
	}
	if run.FinishedAt == nil || !run.FinishedAt.Equal(finishedAt) {
		t.Fatalf("expected original finished_at %v, got %v", finishedAt, run.FinishedAt)
	}
}

func TestInferRunStatusFromAssetNodes(t *testing.T) {
	status, ok := inferRunStatusFromAssetNodes([]models.PipelineRunAssetNode{
		{Status: "Succeeded"},
		{Status: "Skipped"},
	})
	if !ok || status != "Succeeded" {
		t.Fatalf("expected Succeeded, got %q ok=%v", status, ok)
	}
	status, ok = inferRunStatusFromAssetNodes([]models.PipelineRunAssetNode{
		{Status: "Succeeded"},
		{Status: "Failed"},
	})
	if !ok || status != "Failed" {
		t.Fatalf("expected Failed, got %q ok=%v", status, ok)
	}
	status, ok = inferRunStatusFromAssetNodes([]models.PipelineRunAssetNode{
		{Status: "Error", Message: "Unschedulable: 0/12 nodes are available: 2 Insufficient ephemeral-storage."},
		{Status: "Pending"},
	})
	if !ok || status != "Error" {
		t.Fatalf("expected Error with downstream pending placeholders, got %q ok=%v", status, ok)
	}
	status, ok = inferRunStatusFromAssetNodes([]models.PipelineRunAssetNode{
		{Status: "Pending", Message: "Unschedulable: 0/12 nodes are available: 2 Insufficient ephemeral-storage."},
		{Status: "Pending"},
	})
	if ok {
		t.Fatalf("expected pending scheduler diagnostic to stay non-terminal, got %q", status)
	}
	_, ok = inferRunStatusFromAssetNodes([]models.PipelineRunAssetNode{{Status: "Running"}})
	if ok {
		t.Fatal("expected non-terminal when nodes still running")
	}
}

func TestProjectTerminalActiveAssetNodesMarksRunningNodeError(t *testing.T) {
	finishedAt := time.Date(2026, 6, 18, 18, 46, 0, 0, time.UTC)
	result := &models.PipelineRunAssetNodeListResult{
		Items: []models.PipelineRunAssetNode{
			{RunID: "run-1", AssetID: "asset-1", PipelineNodeID: "step-a", Status: "Succeeded"},
			{RunID: "run-1", AssetID: "asset-1", PipelineNodeID: "step-b", Status: "Running"},
			{RunID: "run-1", AssetID: "asset-1", PipelineNodeID: "step-c", Status: "Pending"},
		},
		Summary: models.PipelineRunAssetNodeSummary{
			Statuses: map[string]int{"Succeeded": 1, "Running": 1, "Pending": 1},
		},
	}
	run := &models.PipelineRun{
		ID:         "run-1",
		Status:     "Error",
		Message:    staleWorkflowTTLCleanupMessage,
		FinishedAt: &finishedAt,
	}

	projectTerminalActiveAssetNodes(result, run)

	if result.Items[1].Status != "Error" {
		t.Fatalf("running node status = %q, want Error", result.Items[1].Status)
	}
	if result.Items[1].Message != staleWorkflowTTLCleanupMessage {
		t.Fatalf("message = %q", result.Items[1].Message)
	}
	if result.Items[1].FinishedAt == nil || !result.Items[1].FinishedAt.Equal(finishedAt) {
		t.Fatalf("finishedAt = %v, want %v", result.Items[1].FinishedAt, finishedAt)
	}
	if result.Summary.Statuses["Error"] != 1 || result.Summary.Statuses["Running"] != 0 {
		t.Fatalf("summary statuses = %#v", result.Summary.Statuses)
	}
}

func TestPipelineRunNodePhase_KeepsUnschedulablePendingActive(t *testing.T) {
	phase := pipelineRunNodePhase(wfv1.NodeStatus{
		Phase:   wfv1.NodePending,
		Message: "Unschedulable: 0/12 nodes are available: 2 Insufficient ephemeral-storage.",
	})
	if phase != string(wfv1.NodePending) {
		t.Fatalf("phase = %q, want Pending", phase)
	}
}

func TestGetRun_UsesArgoFinishedAtOnTerminalWorkflow(t *testing.T) {
	ctx := context.Background()
	actualFinish := time.Date(2026, 6, 16, 9, 39, 43, 0, time.UTC)
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
				FinishedAt: metav1.Time{Time: actualFinish},
			},
		}, nil
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetRunEventRepo(&mockRunEventRepo{})

	run, err := uc.GetRun(ctx, "run-1")
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if run.FinishedAt == nil || !run.FinishedAt.Equal(actualFinish) {
		t.Fatalf("expected Argo finished_at %v, got %v", actualFinish, run.FinishedAt)
	}
}

func TestRefreshRunForList_ClearsFinishedAtForActiveRun(t *testing.T) {
	ctx := context.Background()
	pollutedFinish := time.Date(2026, 6, 17, 3, 28, 23, 0, time.UTC)
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:           "run-1",
				WorkflowName: "wf-1",
				Status:       "Running",
				FinishedAt:   &pollutedFinish,
			},
		},
	}
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, name, _ string) (*wfv1.Workflow, error) {
		if name != "wf-1" {
			t.Fatalf("unexpected workflow name %q", name)
		}
		return &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "wf-1", UID: "uid-1"},
			Status:     wfv1.WorkflowStatus{Phase: wfv1.WorkflowRunning},
		}, nil
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})

	run := runRepo.byID["run-1"]
	uc.RefreshRunForList(ctx, run)
	if run.FinishedAt != nil {
		t.Fatalf("expected finished_at cleared for active run, got %v", *run.FinishedAt)
	}
	if run.Status != string(wfv1.WorkflowRunning) {
		t.Fatalf("expected Running status, got %q", run.Status)
	}
}

func TestRefreshRunForList_DerivesFailedStatusFromShutdownNode(t *testing.T) {
	ctx := context.Background()
	finishedAt := time.Date(2026, 6, 17, 10, 12, 13, 0, time.UTC)
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:           "run-1",
				WorkflowName: "wf-1",
				Status:       "Running",
			},
		},
	}
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, name, _ string) (*wfv1.Workflow, error) {
		if name != "wf-1" {
			t.Fatalf("unexpected workflow name %q", name)
		}
		return &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "wf-1", UID: "uid-1"},
			Status: wfv1.WorkflowStatus{
				Phase: wfv1.WorkflowRunning,
				Nodes: map[string]wfv1.NodeStatus{
					"node-1": {
						ID:           "node-1",
						Name:         "wf-1-step",
						DisplayName:  "step",
						Type:         wfv1.NodeTypePod,
						Phase:        wfv1.NodeFailed,
						Message:      "workflow shutdown with strategy: Stop",
						FinishedAt:   metav1.Time{Time: finishedAt},
						TemplateName: "step",
					},
				},
			},
		}, nil
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})

	run := runRepo.byID["run-1"]
	uc.RefreshRunForList(ctx, run)
	if run.Status != string(wfv1.WorkflowFailed) {
		t.Fatalf("expected derived Failed status, got %q", run.Status)
	}
	if run.Message != "workflow shutdown with strategy: Stop" {
		t.Fatalf("expected shutdown message, got %q", run.Message)
	}
	if run.FinishedAt == nil || !run.FinishedAt.Equal(finishedAt) {
		t.Fatalf("expected finished_at %v, got %v", finishedAt, run.FinishedAt)
	}
}

func TestRefreshRunForList_KeepsShortUnschedulablePendingActive(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 18, 10, 0, 0, 0, time.UTC)
	startedAt := now.Add(-5 * time.Minute)
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:           "run-1",
				WorkflowName: "wf-1",
				Status:       "Running",
				CreatedAt:    startedAt,
			},
		},
	}
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, name, _ string) (*wfv1.Workflow, error) {
		if name != "wf-1" {
			t.Fatalf("unexpected workflow name %q", name)
		}
		return &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "wf-1", UID: "uid-1", CreationTimestamp: metav1.Time{Time: startedAt}},
			Status: wfv1.WorkflowStatus{
				Phase:     wfv1.WorkflowRunning,
				StartedAt: metav1.Time{Time: startedAt},
				Nodes: map[string]wfv1.NodeStatus{
					"node-1": {
						ID:           "node-1",
						Name:         "wf-1-step",
						DisplayName:  "step",
						Type:         wfv1.NodeTypePod,
						Phase:        wfv1.NodePending,
						Message:      "0/11 nodes are available: 1 Insufficient cpu.",
						StartedAt:    metav1.Time{Time: startedAt},
						TemplateName: "step",
					},
				},
			},
		}, nil
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetResourceGuardConfig(ResourceGuardConfig{UnschedulablePendingThreshold: 15 * time.Minute})
	uc.now = func() time.Time { return now }

	run := runRepo.byID["run-1"]
	uc.RefreshRunForList(ctx, run)
	if run.Status != string(wfv1.WorkflowRunning) {
		t.Fatalf("expected Running below threshold, got %q", run.Status)
	}
	if run.Message != "" {
		t.Fatalf("expected no terminal message below threshold, got %q", run.Message)
	}
	if run.FinishedAt != nil {
		t.Fatalf("expected no finished_at below threshold, got %v", run.FinishedAt)
	}
}

func TestRefreshRunForList_MarksLongUnschedulablePendingError(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 18, 10, 0, 0, 0, time.UTC)
	startedAt := now.Add(-20 * time.Minute)
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:           "run-1",
				WorkflowName: "wf-1",
				Status:       "Running",
				CreatedAt:    startedAt,
			},
		},
	}
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, name, _ string) (*wfv1.Workflow, error) {
		if name != "wf-1" {
			t.Fatalf("unexpected workflow name %q", name)
		}
		return &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "wf-1", UID: "uid-1", CreationTimestamp: metav1.Time{Time: startedAt}},
			Status: wfv1.WorkflowStatus{
				Phase:     wfv1.WorkflowRunning,
				StartedAt: metav1.Time{Time: startedAt},
				Nodes: map[string]wfv1.NodeStatus{
					"node-1": {
						ID:           "node-1",
						Name:         "wf-1-step",
						DisplayName:  "step",
						Type:         wfv1.NodeTypePod,
						Phase:        wfv1.NodePending,
						Message:      "0/11 nodes are available: 1 Insufficient cpu, 2 Insufficient memory.",
						StartedAt:    metav1.Time{Time: startedAt},
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
	uc.SetResourceGuardConfig(ResourceGuardConfig{UnschedulablePendingThreshold: 15 * time.Minute})
	uc.now = func() time.Time { return now }

	run := runRepo.byID["run-1"]
	uc.RefreshRunForList(ctx, run)
	if run.Status != string(wfv1.WorkflowError) {
		t.Fatalf("expected Error for over-threshold unschedulable run, got %q", run.Status)
	}
	if !strings.Contains(run.Message, "Insufficient cpu") || !strings.Contains(run.Message, "Pending 20m0s") {
		t.Fatalf("expected scheduler diagnostics in message, got %q", run.Message)
	}
	if run.FinishedAt == nil || !run.FinishedAt.Equal(now) {
		t.Fatalf("expected finished_at %v, got %v", now, run.FinishedAt)
	}
	foundEvent := false
	for _, event := range eventRepo.events {
		if event.EventType == runEventFailed && event.Reason == "unschedulable" {
			foundEvent = true
			break
		}
	}
	if !foundEvent {
		t.Fatalf("expected unschedulable run_failed event, got %#v", eventRepo.events)
	}
}

func TestRefreshRunForList_MarksLongInvalidImageNamePendingError(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 18, 10, 0, 0, 0, time.UTC)
	startedAt := now.Add(-20 * time.Minute)
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:            "run-1",
				WorkflowName:  "wf-1",
				Status:        "Running",
				ArgoNamespace: "video-proc-dev",
				CreatedAt:     startedAt,
			},
		},
	}
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, name, namespace string) (*wfv1.Workflow, error) {
		if name != "wf-1" || namespace != "video-proc-dev" {
			t.Fatalf("unexpected workflow lookup name=%q namespace=%q", name, namespace)
		}
		return &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "wf-1", UID: "uid-1", CreationTimestamp: metav1.Time{Time: startedAt}},
			Status: wfv1.WorkflowStatus{
				Phase:     wfv1.WorkflowRunning,
				StartedAt: metav1.Time{Time: startedAt},
				Nodes: map[string]wfv1.NodeStatus{
					"node-1": {
						ID:           "node-1",
						Name:         "wf-1-step",
						DisplayName:  "step",
						Type:         wfv1.NodeTypePod,
						Phase:        wfv1.NodePending,
						Message:      `InvalidImageName: Failed to apply default image tag "repo/image@sha256:1234": couldn't parse image name "repo/image@sha256:1234": invalid reference format`,
						StartedAt:    metav1.Time{Time: startedAt},
						TemplateName: "step",
					},
				},
			},
		}, nil
	}
	eventRepo := &mockRunEventRepo{}
	nodeRepo := &mockRunNodeRepo{}
	assetNodeRepo := &mockAssetNodeRepo{}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, nodeRepo)
	uc.SetRunEventRepo(eventRepo)
	uc.SetObservabilityRepositories(assetNodeRepo, nil, nil)
	uc.SetResourceGuardConfig(ResourceGuardConfig{UnschedulablePendingThreshold: 15 * time.Minute})
	uc.now = func() time.Time { return now }

	run := runRepo.byID["run-1"]
	uc.RefreshRunForList(ctx, run)
	if run.Status != string(wfv1.WorkflowError) {
		t.Fatalf("expected Error for over-threshold invalid image run, got %q", run.Status)
	}
	if !strings.Contains(run.Message, "Kubernetes 镜像启动失败") || !strings.Contains(run.Message, "InvalidImageName") {
		t.Fatalf("expected image diagnostics in message, got %q", run.Message)
	}
	if run.FinishedAt == nil || !run.FinishedAt.Equal(now) {
		t.Fatalf("expected finished_at %v, got %v", now, run.FinishedAt)
	}
	if len(nodeRepo.byRun["run-1"]) == 0 {
		t.Fatal("expected node snapshots to be stored")
	}
	foundNodeError := false
	for _, node := range nodeRepo.byRun["run-1"] {
		if node.DisplayName == "step" && node.Phase == string(wfv1.NodeError) && strings.Contains(node.Message, "InvalidImageName") {
			foundNodeError = true
			if node.FinishedAt == nil {
				t.Fatal("expected derived image-startup node to have finished_at")
			}
			break
		}
	}
	if !foundNodeError {
		t.Fatalf("expected invalid-image node snapshot Error, got %#v", nodeRepo.byRun["run-1"])
	}
	if len(assetNodeRepo.byRun["run-1"]) == 0 {
		t.Fatal("expected asset-node rows to be stored")
	}
	foundAssetNodeError := false
	for _, row := range assetNodeRepo.byRun["run-1"] {
		if row.DisplayName == "step" && row.Status == string(wfv1.NodeError) && strings.Contains(row.Message, "InvalidImageName") {
			foundAssetNodeError = true
			break
		}
	}
	if !foundAssetNodeError {
		t.Fatalf("expected invalid-image asset-node Error, got %#v", assetNodeRepo.byRun["run-1"])
	}
	foundEvent := false
	for _, event := range eventRepo.events {
		if event.EventType == runEventFailed && event.Reason == "image_startup" {
			foundEvent = true
			break
		}
	}
	if !foundEvent {
		t.Fatalf("expected image_startup run_failed event, got %#v", eventRepo.events)
	}
}

func TestRefreshRunForList_DoesNotAdvanceFinishedAtOnStaleMessageReconcile(t *testing.T) {
	ctx := context.Background()
	correctFinish := time.Date(2026, 6, 16, 9, 39, 43, 0, time.UTC)
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:           "run-1",
				WorkflowName: "wf-1",
				Status:       "Succeeded",
				FinishedAt:   &correctFinish,
				Message:      messageWorkflowUnavailable,
			},
		},
	}
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, name, _ string) (*wfv1.Workflow, error) {
		if name != "wf-1" {
			t.Fatalf("unexpected workflow name %q", name)
		}
		return &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "wf-1", UID: "uid-1"},
			Status:     wfv1.WorkflowStatus{Phase: wfv1.WorkflowSucceeded},
		}, nil
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})

	run := runRepo.byID["run-1"]
	before := *run.FinishedAt
	uc.RefreshRunForList(ctx, run)
	if run.FinishedAt == nil {
		t.Fatal("expected finished_at to remain set")
	}
	if run.FinishedAt.After(before) {
		t.Fatalf("finished_at advanced from %v to %v", before, *run.FinishedAt)
	}
	if !run.FinishedAt.Equal(correctFinish) {
		t.Fatalf("expected finished_at %v, got %v", correctFinish, *run.FinishedAt)
	}
}

func TestPersistRunObservation_ActiveToTerminalOverwritesStaleFinishedAt(t *testing.T) {
	ctx := context.Background()
	staleFinish := time.Date(2026, 6, 17, 12, 18, 55, 0, time.UTC)
	terminalFinish := time.Date(2026, 6, 18, 4, 43, 39, 0, time.UTC)
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:         "run-1",
				Status:     "Running",
				FinishedAt: &staleFinish,
			},
		},
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})

	observed := &models.PipelineRun{
		ID:         "run-1",
		Status:     string(wfv1.WorkflowError),
		Message:    "Kubernetes 调度失败",
		FinishedAt: &terminalFinish,
	}
	uc.persistRunObservation(ctx, observed)

	got := runRepo.byID["run-1"]
	if got.FinishedAt == nil || !got.FinishedAt.Equal(terminalFinish) {
		t.Fatalf("expected active-to-terminal finished_at %v, got %v", terminalFinish, got.FinishedAt)
	}
}

func TestGetRun_RepairsPollutedFinishedAtFromArgo(t *testing.T) {
	ctx := context.Background()
	correctFinish := time.Date(2026, 6, 16, 9, 39, 43, 0, time.UTC)
	pollutedFinish := correctFinish.Add(16 * time.Minute)
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:           "run-1",
				WorkflowName: "wf-1",
				Status:       "Succeeded",
				FinishedAt:   &pollutedFinish,
				Message:      messageWorkflowUnavailable,
			},
		},
	}
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
		return &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "wf-1", UID: "uid-1"},
			Status: wfv1.WorkflowStatus{
				Phase:      wfv1.WorkflowSucceeded,
				FinishedAt: metav1.Time{Time: correctFinish},
			},
		}, nil
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetRunEventRepo(&mockRunEventRepo{})

	run, err := uc.GetRun(ctx, "run-1")
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if run.FinishedAt == nil || !run.FinishedAt.Equal(correctFinish) {
		t.Fatalf("expected repaired finished_at %v, got %v", correctFinish, run.FinishedAt)
	}
}
