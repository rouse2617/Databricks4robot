package pipeline

import (
	"context"
	"testing"

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

func strPtr(s string) *string { return &s }
