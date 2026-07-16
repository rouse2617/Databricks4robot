package workflow

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

	"github.com/CyberOrigin2077/cyber-databrew/internal/argo"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// ── Mocks ───────────────────────────────────────────────────────────────────

// stubArgoFactory implements argo.ClientFactory. Records the cluster IDs it was
// asked for and returns a pre-canned client per ID (mirrors the pipeline
// usecase's routing test stub).
type stubArgoFactory struct {
	seen   []string
	byID   map[string]argo.WorkflowClient
	err    error
	forced bool // when true, ignore byID and always return err
}

func (s *stubArgoFactory) ForCluster(_ context.Context, id string) (argo.WorkflowClient, error) {
	s.seen = append(s.seen, id)
	if s.forced {
		return nil, s.err
	}
	c, ok := s.byID[id]
	if !ok {
		return nil, errors.New("cluster not found in stub: " + id)
	}
	return c, nil
}

func (s *stubArgoFactory) ForTarget(ctx context.Context, t *models.ExecutionTarget) (argo.WorkflowClient, error) {
	id := clusterDefault
	if t != nil && t.ClusterID != "" {
		id = t.ClusterID
	}
	return s.ForCluster(ctx, id)
}

func (s *stubArgoFactory) Invalidate(string) {}

var _ argo.ClientFactory = (*stubArgoFactory)(nil)

// mockTargetRepo implements repository.ExecutionTargetRepository for the
// ExecutionTargetID → cluster_id lookup path.
type mockTargetRepo struct {
	byID  map[string]*models.ExecutionTarget
	calls int
}

func (m *mockTargetRepo) Save(context.Context, *models.ExecutionTarget) error { return nil }
func (m *mockTargetRepo) FindAll(context.Context) ([]models.ExecutionTarget, error) {
	return nil, nil
}
func (m *mockTargetRepo) FindByID(_ context.Context, id string) (*models.ExecutionTarget, error) {
	m.calls++
	return m.byID[id], nil
}
func (m *mockTargetRepo) FindDefault(context.Context) (*models.ExecutionTarget, error) {
	return nil, nil
}
func (m *mockTargetRepo) Delete(context.Context, string) error { return nil }

var _ repository.ExecutionTargetRepository = (*mockTargetRepo)(nil)

// ── resolveRunClusterID ─────────────────────────────────────────────────────

// A populated ExecutionTarget wins and no target lookup happens.
func TestResolveRunClusterID_ObjectFirst(t *testing.T) {
	repo := &mockTargetRepo{byID: map[string]*models.ExecutionTarget{}}
	h := &Handler{targetRepo: repo}
	run := &models.PipelineRun{ExecutionTarget: &models.ExecutionTarget{ClusterID: "cluster-delivery"}}
	if got := h.resolveRunClusterID(context.Background(), run); got != "cluster-delivery" {
		t.Errorf("want cluster-delivery, got %q", got)
	}
	if repo.calls != 0 {
		t.Errorf("target repo must not be consulted when ExecutionTarget is populated (calls=%d)", repo.calls)
	}
}

// The FindByWorkflowName path only has ExecutionTargetID (no nested object), so
// the cluster must be resolved via the target lookup — and memoized.
func TestResolveRunClusterID_LookupByTargetIDAndCache(t *testing.T) {
	repo := &mockTargetRepo{byID: map[string]*models.ExecutionTarget{
		"delivery-clust-dev": {ID: "delivery-clust-dev", ClusterID: "cluster-delivery"},
	}}
	h := &Handler{targetRepo: repo}
	run := &models.PipelineRun{ExecutionTargetID: "delivery-clust-dev"}
	if got := h.resolveRunClusterID(context.Background(), run); got != "cluster-delivery" {
		t.Fatalf("want cluster-delivery via target lookup, got %q", got)
	}
	// Second call must hit the cache: delete the repo row, still resolves.
	delete(repo.byID, "delivery-clust-dev")
	if got := h.resolveRunClusterID(context.Background(), run); got != "cluster-delivery" {
		t.Errorf("cache miss: want cluster-delivery from cache, got %q", got)
	}
	if repo.calls != 1 {
		t.Errorf("target repo should be consulted once then cached, calls=%d", repo.calls)
	}
}

// nil run, no target id, and target-not-found all resolve to cluster-default;
// a not-found is NOT cached (so a transient error can't pin the run).
func TestResolveRunClusterID_Fallbacks(t *testing.T) {
	repo := &mockTargetRepo{byID: map[string]*models.ExecutionTarget{}}
	h := &Handler{targetRepo: repo}
	ctx := context.Background()
	if got := h.resolveRunClusterID(ctx, nil); got != clusterDefault {
		t.Errorf("nil run: want %q, got %q", clusterDefault, got)
	}
	if got := h.resolveRunClusterID(ctx, &models.PipelineRun{}); got != clusterDefault {
		t.Errorf("no target id: want %q, got %q", clusterDefault, got)
	}
	run := &models.PipelineRun{ExecutionTargetID: "ghost"}
	if got := h.resolveRunClusterID(ctx, run); got != clusterDefault {
		t.Errorf("missing target: want %q, got %q", clusterDefault, got)
	}
	if _, cached := h.targetClusterCache.Load("ghost"); cached {
		t.Errorf("a not-found target must not be cached")
	}
	// Now add it → resolves (proves the miss wasn't cached).
	repo.byID["ghost"] = &models.ExecutionTarget{ID: "ghost", ClusterID: "cluster-x"}
	if got := h.resolveRunClusterID(ctx, run); got != "cluster-x" {
		t.Errorf("after adding target: want cluster-x, got %q", got)
	}
}

// No target repo wired → ExecutionTargetID can't be resolved → cluster-default.
func TestResolveRunClusterID_NoTargetRepo(t *testing.T) {
	h := &Handler{}
	run := &models.PipelineRun{ExecutionTargetID: "delivery-clust-dev"}
	if got := h.resolveRunClusterID(context.Background(), run); got != clusterDefault {
		t.Errorf("no target repo: want %q, got %q", clusterDefault, got)
	}
}

// ── argoClientForRun ────────────────────────────────────────────────────────

func TestArgoClientForRun_FactoryRoutesByCluster(t *testing.T) {
	delivery := &mockWorkflowClient{}
	factory := &stubArgoFactory{byID: map[string]argo.WorkflowClient{"cluster-delivery": delivery}}
	singleton := &mockWorkflowClient{}
	h := &Handler{wfClient: singleton, argoFactory: factory}
	run := &models.PipelineRun{ExecutionTarget: &models.ExecutionTarget{ClusterID: "cluster-delivery"}}
	if got := h.argoClientForRun(context.Background(), run); got != delivery {
		t.Errorf("want delivery client, got singleton/other")
	}
	if len(factory.seen) != 1 || factory.seen[0] != "cluster-delivery" {
		t.Errorf("factory saw wrong cluster IDs: %v", factory.seen)
	}
}

func TestArgoClientForRun_NoFactoryReturnsSingleton(t *testing.T) {
	singleton := &mockWorkflowClient{}
	h := &Handler{wfClient: singleton} // no factory
	run := &models.PipelineRun{ExecutionTarget: &models.ExecutionTarget{ClusterID: "cluster-delivery"}}
	if got := h.argoClientForRun(context.Background(), run); got != singleton {
		t.Errorf("no factory must return the singleton wfClient")
	}
}

// A factory error (misconfigured/unknown cluster) degrades to the singleton so
// the request keeps working with prior behavior instead of erroring.
func TestArgoClientForRun_FactoryErrorFallsBackToSingleton(t *testing.T) {
	singleton := &mockWorkflowClient{}
	factory := &stubArgoFactory{err: errors.New("cluster misconfigured"), forced: true}
	h := &Handler{wfClient: singleton, argoFactory: factory}
	run := &models.PipelineRun{ExecutionTarget: &models.ExecutionTarget{ClusterID: "cluster-broken"}}
	if got := h.argoClientForRun(context.Background(), run); got != singleton {
		t.Errorf("factory error must fall back to the singleton wfClient")
	}
}

// Empty/blank cluster ids resolve through cluster-default in the factory.
func TestArgoClientForCluster_BlankDefaultsToClusterDefault(t *testing.T) {
	def := &mockWorkflowClient{}
	factory := &stubArgoFactory{byID: map[string]argo.WorkflowClient{clusterDefault: def}}
	h := &Handler{argoFactory: factory}
	if got := h.argoClientForCluster(context.Background(), ""); got != def {
		t.Errorf("blank cluster id must resolve to %q client", clusterDefault)
	}
	if len(factory.seen) != 1 || factory.seen[0] != clusterDefault {
		t.Errorf("factory saw %v", factory.seen)
	}
}

// ── HTTP-level routing ──────────────────────────────────────────────────────

// A run whose target resolves to a non-default cluster routes GetWorkflow to
// that cluster's client and namespace — the CYB-3486 fix. The singleton client
// fails the test if consulted.
func TestGetWorkflow_RoutesToOwningClusterByRun(t *testing.T) {
	var nsSeen string
	delivery := &mockWorkflowClient{
		getFn: func(_ context.Context, name, namespace string) (*wfv1.Workflow, error) {
			nsSeen = namespace
			return makeWorkflow(name, "Running", 1), nil
		},
	}
	singleton := &mockWorkflowClient{
		getFn: func(context.Context, string, string) (*wfv1.Workflow, error) {
			t.Fatal("singleton client must NOT be used for a non-default-cluster workflow")
			return nil, nil
		},
	}
	factory := &stubArgoFactory{byID: map[string]argo.WorkflowClient{"cluster-delivery": delivery}}

	h := New(singleton, "default")
	h.SetArgoFactory(factory)
	h.SetExecutionTargetRepo(&mockTargetRepo{byID: map[string]*models.ExecutionTarget{
		"delivery-clust-dev": {ID: "delivery-clust-dev", ClusterID: "cluster-delivery"},
	}})
	h.SetRunRepositories(&mockRunRepo{byWorkflow: map[string]*models.PipelineRun{
		"wf-delivery": {
			ID:                "run-d",
			WorkflowName:      "wf-delivery",
			Status:            "Running", // active → skip DB reconstruct, hit live Argo
			ExecutionTargetID: "delivery-clust-dev",
			ArgoNamespace:     "cyber-delivery-dev",
		},
	}}, nil, nil)

	r := setupRouter(h)
	req := httptest.NewRequest(http.MethodGet, "/workflows/wf-delivery", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if nsSeen != "cyber-delivery-dev" {
		t.Fatalf("delivery client saw namespace %q, want cyber-delivery-dev", nsSeen)
	}
	if len(factory.seen) == 0 || factory.seen[len(factory.seen)-1] != "cluster-delivery" {
		t.Fatalf("factory should have resolved cluster-delivery, saw %v", factory.seen)
	}
}

// With no factory wired (e.g. no Postgres), GetWorkflow uses the singleton
// client — unchanged pre-3486 behavior.
func TestGetWorkflow_NoFactoryUsesSingleton(t *testing.T) {
	var used bool
	singleton := &mockWorkflowClient{
		getFn: func(_ context.Context, name, _ string) (*wfv1.Workflow, error) {
			used = true
			return makeWorkflow(name, "Running", 1), nil
		},
	}
	h := New(singleton, "default") // no SetArgoFactory
	h.SetRunRepositories(&mockRunRepo{byWorkflow: map[string]*models.PipelineRun{
		"wf-x": {ID: "r", WorkflowName: "wf-x", Status: "Running", ExecutionTargetID: "t1"},
	}}, nil, nil)

	r := setupRouter(h)
	req := httptest.NewRequest(http.MethodGet, "/workflows/wf-x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !used {
		t.Fatal("singleton client must be used when no factory is wired")
	}
}

// Lifecycle operations (via workflowOperation) route to the owning cluster too.
func TestStopWorkflow_RoutesToOwningCluster(t *testing.T) {
	delivery := &mockWorkflowClient{}
	singleton := &mockWorkflowClient{}
	factory := &stubArgoFactory{byID: map[string]argo.WorkflowClient{"cluster-delivery": delivery}}

	h := New(singleton, "default")
	h.SetArgoFactory(factory)
	h.SetExecutionTargetRepo(&mockTargetRepo{byID: map[string]*models.ExecutionTarget{
		"delivery-clust-dev": {ID: "delivery-clust-dev", ClusterID: "cluster-delivery"},
	}})
	h.SetRunRepositories(&mockRunRepo{byWorkflow: map[string]*models.PipelineRun{
		"wf-delivery": {
			ID: "run-d", WorkflowName: "wf-delivery", Status: "Running",
			ExecutionTargetID: "delivery-clust-dev", ArgoNamespace: "cyber-delivery-dev",
		},
	}}, &mockRunEventRepo{}, nil)

	r := setupRouter(h)
	req := httptest.NewRequest(http.MethodPost, "/workflows/wf-delivery/stop", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if delivery.operation != "stop" {
		t.Errorf("stop must be issued on the delivery client, got operation %q", delivery.operation)
	}
	if delivery.namespace != "cyber-delivery-dev" {
		t.Errorf("stop namespace = %q, want cyber-delivery-dev", delivery.namespace)
	}
	if singleton.operation != "" {
		t.Errorf("singleton client must not receive the stop op, got %q", singleton.operation)
	}
}
