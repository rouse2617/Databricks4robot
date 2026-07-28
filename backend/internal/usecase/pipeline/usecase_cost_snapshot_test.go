package pipeline

import (
	"testing"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

func TestRunCostSnapshotMissing(t *testing.T) {
	pricing := &PricingConfig{Units: map[string]map[string]float64{}}
	uc := &Usecase{pricing: pricing}

	if !uc.runCostSnapshotMissing(&models.PipelineRun{}) {
		t.Fatalf("expected empty node snapshot to be considered missing when pricing is configured")
	}

	if !uc.runCostSnapshotMissing(&models.PipelineRun{
		NodeCount: 2,
		Nodes: []models.PipelineRunNode{
			{
				PipelineNodeID:    "step-1",
				ResourcesDuration: map[string]interface{}{"cpu": 10},
			},
		},
	}) {
		t.Fatalf("expected resource duration without estimated cost to be considered missing")
	}

	cost := 0.01
	if uc.runCostSnapshotMissing(&models.PipelineRun{
		NodeCount: 1,
		Nodes: []models.PipelineRunNode{
			{
				PipelineNodeID:    "step-1",
				ResourcesDuration: map[string]interface{}{"cpu": 10},
				EstimatedCostUSD:  &cost,
			},
		},
	}) {
		t.Fatalf("expected populated estimated cost snapshot to be considered present")
	}

	ucWithoutPricing := &Usecase{}
	if !ucWithoutPricing.runCostSnapshotMissing(&models.PipelineRun{
		NodeCount: 2,
		Nodes: []models.PipelineRunNode{
			{PipelineNodeID: "step-1"},
		},
	}) {
		t.Fatalf("expected missing node snapshot to be rebuilt even when pricing is not configured")
	}
}

func TestAppendMissingStaticRunNodesUsesNonNilChildren(t *testing.T) {
	wf := &wfv1.Workflow{
		Spec: wfv1.WorkflowSpec{
			Templates: []wfv1.Template{
				{
					Name: "dag",
					DAG: &wfv1.DAGTemplate{
						Tasks: []wfv1.DAGTask{
							{Name: "step-prepare", Template: "step-prepare"},
						},
					},
				},
				{
					Name:      "step-prepare",
					Container: &corev1.Container{},
				},
			},
		},
		Status: wfv1.WorkflowStatus{Phase: wfv1.WorkflowSucceeded},
	}

	nodes := appendMissingStaticRunNodesFromWorkflow("run-1", wf, nil)
	if len(nodes) != 1 {
		t.Fatalf("expected one static node, got %d", len(nodes))
	}
	if nodes[0].Children == nil {
		t.Fatalf("expected static node children to be an empty slice, not nil")
	}
}

func TestDeriveAssetNodeRowsDeduplicatesAndPrefersRuntimePod(t *testing.T) {
	cost := 0.02
	run := &models.PipelineRun{ID: "run-1", AssetIDs: []string{"asset-1"}}
	nodes := []models.PipelineRunNode{
		{
			PipelineNodeID: "dag",
			ArgoNodeID:     "argo-dag",
			DisplayName:    "workflow-root",
			Type:           string(wfv1.NodeTypeDAG),
			Phase:          string(wfv1.NodeSucceeded),
		},
		{
			PipelineNodeID: "step-qc",
			ArgoNodeID:     "static:step-qc",
			DisplayName:    "step-qc",
			Phase:          string(wfv1.NodePending),
		},
		{
			PipelineNodeID:    "step-qc",
			ArgoNodeID:        "argo-node-1",
			DisplayName:       "step-qc",
			Phase:             string(wfv1.NodeSucceeded),
			PodName:           "wf.step-qc",
			EstimatedCostUSD:  &cost,
			ResourcesDuration: map[string]interface{}{"cpu": 10},
		},
	}

	rows := deriveAssetNodeRows(run, nodes)
	if len(rows) != 1 {
		t.Fatalf("expected one deduped asset-node row, got %d", len(rows))
	}
	if rows[0].ArgoNodeID != "argo-node-1" || rows[0].PodName != "wf.step-qc" || rows[0].EstimatedCostUSD == nil {
		t.Fatalf("expected runtime pod row to win, got %#v", rows[0])
	}
}
