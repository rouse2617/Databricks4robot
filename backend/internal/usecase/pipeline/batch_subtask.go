package pipeline

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// BatchSubtaskRunInput describes a batch subtask ledger row that should look
// like any other pipeline run in list/detail views.
type BatchSubtaskRunInput struct {
	TemplateID      string
	TemplateVersion int
	BatchJobID      string
	AssetID         string
	RunID           string
	Status          string
	Message         string
	WorkflowName    string
}

// UpsertBatchSubtaskRun creates or updates a first-class pipeline run for a
// batch subtask so batch detail lists behave like normal execution records.
func (uc *Usecase) UpsertBatchSubtaskRun(ctx context.Context, in BatchSubtaskRunInput) (string, string, error) {
	if uc.runRepo == nil {
		return "", "", fmt.Errorf("pipeline run repository is not configured")
	}
	templateID := strings.TrimSpace(in.TemplateID)
	batchJobID := strings.TrimSpace(in.BatchJobID)
	assetID := strings.TrimSpace(in.AssetID)
	if templateID == "" || batchJobID == "" || assetID == "" {
		return "", "", fmt.Errorf("%w: templateId, batchJobId and assetId are required", ErrInvalidArgument)
	}

	t, err := uc.templateRepo.FindByID(ctx, templateID)
	if err != nil {
		return "", "", err
	}
	if t == nil {
		return "", "", ErrTemplateNotFound
	}
	if in.TemplateVersion > 0 && in.TemplateVersion != t.Version {
		versioned, err := uc.templateRepo.FindByNameAndVersion(ctx, t.Name, in.TemplateVersion)
		if err != nil {
			return "", "", err
		}
		if versioned == nil {
			return "", "", ErrTemplateNotFound
		}
		t = versioned
		templateID = t.ID
	}

	runID := strings.TrimSpace(in.RunID)
	if runID == "" {
		if existing, err := uc.runRepo.FindByBatchJobAndAssetID(ctx, batchJobID, assetID); err != nil {
			return "", "", err
		} else if existing != nil {
			runID = existing.ID
		}
	}
	if runID == "" {
		runID = uuid.New().String()
	}

	status := strings.TrimSpace(in.Status)
	if status == "" {
		status = "Pending"
	}

	workflowName := strings.TrimSpace(in.WorkflowName)
	if workflowName == "" {
		workflowName = batchSubtaskWorkflowName(t.Name, assetID, runID)
	}

	target, err := uc.resolveExecutionTarget(ctx, "")
	if err != nil {
		return "", "", err
	}

	now := time.Now().UTC()
	templateIDCopy := templateID
	var templateVersion *int
	if t.Version > 0 {
		v := t.Version
		templateVersion = &v
	}

	var finishedAt *time.Time
	if isTerminalBatchSubtaskStatus(status) {
		finishedAt = &now
	}

	run := &models.PipelineRun{
		ID:                runID,
		TemplateID:        &templateIDCopy,
		TemplateVersion:   templateVersion,
		PipelineName:      t.Name,
		WorkflowName:      workflowName,
		ExecutionTargetID: target.ID,
		TargetSnapshot:    executionTargetSnapshot(target),
		Status:            status,
		NodeCount:         t.NodeCount,
		AssetIDs:          []string{assetID},
		AssetCount:        1,
		ArgoNamespace:     target.Namespace,
		ExecutionTarget:   target,
		Scope:             t.Scope,
		BatchJobID:        &batchJobID,
		Message:           strings.TrimSpace(in.Message),
		CreatedAt:         now,
		UpdatedAt:         now,
		FinishedAt:        finishedAt,
	}
	if run.ArgoNamespace == "" {
		run.ArgoNamespace = uc.namespace
	}

	if existing, err := uc.runRepo.FindByID(ctx, runID); err == nil && existing != nil {
		run.CreatedAt = existing.CreatedAt
		switch strings.ToLower(strings.TrimSpace(status)) {
		case "pending":
			run.StartedAt = nil
			run.FinishedAt = nil
		case "running":
			run.FinishedAt = nil
			if existing.StartedAt != nil {
				run.StartedAt = existing.StartedAt
			}
		default:
			if existing.StartedAt != nil {
				run.StartedAt = existing.StartedAt
			}
		}
	}

	if err := uc.runRepo.Save(ctx, run); err != nil {
		return "", "", err
	}
	return runID, workflowName, nil
}

func isBatchSubtaskPlaceholderWorkflowName(name string) bool {
	return strings.Contains(strings.TrimSpace(name), "-batch-")
}

func batchSubtaskWorkflowName(pipelineName, assetID, runID string) string {
	suffix := strings.TrimSpace(assetID)
	if len(suffix) > 12 {
		suffix = suffix[len(suffix)-12:]
	}
	if suffix == "" {
		suffix = runID[:6]
	}
	return fmt.Sprintf("%s-batch-%s", pipelineName, suffix)
}

func isTerminalBatchSubtaskStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "failed", "error", "succeeded", "success":
		return true
	default:
		return false
	}
}
