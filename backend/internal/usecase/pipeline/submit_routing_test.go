package pipeline

import (
	"context"
	"errors"
	"testing"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/CyberOrigin2077/cyber-databrew/internal/argo"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// stubArgoFactory implements argo.ClientFactory for submit-routing tests.
// It records which cluster ID was requested and returns the pre-canned client
// when the cluster matches, or an error otherwise.
type stubArgoFactory struct {
	seen    []string
	byID    map[string]argo.WorkflowClient
	err     error
	forced  bool // when true, ignore byID and return err
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
	id := "cluster-default"
	if t != nil && t.ClusterID != "" {
		id = t.ClusterID
	}
	return s.ForCluster(ctx, id)
}

func (s *stubArgoFactory) Invalidate(string) {}

var _ argo.ClientFactory = (*stubArgoFactory)(nil)

// TestSubmitRuntimeWorkflow_FactoryRoutesByTarget checks that when argoFactory
// is wired, submit resolves the client via ForTarget(target) and uses that
// client's CreateWorkflow — proving multi-cluster routing lands on the correct
// cluster (CYB-3486 PR 4c).
func TestSubmitRuntimeWorkflow_FactoryRoutesByTarget(t *testing.T) {
	defaultClient := &mockWorkflowClient{}
	deliveryClient := &mockWorkflowClient{}
	var deliveryCalled bool
	deliveryClient.createWorkflowFn = func(_ context.Context, _ *wfv1.Workflow, _ string) error {
		deliveryCalled = true
		return nil
	}
	var defaultCalled bool
	defaultClient.createWorkflowFn = func(_ context.Context, _ *wfv1.Workflow, _ string) error {
		defaultCalled = true
		return nil
	}

	factory := &stubArgoFactory{
		byID: map[string]argo.WorkflowClient{
			"cluster-default":  defaultClient,
			"cluster-delivery": deliveryClient,
		},
	}

	// wfClient set to the "default" mock as a fallback the factory should NOT
	// use. If the factory branch is skipped by accident, defaultCalled would
	// still flip — so instead we use a third distinct singleton that must never
	// fire.
	singleton := &mockWorkflowClient{
		createWorkflowFn: func(context.Context, *wfv1.Workflow, string) error {
			t.Fatal("singleton wfClient must not be called when factory is wired")
			return nil
		},
	}

	uc := &Usecase{
		wfClient:    singleton,
		argoFactory: factory,
	}

	wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "wf-1", UID: "uid-1"}}
	target := &models.ExecutionTarget{ID: "target-x", ClusterID: "cluster-delivery"}
	job, err := uc.submitRuntimeWorkflow(context.Background(), target, "run-1", "run-name", wf, "ns-1")
	if err != nil {
		t.Fatalf("submit failed: %v", err)
	}
	if job == nil || job.Ref.Name != "wf-1" {
		t.Fatalf("unexpected job: %+v", job)
	}
	if !deliveryCalled {
		t.Fatal("delivery cluster client was not called")
	}
	if defaultCalled {
		t.Fatal("default cluster client was called; routing did not honor ClusterID")
	}
	if len(factory.seen) != 1 || factory.seen[0] != "cluster-delivery" {
		t.Errorf("factory saw wrong cluster IDs: %v", factory.seen)
	}
}

// TestSubmitRuntimeWorkflow_FactoryEmptyClusterIDFallsToDefault verifies that
// a target with empty ClusterID (pre-CYB-3486 rows) still resolves through the
// factory's cluster-default seat — no behavior regression on the migration path.
func TestSubmitRuntimeWorkflow_FactoryEmptyClusterIDFallsToDefault(t *testing.T) {
	var defaultCalled bool
	defaultClient := &mockWorkflowClient{
		createWorkflowFn: func(context.Context, *wfv1.Workflow, string) error {
			defaultCalled = true
			return nil
		},
	}
	factory := &stubArgoFactory{byID: map[string]argo.WorkflowClient{
		"cluster-default": defaultClient,
	}}
	uc := &Usecase{argoFactory: factory}
	target := &models.ExecutionTarget{ID: "target-legacy", ClusterID: ""}
	if _, err := uc.submitRuntimeWorkflow(context.Background(), target, "r", "n",
		&wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "wf"}}, "ns"); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if !defaultCalled {
		t.Fatal("empty ClusterID must route to cluster-default")
	}
}

// TestSubmitRuntimeWorkflow_FactoryErrorWrapped confirms that a factory failure
// bubbles up as ErrWorkflowUnavailable with the cluster ID in the message so
// operators can pinpoint the misconfigured row.
func TestSubmitRuntimeWorkflow_FactoryErrorWrapped(t *testing.T) {
	factory := &stubArgoFactory{err: errors.New("cluster misconfigured: missing audience"), forced: true}
	uc := &Usecase{argoFactory: factory}
	target := &models.ExecutionTarget{ID: "target-bad", ClusterID: "cluster-broken"}
	_, err := uc.submitRuntimeWorkflow(context.Background(), target, "r", "n",
		&wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "wf"}}, "ns")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrWorkflowUnavailable) {
		t.Errorf("expected ErrWorkflowUnavailable, got %v", err)
	}
}

// TestSubmitRuntimeWorkflow_NoFactoryFallsToLegacy checks the compat path: when
// the factory is nil, the legacy runtimeAdapter/wfClient flow runs unchanged.
func TestSubmitRuntimeWorkflow_NoFactoryFallsToLegacy(t *testing.T) {
	var singletonCalled bool
	singleton := &mockWorkflowClient{
		createWorkflowFn: func(context.Context, *wfv1.Workflow, string) error {
			singletonCalled = true
			return nil
		},
	}
	uc := &Usecase{wfClient: singleton}
	target := &models.ExecutionTarget{ID: "t", ClusterID: "cluster-anything"}
	if _, err := uc.submitRuntimeWorkflow(context.Background(), target, "r", "n",
		&wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "wf"}}, "ns"); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if !singletonCalled {
		t.Fatal("with factory nil, legacy singleton path must run")
	}
}
