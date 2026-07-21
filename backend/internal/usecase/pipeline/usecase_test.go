package pipeline

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
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
	runtimeadapter "github.com/CyberOrigin2077/cyber-databrew/internal/runtimeos/adapter"
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

func (m *mockTemplateRepo) GetUserPipelineStatsAfter(_ context.Context, _ string, _ time.Time) (*models.PipelineUserStats, error) {
	return &models.PipelineUserStats{
		ClickCounts:     make(map[string]int),
		RunCounts:       make(map[string]int),
		ExecutionTimes:  make(map[string]int64),
		LastAccessTimes: make(map[string]time.Time),
		Window:          "30d",
		ComputedAt:      time.Now(),
	}, nil
}

func (m *mockTemplateRepo) GetLatestVersionWithConflictCheck(_ context.Context, id string, baseVersion int) (*models.PipelineTemplate, bool, error) {
	t := m.byID[id]
	if t == nil {
		return nil, false, nil
	}
	conflict := (baseVersion > 0 && t.Version != baseVersion)
	return t, conflict, nil
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
	listWorkflowsFn     func(ctx context.Context, namespace, labelSelector string) ([]wfv1.Workflow, error)
	stopCalls           []string
	stopErr             error
}

func (m *mockWorkflowClient) CreateWorkflow(ctx context.Context, wf *wfv1.Workflow, namespace string) error {
	if m.createWorkflowFn != nil {
		return m.createWorkflowFn(ctx, wf, namespace)
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
func (m *mockWorkflowClient) ListWorkflows(ctx context.Context, namespace string, labelSelector string) ([]wfv1.Workflow, error) {
	if m.listWorkflowsFn != nil {
		return m.listWorkflowsFn(ctx, namespace, labelSelector)
	}
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
func (m *mockWorkflowClient) StopWorkflow(_ context.Context, name, namespace string) error {
	m.stopCalls = append(m.stopCalls, namespace+"/"+name)
	return m.stopErr
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

type mockRuntimeAdapter struct {
	submitRuns    []runtimeadapter.RunRef
	submitSpecs   []runtimeadapter.RuntimeSpec
	submitJob     *runtimeadapter.RuntimeJob
	submitErr     error
	retryRefs     []runtimeadapter.RuntimeRef
	stopRefs      []runtimeadapter.RuntimeRef
	suspendRefs   []runtimeadapter.RuntimeRef
	resumeRefs    []runtimeadapter.RuntimeRef
	terminateRefs []runtimeadapter.RuntimeRef
	retryErr      error
	stopErr       error
}

func (m *mockRuntimeAdapter) Submit(_ context.Context, run runtimeadapter.RunRef, spec runtimeadapter.RuntimeSpec) (*runtimeadapter.RuntimeJob, error) {
	m.submitRuns = append(m.submitRuns, run)
	m.submitSpecs = append(m.submitSpecs, spec)
	if m.submitErr != nil {
		return nil, m.submitErr
	}
	if m.submitJob != nil {
		return m.submitJob, nil
	}
	return &runtimeadapter.RuntimeJob{
		Ref: runtimeadapter.RuntimeRef{
			RuntimeType: spec.RuntimeType,
			Name:        "adapter-workflow",
			Namespace:   spec.Namespace,
			UID:         "adapter-workflow-uid",
		},
	}, nil
}
func (m *mockRuntimeAdapter) Get(_ context.Context, ref runtimeadapter.RuntimeRef) (*runtimeadapter.RuntimeJobStatus, error) {
	return &runtimeadapter.RuntimeJobStatus{Ref: ref}, nil
}
func (m *mockRuntimeAdapter) Stop(_ context.Context, ref runtimeadapter.RuntimeRef) error {
	m.stopRefs = append(m.stopRefs, ref)
	return m.stopErr
}
func (m *mockRuntimeAdapter) Suspend(_ context.Context, ref runtimeadapter.RuntimeRef) error {
	m.suspendRefs = append(m.suspendRefs, ref)
	return nil
}
func (m *mockRuntimeAdapter) Resume(_ context.Context, ref runtimeadapter.RuntimeRef) error {
	m.resumeRefs = append(m.resumeRefs, ref)
	return nil
}
func (m *mockRuntimeAdapter) Terminate(_ context.Context, ref runtimeadapter.RuntimeRef) error {
	m.terminateRefs = append(m.terminateRefs, ref)
	return nil
}
func (m *mockRuntimeAdapter) Retry(_ context.Context, ref runtimeadapter.RuntimeRef, _ runtimeadapter.RetryOptions) (*runtimeadapter.RuntimeJob, error) {
	m.retryRefs = append(m.retryRefs, ref)
	if m.retryErr != nil {
		return nil, m.retryErr
	}
	return &runtimeadapter.RuntimeJob{Ref: ref}, nil
}
func (m *mockRuntimeAdapter) Resubmit(_ context.Context, ref runtimeadapter.RuntimeRef, _ runtimeadapter.ResubmitOptions) (*runtimeadapter.RuntimeJob, error) {
	return &runtimeadapter.RuntimeJob{Ref: ref}, nil
}
func (m *mockRuntimeAdapter) Logs(context.Context, runtimeadapter.RuntimeRef, string, runtimeadapter.LogOptions) (*runtimeadapter.LogResult, error) {
	return &runtimeadapter.LogResult{}, nil
}
func (m *mockRuntimeAdapter) LogStream(context.Context, runtimeadapter.RuntimeRef, string, runtimeadapter.LogOptions) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

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

type mockRunRelationRepo struct {
	relations []models.RunRelation
}

func (m *mockRunRelationRepo) Upsert(_ context.Context, relation *models.RunRelation) error {
	if relation == nil {
		return nil
	}
	for i := range m.relations {
		existing := &m.relations[i]
		if existing.ParentRunID == relation.ParentRunID &&
			existing.ChildRunID == relation.ChildRunID &&
			existing.RelationType == relation.RelationType {
			m.relations[i] = *relation
			return nil
		}
	}
	m.relations = append(m.relations, *relation)
	return nil
}

func (m *mockRunRelationRepo) ListByParentRunID(_ context.Context, parentRunID string) ([]models.RunRelation, error) {
	out := []models.RunRelation{}
	for _, relation := range m.relations {
		if relation.ParentRunID == parentRunID {
			out = append(out, relation)
		}
	}
	return out, nil
}

type mockRunInputRepo struct {
	inputs []models.RunInput
}

func (m *mockRunInputRepo) UpsertMany(_ context.Context, inputs []models.RunInput) error {
	for _, input := range inputs {
		replaced := false
		for i := range m.inputs {
			existing := &m.inputs[i]
			if existing.RunID == input.RunID &&
				existing.Type == input.Type &&
				existing.NodeID == input.NodeID &&
				existing.RefID == input.RefID &&
				existing.RefVersion == input.RefVersion &&
				existing.MountPath == input.MountPath &&
				existing.TargetFilename == input.TargetFilename &&
				existing.ProjectionKey == input.ProjectionKey {
				m.inputs[i] = input
				replaced = true
				break
			}
		}
		if !replaced {
			m.inputs = append(m.inputs, input)
		}
	}
	return nil
}

func (m *mockRunInputRepo) ListByRunID(_ context.Context, runID string) ([]models.RunInput, error) {
	out := []models.RunInput{}
	for _, input := range m.inputs {
		if input.RunID == runID {
			out = append(out, input)
		}
	}
	return out, nil
}
func (m *mockPipelineConfigRepo) FindVersions(context.Context, string) ([]models.PipelineConfigVersion, error) {
	return nil, nil
}
func (m *mockPipelineConfigRepo) Deprecate(context.Context, string) error {
	return nil
}
func (m *mockPipelineConfigRepo) UpdateVersionStatus(_ context.Context, configID string, version int, status string) (*models.PipelineConfigVersion, error) {
	if v := m.versions[fmt.Sprintf("%s:%d", configID, version)]; v != nil {
		v.Status = status
		return v, nil
	}
	return nil, nil
}
func (m *mockPipelineConfigRepo) UpdateLifecycle(context.Context, string, string) error {
	return nil
}
func (m *mockPipelineConfigRepo) UpdateVersionContent(_ context.Context, configID string, version int, content string, summary string) (*models.PipelineConfigVersion, error) {
	if v := m.versions[fmt.Sprintf("%s:%d", configID, version)]; v != nil {
		v.Content = content
		v.Summary = summary
		return v, nil
	}
	return nil, nil
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

type mockRuntimeConfigStoreFactory struct {
	store      RuntimeConfigStore
	lastTarget *models.ExecutionTarget
	err        error
}

func (f *mockRuntimeConfigStoreFactory) ForTarget(_ context.Context, target *models.ExecutionTarget) (RuntimeConfigStore, error) {
	f.lastTarget = target
	if f.err != nil {
		return nil, f.err
	}
	return f.store, nil
}

// ── Tests ─────────────────────────────────────────────────────────────────

// CYB-3486: the runtime-config ConfigMap must be created on the TARGET's
// cluster. When a per-cluster factory is wired, resolveRuntimeConfigStore must
// route through it (threading the target) instead of the default-cluster
// singleton; with no factory it falls back to the singleton (no-PG/test path).
func TestResolveRuntimeConfigStore_FactoryFirstElseSingleton(t *testing.T) {
	ctx := context.Background()
	target := &models.ExecutionTarget{ID: "delivery-mid", ClusterID: "c-delivery"}
	perTarget := &mockRuntimeConfigStore{volumeName: "per-target"}
	singleton := &mockRuntimeConfigStore{volumeName: "singleton"}

	factory := &mockRuntimeConfigStoreFactory{store: perTarget}
	uc := &Usecase{runtimeConfigStore: singleton, runtimeConfigStoreFactory: factory}
	got, err := uc.resolveRuntimeConfigStore(ctx, target)
	if err != nil {
		t.Fatalf("factory path: unexpected err: %v", err)
	}
	if got != perTarget {
		t.Fatalf("factory path: expected per-target store, got %#v", got)
	}
	if factory.lastTarget != target {
		t.Fatalf("factory path: target was not threaded to ForTarget")
	}

	ucNoFactory := &Usecase{runtimeConfigStore: singleton}
	got2, err := ucNoFactory.resolveRuntimeConfigStore(ctx, target)
	if err != nil {
		t.Fatalf("singleton path: unexpected err: %v", err)
	}
	if got2 != singleton {
		t.Fatalf("singleton path: expected singleton fallback, got %#v", got2)
	}
}

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

func TestDeploy_UsesRuntimeAdapterSubmitWhenConfigured(t *testing.T) {
	ctx := context.Background()
	pipe := map[string]interface{}{
		"name": "adapter-submit-pipe",
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
	adapter := &mockRuntimeAdapter{
		submitJob: &runtimeadapter.RuntimeJob{
			Ref: runtimeadapter.RuntimeRef{
				RuntimeType: "argo",
				Name:        "adapter-workflow",
				Namespace:   "runtime-ns",
				UID:         "adapter-uid",
			},
			Raw: &wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "adapter-workflow",
					Namespace: "runtime-ns",
					UID:       "adapter-uid",
				},
				Status: wfv1.WorkflowStatus{Phase: wfv1.WorkflowRunning},
			},
		},
	}
	depRepo := &mockDeploymentRepo{}
	runRepo := &mockRunRepo{}
	uc := New(&mockTemplateRepo{}, depRepo, newMockAssetRepo(), nil, "runtime-ns")
	uc.SetRuntimeAdapter(adapter)
	uc.SetRunRepositories(nil, runRepo, &mockRunNodeRepo{})

	dep, err := uc.Deploy(ctx, pipe, "", nil)
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if dep == nil {
		t.Fatal("expected deployment")
	}
	if len(adapter.submitRuns) != 1 || len(adapter.submitSpecs) != 1 {
		t.Fatalf("expected one adapter submit, got runs=%d specs=%d", len(adapter.submitRuns), len(adapter.submitSpecs))
	}
	if adapter.submitRuns[0].ID != dep.ID || adapter.submitRuns[0].Name != dep.PipelineName {
		t.Fatalf("unexpected run ref %#v for dep %#v", adapter.submitRuns[0], dep)
	}
	spec := adapter.submitSpecs[0]
	if spec.RuntimeType != "argo" || spec.Namespace != "runtime-ns" {
		t.Fatalf("unexpected runtime spec %#v", spec)
	}
	wf, ok := spec.Manifest.(*wfv1.Workflow)
	if !ok || wf == nil {
		t.Fatalf("expected workflow manifest, got %#v", spec.Manifest)
	}
	if wf.Name != dep.WorkflowName {
		t.Fatalf("expected submitted workflow %q, got %q", dep.WorkflowName, wf.Name)
	}
	if dep.Status != string(wfv1.WorkflowRunning) {
		t.Fatalf("expected status Running from adapter job, got %q", dep.Status)
	}
	run := runRepo.byID[dep.ID]
	if run == nil {
		t.Fatal("expected pipeline run to be saved")
	}
	if run.ArgoWorkflowUID != "adapter-uid" {
		t.Fatalf("expected adapter workflow uid, got %q", run.ArgoWorkflowUID)
	}
	if run.Status != string(wfv1.WorkflowRunning) || run.WorkflowName != dep.WorkflowName {
		t.Fatalf("unexpected saved run %#v", run)
	}
}

func TestDeploy_FallsBackToWorkflowClientSubmitWithoutAdapter(t *testing.T) {
	ctx := context.Background()
	pipe := map[string]interface{}{
		"name": "fallback-submit-pipe",
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
	var createdName, createdNamespace string
	wfClient := &mockWorkflowClient{
		createWorkflowFn: func(_ context.Context, wf *wfv1.Workflow, namespace string) error {
			if wf != nil {
				createdName = wf.Name
			}
			createdNamespace = namespace
			return nil
		},
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, newMockAssetRepo(), wfClient, "fallback-ns")

	dep, err := uc.Deploy(ctx, pipe, "", nil)
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if createdName != dep.WorkflowName {
		t.Fatalf("expected workflow client to create %q, got %q", dep.WorkflowName, createdName)
	}
	if createdNamespace != "fallback-ns" {
		t.Fatalf("expected namespace fallback-ns, got %q", createdNamespace)
	}
}

func TestDeploy_DryRunDoesNotSubmitRuntimeAdapter(t *testing.T) {
	ctx := context.Background()
	pipe := map[string]interface{}{
		"name": "dry-run-submit-pipe",
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
	adapter := &mockRuntimeAdapter{}
	depRepo := &mockDeploymentRepo{}
	uc := New(&mockTemplateRepo{}, depRepo, newMockAssetRepo(), nil, "dry-run-ns")
	uc.SetRuntimeAdapter(adapter)

	dep, err := uc.Deploy(ctx, pipe, "", nil, DeployOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Deploy dry-run: %v", err)
	}
	if dep == nil || dep.Status != "Preview" || dep.Manifest == nil {
		t.Fatalf("expected preview deployment with manifest, got %#v", dep)
	}
	if len(adapter.submitRuns) != 0 {
		t.Fatalf("expected dry-run not to submit runtime, got %d submits", len(adapter.submitRuns))
	}
	if len(depRepo.saved) != 0 {
		t.Fatalf("expected dry-run not to persist deployment, got %d saves", len(depRepo.saved))
	}
}

func TestDeploy_RuntimeAdapterSubmitFailureDoesNotPersistRun(t *testing.T) {
	ctx := context.Background()
	pipe := map[string]interface{}{
		"name": "adapter-submit-fail-pipe",
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
	adapter := &mockRuntimeAdapter{submitErr: errors.New("adapter submit denied")}
	depRepo := &mockDeploymentRepo{}
	runRepo := &mockRunRepo{}
	uc := New(&mockTemplateRepo{}, depRepo, newMockAssetRepo(), nil, "runtime-ns")
	uc.SetRuntimeAdapter(adapter)
	uc.SetRunRepositories(nil, runRepo, &mockRunNodeRepo{})

	_, err := uc.Deploy(ctx, pipe, "", nil)
	if err == nil {
		t.Fatal("expected adapter submit error")
	}
	if !strings.Contains(err.Error(), "create workflow") || !strings.Contains(err.Error(), "adapter submit denied") {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(adapter.submitRuns) != 1 {
		t.Fatalf("expected one submit attempt, got %d", len(adapter.submitRuns))
	}
	if len(depRepo.saved) != 0 {
		t.Fatalf("expected deployment not to be saved, got %d saves", len(depRepo.saved))
	}
	if len(runRepo.byID) != 0 {
		t.Fatalf("expected run not to be saved, got %d saves", len(runRepo.byID))
	}
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
	// CYB-3680: content-addressed CMs are shared and owner-less; the CM is
	// ensured BEFORE the Workflow so no owner UID can exist yet.
	if store.lastOwner != nil {
		t.Fatalf("expected owner-less runtime config (CYB-3680), got %#v", store.lastOwner)
	}
	if store.lastProjection.ContentHash == "" {
		t.Fatal("expected content-addressed projection hash")
	}
	manifest := *dep.Manifest
	if !strings.Contains(manifest, store.lastProjection.VolumeName) {
		t.Fatalf("expected runtime config volume in manifest, got %s", manifest)
	}
	if !strings.Contains(manifest, "PIPELINE_CONFIG_PATH") || !strings.Contains(manifest, "/workspace/configs/effective.yaml") {
		t.Fatalf("expected config env path in manifest, got %s", manifest)
	}
	if !strings.Contains(manifest, "subPath: effective.yaml") {
		t.Fatalf("expected config subPath mount in manifest, got %s", manifest)
	}
}

func TestDeploy_RuntimeAdapterSubmitPreservesRuntimeConfigOwnerLookup(t *testing.T) {
	ctx := context.Background()
	pipe := map[string]interface{}{
		"name": "adapter-config-pipe",
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
	createdThroughWorkflowClient := false
	wfClient := &mockWorkflowClient{
		createWorkflowFn: func(context.Context, *wfv1.Workflow, string) error {
			createdThroughWorkflowClient = true
			return nil
		},
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, newMockAssetRepo(), wfClient, "runtime-ns")
	uc.SetRuntimeAdapter(&mockRuntimeAdapter{})
	uc.pipelineConfigRepo = &mockPipelineConfigRepo{
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
	store := &mockRuntimeConfigStore{volumeName: "runtime-config-adapter"}
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
	if dep == nil {
		t.Fatal("expected deployment")
	}
	if createdThroughWorkflowClient {
		t.Fatal("expected submit to use runtime adapter, not workflow client")
	}
	// CYB-3680: owner-less + created before the Workflow (no lookup needed).
	if store.lastOwner != nil {
		t.Fatalf("expected owner-less runtime config (CYB-3680), got %#v", store.lastOwner)
	}
	if store.lastNamespace != "runtime-ns" || store.lastDeploymentID != dep.ID {
		t.Fatalf("unexpected runtime config store target namespace=%q deployment=%q", store.lastNamespace, store.lastDeploymentID)
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
	// CYB-3680: content-addressed CMs are shared and owner-less; the CM is
	// ensured BEFORE the Workflow so no owner UID can exist yet.
	if store.lastOwner != nil {
		t.Fatalf("expected owner-less runtime config (CYB-3680), got %#v", store.lastOwner)
	}
	if store.lastProjection.ContentHash == "" {
		t.Fatal("expected content-addressed projection hash")
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
	// CYB-3680: content-addressed CMs are shared and owner-less; the CM is
	// ensured BEFORE the Workflow so no owner UID can exist yet.
	if store.lastOwner != nil {
		t.Fatalf("expected owner-less runtime config (CYB-3680), got %#v", store.lastOwner)
	}
	if store.lastProjection.ContentHash == "" {
		t.Fatal("expected content-addressed projection hash")
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
		"subPath: 01-step-a-a.yaml",
		"subPath: 02-step-b-b.yaml",
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

func TestListRunInputs_IncludesMaterializedRuntimeConfigs(t *testing.T) {
	ctx := context.Background()
	pipe := map[string]interface{}{
		"name": "run-input-config-pipe",
		"nodes": []interface{}{
			map[string]interface{}{
				"id": "step-a",
				"component": map[string]interface{}{
					"name":  "detector",
					"image": "busybox",
				},
				"runtimeConfig": map[string]interface{}{
					"mode":           "saved",
					"configId":       "cfg-node",
					"version":        2,
					"fileName":       "node-source.yaml",
					"mountPath":      "/workspace/configs",
					"targetFilename": "node.yaml",
				},
			},
			map[string]interface{}{
				"id": "step-b",
				"component": map[string]interface{}{
					"name":  "consumer",
					"image": "busybox",
				},
			},
		},
		"edges": []interface{}{},
	}
	runRepo := &mockRunRepo{}
	uc := newUsecase(newMockAssetRepo())
	uc.SetRunRepositories(nil, runRepo, &mockRunNodeRepo{})
	uc.pipelineConfigRepo = &mockPipelineConfigRepo{
		configs: map[string]*models.PipelineConfig{
			"cfg-global": {ID: "cfg-global", Name: "global-source.yaml", Lifecycle: "ready"},
			"cfg-node":   {ID: "cfg-node", Name: "node-source.yaml", Lifecycle: "ready"},
		},
		versions: map[string]*models.PipelineConfigVersion{
			"cfg-global:1": {ConfigID: "cfg-global", Version: 1, Status: "ready", Content: "global: true\n"},
			"cfg-node:2":   {ConfigID: "cfg-node", Version: 2, Status: "ready", Content: "node: true\n"},
		},
	}
	uc.runtimeConfigStore = &mockRuntimeConfigStore{volumeName: "runtime-config-inputs"}

	dep, err := uc.Deploy(ctx, pipe, "", nil, DeployOptions{
		ConfigSelection: &RuntimeConfigSelection{
			Mode:           "saved",
			ConfigID:       "cfg-global",
			Version:        1,
			FileName:       "global-source.yaml",
			MountPath:      "/workspace/global",
			TargetFilename: "global.yaml",
		},
	})
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	rawMaterialized, ok := dep.PipelineJSON[runConfigInputsPipelineJSONKey].([]interface{})
	if !ok || len(rawMaterialized) != 2 {
		t.Fatalf("expected two materialized config inputs, got %#v", dep.PipelineJSON[runConfigInputsPipelineJSONKey])
	}

	inputs, err := uc.ListRunInputs(ctx, dep.ID)
	if err != nil {
		t.Fatalf("ListRunInputs: %v", err)
	}
	configInputs := make([]models.RunInput, 0, 2)
	for _, item := range inputs.Items {
		if item.Type == "config" {
			configInputs = append(configInputs, item)
		}
	}
	if len(configInputs) != 2 {
		t.Fatalf("expected two config inputs, got %#v", configInputs)
	}
	byRef := map[string]models.RunInput{}
	for _, item := range configInputs {
		byRef[item.RefID] = item
		if item.ContentHash == "" || !strings.HasPrefix(item.ContentHash, "sha256:") {
			t.Fatalf("expected content hash on %#v", item)
		}
		if _, ok := item.Snapshot["content"]; ok {
			t.Fatalf("expected raw content to be omitted from snapshot %#v", item.Snapshot)
		}
		if item.ProjectionKey == "" {
			t.Fatalf("expected projection key on %#v", item)
		}
	}
	global := byRef["cfg-global"]
	if global.NodeID != "" || global.RefVersion != "1" || global.FileName != "global-source.yaml" || global.TargetFilename != "global.yaml" || global.MountPath != "/workspace/global" {
		t.Fatalf("unexpected global config input %#v", global)
	}
	if global.ContentHash != runtimeConfigContentHash("global: true\n") {
		t.Fatalf("unexpected global content hash %q", global.ContentHash)
	}
	node := byRef["cfg-node"]
	if node.NodeID != "step-a" || node.RefVersion != "2" || node.FileName != "node-source.yaml" || node.TargetFilename != "node.yaml" {
		t.Fatalf("unexpected node config input %#v", node)
	}
	if node.ContentHash != runtimeConfigContentHash("node: true\n") {
		t.Fatalf("unexpected node content hash %q", node.ContentHash)
	}
}

func TestListRunInputs_LegacyRuntimeConfigFallbackOmitsContent(t *testing.T) {
	ctx := context.Background()
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-legacy-config": {
				ID:           "run-legacy-config",
				PipelineName: "legacy-config",
				Status:       "Running",
				PipelineJSON: map[string]interface{}{
					"nodes": []interface{}{
						map[string]interface{}{
							"id": "step-a",
							"runtimeConfig": map[string]interface{}{
								"mode":           "inline",
								"fileName":       "inline.yaml",
								"mountPath":      "/workspace/configs",
								"targetFilename": "inline.yaml",
								"content":        "secret: true\n",
							},
						},
					},
				},
			},
		},
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, newMockAssetRepo(), &mockWorkflowClient{}, "default")
	uc.SetRunRepositories(nil, runRepo, &mockRunNodeRepo{})

	inputs, err := uc.ListRunInputs(ctx, "run-legacy-config")
	if err != nil {
		t.Fatalf("ListRunInputs: %v", err)
	}
	var configInput *models.RunInput
	for i := range inputs.Items {
		if inputs.Items[i].Type == "config" {
			configInput = &inputs.Items[i]
			break
		}
	}
	if configInput == nil {
		t.Fatalf("expected config input, got %#v", inputs.Items)
	}
	if configInput.ContentHash != runtimeConfigContentHash("secret: true\n") {
		t.Fatalf("unexpected content hash %q", configInput.ContentHash)
	}
	if _, ok := configInput.Snapshot["content"]; ok {
		t.Fatalf("expected raw content to be omitted from snapshot %#v", configInput.Snapshot)
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
	if !strings.Contains(*dep.Manifest, "serviceAccountName: cyber-databrew-backend-argo") {
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
	summaryByID  map[string]*models.PipelineRun
	byWf         map[string]*models.PipelineRun
	findAllErr   error
	findAllCalls int
	listFilters  []models.PipelineRunListFilter
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
	// Deterministic order: the real repo is SQL-ordered; map iteration is
	// random and would make ordering-sensitive tests (watcher rotation) flaky.
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
func (m *mockRunRepo) FindActiveRunSummaries(_ context.Context, _ int) ([]models.PipelineRun, error) {
	items, err := m.FindAll(context.Background())
	if err != nil {
		return nil, err
	}
	active := make([]models.PipelineRun, 0, len(items))
	for _, it := range items {
		if isActiveDeploymentStatus(it.Status) {
			active = append(active, it)
		}
	}
	return active, nil
}

func (m *mockRunRepo) FindActiveRunSummariesAfter(_ context.Context, afterCreatedAt time.Time, afterID string, limit int) ([]models.PipelineRun, error) {
	items, err := m.FindAll(context.Background())
	if err != nil {
		return nil, err
	}
	// Reproduce the postgres implementation's newest-first (created_at, id) DESC
	// ordering + cursor semantics in memory so unit tests exercise the same
	// contract the paginated loader relies on.
	active := make([]models.PipelineRun, 0, len(items))
	for _, it := range items {
		if !isActiveDeploymentStatus(it.Status) {
			continue
		}
		if afterID != "" {
			if it.CreatedAt.After(afterCreatedAt) {
				continue
			}
			if it.CreatedAt.Equal(afterCreatedAt) && it.ID >= afterID {
				continue
			}
		}
		active = append(active, it)
	}
	sort.SliceStable(active, func(i, j int) bool {
		if !active[i].CreatedAt.Equal(active[j].CreatedAt) {
			return active[i].CreatedAt.After(active[j].CreatedAt)
		}
		return active[i].ID > active[j].ID
	})
	if limit > 0 && len(active) > limit {
		active = active[:limit]
	}
	return active, nil
}

func (m *mockRunRepo) ListSummaries(_ context.Context, filter models.PipelineRunListFilter) ([]models.PipelineRun, int, error) {
	m.listFilters = append(m.listFilters, filter)
	items, err := m.FindAll(context.Background())
	if err != nil {
		return nil, 0, err
	}
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
	return filtered, len(filtered), nil
}
func (m *mockRunRepo) FindByID(_ context.Context, id string) (*models.PipelineRun, error) {
	if m.byID == nil {
		return nil, nil
	}
	return m.byID[id], nil
}
func (m *mockRunRepo) FindSummaryByID(_ context.Context, id string) (*models.PipelineRun, error) {
	if m.summaryByID != nil {
		return m.summaryByID[id], nil
	}
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

func TestListRunChildrenReturnsRelationsAndSummary(t *testing.T) {
	t.Parallel()

	const parentID = "batch-1"
	childBatchID := parentID
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			parentID: {
				ID:           parentID,
				PipelineName: "Batch 1",
				Status:       "Running",
				CreatedAt:    time.Now().UTC(),
			},
			"child-running": {
				ID:         "child-running",
				Status:     "Running",
				BatchJobID: &childBatchID,
				AssetIDs:   []string{"asset-1"},
				CreatedAt:  time.Now().UTC(),
			},
			"child-failed": {
				ID:           "child-failed",
				Status:       "Failed",
				Message:      "Unschedulable: 0/12 nodes are available: 2 Insufficient memory.",
				WorkflowName: "wf-child-failed",
				BatchJobID:   &childBatchID,
				AssetIDs:     []string{"asset-2"},
				CreatedAt:    time.Now().UTC(),
			},
			"unrelated": {
				ID:        "unrelated",
				Status:    "Succeeded",
				CreatedAt: time.Now().UTC(),
			},
		},
	}

	uc := New(&mockTemplateRepo{}, nil, nil, nil, "cyber-databrew-dev")
	uc.SetRunRepositories(nil, runRepo, nil)

	got, err := uc.ListRunChildren(context.Background(), parentID)
	if err != nil {
		t.Fatalf("ListRunChildren() error = %v", err)
	}
	if got.RunID != parentID || got.Total != 2 || len(got.Items) != 2 {
		t.Fatalf("children = %+v, want 2 children for %s", got, parentID)
	}
	if len(got.Relations) != 2 {
		t.Fatalf("relations = %d, want 2: %+v", len(got.Relations), got.Relations)
	}
	if got.Summary.Total != 2 || got.Summary.AggregateStatus != "Running" {
		t.Fatalf("summary = %+v, want Running total=2", got.Summary)
	}
	if got.Summary.RunningCount != 1 || got.Summary.FailedCount != 1 {
		t.Fatalf("summary counts = %+v, want running=1 failed=1", got.Summary)
	}
	if len(got.Summary.TopFailureReasons) != 1 || got.Summary.TopFailureReasons[0].Reason != "unschedulable" {
		t.Fatalf("top failure reasons = %+v, want unschedulable", got.Summary.TopFailureReasons)
	}
	if got.Items[1].FailureReason == "" && got.Items[0].FailureReason == "" {
		t.Fatalf("expected child failure reason annotation in %+v", got.Items)
	}
}

func TestListRunChildrenFallsBackToBatchChildrenWithoutParentRun(t *testing.T) {
	t.Parallel()

	const batchID = "legacy-batch"
	childBatchID := batchID
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"child-running": {
				ID:         "child-running",
				Status:     "Running",
				BatchJobID: &childBatchID,
				AssetIDs:   []string{"asset-1"},
				CreatedAt:  time.Now().UTC(),
			},
			"child-failed": {
				ID:           "child-failed",
				Status:       "Failed",
				Message:      "InvalidImageName: could not parse image name",
				WorkflowName: "wf-child-failed",
				BatchJobID:   &childBatchID,
				AssetIDs:     []string{"asset-2"},
				CreatedAt:    time.Now().UTC(),
			},
		},
	}

	uc := New(&mockTemplateRepo{}, nil, nil, nil, "cyber-databrew-dev")
	uc.SetRunRepositories(nil, runRepo, nil)

	got, err := uc.ListRunChildren(context.Background(), batchID)
	if err != nil {
		t.Fatalf("ListRunChildren() error = %v", err)
	}
	if got.RunID != batchID || got.Total != 2 || len(got.Items) != 2 {
		t.Fatalf("children = %+v, want legacy batch children", got)
	}
	if len(got.Relations) != 2 || got.Relations[0].ParentRunID != batchID {
		t.Fatalf("relations = %+v, want batch parent relations", got.Relations)
	}
	if got.Summary.AggregateStatus != "Running" || !got.Summary.HasFailures {
		t.Fatalf("summary = %+v, want running with failures", got.Summary)
	}
	if len(got.Summary.TopFailureReasons) != 1 || got.Summary.TopFailureReasons[0].Reason != "image_startup" {
		t.Fatalf("top failure reasons = %+v, want image_startup", got.Summary.TopFailureReasons)
	}
	// CYB-3490: children listing is a pure read — no refresh requested.
	if len(runRepo.listFilters) != 1 || runRepo.listFilters[0].RefreshActive {
		t.Fatalf("ListRunChildren batch filter = %+v, want a single pure (no-refresh) listing", runRepo.listFilters)
	}
}

func TestListRunChildrenIncludesEventDerivedRelations(t *testing.T) {
	t.Parallel()

	const parentID = "run-parent"
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			parentID: {
				ID:           parentID,
				PipelineName: "source",
				Status:       "Succeeded",
				CreatedAt:    time.Now().UTC(),
			},
			"run-rerun": {
				ID:           "run-rerun",
				PipelineName: "source-rerun",
				Status:       "Running",
				CreatedAt:    time.Now().UTC(),
			},
			"run-resubmit": {
				ID:           "run-resubmit",
				PipelineName: "source-resubmit",
				Status:       "Succeeded",
				CreatedAt:    time.Now().UTC(),
			},
		},
	}
	eventRepo := &mockRunEventRepo{
		events: []models.PipelineRunEvent{
			{
				RunID:       parentID,
				EventType:   runEventRerunCreated,
				SubjectType: "run",
				SubjectID:   "run-rerun",
				Payload: map[string]interface{}{
					"sourceRunId": parentID,
					"childRunId":  "run-rerun",
					"relation":    "rerun_of",
				},
			},
			{
				RunID:       parentID,
				EventType:   runEventResubmitted,
				SubjectType: "run",
				SubjectID:   "run-resubmit",
				Payload: map[string]interface{}{
					"sourceRunId": parentID,
					"childRunId":  "run-resubmit",
					"relation":    "resubmit_of",
				},
			},
		},
	}

	uc := New(&mockTemplateRepo{}, nil, nil, nil, "cyber-databrew-dev")
	uc.SetRunRepositories(nil, runRepo, nil)
	uc.SetRunEventRepo(eventRepo)

	got, err := uc.ListRunChildren(context.Background(), parentID)
	if err != nil {
		t.Fatalf("ListRunChildren() error = %v", err)
	}
	if got.Total != 2 || len(got.Items) != 2 || len(got.Relations) != 2 {
		t.Fatalf("unexpected event-derived children: %+v", got)
	}
	relations := map[string]string{}
	for _, relation := range got.Relations {
		relations[relation.ChildRunID] = relation.RelationType
		if relation.Source != "pipeline_run_events" {
			t.Fatalf("unexpected relation source: %+v", relation)
		}
	}
	if relations["run-rerun"] != "rerun_of" || relations["run-resubmit"] != "resubmit_of" {
		t.Fatalf("unexpected event-derived relations: %+v", got.Relations)
	}
	if got.Summary.Total != 2 || got.Summary.AggregateStatus != "Running" {
		t.Fatalf("unexpected summary: %+v", got.Summary)
	}
}

func TestListRunChildrenPrefersDurableRelations(t *testing.T) {
	t.Parallel()

	const parentID = "run-parent"
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			parentID: {
				ID:           parentID,
				PipelineName: "source",
				Status:       "Succeeded",
				CreatedAt:    time.Now().UTC(),
			},
			"run-child": {
				ID:           "run-child",
				PipelineName: "source-rerun",
				Status:       "Failed",
				CreatedAt:    time.Now().UTC(),
			},
		},
	}
	relationRepo := &mockRunRelationRepo{relations: []models.RunRelation{
		{
			ID:           "rel-1",
			ParentRunID:  parentID,
			ChildRunID:   "run-child",
			RelationType: "rerun_of",
			Source:       "run_kernel",
		},
	}}

	uc := New(&mockTemplateRepo{}, nil, nil, nil, "cyber-databrew-dev")
	uc.SetRunRepositories(nil, runRepo, nil)
	uc.SetRunFactRepositories(relationRepo, nil)

	got, err := uc.ListRunChildren(context.Background(), parentID)
	if err != nil {
		t.Fatalf("ListRunChildren() error = %v", err)
	}
	if got.Total != 1 || len(got.Items) != 1 || got.Items[0].ID != "run-child" {
		t.Fatalf("durable children = %+v, want run-child", got)
	}
	if len(got.Relations) != 1 || got.Relations[0].RelationType != "rerun_of" || got.Relations[0].Source != "run_kernel" {
		t.Fatalf("durable relations = %+v, want rerun_of from run_kernel", got.Relations)
	}
	if got.Summary.HealthStatus != "failed" {
		t.Fatalf("summary health = %q, want failed", got.Summary.HealthStatus)
	}
}

func TestListRunChildrenPaginatesDurableRelations(t *testing.T) {
	t.Parallel()

	const parentID = "run-parent"
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			parentID: {
				ID:        parentID,
				Status:    "Running",
				CreatedAt: time.Now().UTC(),
			},
		},
	}
	relationRepo := &mockRunRelationRepo{}
	for i := 1; i <= 25; i++ {
		childID := fmt.Sprintf("run-child-%02d", i)
		runRepo.byID[childID] = &models.PipelineRun{
			ID:        childID,
			Status:    "Pending",
			CreatedAt: time.Now().UTC(),
		}
		relationRepo.relations = append(relationRepo.relations, models.RunRelation{
			ID:           fmt.Sprintf("rel-%02d", i),
			ParentRunID:  parentID,
			ChildRunID:   childID,
			RelationType: "batch_child",
			Source:       "run_relations",
		})
	}

	uc := New(&mockTemplateRepo{}, nil, nil, nil, "cyber-databrew-dev")
	uc.SetRunRepositories(nil, runRepo, nil)
	uc.SetRunFactRepositories(relationRepo, nil)

	got, err := uc.ListRunChildren(context.Background(), parentID, models.PipelineRunListFilter{
		Page:     2,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("ListRunChildren() error = %v", err)
	}
	if got.Total != 25 || got.Page != 2 || got.PageSize != 10 {
		t.Fatalf("pagination metadata = %+v, want total=25 page=2 pageSize=10", got)
	}
	if len(got.Items) != 10 || got.Items[0].ID != "run-child-11" || got.Items[9].ID != "run-child-20" {
		t.Fatalf("durable page children = %+v, want children 11-20", got.Items)
	}
	if len(got.Relations) != 10 || got.Relations[0].ChildRunID != "run-child-11" {
		t.Fatalf("durable page relations = %+v, want child 11 first", got.Relations)
	}
}

func TestListRunInputsPrefersDurableInputs(t *testing.T) {
	t.Parallel()

	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:                "run-1",
				PipelineName:      "pipe",
				Status:            "Running",
				ExecutionTargetID: "target-1",
				AssetIDs:          []string{"legacy-asset"},
				PipelineJSON: map[string]interface{}{
					"nodes": []interface{}{},
				},
				CreatedAt: time.Now().UTC(),
			},
		},
	}
	inputRepo := &mockRunInputRepo{inputs: []models.RunInput{
		{
			ID:     "input-1",
			RunID:  "run-1",
			Type:   "asset",
			RefID:  "durable-asset",
			Source: "run_inputs",
		},
	}}

	uc := New(&mockTemplateRepo{}, nil, nil, nil, "cyber-databrew-dev")
	uc.SetRunRepositories(nil, runRepo, nil)
	uc.SetRunFactRepositories(nil, inputRepo)

	got, err := uc.ListRunInputs(context.Background(), "run-1")
	if err != nil {
		t.Fatalf("ListRunInputs() error = %v", err)
	}
	if got.Total != 1 || len(got.Items) != 1 || got.Items[0].RefID != "durable-asset" {
		t.Fatalf("inputs = %+v, want durable input only", got)
	}
}

func TestPersistRunInputs_CleansInvalidUUIDIDs(t *testing.T) {
	t.Parallel()

	run := &models.PipelineRun{
		ID:                "run-invalid-input-ids",
		Status:            "Succeeded",
		AssetIDs:          []string{"asset-1", "asset-2"},
		ExecutionTargetID: "target-1",
		TargetSnapshot:    map[string]interface{}{"region": "us-east1"},
		PipelineJSON:      map[string]interface{}{"nodes": []interface{}{}},
	}
	inputRepo := &mockRunInputRepo{}

	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, newMockAssetRepo(), &mockWorkflowClient{}, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, &mockRunRepo{byID: map[string]*models.PipelineRun{run.ID: run}}, &mockRunNodeRepo{})
	uc.SetRunFactRepositories(&mockRunRelationRepo{}, inputRepo)

	if err := uc.persistRunInputs(context.Background(), run); err != nil {
		t.Fatalf("persistRunInputs: %v", err)
	}
	if len(inputRepo.inputs) != 3 {
		t.Fatalf("expected 3 run inputs, got %d", len(inputRepo.inputs))
	}
	for _, input := range inputRepo.inputs {
		if strings.TrimSpace(input.ID) != "" {
			t.Fatalf("expected sanitized UUID IDs, got %q", input.ID)
		}
	}
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
func (m *mockTargetRepo) Delete(_ context.Context, id string) error {
	delete(m.byID, id)
	return nil
}

type mockRunNodeRepo struct {
	byRun        map[string][]models.PipelineRunNode
	replaceCalls int
}

func (m *mockRunNodeRepo) ReplaceByRunID(_ context.Context, runID string, nodes []models.PipelineRunNode) error {
	m.replaceCalls++
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

func assertRunEvents(t *testing.T, events []models.PipelineRunEvent, runID string, expected ...string) {
	t.Helper()
	seen := map[string]bool{}
	for _, event := range events {
		if event.RunID == runID {
			seen[event.EventType] = true
		}
	}
	for _, eventType := range expected {
		if !seen[eventType] {
			t.Fatalf("expected run %s event %s in %#v", runID, eventType, events)
		}
	}
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

type watcherBackfillRepo struct {
	repository.BackfillRepository
	itemsByRunID map[string]*models.BackfillItem
	updated      []models.BackfillItem
}

func (m *watcherBackfillRepo) FindItemByPipelineRunID(_ context.Context, pipelineRunID string) (*models.BackfillItem, error) {
	if m.itemsByRunID == nil {
		return nil, nil
	}
	return m.itemsByRunID[pipelineRunID], nil
}

func (m *watcherBackfillRepo) UpdateItemStatus(_ context.Context, id, status, workflowName, errorMsg string) error {
	for _, item := range m.itemsByRunID {
		if item == nil || item.ID != id {
			continue
		}
		item.Status = status
		if workflowName != "" {
			item.WorkflowName = &workflowName
		}
		if errorMsg != "" {
			item.ErrorMessage = &errorMsg
		} else {
			item.ErrorMessage = nil
		}
		m.updated = append(m.updated, *item)
		return nil
	}
	return nil
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

func TestStopRun_AppendsSucceededAndFailedEvents(t *testing.T) {
	ctx := context.Background()
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-ok": {
				ID:            "run-ok",
				WorkflowName:  "wf-ok",
				Status:        "Running",
				ArgoNamespace: "video-proc-dev",
			},
			"run-fail": {
				ID:            "run-fail",
				WorkflowName:  "wf-fail",
				Status:        "Running",
				ArgoNamespace: "video-proc-dev",
			},
		},
	}
	eventRepo := &mockRunEventRepo{}
	wfClient := &mockWorkflowClient{}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetRunEventRepo(eventRepo)

	if err := uc.StopRun(ctx, "run-ok"); err != nil {
		t.Fatalf("StopRun success path returned error: %v", err)
	}
	if len(wfClient.stopCalls) != 1 || wfClient.stopCalls[0] != "video-proc-dev/wf-ok" {
		t.Fatalf("unexpected stop calls after success: %#v", wfClient.stopCalls)
	}
	assertRunEvents(t, eventRepo.events, "run-ok", runEventStopRequested, runEventStopSucceeded)

	wfClient.stopErr = errors.New("argo stop failed")
	err := uc.StopRun(ctx, "run-fail")
	if err == nil {
		t.Fatal("expected StopRun failure")
	}
	if len(wfClient.stopCalls) != 2 || wfClient.stopCalls[1] != "video-proc-dev/wf-fail" {
		t.Fatalf("unexpected stop calls after failure: %#v", wfClient.stopCalls)
	}
	assertRunEvents(t, eventRepo.events, "run-fail", runEventStopRequested, runEventStopFailed)
	var failedEvent *models.PipelineRunEvent
	for i := range eventRepo.events {
		event := &eventRepo.events[i]
		if event.RunID == "run-fail" && event.EventType == runEventStopFailed {
			failedEvent = event
			break
		}
	}
	if failedEvent == nil || failedEvent.Reason != "argo stop failed" {
		t.Fatalf("expected stop failed event reason, got %#v", failedEvent)
	}
}

func TestStopRun_UsesRuntimeAdapterWhenConfigured(t *testing.T) {
	ctx := context.Background()
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-ok": {
				ID:              "run-ok",
				WorkflowName:    "wf-ok",
				Status:          "Running",
				ArgoNamespace:   "video-proc-dev",
				ArgoWorkflowUID: "uid-ok",
			},
			"run-fail": {
				ID:            "run-fail",
				WorkflowName:  "wf-fail",
				Status:        "Running",
				ArgoNamespace: "video-proc-dev",
			},
		},
	}
	eventRepo := &mockRunEventRepo{}
	wfClient := &mockWorkflowClient{
		getWorkflowFn: func(_ context.Context, name, namespace string) (*wfv1.Workflow, error) {
			return &wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, UID: "uid-ok"},
			}, nil
		},
	}
	adapter := &mockRuntimeAdapter{}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRuntimeAdapter(adapter)
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetRunEventRepo(eventRepo)

	if err := uc.StopRun(ctx, "run-ok"); err != nil {
		t.Fatalf("StopRun success path returned error: %v", err)
	}
	if len(adapter.stopRefs) != 1 {
		t.Fatalf("expected adapter stop call, got %#v", adapter.stopRefs)
	}
	ref := adapter.stopRefs[0]
	if ref.Name != "wf-ok" || ref.Namespace != "video-proc-dev" || ref.UID != "uid-ok" {
		t.Fatalf("unexpected adapter stop ref: %#v", ref)
	}
	if len(wfClient.stopCalls) != 0 {
		t.Fatalf("expected legacy workflow client to be bypassed, got %#v", wfClient.stopCalls)
	}
	assertRunEvents(t, eventRepo.events, "run-ok", runEventStopRequested, runEventStopSucceeded)

	adapter.stopErr = errors.New("runtime stop failed")
	err := uc.StopRun(ctx, "run-fail")
	if err == nil {
		t.Fatal("expected StopRun failure")
	}
	assertRunEvents(t, eventRepo.events, "run-fail", runEventStopRequested, runEventStopFailed)
	var failedEvent *models.PipelineRunEvent
	for i := range eventRepo.events {
		event := &eventRepo.events[i]
		if event.RunID == "run-fail" && event.EventType == runEventStopFailed {
			failedEvent = event
			break
		}
	}
	if failedEvent == nil || failedEvent.Reason != "runtime stop failed" {
		t.Fatalf("expected adapter stop failure reason, got %#v", failedEvent)
	}
}

func TestRuntimeRetryRun_UsesRuntimeAdapterWithoutWorkflowClient(t *testing.T) {
	ctx := context.Background()
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-retry": {
				ID:            "run-retry",
				WorkflowName:  "wf-retry",
				Status:        "Failed",
				ArgoNamespace: "video-proc-dev",
			},
		},
	}
	eventRepo := &mockRunEventRepo{}
	adapter := &mockRuntimeAdapter{}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	uc.SetRuntimeAdapter(adapter)
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetRunEventRepo(eventRepo)

	run, err := uc.RuntimeRetryRun(ctx, "run-retry")
	if err != nil {
		t.Fatalf("RuntimeRetryRun returned error: %v", err)
	}
	if run == nil || run.ID != "run-retry" {
		t.Fatalf("expected same run identity, got %#v", run)
	}
	if len(adapter.retryRefs) != 1 || adapter.retryRefs[0].Name != "wf-retry" || adapter.retryRefs[0].Namespace != "video-proc-dev" {
		t.Fatalf("unexpected adapter retry refs: %#v", adapter.retryRefs)
	}
	assertRunEvents(t, eventRepo.events, "run-retry", runEventRuntimeRetryRequested, runEventRuntimeRetrySucceeded)
}

func TestRuntimeRetryRun_RejectsNonRetryableStatus(t *testing.T) {
	ctx := context.Background()
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-running": {
				ID:            "run-running",
				WorkflowName:  "wf-running",
				Status:        "Running",
				ArgoNamespace: "video-proc-dev",
			},
		},
	}
	eventRepo := &mockRunEventRepo{}
	adapter := &mockRuntimeAdapter{}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	uc.SetRuntimeAdapter(adapter)
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetRunEventRepo(eventRepo)

	_, err := uc.RuntimeRetryRun(ctx, "run-running")
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("expected ErrInvalidArgument, got: %v", err)
	}
	if !strings.Contains(err.Error(), "runtime retry only supports failed or errored runs") {
		t.Fatalf("unexpected retry error: %v", err)
	}
	if len(adapter.retryRefs) != 0 {
		t.Fatalf("runtime adapter should not be called when status is not retryable, refs: %#v", adapter.retryRefs)
	}
	assertRunEvents(t, eventRepo.events, "run-running")
}

func TestRuntimeRetryRun_MapsRetryRuntimeErrorToInvalidArgument(t *testing.T) {
	ctx := context.Background()
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-failed": {
				ID:            "run-failed",
				WorkflowName:  "wf-failed",
				Status:        "Failed",
				ArgoNamespace: "video-proc-dev",
			},
		},
	}
	eventRepo := &mockRunEventRepo{}
	adapter := &mockRuntimeAdapter{
		retryErr: errors.New("To retry a succeeded workflow"),
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	uc.SetRuntimeAdapter(adapter)
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetRunEventRepo(eventRepo)

	_, err := uc.RuntimeRetryRun(ctx, "run-failed")
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("expected ErrInvalidArgument, got: %v", err)
	}
	if len(eventRepo.events) == 0 || eventRepo.events[len(eventRepo.events)-1].EventType != runEventRuntimeRetryFailed {
		t.Fatalf("expected run_runtime_retry_failed event, got %#v", eventRepo.events)
	}
	last := eventRepo.events[len(eventRepo.events)-1]
	if !strings.Contains(last.Reason, "to retry a succeeded workflow") {
		t.Fatalf("expected mapped retry reason, got %q", last.Reason)
	}
}

func TestRuntimeControlRejectsMissingWorkflowBeforeAdapterCall(t *testing.T) {
	ctx := context.Background()
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-no-runtime": {
				ID:     "run-no-runtime",
				Status: "Running",
			},
		},
	}
	eventRepo := &mockRunEventRepo{}
	adapter := &mockRuntimeAdapter{}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	uc.SetRuntimeAdapter(adapter)
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetRunEventRepo(eventRepo)

	err := uc.StopRun(ctx, "run-no-runtime")
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
	}
	if len(adapter.stopRefs) != 0 {
		t.Fatalf("adapter should not be called without workflowName: %#v", adapter.stopRefs)
	}
	if len(eventRepo.events) != 0 {
		t.Fatalf("no events should be written before runtime ref validation, got %#v", eventRepo.events)
	}
}

func TestRerunRun_CreatesNewRunAndEvents(t *testing.T) {
	ctx := context.Background()
	validPipe := map[string]interface{}{
		"name": "rerun-pipe",
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
		runConfigInputsPipelineJSONKey: []interface{}{
			map[string]interface{}{
				"scope":          "global",
				"mode":           "saved",
				"configId":       "cfg-global",
				"version":        1,
				"fileName":       "global.yaml",
				"mountPath":      "/workspace/configs",
				"targetFilename": "global.yaml",
				"contentHash":    runtimeConfigContentHash("global: true\n"),
				"projectionKey":  "runtime-config-global.yaml",
			},
		},
	}
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-source": {
				ID:                "run-source",
				PipelineName:      "rerun-pipe",
				PipelineJSON:      validPipe,
				Status:            "Succeeded",
				ExecutionTargetID: "default",
			},
		},
	}
	eventRepo := &mockRunEventRepo{}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, newMockAssetRepo(), &mockWorkflowClient{}, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetRunEventRepo(eventRepo)
	uc.pipelineConfigRepo = &mockPipelineConfigRepo{
		configs: map[string]*models.PipelineConfig{
			"cfg-global": {ID: "cfg-global", Name: "global.yaml", Lifecycle: "ready"},
		},
		versions: map[string]*models.PipelineConfigVersion{
			"cfg-global:1": {ConfigID: "cfg-global", Version: 1, Status: "ready", Content: "global: true\n"},
		},
	}
	uc.runtimeConfigStore = &mockRuntimeConfigStore{volumeName: "runtime-config-rerun"}

	next, err := uc.RerunRun(ctx, "run-source")
	if err != nil {
		t.Fatalf("RerunRun: %v", err)
	}
	if next == nil || next.ID == "run-source" {
		t.Fatalf("expected new run, got %#v", next)
	}
	if !strings.Contains(next.PipelineName, "rerun") {
		t.Fatalf("expected rerun suffix in pipeline name, got %q", next.PipelineName)
	}
	inputs, err := uc.ListRunInputs(ctx, next.ID)
	if err != nil {
		t.Fatalf("ListRunInputs: %v", err)
	}
	foundConfig := false
	for _, input := range inputs.Items {
		if input.Type == "config" && input.RefID == "cfg-global" {
			foundConfig = true
			if input.ContentHash != runtimeConfigContentHash("global: true\n") {
				t.Fatalf("unexpected rerun config hash %q", input.ContentHash)
			}
		}
	}
	if !foundConfig {
		t.Fatalf("expected rerun to preserve saved deploy-level config input, got %#v", inputs.Items)
	}
	assertRunEvents(t, eventRepo.events, "run-source", runEventRerunRequested)
	assertRunEvents(t, eventRepo.events, "run-source", runEventRerunCreated)
	assertRunEvents(t, eventRepo.events, next.ID, runEventRerunCreated)
	for _, event := range eventRepo.events {
		if event.RunID == next.ID && event.EventType == runEventRerunCreated {
			if event.Payload["sourceRunId"] != "run-source" || event.Payload["childRunId"] != next.ID || event.Payload["relation"] != "rerun_of" {
				t.Fatalf("unexpected rerun payload %#v", event.Payload)
			}
		}
		if event.RunID == "run-source" && event.EventType == runEventRerunCreated {
			if event.Payload["sourceRunId"] != "run-source" || event.Payload["childRunId"] != next.ID || event.Payload["relation"] != "rerun_of" {
				t.Fatalf("unexpected source rerun payload %#v", event.Payload)
			}
		}
	}
	children, err := uc.ListRunChildren(ctx, "run-source")
	if err != nil {
		t.Fatalf("ListRunChildren: %v", err)
	}
	if children.Total != 1 || len(children.Items) != 1 || children.Items[0].ID != next.ID {
		t.Fatalf("unexpected rerun children: %+v", children)
	}
	if len(children.Relations) != 1 || children.Relations[0].RelationType != "rerun_of" || children.Relations[0].ChildRunID != next.ID {
		t.Fatalf("unexpected rerun relations: %+v", children.Relations)
	}
}

func TestGetRunRuntime_OmitsDebugURLWithoutWorkflowName(t *testing.T) {
	ctx := context.Background()
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-empty-workflow": {
				ID:     "run-empty-workflow",
				Status: "Pending",
			},
			"run-with-workflow": {
				ID:            "run-with-workflow",
				WorkflowName:  "wf-with-name",
				Status:        "Running",
				ArgoNamespace: "video-proc-dev",
			},
		},
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})

	emptyRuntime, err := uc.GetRunRuntime(ctx, "run-empty-workflow")
	if err != nil {
		t.Fatalf("GetRunRuntime empty workflow returned error: %v", err)
	}
	if emptyRuntime.Runtime.WorkflowName != "" {
		t.Fatalf("expected empty workflow name, got %q", emptyRuntime.Runtime.WorkflowName)
	}
	if emptyRuntime.Runtime.DebugURL != "" {
		t.Fatalf("expected empty debug URL for missing workflowName, got %q", emptyRuntime.Runtime.DebugURL)
	}

	namedRuntime, err := uc.GetRunRuntime(ctx, "run-with-workflow")
	if err != nil {
		t.Fatalf("GetRunRuntime named workflow returned error: %v", err)
	}
	if namedRuntime.Runtime.DebugURL != "/api/v1/workflows/wf-with-name" {
		t.Fatalf("unexpected debug URL: %q", namedRuntime.Runtime.DebugURL)
	}
}

func TestGetRunRuntime_BatchParentRunHasNoRuntime(t *testing.T) {
	ctx := context.Background()
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"batch-parent": {
				ID:           "batch-parent",
				Status:       "Succeeded",
				WorkflowName: "batch-parent-batch-parent",
			},
		},
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})

	runtime, err := uc.GetRunRuntime(ctx, "batch-parent")
	if err != nil {
		t.Fatalf("GetRunRuntime batch parent returned error: %v", err)
	}
	if runtime.Runtime.RuntimeType != "none" {
		t.Fatalf("runtimeType = %q, want none", runtime.Runtime.RuntimeType)
	}
	if runtime.Runtime.WorkflowName != "" {
		t.Fatalf("expected batch parent workflowName empty, got %q", runtime.Runtime.WorkflowName)
	}
	if runtime.Runtime.DebugURL != "" {
		t.Fatalf("expected no debug URL for batch parent, got %q", runtime.Runtime.DebugURL)
	}
}

func TestRetryAndResubmitRun_AppendFailedEventsWhenCreateRunFails(t *testing.T) {
	ctx := context.Background()
	validPipe := map[string]interface{}{
		"name": "event-pipe",
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
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-retry": {
				ID:           "run-retry",
				PipelineName: "pipe",
				PipelineJSON: validPipe,
				AssetIDs:     []string{"missing-asset"},
				Status:       "Failed",
			},
			"run-resubmit": {
				ID:           "run-resubmit",
				PipelineName: "pipe",
				PipelineJSON: validPipe,
				AssetIDs:     []string{"missing-asset"},
				Status:       "Failed",
			},
			"run-rerun": {
				ID:           "run-rerun",
				PipelineName: "pipe",
				PipelineJSON: validPipe,
				AssetIDs:     []string{"missing-asset"},
				Status:       "Failed",
			},
		},
	}
	eventRepo := &mockRunEventRepo{}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, newMockAssetRepo(), &mockWorkflowClient{}, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetRunEventRepo(eventRepo)

	if _, err := uc.RetryRun(ctx, "run-retry"); err == nil {
		t.Fatal("expected RetryRun to fail when source assets are missing")
	}
	assertRunEvents(t, eventRepo.events, "run-retry", runEventRetryRequested, runEventRetryFailed)

	if _, err := uc.ResubmitRun(ctx, "run-resubmit"); err == nil {
		t.Fatal("expected ResubmitRun to fail when source assets are missing")
	}
	assertRunEvents(t, eventRepo.events, "run-resubmit", runEventResubmitRequested, runEventResubmitFailed)
	if _, err := uc.RerunRun(ctx, "run-rerun"); err == nil {
		t.Fatal("expected RerunRun to fail when source assets are missing")
	}
	assertRunEvents(t, eventRepo.events, "run-rerun", runEventRerunRequested, runEventRerunFailed)
	for _, event := range eventRepo.events {
		if (event.EventType == runEventRetryFailed || event.EventType == runEventResubmitFailed || event.EventType == runEventRerunFailed) &&
			!strings.Contains(event.Reason, "asset not found") {
			t.Fatalf("expected failed event reason to include asset validation failure, got %#v", event)
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
	if status.LedgerHealth.TotalRuns != 1 || status.LedgerHealth.RunsWithEvents != 0 {
		t.Fatalf("unexpected cached ledger health: %#v", status.LedgerHealth)
	}
}

// CYB-3490 P1b: continuous observation is workflow-level only — node rows are
// written once at terminal (or live on single-run drill-in), never per cycle
// while the workflow runs.
func TestProjectRunNodes_TerminalArchiveOncePolicy(t *testing.T) {
	ctx := context.Background()
	newUC := func(runStatus string) (*Usecase, *mockRunRepo, *mockRunNodeRepo) {
		runRepo := &mockRunRepo{byID: map[string]*models.PipelineRun{
			"run-1": {ID: "run-1", WorkflowName: "wf-1", Status: runStatus},
		}}
		nodeRepo := &mockRunNodeRepo{}
		uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, &mockWorkflowClient{}, "default")
		uc.SetRunRepositories(&mockTargetRepo{}, runRepo, nodeRepo)
		uc.SetRunEventRepo(&mockRunEventRepo{})
		return uc, runRepo, nodeRepo
	}
	wfWith := func(phase wfv1.WorkflowPhase) *wfv1.Workflow {
		return &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "wf-1", UID: "uid-1"},
			Status: wfv1.WorkflowStatus{
				Phase:    phase,
				Progress: wfv1.Progress("37/100"),
				Nodes: map[string]wfv1.NodeStatus{
					"n1": {ID: "n1", Name: "wf-1.step", Phase: wfv1.NodeRunning},
				},
			},
		}
	}

	// 1. Running + terminal-archive mode: zero node writes (the old code
	// delete+inserted every node on every cycle — the 100k-node amplifier).
	uc, runRepo, nodeRepo := newUC("Running")
	uc.applyWorkflowToRun(ctx, runRepo.byID["run-1"], wfWith(wfv1.WorkflowRunning), nodeProjectTerminalArchive)
	if nodeRepo.replaceCalls != 0 {
		t.Fatalf("running+terminalArchive: expected 0 node writes, got %d", nodeRepo.replaceCalls)
	}
	// ...but the workflow-level progress IS projected.
	if runRepo.byID["run-1"].Progress != "37/100" {
		t.Fatalf("expected progress 37/100 projected, got %q", runRepo.byID["run-1"].Progress)
	}

	// 2. Running + live (single-run drill-in): nodes projected.
	uc, runRepo, nodeRepo = newUC("Running")
	uc.applyWorkflowToRun(ctx, runRepo.byID["run-1"], wfWith(wfv1.WorkflowRunning), nodeProjectLive)
	if nodeRepo.replaceCalls != 1 {
		t.Fatalf("running+live: expected 1 node write, got %d", nodeRepo.replaceCalls)
	}

	// 3. Active -> terminal transition: archived exactly once...
	uc, runRepo, nodeRepo = newUC("Running")
	uc.applyWorkflowToRun(ctx, runRepo.byID["run-1"], wfWith(wfv1.WorkflowSucceeded), nodeProjectTerminalArchive)
	if nodeRepo.replaceCalls != 1 {
		t.Fatalf("transition: expected 1 archive write, got %d", nodeRepo.replaceCalls)
	}
	// ...and a re-observation of the already-terminal run does not rewrite.
	uc.applyWorkflowToRun(ctx, runRepo.byID["run-1"], wfWith(wfv1.WorkflowSucceeded), nodeProjectTerminalArchive)
	if nodeRepo.replaceCalls != 1 {
		t.Fatalf("terminal re-observation with archive: expected no rewrite, got %d", nodeRepo.replaceCalls)
	}

	// 4. Already-terminal run whose archive is missing (lost webhook or a
	// restart between persist and archive): the observation backfills it.
	uc, runRepo, nodeRepo = newUC("Succeeded")
	uc.applyWorkflowToRun(ctx, runRepo.byID["run-1"], wfWith(wfv1.WorkflowSucceeded), nodeProjectTerminalArchive)
	if nodeRepo.replaceCalls != 1 {
		t.Fatalf("terminal without archive: expected 1 backfill write, got %d", nodeRepo.replaceCalls)
	}
}

// CYB-3490 regression: persistRunObservation copies observed fields onto the
// stored row — Progress must survive that copy. The watcher observes COPIES
// (FindAllSummaries), so this test must not alias the stored pointer, or the
// copy bug is invisible (dev incident: progress never persisted).
func TestPersistRunObservation_CarriesProgressOntoStoredRow(t *testing.T) {
	ctx := context.Background()
	stored := &models.PipelineRun{ID: "run-1", WorkflowName: "wf-1", Status: "Running"}
	runRepo := &mockRunRepo{byID: map[string]*models.PipelineRun{"run-1": stored}}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, &mockWorkflowClient{}, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetRunEventRepo(&mockRunEventRepo{})

	observed := *stored // watcher-style detached copy
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "wf-1", UID: "uid-1"},
		Status: wfv1.WorkflowStatus{
			Phase:    wfv1.WorkflowRunning,
			Progress: wfv1.Progress("42/100"),
		},
	}
	uc.applyWorkflowToRun(ctx, &observed, wf, nodeProjectTerminalArchive)
	got := runRepo.byID["run-1"]
	if got == nil || got.Progress != "42/100" {
		t.Fatalf("expected stored row to carry progress 42/100, got %+v", got)
	}
	// An observation with empty progress must not wipe the stored value.
	observed2 := *got
	wf.Status.Progress = ""
	uc.applyWorkflowToRun(ctx, &observed2, wf, nodeProjectTerminalArchive)
	if runRepo.byID["run-1"].Progress != "42/100" {
		t.Fatalf("empty observation wiped progress: %q", runRepo.byID["run-1"].Progress)
	}
}

// CYB-3490 P1b: batch lists surface near-realtime progress from the
// workflow-level column when no node rows exist for a running item.
func TestAttachBatchNodeProgress_FallsBackToWorkflowProgress(t *testing.T) {
	ctx := context.Background()
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, &mockWorkflowClient{}, "default")
	uc.SetObservabilityRepositories(&mockAssetNodeRepo{}, nil, nil)
	items := []models.PipelineRun{
		{ID: "run-1", Status: "Running", Progress: "37/100"},
		{ID: "run-2", Status: "Succeeded", Progress: "100/100"},
	}
	uc.attachBatchNodeProgress(ctx, items)
	if items[0].NodeProgress == nil || items[0].NodeProgress.Label != "37/100" {
		t.Fatalf("running item: expected progress label fallback, got %+v", items[0].NodeProgress)
	}
	// Terminal items keep the ledger-derived summary (已完成), not the raw N/M.
	if items[1].NodeProgress == nil || items[1].NodeProgress.Label != "已完成" {
		t.Fatalf("terminal item: expected ledger label, got %+v", items[1].NodeProgress)
	}
}

// CYB-3490: with more active runs than the per-cycle cap, the refresh window
// must rotate so no active run starves. 3 actives, cap 2 — after two cycles
// every workflow has been refreshed at least once (the old always-from-0 scan
// would never reach the third).
func TestSyncActiveRunEvents_RotatesActiveWindowNoStarvation(t *testing.T) {
	t.Setenv(watcherModeEnv, watcherModeLegacy) // rotation is the legacy path (CYB-3681)
	ctx := context.Background()
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {ID: "run-1", WorkflowName: "wf-1", Status: "Running"},
			"run-2": {ID: "run-2", WorkflowName: "wf-2", Status: "Running"},
			"run-3": {ID: "run-3", WorkflowName: "wf-3", Status: "Running"},
		},
	}
	refreshedByName := map[string]int{}
	wfClient := &mockWorkflowClient{
		getWorkflowFn: func(_ context.Context, name, namespace string) (*wfv1.Workflow, error) {
			refreshedByName[name]++
			return &wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
				Status:     wfv1.WorkflowStatus{Phase: wfv1.WorkflowRunning},
			}, nil
		},
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetRunEventRepo(&mockRunEventRepo{})

	for cycle := 0; cycle < 2; cycle++ {
		if _, err := uc.SyncActiveRunEvents(ctx, 2); err != nil {
			t.Fatalf("SyncActiveRunEvents cycle %d: %v", cycle, err)
		}
	}
	for _, name := range []string{"wf-1", "wf-2", "wf-3"} {
		if refreshedByName[name] == 0 {
			t.Fatalf("active run %s starved: never refreshed across cycles (got %v)", name, refreshedByName)
		}
	}
}

func TestSyncActiveRunEvents_ReconcilesMisclassifiedTerminalRun(t *testing.T) {
	ctx := context.Background()
	createdAt := time.Now().UTC().Add(-10 * time.Minute)
	finishedAt := time.Now().UTC().Add(-5 * time.Minute)
	batchJobID := "batch-1"
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:           "run-1",
				WorkflowName: "wf-1",
				Status:       "Error",
				Message:      staleWorkflowTTLCleanupMessage,
				CreatedAt:    createdAt,
				FinishedAt:   &finishedAt,
				BatchJobID:   &batchJobID,
			},
		},
	}
	runID := "run-1"
	backfillRepo := &watcherBackfillRepo{
		itemsByRunID: map[string]*models.BackfillItem{
			runID: {
				ID:            "item-1",
				JobID:         batchJobID,
				AssetID:       "asset-1",
				Status:        "failed",
				PipelineRunID: &runID,
			},
		},
	}
	wfClient := &mockWorkflowClient{}
	getWorkflowCalls := 0
	wfClient.getWorkflowFn = func(_ context.Context, name, namespace string) (*wfv1.Workflow, error) {
		getWorkflowCalls++
		if name != "wf-1" {
			t.Fatalf("unexpected workflow name %q", name)
		}
		return &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "wf-1", Namespace: namespace, UID: "uid-1"},
			Status: wfv1.WorkflowStatus{
				Phase:      wfv1.WorkflowSucceeded,
				FinishedAt: metav1.Time{Time: finishedAt},
			},
		}, nil
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetRunEventRepo(&mockRunEventRepo{})
	uc.SetBackfillRepo(backfillRepo)

	synced, err := uc.SyncActiveRunEvents(ctx, 25)
	if err != nil {
		t.Fatalf("SyncActiveRunEvents: %v", err)
	}
	if synced == 0 {
		t.Fatal("expected watcher to count the reconciled anomalous run")
	}
	if getWorkflowCalls == 0 {
		t.Fatal("expected watcher to reconcile terminal anomaly through Argo")
	}
	if got := runRepo.byID["run-1"].Status; got != string(wfv1.WorkflowSucceeded) {
		t.Fatalf("expected Succeeded after watcher reconcile, got %q", got)
	}
	if runRepo.byID["run-1"].Message != "" {
		t.Fatalf("expected stale runtime message cleared, got %q", runRepo.byID["run-1"].Message)
	}
	if got := backfillRepo.itemsByRunID[runID].Status; got != "completed" {
		t.Fatalf("expected batch item completed after run repair, got %q", got)
	}
	if len(backfillRepo.updated) != 1 {
		t.Fatalf("expected one batch item status update, got %d", len(backfillRepo.updated))
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

	uc.refreshRunStatus(ctx, runRepo.byID["run-batch"], nodeProjectTerminalArchive)
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

	uc.refreshRunStatus(ctx, runRepo.byID["run-batch"], nodeProjectTerminalArchive)
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

	uc.refreshRunStatus(ctx, runRepo.byID["run-1"], nodeProjectTerminalArchive)

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

func TestRefreshRunStatus_PreservesRecentActiveRunBeforeStaleLedger(t *testing.T) {
	ctx := context.Background()
	finishedAt := time.Now().UTC().Add(-1 * time.Minute)
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:           "run-1",
				WorkflowName: "wf-1",
				Status:       "Running",
				FinishedAt:   &finishedAt,
				Message:      staleWorkflowTTLCleanupMessage,
				CreatedAt:    time.Now().UTC().Add(-10 * time.Minute),
			},
		},
	}
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, name, namespace string) (*wfv1.Workflow, error) {
		if name != "wf-1" {
			t.Fatalf("unexpected workflow name %q", name)
		}
		return nil, argo.ErrNotFound
	}
	assetNodeRepo := &mockAssetNodeRepo{
		byRun: map[string][]models.PipelineRunAssetNode{
			"run-1": {
				{RunID: "run-1", PipelineNodeID: "step-1", Status: "Error", Message: staleWorkflowTTLCleanupMessage},
			},
		},
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetObservabilityRepositories(assetNodeRepo, nil, nil)

	uc.refreshRunStatus(ctx, runRepo.byID["run-1"], nodeProjectTerminalArchive)

	run := runRepo.byID["run-1"]
	if run.Status != "Running" {
		t.Fatalf("expected recent active run to stay Running, got %q", run.Status)
	}
	if run.Message != "" {
		t.Fatalf("expected stale TTL message cleared, got %q", run.Message)
	}
	if run.FinishedAt != nil {
		t.Fatalf("expected active run finished_at cleared, got %v", run.FinishedAt)
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

func TestListRunSummaries_NormalizesActiveStaleTerminalFields(t *testing.T) {
	ctx := context.Background()
	finishedAt := time.Now().UTC().Add(-1 * time.Minute)
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:           "run-1",
				WorkflowName: "wf-1",
				Status:       "Running",
				FinishedAt:   &finishedAt,
				Message:      staleWorkflowTTLCleanupMessage,
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

	items, total, err := uc.ListRunSummaries(ctx, models.PipelineRunListFilter{ExcludeBatch: true})
	if err != nil {
		t.Fatalf("ListRunSummaries: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("expected one summary, total=%d len=%d", total, len(items))
	}
	if getWorkflowCalls != 0 {
		t.Fatalf("default summary should not refresh Argo, got %d calls", getWorkflowCalls)
	}
	if items[0].FinishedAt != nil {
		t.Fatalf("expected active summary finished_at cleared, got %v", items[0].FinishedAt)
	}
	if items[0].Message != "" {
		t.Fatalf("expected stale active summary message cleared, got %q", items[0].Message)
	}
}

// CYB-3491: an unfiltered ListRunSummaries must never full-scan pipeline_runs.
// It now routes through the bounded paginated path (ListSummaries with a
// default page size) instead of the old unbounded FindAllSummaries.
func TestListRunSummaries_UnfilteredIsBounded(t *testing.T) {
	ctx := context.Background()
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:           "run-1",
				WorkflowName: "wf-1",
				Status:       "Succeeded",
				CreatedAt:    time.Now().UTC(),
			},
		},
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})

	if _, _, err := uc.ListRunSummaries(ctx); err != nil {
		t.Fatalf("ListRunSummaries (no filter): %v", err)
	}

	// The unfiltered call must go through ListSummaries (bounded), not the old
	// FindAllSummaries full-scan.
	if len(runRepo.listFilters) != 1 {
		t.Fatalf("expected exactly one bounded ListSummaries call, got %d", len(runRepo.listFilters))
	}
	if got := runRepo.listFilters[0].PageSize; got != defaultUnfilteredRunSummaryPageSize {
		t.Fatalf("unfiltered list PageSize = %d, want %d (bounded default)", got, defaultUnfilteredRunSummaryPageSize)
	}
}

// CYB-3490: list reads are pure — RefreshActive is accepted but ignored,
// and no Argo call happens on the request path.
func TestListRunSummaries_PureRead_IgnoresRefreshActive(t *testing.T) {
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
	if getWorkflowCalls != 0 {
		t.Fatalf("expected pure list read (0 Argo calls), got %d", getWorkflowCalls)
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

func TestRefreshRunForList_PersistsWorkflowStartedAt(t *testing.T) {
	ctx := context.Background()
	createdAt := time.Date(2026, 6, 28, 4, 0, 0, 0, time.UTC)
	startedAt := createdAt.Add(2 * time.Minute)
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:           "run-1",
				WorkflowName: "wf-1",
				Status:       "Pending",
				CreatedAt:    createdAt,
			},
		},
	}
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, name, namespace string) (*wfv1.Workflow, error) {
		return &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, UID: "uid-1"},
			Status: wfv1.WorkflowStatus{
				Phase:     wfv1.WorkflowRunning,
				StartedAt: metav1.Time{Time: startedAt},
			},
		}, nil
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})

	run := runRepo.byID["run-1"]
	uc.RefreshRunForList(ctx, run)
	if run.StartedAt == nil || !run.StartedAt.Equal(startedAt) {
		t.Fatalf("expected startedAt %v, got %v", startedAt, run.StartedAt)
	}
	saved := runRepo.byID["run-1"]
	if saved.StartedAt == nil || !saved.StartedAt.Equal(startedAt) {
		t.Fatalf("expected persisted startedAt %v, got %v", startedAt, saved.StartedAt)
	}
}

// CYB-3491(原则):任务可以等——只有任务自己报错才是失败。48h 无更新的
// active run 不再被硬翻成 Failed(以前这里会,而底下的 Argo workflow 从
// 未被停过、也没 GC)。停滞是"读侧的展示提示",不是"写侧的状态判决":
// 通过 runstate.AnnotateRunDiagnostics 标注 BlockingReason,而 status 保
// 持 Argo 真值,若 webhook/poll 后来交回更新自然回归正常。
func TestRefreshRunStatus_StaleActiveRunKeepsStatusNoZombieMark(t *testing.T) {
	ctx := context.Background()
	createdAt := time.Now().UTC().Add(-staleActiveRunMaxAge - 2*time.Hour)
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:           "run-1",
				WorkflowName: "wf-1",
				Status:       "Running",
				CreatedAt:    createdAt,
				UpdatedAt:    createdAt,
			},
		},
	}
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
		return &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "wf-1", UID: "uid-1"},
			Status: wfv1.WorkflowStatus{
				Phase:     wfv1.WorkflowRunning,
				StartedAt: metav1.Time{Time: createdAt.Add(time.Hour)},
			},
		}, nil
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})

	run := runRepo.byID["run-1"]
	uc.refreshRunStatus(ctx, run, nodeProjectTerminalArchive)
	if run.Status != string(wfv1.WorkflowRunning) {
		t.Fatalf("stale-but-still-active run must keep Argo truth (Running), got %q", run.Status)
	}
	if strings.Contains(run.Message, "stale run:") {
		t.Fatalf("stale zombie message must NOT be written to the ledger, got %q", run.Message)
	}
	saved := runRepo.byID["run-1"]
	if saved.FinishedAt != nil {
		t.Fatalf("stale-but-active run must not carry finished_at, got %v", saved.FinishedAt)
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

// CYB-3490: batch-asset attempt list is a pure ledger read.
func TestListBatchAssetRuns_PureRead_NoArgoRefresh(t *testing.T) {
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
		t.Fatalf("unexpected Argo call on pure list read (workflow %q)", name)
		return nil, nil
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
	// CYB-3490: pure read — the ledger row is returned as persisted; healing
	// belongs to the background watcher, not the list request.
	if runs[0].Status != "Error" {
		t.Fatalf("expected as-persisted status Error, got %q", runs[0].Status)
	}
	if runRepo.byID["run-1"].Status != "Error" {
		t.Fatalf("expected persisted run untouched (Error), got %q", runRepo.byID["run-1"].Status)
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

func TestGetRun_ReturnsBackfillOnlyFailedSummary(t *testing.T) {
	ctx := context.Background()
	batchJobID := "batch-1"
	finishedAt := time.Now().UTC()
	runRepo := &mockRunRepo{
		summaryByID: map[string]*models.PipelineRun{
			"item-1": {
				ID:            "item-1",
				Status:        "Failed",
				Message:       "invalid argument: node head_tracking runtime secret video-proc-dev-db-creds is not available",
				FailureReason: "run_failed",
				BatchJobID:    &batchJobID,
				AssetIDs:      []string{"asset-1"},
				AssetCount:    1,
				CreatedAt:     time.Now().UTC(),
				FinishedAt:    &finishedAt,
			},
		},
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})

	run, err := uc.GetRun(ctx, "item-1")
	if err != nil {
		t.Fatalf("GetRun() error = %v", err)
	}
	if run == nil {
		t.Fatal("GetRun() returned nil for summary-only failed batch item")
	}
	if run.Status != "Failed" || run.Message == "" {
		t.Fatalf("run = %+v, want failed summary with message", run)
	}
	if run.WorkflowName != "" {
		t.Fatalf("workflowName = %q, want empty for pre-submit failure", run.WorkflowName)
	}
}

func TestListRunEvents_ReturnsEmptyForBackfillOnlyFailedSummary(t *testing.T) {
	ctx := context.Background()
	batchJobID := "batch-1"
	finishedAt := time.Now().UTC()
	runRepo := &mockRunRepo{
		summaryByID: map[string]*models.PipelineRun{
			"item-1": {
				ID:         "item-1",
				Status:     "Failed",
				Message:    "invalid argument: runtime secret is not available",
				BatchJobID: &batchJobID,
				AssetIDs:   []string{"asset-1"},
				AssetCount: 1,
				CreatedAt:  time.Now().UTC(),
				FinishedAt: &finishedAt,
			},
		},
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, nil, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})

	result, err := uc.ListRunEvents(ctx, "item-1", models.PipelineRunEventListOptions{Limit: 100})
	if err != nil {
		t.Fatalf("ListRunEvents() error = %v", err)
	}
	if result == nil || len(result.Items) != 0 {
		t.Fatalf("events = %+v, want empty event list", result)
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

func TestGetRun_ReconcilesMisclassifiedFailedWithRunningNodes(t *testing.T) {
	ctx := context.Background()
	finishedAt := time.Now().UTC().Add(-10 * time.Minute)
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:           "run-1",
				WorkflowName: "wf-1",
				Status:       "Failed",
				FinishedAt:   &finishedAt,
				Message:      "run_failed",
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
			Status: wfv1.WorkflowStatus{
				Nodes: map[string]wfv1.NodeStatus{
					"wf-1": {
						ID:          "wf-1",
						Name:        "wf-1",
						DisplayName: "wf-1",
						Type:        wfv1.NodeTypeDAG,
						Phase:       wfv1.NodeRunning,
					},
					"wf-1-123": {
						ID:           "wf-1-123",
						Name:         "wf-1.step-a",
						DisplayName:  "step-a",
						TemplateName: "step-a",
						Type:         wfv1.NodeTypePod,
						Phase:        wfv1.NodeRunning,
					},
				},
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
	if run.Status != "Running" {
		t.Fatalf("expected reconciled Running status, got %q", run.Status)
	}
	if run.FinishedAt != nil {
		t.Fatalf("expected active run finished_at cleared, got %v", run.FinishedAt)
	}
	if len(run.Nodes) == 0 || run.Nodes[0].Phase != "Running" {
		t.Fatalf("expected running nodes to be refreshed, got %#v", run.Nodes)
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
	createdAt := now.Add(-staleActiveRunMaxAge - time.Minute)
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

// CYB-3491(语义): scheduling starvation is WAITING, not failure. An
// over-threshold unschedulable run keeps its Argo-truth active phase — pods
// that cannot schedule today schedule when capacity frees. Only the
// workload's own errors are terminal.
func TestRefreshRunForList_UnschedulableStaysWaiting(t *testing.T) {
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
		return &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "wf-1", UID: "uid-1", CreationTimestamp: metav1.Time{Time: startedAt}},
			Status: wfv1.WorkflowStatus{
				Phase:     wfv1.WorkflowRunning,
				StartedAt: metav1.Time{Time: startedAt},
				Nodes: map[string]wfv1.NodeStatus{
					"node-1": {
						ID: "node-1", Name: "wf-1-step", DisplayName: "step",
						Type: wfv1.NodeTypePod, Phase: wfv1.NodePending,
						Message:   "0/11 nodes are available: 1 Insufficient cpu, 2 Insufficient memory.",
						StartedAt: metav1.Time{Time: startedAt}, TemplateName: "step",
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
	if run.Status != string(wfv1.WorkflowRunning) {
		t.Fatalf("unschedulable run must stay active (waiting), got %q", run.Status)
	}
	if run.FinishedAt != nil {
		t.Fatalf("waiting run must not carry finished_at, got %v", run.FinishedAt)
	}
	for _, event := range eventRepo.events {
		if event.EventType == runEventFailed && event.Reason == "unschedulable" {
			t.Fatalf("waiting run must not emit an unschedulable run_failed event")
		}
	}
}

// Legacy verdicts minted by the removed guard ("Kubernetes 调度失败:…") are
// NOT definitive — the workflow was never stopped in Argo. The misclassified
// reconciler restores them to their true active phase.
func TestReconcileMisclassified_RevivesLegacyUnschedulableVerdict(t *testing.T) {
	ctx := context.Background()
	finished := time.Date(2026, 7, 15, 14, 34, 0, 0, time.UTC)
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:              "run-1",
				WorkflowName:    "wf-1",
				ArgoWorkflowUID: "uid-1",
				Status:          "Error",
				Message:         "Kubernetes 调度失败：节点 \"step-x\" 已 Pending 2h，Unschedulable: 0/239 nodes are available",
				FinishedAt:      &finished,
				CreatedAt:       finished.Add(-3 * time.Hour),
			},
		},
	}
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, name, _ string) (*wfv1.Workflow, error) {
		return &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "wf-1", UID: "uid-1"},
			Status:     wfv1.WorkflowStatus{Phase: wfv1.WorkflowRunning},
		}, nil
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetRunEventRepo(&mockRunEventRepo{})

	run := runRepo.byID["run-1"]
	uc.reconcileMisclassifiedRunFromArgo(ctx, run, nodeProjectTerminalArchive)
	got := runRepo.byID["run-1"]
	if got.Status != string(wfv1.WorkflowRunning) {
		t.Fatalf("legacy unschedulable verdict must revive to Argo truth, got %q", got.Status)
	}

	// A genuine failure verdict stays definitive and is NOT revived.
	if isUnschedulableGuardVerdict("component exited with code 1") {
		t.Fatal("real failure message must not match the guard-verdict matcher")
	}
	if !isUnschedulableGuardVerdict("Kubernetes 调度失败：节点 \"x\" 已 Pending 1h") {
		t.Fatal("guard verdict matcher must match the legacy prefix")
	}
}

// CYB-3672: past-revival-age terminal misclassified runs must NOT call Argo
// GetWorkflow — the workflow was TTL-GCed long ago, every lookup returns 404
// and floods argo-server's ERROR log. Observed on shared argo-server: ~1 req/s
// steady load per legacy stranded run across all namespaces.
func TestReconcileMisclassified_SkipsArgo_WhenPastRevivalAge(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	// 60 days old ≫ staleActiveRunMaxAge, so age gate must reject.
	createdAt := now.Add(-60 * 24 * time.Hour)
	finished := createdAt.Add(1 * time.Hour)
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:           "run-1",
				WorkflowName: "youxin-1782182330729-legacy",
				Status:       "Error",
				Message:      "", // empty; also covers real-diagnostic case since we don't touch message
				FinishedAt:   &finished,
				CreatedAt:    createdAt,
			},
		},
	}
	wfClient := &mockWorkflowClient{}
	getCalls := 0
	wfClient.getWorkflowFn = func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
		getCalls++
		return nil, argo.ErrNotFound
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetRunEventRepo(&mockRunEventRepo{})

	uc.reconcileMisclassifiedRunFromArgo(ctx, runRepo.byID["run-1"], nodeProjectTerminalArchive)
	if getCalls != 0 {
		t.Fatalf("past-revival-age terminal run must not call Argo GetWorkflow, got %d", getCalls)
	}
	// Status + message untouched.
	if runRepo.byID["run-1"].Status != "Error" {
		t.Fatalf("status must be preserved, got %q", runRepo.byID["run-1"].Status)
	}
	if runRepo.byID["run-1"].Message != "" {
		t.Fatalf("message must not be stomped, got %q", runRepo.byID["run-1"].Message)
	}
}

// CYB-3672 negative: within-revival-age terminal misclassified runs must still
// call Argo — they might genuinely need reviving (e.g., "Kubernetes 调度失败"
// verdict on a workflow that's actually still Running).
func TestReconcileMisclassified_CallsArgo_WithinRevivalAge(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	// 30 min old, well within staleActiveRunMaxAge — Argo must be consulted.
	createdAt := now.Add(-30 * time.Minute)
	finished := createdAt.Add(1 * time.Minute)
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			"run-1": {
				ID:              "run-1",
				WorkflowName:    "wf-fresh",
				ArgoWorkflowUID: "uid-1",
				Status:          "Error",
				Message:         "Kubernetes 调度失败：节点 x Pending 1h",
				FinishedAt:      &finished,
				CreatedAt:       createdAt,
			},
		},
	}
	wfClient := &mockWorkflowClient{}
	getCalls := 0
	wfClient.getWorkflowFn = func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
		getCalls++
		return &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "wf-fresh", UID: "uid-1"},
			Status:     wfv1.WorkflowStatus{Phase: wfv1.WorkflowRunning},
		}, nil
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetRunEventRepo(&mockRunEventRepo{})

	uc.reconcileMisclassifiedRunFromArgo(ctx, runRepo.byID["run-1"], nodeProjectTerminalArchive)
	if getCalls != 1 {
		t.Fatalf("fresh terminal run must call Argo GetWorkflow once, got %d", getCalls)
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

func TestRefreshRunForList_MarksTerminalImageFailureImmediately(t *testing.T) {
	// ImagePullBackOff / InvalidImageName should mark the run as Error
	// immediately, without waiting for UnschedulablePendingThreshold.
	ctx := context.Background()
	now := time.Date(2026, 6, 18, 10, 1, 0, 0, time.UTC)
	startedAt := now.Add(-30 * time.Second) // only 30 seconds ago
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
		return &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "wf-1", UID: "uid-1", CreationTimestamp: metav1.Time{Time: startedAt}},
			Status: wfv1.WorkflowStatus{
				Phase:     wfv1.WorkflowRunning,
				StartedAt: metav1.Time{Time: startedAt},
				Nodes: map[string]wfv1.NodeStatus{
					"node-1": {
						ID:           "node-1",
						Name:         "wf-1-smoke",
						DisplayName:  "smoke-task",
						Type:         wfv1.NodeTypePod,
						Phase:        wfv1.NodePending,
						Message:      `ImagePullBackOff: Back-off pulling image "registry.example.com/smoke-task"`,
						StartedAt:    metav1.Time{Time: startedAt},
						TemplateName: "smoke",
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
	// Set a 15 min threshold — the test proves we don't wait that long.
	uc.SetResourceGuardConfig(ResourceGuardConfig{UnschedulablePendingThreshold: 15 * time.Minute})
	uc.now = func() time.Time { return now }

	run := runRepo.byID["run-1"]
	uc.RefreshRunForList(ctx, run)
	if run.Status != string(wfv1.WorkflowError) {
		t.Fatalf("expected immediate Error for ImagePullBackOff, got %q (should not wait 15 min)", run.Status)
	}
	if !strings.Contains(run.Message, "镜像启动失败") || !strings.Contains(run.Message, "ImagePullBackOff") {
		t.Fatalf("expected image diagnostics in message, got %q", run.Message)
	}
	if run.FinishedAt == nil || !run.FinishedAt.Equal(now) {
		t.Fatalf("expected finished_at %v, got %v", now, run.FinishedAt)
	}
	// Verify the pending duration in the message is short (~30s), not 15+ minutes.
	if !strings.Contains(run.Message, "30s") && !strings.Contains(run.Message, "29s") {
		t.Logf("note: pending duration in message: %q (expected ~30s)", run.Message)
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

func TestRefreshRunFromWorkflowByName_AppliesArgoTruth(t *testing.T) {
	ctx := context.Background()
	run := &models.PipelineRun{ID: "run-1", WorkflowName: "wf-1", Status: "Running", ArgoWorkflowUID: "uid-1"}
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{"run-1": run},
		byWf: map[string]*models.PipelineRun{"wf-1": run},
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

	got, err := uc.RefreshRunFromWorkflowByName(ctx, "wf-1", "uid-1")
	if err != nil {
		t.Fatalf("RefreshRunFromWorkflowByName: %v", err)
	}
	if got == nil {
		t.Fatalf("expected a run, got nil")
	}
	if !isSucceededRunStatus(got.Status) {
		t.Fatalf("expected succeeded status, got %q", got.Status)
	}
}

func TestRefreshRunFromWorkflowByName_UnknownWorkflowReturnsNil(t *testing.T) {
	ctx := context.Background()
	runRepo := &mockRunRepo{byWf: map[string]*models.PipelineRun{}}
	wfClient := &mockWorkflowClient{}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})

	got, err := uc.RefreshRunFromWorkflowByName(ctx, "no-such-wf", "")
	if err != nil {
		t.Fatalf("RefreshRunFromWorkflowByName: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil run for unknown workflow, got %+v", got)
	}
}

func TestRefreshRunFromWorkflowByName_UIDMismatchDoesNotApply(t *testing.T) {
	ctx := context.Background()
	run := &models.PipelineRun{ID: "run-1", WorkflowName: "wf-1", Status: "Running", ArgoWorkflowUID: "uid-1"}
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{"run-1": run},
		byWf: map[string]*models.PipelineRun{"wf-1": run},
	}
	called := false
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
		called = true
		return &wfv1.Workflow{Status: wfv1.WorkflowStatus{Phase: wfv1.WorkflowSucceeded}}, nil
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})

	got, err := uc.RefreshRunFromWorkflowByName(ctx, "wf-1", "uid-2-different")
	if err != nil {
		t.Fatalf("RefreshRunFromWorkflowByName: %v", err)
	}
	if got == nil || got.Status != "Running" {
		t.Fatalf("expected run untouched (Running), got %+v", got)
	}
	if called {
		t.Fatalf("GetWorkflow must not be called on UID mismatch")
	}
}

func TestPersistRunObservation_SucceededNotRegressedToActive(t *testing.T) {
	ctx := context.Background()
	existing := &models.PipelineRun{ID: "run-1", WorkflowName: "wf-1", Status: "Succeeded"}
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{"run-1": existing},
		byWf: map[string]*models.PipelineRun{"wf-1": existing},
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, &mockWorkflowClient{}, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})

	// A late/out-of-order observation tries to move the run back to Running.
	stale := &models.PipelineRun{ID: "run-1", WorkflowName: "wf-1", Status: "Running"}
	uc.persistRunObservation(ctx, stale)

	if runRepo.byID["run-1"].Status != "Succeeded" {
		t.Fatalf("succeeded run must not regress, got %q", runRepo.byID["run-1"].Status)
	}
	if stale.Status != "Succeeded" {
		t.Fatalf("returned run should reflect terminal status, got %q", stale.Status)
	}
}

const resourceRejectionMessage = `invalid argument: 执行目标 "Default Argo target"不支持该资源规格：节点 "head-track-pycuvslam" 请求 cpu=14000m，最大可用 cpu=8`

// TestPersistRunObservation_DefinitiveFailureNotRegressedToActive guards
// CYB-3080 at the central choke point: a run rejected before submission (real
// Failed + a substantive message) must not be reverted to an active phase by a
// "waiting for workflow creation" revival, which would leave it polling a
// workflow that will never exist.
func TestPersistRunObservation_DefinitiveFailureNotRegressedToActive(t *testing.T) {
	ctx := context.Background()
	existing := &models.PipelineRun{ID: "run-1", WorkflowName: "wf-1", Status: "Failed", Message: resourceRejectionMessage}
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{"run-1": existing},
		byWf: map[string]*models.PipelineRun{"wf-1": existing},
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, &mockWorkflowClient{}, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})

	// A "still waiting for workflow creation" observation tries to revive it.
	revive := &models.PipelineRun{ID: "run-1", WorkflowName: "wf-1", Status: "Pending", Message: ""}
	uc.persistRunObservation(ctx, revive)

	if runRepo.byID["run-1"].Status != "Failed" {
		t.Fatalf("definitively-failed run must not regress to active, got %q", runRepo.byID["run-1"].Status)
	}
	if runRepo.byID["run-1"].Message != resourceRejectionMessage {
		t.Fatalf("failure message must be preserved, got %q", runRepo.byID["run-1"].Message)
	}
	if revive.Status != "Failed" {
		t.Fatalf("returned run should reflect the preserved terminal status, got %q", revive.Status)
	}
}

// TestPersistRunObservation_MisclassifiedFailureStillRevivable confirms the new
// guard does NOT over-block: a terminal run whose message is a known
// stale/transient "workflow unavailable" message (TTL-cleanup false positive)
// is a misclassification, not a definitive failure, and remains revivable.
func TestPersistRunObservation_MisclassifiedFailureStillRevivable(t *testing.T) {
	ctx := context.Background()
	existing := &models.PipelineRun{ID: "run-1", WorkflowName: "wf-1", Status: "Error", Message: staleWorkflowTTLCleanupMessage}
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{"run-1": existing},
		byWf: map[string]*models.PipelineRun{"wf-1": existing},
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, &mockWorkflowClient{}, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})

	revive := &models.PipelineRun{ID: "run-1", WorkflowName: "wf-1", Status: "Running", Message: ""}
	uc.persistRunObservation(ctx, revive)

	if runRepo.byID["run-1"].Status != "Running" {
		t.Fatalf("misclassified (stale-message) run should stay revivable, got %q", runRepo.byID["run-1"].Status)
	}
}

// TestNeedsWatcherAnomalyReconcile_SkipsAgedRunWithRecentUpdatedAt covers the
// self-perpetuating reconcile loop: an old TTL-cleaned run whose UpdatedAt keeps
// getting bumped by ordinary observation writes must not be picked back into the
// anomaly reconcile set. runObservedRecently is expected to use only lifecycle
// timestamps (FinishedAt → StartedAt → CreatedAt), not UpdatedAt.
func TestNeedsWatcherAnomalyReconcile_SkipsAgedRunWithRecentUpdatedAt(t *testing.T) {
	now := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	oldFinished := now.AddDate(0, -1, 0) // ~30 days ago
	oldCreated := now.AddDate(0, -1, -1)
	freshFinished := now.Add(-30 * time.Minute)
	// UpdatedAt is recent — simulating a repo Save from any code path that
	// touched the row (persistRunObservation, syncBackfillItemStatusFromRun,
	// UpdateLedgerState, etc.). Under the old ref chain this would drag the
	// run back into the anomaly set forever.
	recentUpdated := now.Add(-2 * time.Hour)

	cases := []struct {
		name        string
		run         models.PipelineRun
		want        bool
		description string
	}{
		{
			name: "aged TTL-cleaned run with recent UpdatedAt is not observed recently",
			run: models.PipelineRun{
				ID:           "run-aged",
				WorkflowName: "youxin-1782-aged",
				Status:       "Expired",
				Message:      staleWorkflowTTLCleanupMessage,
				LedgerState:  "pending",
				CreatedAt:    oldCreated,
				UpdatedAt:    recentUpdated,
				FinishedAt:   &oldFinished,
			},
			want:        false,
			description: "FinishedAt is the reference, not UpdatedAt",
		},
		{
			name: "aged run with nil FinishedAt still not selected via UpdatedAt fallback",
			run: models.PipelineRun{
				ID:           "run-aged-no-fin",
				WorkflowName: "youxin-1782-nofin",
				Status:       "Failed",
				Message:      staleWorkflowTTLCleanupMessage,
				LedgerState:  "",
				CreatedAt:    oldCreated,
				UpdatedAt:    recentUpdated,
			},
			want:        false,
			description: "falls through to CreatedAt (StartedAt nil), which is 30d ago; UpdatedAt is skipped",
		},
		{
			name: "recently finished run is still eligible for reconcile",
			run: models.PipelineRun{
				ID:           "run-fresh",
				WorkflowName: "wf-fresh",
				Status:       "Failed",
				Message:      staleWorkflowTTLCleanupMessage,
				LedgerState:  "pending",
				CreatedAt:    now.Add(-6 * time.Hour),
				UpdatedAt:    now.Add(-1 * time.Minute),
				FinishedAt:   &freshFinished,
			},
			want:        true,
			description: "FinishedAt is inside the 7-day window",
		},
		{
			name: "ledger already resolved to no_ledger short-circuits regardless of dates",
			run: models.PipelineRun{
				ID:           "run-noledger",
				WorkflowName: "wf-noledger",
				Status:       "Failed",
				Message:      staleWorkflowTTLCleanupMessage,
				LedgerState:  "no_ledger",
				CreatedAt:    now.Add(-1 * time.Hour),
				UpdatedAt:    now.Add(-1 * time.Minute),
				FinishedAt:   &freshFinished,
			},
			want:        false,
			description: "no_ledger is a definitive answer — do not re-poll Argo",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := needsWatcherAnomalyReconcile(&tc.run, now)
			if got != tc.want {
				t.Fatalf("needsWatcherAnomalyReconcile(%s) = %v, want %v (%s)",
					tc.name, got, tc.want, tc.description)
			}
		})
	}
}

// TestReconcileMisclassifiedRunFromArgo_DoesNotReviveResourceRejectedRun
// reproduces the actual reported path (batch detail page → GetRun →
// reconcileMisclassifiedRunFromArgo): a resource-guard-rejected batch subtask
// run has a placeholder "-batch-" workflow name and no UID, so the
// "pending creation" heuristic matches it; without the persist guard it gets
// revived to Pending. It must stay Failed. (CYB-3080)
func TestReconcileMisclassifiedRunFromArgo_DoesNotReviveResourceRejectedRun(t *testing.T) {
	ctx := context.Background()
	batchJobID := "batch-1"
	existing := &models.PipelineRun{
		ID:           "run-1",
		WorkflowName: "youxin-all-batch-82470acae720", // placeholder batch name, no UID
		Status:       "Failed",
		Message:      resourceRejectionMessage,
		BatchJobID:   &batchJobID,
		CreatedAt:    time.Now().UTC(), // within the creation grace period
	}
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{"run-1": existing},
		byWf: map[string]*models.PipelineRun{"youxin-all-batch-82470acae720": existing},
	}
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, _, _ string) (*wfv1.Workflow, error) {
		return nil, argo.ErrNotFound
	}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})

	// Operate on a fresh copy, as GetRun does after FindByID.
	run := *existing
	uc.reconcileMisclassifiedRunFromArgo(ctx, &run, nodeProjectTerminalArchive)

	if runRepo.byID["run-1"].Status != "Failed" {
		t.Fatalf("resource-rejected run must stay Failed, got %q", runRepo.byID["run-1"].Status)
	}
	if runRepo.byID["run-1"].Message != resourceRejectionMessage {
		t.Fatalf("failure message must be preserved, got %q", runRepo.byID["run-1"].Message)
	}
}
