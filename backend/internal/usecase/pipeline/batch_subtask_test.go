package pipeline

import (
	"context"
	"testing"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

func TestUpsertBatchSubtaskRunReusesExistingRunForBatchAsset(t *testing.T) {
	t.Parallel()

	const (
		batchJobID = "job-1"
		assetID    = "23324"
		existingID = "0139f5b5-b612-42c6-a958-616a755c8fae"
		templateID = "tmpl-1"
	)

	existing := &models.PipelineRun{
		ID:           existingID,
		WorkflowName: "3-batch-23324",
		Status:       "Error",
		AssetIDs:     []string{assetID},
		BatchJobID:   strPtr(batchJobID),
	}
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{existingID: existing},
		byWf: map[string]*models.PipelineRun{"3-batch-23324": existing},
	}
	templateRepo := &mockTemplateRepo{
		byID: map[string]*models.PipelineTemplate{
			templateID: {ID: templateID, Name: "3", Version: 1, NodeCount: 3, Scope: "dev"},
		},
	}

	uc := New(templateRepo, nil, nil, nil, "cyber-databrew-dev")
	uc.SetRunRepositories(nil, runRepo, nil)

	runID, workflowName, err := uc.UpsertBatchSubtaskRun(context.Background(), BatchSubtaskRunInput{
		TemplateID:      templateID,
		TemplateVersion: 1,
		BatchJobID:      batchJobID,
		AssetID:         assetID,
		Status:          "Pending",
	})
	if err != nil {
		t.Fatalf("UpsertBatchSubtaskRun() error = %v", err)
	}
	if runID != existingID {
		t.Fatalf("runID = %q, want existing %q", runID, existingID)
	}
	if workflowName != "3-batch-23324" {
		t.Fatalf("workflowName = %q", workflowName)
	}
	if saved := runRepo.byID[existingID]; saved == nil || saved.Status != "Pending" {
		t.Fatalf("saved status = %q, want Pending", saved.Status)
	}
}

func TestUpsertBatchSubtaskRunForceNewAttemptCreatesFreshRun(t *testing.T) {
	t.Parallel()

	const (
		batchJobID = "job-1"
		assetID    = "23324"
		existingID = "0139f5b5-b612-42c6-a958-616a755c8fae"
		templateID = "tmpl-1"
	)

	existing := &models.PipelineRun{
		ID:           existingID,
		WorkflowName: "3-batch-23324",
		Status:       "Error",
		AssetIDs:     []string{assetID},
		BatchJobID:   strPtr(batchJobID),
	}
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{existingID: existing},
	}
	templateRepo := &mockTemplateRepo{
		byID: map[string]*models.PipelineTemplate{
			templateID: {ID: templateID, Name: "3", Version: 1, NodeCount: 3, Scope: "dev"},
		},
	}

	uc := New(templateRepo, nil, nil, nil, "cyber-databrew-dev")
	uc.SetRunRepositories(nil, runRepo, nil)

	runID, _, err := uc.UpsertBatchSubtaskRun(context.Background(), BatchSubtaskRunInput{
		TemplateID:      templateID,
		TemplateVersion: 1,
		BatchJobID:      batchJobID,
		AssetID:         assetID,
		Status:          "Pending",
		ForceNewAttempt: true,
	})
	if err != nil {
		t.Fatalf("UpsertBatchSubtaskRun() error = %v", err)
	}
	if runID == existingID {
		t.Fatalf("runID = existing %q, want new run id", existingID)
	}
	if runRepo.byID[existingID].Status != "Error" {
		t.Fatalf("existing run status changed to %q", runRepo.byID[existingID].Status)
	}
	if saved := runRepo.byID[runID]; saved == nil || saved.Status != "Pending" {
		t.Fatalf("new run status = %v, want Pending", saved)
	}
}

func TestRecordBatchSubtaskFailurePersistsMessageAndEvent(t *testing.T) {
	t.Parallel()

	const (
		batchJobID = "job-1"
		assetID    = "23324"
		runID      = "run-1"
		templateID = "tmpl-1"
	)

	runRepo := &mockRunRepo{byID: map[string]*models.PipelineRun{}}
	eventRepo := &mockRunEventRepo{}
	templateRepo := &mockTemplateRepo{
		byID: map[string]*models.PipelineTemplate{
			templateID: {ID: templateID, Name: "pipe", Version: 1, NodeCount: 3, Scope: "dev"},
		},
	}

	uc := New(templateRepo, nil, nil, nil, "cyber-databrew-dev")
	uc.SetRunRepositories(nil, runRepo, nil)
	uc.SetRunEventRepo(eventRepo)

	gotRunID, _, err := uc.RecordBatchSubtaskFailure(context.Background(), BatchSubtaskRunInput{
		TemplateID:      templateID,
		TemplateVersion: 1,
		BatchJobID:      batchJobID,
		AssetID:         assetID,
		RunID:           runID,
		Status:          "Failed",
		Message:         "create workflow: permission denied",
	})
	if err != nil {
		t.Fatalf("RecordBatchSubtaskFailure() error = %v", err)
	}
	if gotRunID != runID {
		t.Fatalf("runID = %q, want %q", gotRunID, runID)
	}
	saved := runRepo.byID[runID]
	if saved == nil {
		t.Fatal("expected run to be saved")
	}
	if saved.Status != "Failed" {
		t.Fatalf("status = %q, want Failed", saved.Status)
	}
	if saved.Message != "create workflow: permission denied" {
		t.Fatalf("message = %q", saved.Message)
	}
	if len(eventRepo.events) != 1 {
		t.Fatalf("events = %d, want 1", len(eventRepo.events))
	}
	if eventRepo.events[0].EventType != runEventFailed {
		t.Fatalf("event type = %q, want %q", eventRepo.events[0].EventType, runEventFailed)
	}
}

func TestUpsertBatchSubtaskRunPreservesBoundWorkflowName(t *testing.T) {
	t.Parallel()

	const (
		batchJobID = "job-1"
		assetID    = "23324"
		runID      = "run-1"
		templateID = "tmpl-1"
	)

	existing := &models.PipelineRun{
		ID:              runID,
		WorkflowName:    "3-a93367",
		ArgoWorkflowUID: "uid-1",
		Status:          "Running",
		AssetIDs:        []string{assetID},
		BatchJobID:      strPtr(batchJobID),
	}
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{runID: existing},
	}
	templateRepo := &mockTemplateRepo{
		byID: map[string]*models.PipelineTemplate{
			templateID: {ID: templateID, Name: "3", Version: 1, NodeCount: 3, Scope: "dev"},
		},
	}

	uc := New(templateRepo, nil, nil, nil, "cyber-databrew-dev")
	uc.SetRunRepositories(nil, runRepo, nil)

	_, workflowName, err := uc.UpsertBatchSubtaskRun(context.Background(), BatchSubtaskRunInput{
		TemplateID:      templateID,
		TemplateVersion: 1,
		BatchJobID:      batchJobID,
		AssetID:         assetID,
		RunID:           runID,
		Status:          "Running",
		WorkflowName:    "3-batch-23324",
	})
	if err != nil {
		t.Fatalf("UpsertBatchSubtaskRun() error = %v", err)
	}
	if workflowName != "3-a93367" {
		t.Fatalf("workflowName = %q, want bound name preserved", workflowName)
	}
	if saved := runRepo.byID[runID]; saved == nil || saved.WorkflowName != "3-a93367" {
		t.Fatalf("saved workflowName = %q", saved.WorkflowName)
	}
}

func TestCommitBatchSubtaskDeployBindsLiveWorkflow(t *testing.T) {
	t.Parallel()

	runID := "run-1"
	runRepo := &mockRunRepo{
		byID: map[string]*models.PipelineRun{
			runID: {
				ID:           runID,
				WorkflowName: "3-batch-23324",
				Status:       "Pending",
				BatchJobID:   strPtr("job-1"),
				AssetIDs:     []string{"23324"},
				CreatedAt:    time.Now().UTC(),
			},
		},
	}
	wfClient := &mockWorkflowClient{}
	wfClient.getWorkflowFn = func(_ context.Context, name, _ string) (*wfv1.Workflow, error) {
		if name != "3-a93367" {
			t.Fatalf("unexpected workflow name %q", name)
		}
		return &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: name, UID: "uid-1"},
			Status: wfv1.WorkflowStatus{
				Phase: wfv1.WorkflowRunning,
				Nodes: map[string]wfv1.NodeStatus{
					"n1": {TemplateName: "step-a", Phase: wfv1.NodeRunning, DisplayName: "step-a"},
				},
			},
		}, nil
	}

	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, &mockAssetRepo{}, wfClient, "cyber-databrew-dev")
	uc.SetRunRepositories(&mockTargetRepo{}, runRepo, &mockRunNodeRepo{})
	uc.SetObservabilityRepositories(&mockAssetNodeRepo{}, nil, nil)

	err := uc.CommitBatchSubtaskDeploy(context.Background(), runID, &models.PipelineDeployment{
		ID:           runID,
		WorkflowName: "3-a93367",
		Status:       "Running",
	})
	if err != nil {
		t.Fatalf("CommitBatchSubtaskDeploy() error = %v", err)
	}
	saved := runRepo.byID[runID]
	if saved.WorkflowName != "3-a93367" {
		t.Fatalf("workflowName = %q", saved.WorkflowName)
	}
	if saved.Message != "" {
		t.Fatalf("message = %q, want cleared", saved.Message)
	}
	if saved.Status != "Running" {
		t.Fatalf("status = %q", saved.Status)
	}
}

func strPtr(s string) *string { return &s }
