package pipeline

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	runstate "github.com/CyberOrigin2077/cyber-databrew/internal/runtimeos/state"
)

// BatchParentRunInput describes the inspectable parent Run for a batch/backfill
// job. It does not create a runtime workflow; child Runs remain associated via
// their BatchJobID field.
type BatchParentRunInput struct {
	ID              string
	Name            string
	TemplateID      string
	TemplateVersion int
	TargetID        string
	Status          string
	Message         string
	AssetCount      int
}

// BatchSubtaskRunInput describes a batch subtask ledger row that should look
// like any other pipeline run in list/detail views.
type BatchSubtaskRunInput struct {
	TemplateID      string
	TemplateVersion int
	TargetID        string
	BatchJobID      string
	AssetID         string
	RunID           string
	Status          string
	Message         string
	WorkflowName    string
	// ForceNewAttempt creates a fresh pipeline run instead of reusing the latest
	// batch+asset run (used when rerunning a backfill subtask).
	ForceNewAttempt bool
}

// UpsertBatchParentRun creates or updates the parent Run record for a batch job
// so the batch can be inspected through /runs/:id and /runs/:id/children.
func (uc *Usecase) UpsertBatchParentRun(ctx context.Context, in BatchParentRunInput) error {
	if uc.runRepo == nil {
		return fmt.Errorf("pipeline run repository is not configured")
	}
	runID := strings.TrimSpace(in.ID)
	templateID := strings.TrimSpace(in.TemplateID)
	if runID == "" || templateID == "" {
		return fmt.Errorf("%w: id and templateId are required", ErrInvalidArgument)
	}

	t, err := uc.templateRepo.FindByID(ctx, templateID)
	if err != nil {
		return err
	}
	if t == nil {
		return ErrTemplateNotFound
	}
	if in.TemplateVersion > 0 && in.TemplateVersion != t.Version {
		versioned, err := uc.templateRepo.FindByNameAndVersion(ctx, t.Name, in.TemplateVersion)
		if err != nil {
			return err
		}
		if versioned == nil {
			return ErrTemplateNotFound
		}
		t = versioned
		templateID = t.ID
	}

	target, err := uc.resolveExecutionTarget(ctx, in.TargetID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	templateIDCopy := templateID
	var templateVersion *int
	if t.Version > 0 {
		v := t.Version
		templateVersion = &v
	}
	status := runstate.NormalizeRunStatus(in.Status)
	startedAt, finishedAt := batchParentRunTimestamps(status, now, nil)
	pipelineName := strings.TrimSpace(in.Name)
	if pipelineName == "" {
		pipelineName = t.Name
	}

	run := &models.PipelineRun{
		ID:                runID,
		TemplateID:        &templateIDCopy,
		TemplateVersion:   templateVersion,
		PipelineName:      pipelineName,
		ExecutionTargetID: target.ID,
		TargetSnapshot:    executionTargetSnapshot(target),
		Status:            status,
		NodeCount:         t.NodeCount,
		AssetCount:        in.AssetCount,
		NoAssetRun:        true,
		PipelineJSON:      t.Pipeline,
		ArgoNamespace:     target.Namespace,
		ExecutionTarget:   target,
		Scope:             t.Scope,
		Message:           strings.TrimSpace(in.Message),
		CreatedAt:         now,
		UpdatedAt:         now,
		StartedAt:         startedAt,
		FinishedAt:        finishedAt,
	}
	if run.ArgoNamespace == "" {
		run.ArgoNamespace = uc.namespace
	}

	if existing, err := uc.runRepo.FindByID(ctx, runID); err == nil && existing != nil {
		run.CreatedAt = existing.CreatedAt
		if strings.TrimSpace(existing.WorkflowName) != "" {
			run.WorkflowName = existing.WorkflowName
		}
		if strings.TrimSpace(existing.ArgoWorkflowUID) != "" {
			run.ArgoWorkflowUID = existing.ArgoWorkflowUID
		}
		if strings.TrimSpace(in.Message) == "" {
			run.Message = existing.Message
		}
		run.StartedAt, run.FinishedAt = batchParentRunTimestamps(status, now, existing)
	} else if err != nil {
		return err
	}

	return uc.runRepo.Save(ctx, run)
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
	if runID == "" && !in.ForceNewAttempt {
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
		workflowName = batchSubtaskWorkflowName(t.Name, assetID, runID, in.ForceNewAttempt)
	}

	target, err := uc.resolveExecutionTarget(ctx, in.TargetID)
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
		if shouldPreserveBatchSubtaskWorkflowName(existing, workflowName) {
			run.WorkflowName = existing.WorkflowName
			workflowName = existing.WorkflowName
		}
		if uid := strings.TrimSpace(existing.ArgoWorkflowUID); uid != "" {
			run.ArgoWorkflowUID = uid
		}
		if strings.TrimSpace(in.Message) == "" && isStaleWorkflowUnavailableMessage(existing.Message) {
			run.Message = ""
		}
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

func shouldPreserveBatchSubtaskWorkflowName(existing *models.PipelineRun, incoming string) bool {
	if existing == nil {
		return false
	}
	if strings.TrimSpace(existing.ArgoWorkflowUID) != "" {
		return true
	}
	existingName := strings.TrimSpace(existing.WorkflowName)
	incomingName := strings.TrimSpace(incoming)
	if existingName == "" {
		return false
	}
	if !isBatchSubtaskPlaceholderWorkflowName(existingName) {
		return isBatchSubtaskPlaceholderWorkflowName(incomingName) || incomingName == "" || incomingName != existingName
	}
	return false
}

// CommitBatchSubtaskDeploy binds a preallocated batch ledger row to the live
// Argo workflow created by DeployByTemplateID.
func (uc *Usecase) CommitBatchSubtaskDeploy(ctx context.Context, runID string, dep *models.PipelineDeployment) error {
	if uc.runRepo == nil || dep == nil {
		return nil
	}
	runID = strings.TrimSpace(runID)
	if runID == "" {
		runID = strings.TrimSpace(dep.ID)
	}
	if runID == "" {
		return fmt.Errorf("batch subtask run id is required")
	}
	existing, err := uc.runRepo.FindByID(ctx, runID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrDeploymentNotFound
	}
	wfName := strings.TrimSpace(dep.WorkflowName)
	if wfName == "" {
		return fmt.Errorf("deployment workflow name is required")
	}
	existing.WorkflowName = wfName
	existing.Status = strings.TrimSpace(dep.Status)
	if existing.Status == "" {
		existing.Status = "Pending"
	}
	existing.Message = ""
	existing.FinishedAt = nil
	if existing.StartedAt == nil {
		now := time.Now().UTC()
		existing.StartedAt = &now
	}
	if err := uc.runRepo.Save(ctx, existing); err != nil {
		return err
	}
	uc.refreshRunStatus(ctx, existing)
	return nil
}

// RecordBatchSubtaskFailure persists a preallocated batch subtask failure with
// a durable message and event. This covers failures before an Argo workflow is
// created, where normal watcher events will never arrive.
func (uc *Usecase) RecordBatchSubtaskFailure(ctx context.Context, in BatchSubtaskRunInput) (string, string, error) {
	if strings.TrimSpace(in.Status) == "" {
		in.Status = "Failed"
	}
	runID, workflowName, err := uc.UpsertBatchSubtaskRun(ctx, in)
	if err != nil {
		return "", "", err
	}
	if uc.runRepo == nil {
		return runID, workflowName, nil
	}
	run, err := uc.runRepo.FindByID(ctx, runID)
	if err != nil || run == nil {
		return runID, workflowName, err
	}
	if strings.TrimSpace(in.Message) != "" && strings.TrimSpace(run.Message) == "" {
		run.Message = strings.TrimSpace(in.Message)
		_ = uc.runRepo.Save(ctx, run)
	}
	uc.appendRunEvent(ctx, run, models.PipelineRunEvent{
		EventType:      runEventFailed,
		SubjectType:    "run",
		SubjectID:      run.ID,
		Status:         run.Status,
		Message:        firstNonEmpty(run.Message, "batch subtask failed before workflow creation"),
		Reason:         strings.TrimSpace(in.Message),
		IdempotencyKey: fmt.Sprintf("batch_subtask_failed:%s", run.ID),
		Payload: map[string]interface{}{
			"batchJobId": in.BatchJobID,
			"assetId":    in.AssetID,
		},
	})
	return runID, workflowName, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func batchSubtaskWorkflowName(pipelineName, assetID, runID string, unique bool) string {
	suffix := strings.TrimSpace(assetID)
	if len(suffix) > 12 {
		suffix = suffix[len(suffix)-12:]
	}
	if suffix == "" {
		suffix = runID[:6]
	}
	if unique {
		runSuffix := strings.TrimSpace(runID)
		if len(runSuffix) > 8 {
			runSuffix = runSuffix[:8]
		}
		if runSuffix != "" {
			return fmt.Sprintf("%s-batch-%s-%s", pipelineName, suffix, runSuffix)
		}
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

func batchParentRunTimestamps(status string, now time.Time, existing *models.PipelineRun) (*time.Time, *time.Time) {
	var startedAt *time.Time
	var finishedAt *time.Time
	if existing != nil {
		startedAt = existing.StartedAt
		finishedAt = existing.FinishedAt
	}
	switch {
	case runstate.IsActiveStatus(status):
		finishedAt = nil
		if status == runstate.StatusRunning || status == runstate.StatusSuspended {
			if startedAt == nil {
				startedAt = &now
			}
		}
	case runstate.IsTerminalStatus(status):
		if startedAt == nil {
			startedAt = &now
		}
		if finishedAt == nil {
			finishedAt = &now
		}
	}
	return startedAt, finishedAt
}
