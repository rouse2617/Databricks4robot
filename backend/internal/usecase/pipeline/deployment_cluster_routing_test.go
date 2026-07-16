package pipeline

import (
	"context"
	"testing"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

	"github.com/CyberOrigin2077/cyber-databrew/internal/argo"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// CYB-3486d1: the legacy resource-usage / deployment-status reads used to hit
// the cyber-clust singleton (uc.wfClient) regardless of the run's cluster, so a
// run on a non-default cluster (delivery-clust, CRD mode) 404'd — the #436/#437
// misroute class. These tests wire a stubArgoFactory + targetRepo and assert
// the reads route to the run's own cluster client, never the singleton (which
// fails the test loudly if consulted).

// deliveryClusterFixtures builds the shared cross-cluster setup: a run whose
// execution target resolves to "cluster-delivery" and a factory that only
// knows that cluster.
func deliveryClusterFixtures(runID string) (*stubArgoFactory, *mockRunRepo, *mockTargetRepo) {
	factory := &stubArgoFactory{byID: map[string]argo.WorkflowClient{}}
	runRepo := &mockRunRepo{byWf: map[string]*models.PipelineRun{
		"wf-1": {
			ID:                runID,
			WorkflowName:      "wf-1",
			Status:            "Running",
			ExecutionTargetID: "tgt-delivery",
			ArgoNamespace:     "cyber-delivery-dev",
		},
	}}
	targetRepo := &mockTargetRepo{byID: map[string]*models.ExecutionTarget{
		"tgt-delivery": {ID: "tgt-delivery", ClusterID: "cluster-delivery"},
	}}
	return factory, runRepo, targetRepo
}

// mustNotUseSingletonWorkflow returns a singleton client that fails the test if
// its GetWorkflow is ever called.
func mustNotUseSingletonWorkflow(t *testing.T) *mockWorkflowClient {
	t.Helper()
	return &mockWorkflowClient{
		getWorkflowFn: func(context.Context, string, string) (*wfv1.Workflow, error) {
			t.Fatal("singleton wfClient must not be consulted for a non-default-cluster run")
			return nil, nil
		},
	}
}

// TestGetWorkflowResourceUsage_RoutesToRunCluster covers the live path
// (WorkflowNodeDetailPanel → GET /workflows/:name/nodes/:id/resources).
func TestGetWorkflowResourceUsage_RoutesToRunCluster(t *testing.T) {
	var gotNamespace string
	delivery := &mockWorkflowClient{
		getWorkflowFn: func(_ context.Context, _, ns string) (*wfv1.Workflow, error) {
			gotNamespace = ns
			wf := &wfv1.Workflow{}
			wf.Status.Phase = wfv1.WorkflowRunning
			wf.Status.Nodes = wfv1.Nodes{"pod-1": {ID: "pod-1", Type: wfv1.NodeTypePod}}
			return wf, nil
		},
	}
	factory, runRepo, targetRepo := deliveryClusterFixtures("run-1")
	factory.byID["cluster-delivery"] = delivery
	uc := &Usecase{
		wfClient:    mustNotUseSingletonWorkflow(t),
		argoFactory: factory,
		runRepo:     runRepo,
		targetRepo:  targetRepo,
		namespace:   "default",
	}

	report, err := uc.getWorkflowResourceUsage(context.Background(), "wf-1", "")
	if err != nil {
		t.Fatalf("getWorkflowResourceUsage: %v", err)
	}
	if report.DeploymentID != "run-1" {
		t.Errorf("expected deploymentID run-1 (from runRepo), got %q", report.DeploymentID)
	}
	if gotNamespace != "cyber-delivery-dev" {
		t.Errorf("expected workflow read in cyber-delivery-dev, got %q", gotNamespace)
	}
	if len(factory.seen) != 1 || factory.seen[0] != "cluster-delivery" {
		t.Errorf("factory saw wrong clusters: %v", factory.seen)
	}
}

// TestGetResourceUsage_RoutesToRunCluster covers GET /deployments/:id/resources:
// the deployments row has no cluster, but the dual-written run (same id /
// workflow_name) does, so the read must route through it.
func TestGetResourceUsage_RoutesToRunCluster(t *testing.T) {
	manifest := "kind: Workflow\nspec: {}\n"
	var gotNamespace string
	delivery := &mockWorkflowClient{
		getWorkflowFn: func(_ context.Context, _, ns string) (*wfv1.Workflow, error) {
			gotNamespace = ns
			wf := &wfv1.Workflow{}
			wf.Status.Phase = wfv1.WorkflowRunning
			return wf, nil
		},
	}
	factory, runRepo, targetRepo := deliveryClusterFixtures("dep-1")
	factory.byID["cluster-delivery"] = delivery
	depRepo := &mockDeploymentRepo{byID: map[string]*models.PipelineDeployment{
		"dep-1": {ID: "dep-1", WorkflowName: "wf-1", Status: "Running", Manifest: &manifest},
	}}
	uc := &Usecase{
		wfClient:       mustNotUseSingletonWorkflow(t),
		argoFactory:    factory,
		runRepo:        runRepo,
		targetRepo:     targetRepo,
		deploymentRepo: depRepo,
		namespace:      "default",
	}

	report, err := uc.GetResourceUsage(context.Background(), "dep-1")
	if err != nil {
		t.Fatalf("GetResourceUsage: %v", err)
	}
	if gotNamespace != "cyber-delivery-dev" {
		t.Errorf("expected workflow read in cyber-delivery-dev, got %q", gotNamespace)
	}
	if report.Source.Workflow != "argo-live" {
		t.Errorf("expected argo-live workflow source, got %q", report.Source.Workflow)
	}
	if len(factory.seen) != 1 || factory.seen[0] != "cluster-delivery" {
		t.Errorf("factory saw wrong clusters: %v", factory.seen)
	}
}

// TestRefreshDeploymentStatus_RoutesToRunCluster is the regression guard: a live
// deployment on a non-default cluster must be probed on its own cluster and
// keep its real phase, not get wrongly marked Expired because the singleton
// 404'd it.
func TestRefreshDeploymentStatus_RoutesToRunCluster(t *testing.T) {
	var gotNamespace string
	delivery := &mockWorkflowClient{
		getWorkflowStatusFn: func(_ context.Context, _, ns string) (wfv1.WorkflowPhase, error) {
			gotNamespace = ns
			return wfv1.WorkflowRunning, nil
		},
	}
	singleton := &mockWorkflowClient{
		getWorkflowStatusFn: func(context.Context, string, string) (wfv1.WorkflowPhase, error) {
			t.Fatal("singleton wfClient must not be probed for a non-default-cluster deployment")
			return wfv1.WorkflowUnknown, nil
		},
	}
	factory, runRepo, targetRepo := deliveryClusterFixtures("dep-1")
	factory.byID["cluster-delivery"] = delivery
	dep := &models.PipelineDeployment{ID: "dep-1", WorkflowName: "wf-1", Status: "Running"}
	depRepo := &mockDeploymentRepo{byID: map[string]*models.PipelineDeployment{"dep-1": dep}}
	uc := &Usecase{
		wfClient:       singleton,
		argoFactory:    factory,
		runRepo:        runRepo,
		targetRepo:     targetRepo,
		deploymentRepo: depRepo,
		namespace:      "default",
	}

	uc.refreshDeploymentStatus(context.Background(), dep)

	if dep.Status != string(wfv1.WorkflowRunning) {
		t.Errorf("expected Running (not wrongly Expired), got %q", dep.Status)
	}
	if gotNamespace != "cyber-delivery-dev" {
		t.Errorf("expected status probe in cyber-delivery-dev, got %q", gotNamespace)
	}
	if len(factory.seen) != 1 || factory.seen[0] != "cluster-delivery" {
		t.Errorf("factory saw wrong clusters: %v", factory.seen)
	}
}
