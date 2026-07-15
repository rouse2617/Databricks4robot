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

// stubArgoFactory implements argo.ClientFactory for routing tests. Records
// which cluster ID was requested and returns a pre-canned client per ID.
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
	id := "cluster-default"
	if t != nil && t.ClusterID != "" {
		id = t.ClusterID
	}
	return s.ForCluster(ctx, id)
}

func (s *stubArgoFactory) Invalidate(string) {}

var _ argo.ClientFactory = (*stubArgoFactory)(nil)

// TestResolveArgoClient_FactoryRoutesByTarget confirms the resolve helper hits
// the factory when it's wired and picks the client matching target.ClusterID.
func TestResolveArgoClient_FactoryRoutesByTarget(t *testing.T) {
	delivery := &mockWorkflowClient{}
	factory := &stubArgoFactory{byID: map[string]argo.WorkflowClient{
		"cluster-delivery": delivery,
	}}
	// wfClient must NOT be consulted when factory is set.
	singleton := &mockWorkflowClient{
		createWorkflowFn: func(context.Context, *wfv1.Workflow, string) error {
			t.Fatal("singleton wfClient must not be used when factory is wired")
			return nil
		},
	}
	uc := &Usecase{wfClient: singleton, argoFactory: factory}
	got, err := uc.resolveArgoClient(context.Background(), &models.ExecutionTarget{ClusterID: "cluster-delivery"})
	if err != nil {
		t.Fatalf("resolveArgoClient: %v", err)
	}
	if got != delivery {
		t.Errorf("wrong client returned")
	}
	if len(factory.seen) != 1 || factory.seen[0] != "cluster-delivery" {
		t.Errorf("factory saw wrong cluster IDs: %v", factory.seen)
	}
}

// TestResolveArgoClient_EmptyClusterIDFallsToDefault verifies that pre-3486
// rows (empty ClusterID) resolve through cluster-default in the factory.
func TestResolveArgoClient_EmptyClusterIDFallsToDefault(t *testing.T) {
	def := &mockWorkflowClient{}
	factory := &stubArgoFactory{byID: map[string]argo.WorkflowClient{"cluster-default": def}}
	uc := &Usecase{argoFactory: factory}
	got, err := uc.resolveArgoClient(context.Background(), &models.ExecutionTarget{ClusterID: ""})
	if err != nil {
		t.Fatal(err)
	}
	if got != def {
		t.Errorf("empty ClusterID must resolve to cluster-default")
	}
}

// TestResolveArgoClient_FactoryErrorWrapped ensures factory failures surface
// as ErrWorkflowUnavailable with the cluster ID in the message.
func TestResolveArgoClient_FactoryErrorWrapped(t *testing.T) {
	factory := &stubArgoFactory{err: errors.New("cluster misconfigured: missing audience"), forced: true}
	uc := &Usecase{argoFactory: factory}
	_, err := uc.resolveArgoClient(context.Background(), &models.ExecutionTarget{ClusterID: "cluster-broken"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrWorkflowUnavailable) {
		t.Errorf("expected ErrWorkflowUnavailable, got %v", err)
	}
}

// TestResolveArgoClient_NoFactoryReturnsSingleton verifies the legacy compat
// path: with argoFactory nil, resolveArgoClient returns uc.wfClient verbatim.
func TestResolveArgoClient_NoFactoryReturnsSingleton(t *testing.T) {
	singleton := &mockWorkflowClient{}
	uc := &Usecase{wfClient: singleton}
	got, err := uc.resolveArgoClient(context.Background(), &models.ExecutionTarget{ClusterID: "ignored"})
	if err != nil {
		t.Fatal(err)
	}
	if got != singleton {
		t.Errorf("no factory must return uc.wfClient")
	}
}

// TestResolveArgoClient_NoFactoryNoSingletonReturnsNil covers the adapter-only
// unit-test path — both factory and wfClient nil. Returns nil client, no error;
// caller (Deploy → submitRuntimeWorkflow) then falls through to the runtime
// adapter.
func TestResolveArgoClient_NoFactoryNoSingletonReturnsNil(t *testing.T) {
	uc := &Usecase{}
	got, err := uc.resolveArgoClient(context.Background(), &models.ExecutionTarget{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil client, got %T", got)
	}
}

// TestSubmitRuntimeWorkflow_UsesProvidedClient checks that submit calls
// CreateWorkflow on the caller-provided client (i.e. downstream ops in Deploy
// will use the same client, avoiding cross-cluster misroutes flagged in the
// PR 4c review).
func TestSubmitRuntimeWorkflow_UsesProvidedClient(t *testing.T) {
	var called bool
	client := &mockWorkflowClient{
		createWorkflowFn: func(context.Context, *wfv1.Workflow, string) error {
			called = true
			return nil
		},
	}
	uc := &Usecase{}
	wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "wf", UID: "uid"}}
	job, err := uc.submitRuntimeWorkflow(context.Background(), client, "run", "name", wf, "ns")
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("provided client's CreateWorkflow was not called")
	}
	if job == nil || job.Ref.Name != "wf" {
		t.Errorf("unexpected job: %+v", job)
	}
}

// TestSubmitRuntimeWorkflow_NilClientFallsToAdapter preserves the unit-test
// path that only wires a runtime adapter (no factory, no wfClient). Submit
// with a nil client hands off to adapter.Submit.
func TestSubmitRuntimeWorkflow_NilClientFallsToAdapter(t *testing.T) {
	uc := &Usecase{}
	_, err := uc.submitRuntimeWorkflow(context.Background(), nil, "r", "n",
		&wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "wf"}}, "ns")
	if !errors.Is(err, ErrWorkflowUnavailable) {
		t.Errorf("no adapter + nil client should error ErrWorkflowUnavailable, got %v", err)
	}
}

// TestGetWorkflowWithUID_UsesProvidedClient — this is the guardrail against
// CYB-3486 PR 4c's downstream misroute (spotted by gemini-code-assist in the
// PR #404 review). getWorkflowWithUID must call GetWorkflow on the passed-in
// client, not on any singleton. When we later thread the factory-resolved
// client through Deploy, this test proves the downstream read follows.
func TestGetWorkflowWithUID_UsesProvidedClient(t *testing.T) {
	var providedCalled bool
	provided := &mockWorkflowClient{
		getWorkflowFn: func(_ context.Context, name, ns string) (*wfv1.Workflow, error) {
			providedCalled = true
			return &wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, UID: "the-uid"},
			}, nil
		},
	}
	// Wire a DIFFERENT singleton on the usecase; if the code accidentally
	// reads from it, this test would fail loud.
	singleton := &mockWorkflowClient{
		getWorkflowFn: func(context.Context, string, string) (*wfv1.Workflow, error) {
			t.Fatal("wfClient singleton must not be consulted when a client is provided")
			return nil, nil
		},
	}
	uc := &Usecase{wfClient: singleton}
	wf, err := uc.getWorkflowWithUID(context.Background(), provided, "name", "ns")
	if err != nil {
		t.Fatalf("getWorkflowWithUID: %v", err)
	}
	if !providedCalled {
		t.Fatal("provided client's GetWorkflow was not called")
	}
	if wf == nil || wf.UID != "the-uid" {
		t.Errorf("unexpected wf: %+v", wf)
	}
}
