package pipeline

import (
	"context"
	"fmt"
	"testing"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/transpiler"
)

// buildRetryStageWorkflow returns a Running workflow with `podPhases` step pods
// plus an exit-notify hook pod. It models an in-flight argo-server /retry, which
// deletes the failed step's node from the map — so fewer step pods than the
// pipeline's step count remain, all of them succeeded.
func buildRetryStageWorkflow(name string, podPhases []wfv1.NodePhase) *wfv1.Workflow {
	fin := metav1.NewTime(time.Date(2026, 7, 21, 1, 20, 0, 0, time.UTC))
	nodes := map[string]wfv1.NodeStatus{
		name: {Name: name, Type: wfv1.NodeTypeDAG, Phase: wfv1.NodeRunning},
		// exit-notify hook pod — must be ignored by both counters.
		name + "-exit": {
			Name:         name + "-exit",
			TemplateName: transpiler.ExitNotifyTemplateName,
			Type:         wfv1.NodeTypePod,
			Phase:        wfv1.NodeRunning,
		},
	}
	for i, ph := range podPhases {
		id := fmt.Sprintf("%s-step-%d", name, i)
		nodes[id] = wfv1.NodeStatus{
			Name:         id,
			DisplayName:  fmt.Sprintf("step-%d", i),
			TemplateName: fmt.Sprintf("step-%d", i),
			Type:         wfv1.NodeTypePod,
			Phase:        ph,
			FinishedAt:   fin,
		}
	}
	return &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status:     wfv1.WorkflowStatus{Phase: wfv1.WorkflowRunning, Nodes: nodes},
	}
}

func TestBusinessStepPodCountExcludesExitHookAndNonPods(t *testing.T) {
	wf := buildRetryStageWorkflow("wf-1", []wfv1.NodePhase{
		wfv1.NodeSucceeded, wfv1.NodeSucceeded, wfv1.NodeSucceeded,
	})
	if got := businessStepPodCount(wf); got != 3 {
		t.Fatalf("businessStepPodCount = %d, want 3 (exit hook + DAG excluded)", got)
	}
	if got := businessStepPodCount(nil); got != 0 {
		t.Fatalf("businessStepPodCount(nil) = %d, want 0", got)
	}
}

// TestApplyWorkflowDoesNotFinalizeIncompleteRetry: a mid-retry workflow (one step
// pod deleted, all remaining succeeded, phase Running) must NOT be finalized as
// Succeeded when the pipeline expects more steps — this is the CYB-3707 bug.
func TestApplyWorkflowDoesNotFinalizeIncompleteRetry(t *testing.T) {
	ctx := context.Background()
	run := &models.PipelineRun{ID: "run-1", WorkflowName: "wf-1", Status: "Running", NodeCount: 6}
	runRepo := &mockRunRepo{byID: map[string]*models.PipelineRun{"run-1": run}}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, &mockWorkflowClient{}, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})

	// 5 succeeded step pods but the pipeline has 6 steps — one was deleted by the
	// retry. Must not finalize to Succeeded.
	wf := buildRetryStageWorkflow("wf-1", []wfv1.NodePhase{
		wfv1.NodeSucceeded, wfv1.NodeSucceeded, wfv1.NodeSucceeded, wfv1.NodeSucceeded, wfv1.NodeSucceeded,
	})
	uc.applyWorkflowToRun(ctx, run, wf, nodeProjectLive)
	if isSucceededRunStatus(run.Status) {
		t.Fatalf("run finalized to %q from an incomplete mid-retry workflow; want non-succeeded", run.Status)
	}
}

// TestApplyWorkflowFinalizesCompleteRun: when every expected step pod is present
// and succeeded (phase still Running only transiently), the finalize-early path
// still works — the guard must not over-block genuine completion.
func TestApplyWorkflowFinalizesCompleteRun(t *testing.T) {
	ctx := context.Background()
	run := &models.PipelineRun{ID: "run-2", WorkflowName: "wf-2", Status: "Running", NodeCount: 3}
	runRepo := &mockRunRepo{byID: map[string]*models.PipelineRun{"run-2": run}}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, &mockWorkflowClient{}, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})

	wf := buildRetryStageWorkflow("wf-2", []wfv1.NodePhase{
		wfv1.NodeSucceeded, wfv1.NodeSucceeded, wfv1.NodeSucceeded,
	})
	uc.applyWorkflowToRun(ctx, run, wf, nodeProjectLive)
	if !isSucceededRunStatus(run.Status) {
		t.Fatalf("run status = %q, want Succeeded (all %d step pods present + succeeded)", run.Status, run.NodeCount)
	}
}

// TestRecoverStuckSucceededRunFromLedger: a run mislabeled Succeeded whose
// asset-node ledger shows a Failed leaf is corrected back to Failed.
func TestRecoverStuckSucceededRunFromLedger(t *testing.T) {
	ctx := context.Background()
	run := &models.PipelineRun{ID: "run-3", WorkflowName: "wf-3", Status: "Succeeded"}
	runRepo := &mockRunRepo{byID: map[string]*models.PipelineRun{"run-3": {ID: "run-3", WorkflowName: "wf-3", Status: "Succeeded"}}}
	assetNodes := &mockAssetNodeRepo{byRun: map[string][]models.PipelineRunAssetNode{
		"run-3": {
			{RunID: "run-3", PipelineNodeID: "step-a", Status: "Succeeded"},
			{RunID: "run-3", PipelineNodeID: "step-b", Status: "Failed", Message: "main: Error (exit code 1)"},
		},
	}}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, &mockWorkflowClient{}, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetObservabilityRepositories(assetNodes, nil, nil)

	uc.recoverStuckSucceededRun(ctx, run)
	if !isTerminalFailureRunStatus(run.Status) {
		t.Fatalf("run status = %q, want a terminal failure after ledger recovery", run.Status)
	}
}

// TestRecoverStuckSucceededRunLeavesGenuineSuccess: a Succeeded run whose ledger
// agrees (all leaves succeeded) is NOT flipped.
func TestRecoverStuckSucceededRunLeavesGenuineSuccess(t *testing.T) {
	ctx := context.Background()
	run := &models.PipelineRun{ID: "run-4", WorkflowName: "wf-4", Status: "Succeeded"}
	runRepo := &mockRunRepo{byID: map[string]*models.PipelineRun{"run-4": {ID: "run-4", WorkflowName: "wf-4", Status: "Succeeded"}}}
	assetNodes := &mockAssetNodeRepo{byRun: map[string][]models.PipelineRunAssetNode{
		"run-4": {
			{RunID: "run-4", PipelineNodeID: "step-a", Status: "Succeeded"},
			{RunID: "run-4", PipelineNodeID: "step-b", Status: "Succeeded"},
		},
	}}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, &mockWorkflowClient{}, "default")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetObservabilityRepositories(assetNodes, nil, nil)

	uc.recoverStuckSucceededRun(ctx, run)
	if !isSucceededRunStatus(run.Status) {
		t.Fatalf("run status = %q, want Succeeded left intact", run.Status)
	}
}
