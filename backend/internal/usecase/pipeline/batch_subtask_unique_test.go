package pipeline

import (
	"context"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

func TestUpsertBatchSubtaskRunForceNewAttemptUsesRunUUIDWorkflowName(t *testing.T) {
	templateRepo := &mockTemplateRepo{
		byID: map[string]*models.PipelineTemplate{
			"tpl-1": {ID: "tpl-1", Name: "tpl", Version: 2, NodeCount: 3, Scope: "dev"},
		},
	}
	runRepo := &mockRunRepo{}
	uc := New(templateRepo, nil, nil, nil, "cyber-databrew-dev")
	uc.SetRunRepositories(nil, runRepo, nil)

	runID, workflowName, err := uc.UpsertBatchSubtaskRun(context.Background(), BatchSubtaskRunInput{
		TemplateID:      "tpl-1",
		TemplateVersion: 2,
		BatchJobID:      "job-1",
		AssetID:         "asset-1",
		ForceNewAttempt: true,
		Status:          "Pending",
	})
	if err != nil {
		t.Fatalf("UpsertBatchSubtaskRun() error = %v", err)
	}
	if workflowName != runID {
		t.Fatalf("workflowName = %q, want run UUID %q", workflowName, runID)
	}
	if saved := runRepo.byID[runID]; saved == nil || saved.WorkflowName != workflowName {
		t.Fatalf("saved run workflowName mismatch: %+v", saved)
	}
}
