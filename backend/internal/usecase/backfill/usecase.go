package backfill

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/CyberOrigin2077/cyber-databrew/internal/batchprogress"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	"github.com/CyberOrigin2077/cyber-databrew/internal/usecase/assetvalidation"
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
)

const maxConcurrentBatchItems = 5
const backfillItemChunkSize = 500

var ErrNotFound = errors.New("backfill job not found")
var ErrInvalidRerunScope = errors.New("invalid backfill rerun scope")

// Usecase orchestrates backfill job operations.
type Usecase struct {
	repo       repository.BackfillRepository
	resultRepo repository.BackfillResultRepository
	assetRepo  repository.AssetRepository
	pipelineUC *pipelineUC.Usecase
	// pgClient enables WithTx for transactional SaveJob+SaveItems in CreateBackfill.
	pgClient any // *postgres.Client — set via NewWithPostgres
}

// New creates a Usecase without transaction support.
func New(repo repository.BackfillRepository, pipelineUC *pipelineUC.Usecase) *Usecase {
	return &Usecase{repo: repo, pipelineUC: pipelineUC}
}

// NewWithPostgres creates a Usecase with transaction support via the postgres client.
func NewWithPostgres(repo repository.BackfillRepository, pipelineUC *pipelineUC.Usecase, pgClient any) *Usecase {
	return &Usecase{repo: repo, pipelineUC: pipelineUC, pgClient: pgClient}
}

// SetResultRepositories wires staging upload dependencies.
func (uc *Usecase) SetResultRepositories(resultRepo repository.BackfillResultRepository, assetRepo repository.AssetRepository) {
	uc.resultRepo = resultRepo
	uc.assetRepo = assetRepo
}

type CreateBackfillOptions struct {
	TargetID        string
	TemplateVersion int
	PilotCount      int
	ConfigSelection *pipelineUC.RuntimeConfigSelection
}

type RerunRequest struct {
	Scope           string
	TemplateID      string
	TemplateVersion int
	ItemIDs         []string
	AssetIDs        []string
	PipelineNodeID  string
	DryRun          bool
}

type RerunResult struct {
	Status          string             `json:"status"`
	DryRun          bool               `json:"dryRun"`
	MatchedCount    int                `json:"matchedCount"`
	RetriedCount    int                `json:"retriedCount"`
	TemplateID      string             `json:"templateId"`
	TemplateVersion int                `json:"templateVersion,omitempty"`
	Skipped         []RerunSkippedItem `json:"skipped"`
}

type RerunSkippedItem struct {
	ItemID string `json:"itemId"`
	Reason string `json:"reason"`
}

// CreateBackfill creates a new backfill job and its items.
func (uc *Usecase) CreateBackfill(ctx context.Context, name, templateID string, assetIDs []string, opts ...CreateBackfillOptions) (*models.BackfillJob, error) {
	normalizedAssetIDs, err := assetvalidation.NormalizeAssetIDs("asset_ids", assetIDs)
	if err != nil {
		return nil, fmt.Errorf("validate asset_ids: %w", err)
	}
	assetIDs = normalizedAssetIDs
	if len(assetIDs) > MaxBackfillAssetCount {
		return nil, fmt.Errorf("%w: max %d assets per batch", ErrTooManyAssets, MaxBackfillAssetCount)
	}
	var options CreateBackfillOptions
	if len(opts) > 0 {
		options = opts[0]
	}
	templateVersion := options.TemplateVersion
	if templateVersion <= 0 {
		templateVersion = uc.resolveTemplateVersion(ctx, templateID)
	}
	pilotCount := options.PilotCount
	if pilotCount < 0 {
		pilotCount = 0
	}
	if pilotCount > len(assetIDs) {
		pilotCount = len(assetIDs)
	}
	pilotPhase := "none"
	status := "pending"
	if pilotCount > 0 && pilotCount < len(assetIDs) {
		pilotPhase = "running"
		status = "pilot_running"
	}

	job := &models.BackfillJob{
		ID:              uuid.New().String(),
		Name:            name,
		TemplateID:      templateID,
		TemplateVersion: templateVersion,
		TotalCount:      len(assetIDs),
		PilotCount:      pilotCount,
		PilotPhase:      pilotPhase,
		Status:          status,
		CreatedAt:       time.Now().UTC(),
	}
	filterJSON := map[string]interface{}{}
	if targetID := strings.TrimSpace(options.TargetID); targetID != "" {
		filterJSON["targetId"] = targetID
	}
	if options.ConfigSelection != nil {
		filterJSON["configSelection"] = map[string]interface{}{
			"mode":           options.ConfigSelection.Mode,
			"configId":       options.ConfigSelection.ConfigID,
			"version":        options.ConfigSelection.Version,
			"fileName":       options.ConfigSelection.FileName,
			"content":        options.ConfigSelection.Content,
			"mountPath":      options.ConfigSelection.MountPath,
			"targetFilename": options.ConfigSelection.TargetFilename,
		}
	}
	if len(filterJSON) > 0 {
		job.FilterJSON = filterJSON
	}

	assetIDsCopy := append([]string(nil), assetIDs...)

	// Try to atomically save job + items in one transaction.
	// Falls back to separate calls when pgClient is not available.
	if uc.pgClient != nil {
		if withTx, ok := uc.pgClient.(interface {
			WithTx(ctx context.Context, fn func(ctx context.Context) error) error
		}); ok {
			txCtx := ctx
			if err := withTx.WithTx(txCtx, func(txCtx context.Context) error {
				if err := uc.repo.SaveJob(txCtx, job); err != nil {
					return fmt.Errorf("save backfill job: %w", err)
				}
				for start := 0; start < len(assetIDsCopy); start += backfillItemChunkSize {
					end := start + backfillItemChunkSize
					if end > len(assetIDsCopy) {
						end = len(assetIDsCopy)
					}
					chunk := assetIDsCopy[start:end]
					items := make([]models.BackfillItem, len(chunk))
					for i, aid := range chunk {
						items[i] = models.BackfillItem{
							ID:      uuid.New().String(),
							JobID:   job.ID,
							AssetID: aid,
							Status:  "pending",
						}
					}
					if err := uc.repo.SaveItems(txCtx, items); err != nil {
						return fmt.Errorf("save backfill items: %w", err)
					}
				}
				return nil
			}); err != nil {
				return nil, err
			}
			uc.ensureBatchParentRun(txCtx, job)
			go func() {
				if err := <-uc.materializeAndRunBatch(job.ID, templateID, templateVersion, pilotCount, assetIDsCopy); err != nil {
					slog.Error("CreateBackfill: materializeAndRunBatch failed", "jobID", job.ID, "err", err)
				}
			}()
			return job, nil
		}
	}

	// Fallback: save job, then items (not atomic; used when pgClient unavailable).
	if err := uc.repo.SaveJob(ctx, job); err != nil {
		return nil, fmt.Errorf("save backfill job: %w", err)
	}
	for start := 0; start < len(assetIDsCopy); start += backfillItemChunkSize {
		end := start + backfillItemChunkSize
		if end > len(assetIDsCopy) {
			end = len(assetIDsCopy)
		}
		chunk := assetIDsCopy[start:end]
		items := make([]models.BackfillItem, len(chunk))
		for i, aid := range chunk {
			items[i] = models.BackfillItem{
				ID:      uuid.New().String(),
				JobID:   job.ID,
				AssetID: aid,
				Status:  "pending",
			}
		}
		if err := uc.repo.SaveItems(ctx, items); err != nil {
			_ = uc.repo.UpdateJobStatus(ctx, job.ID, "failed")
			return nil, fmt.Errorf("save backfill items: %w", err)
		}
	}
	uc.ensureBatchParentRun(ctx, job)
	go func() {
		if err := <-uc.materializeAndRunBatch(job.ID, templateID, templateVersion, pilotCount, assetIDsCopy); err != nil {
			slog.Error("CreateBackfill (fallback): materializeAndRunBatch failed", "jobID", job.ID, "err", err)
		}
	}()

	return job, nil
}

func (uc *Usecase) resolveTemplateVersion(ctx context.Context, templateID string) int {
	if uc.pipelineUC == nil {
		return 0
	}
	t, err := uc.pipelineUC.GetTemplate(ctx, templateID)
	if err != nil || t == nil {
		return 0
	}
	if t.ActiveVersion > 0 {
		return t.ActiveVersion
	}
	return t.Version
}

func targetIDFromBackfillJob(job *models.BackfillJob) string {
	if job == nil || job.FilterJSON == nil {
		return ""
	}
	for _, key := range []string{"targetId", "target_id"} {
		if raw, ok := job.FilterJSON[key]; ok {
			if value, ok := raw.(string); ok {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}

// materializeAndRunBatch materializes items and schedules execution.
// Returns a channel that delivers any fatal errors encountered during background execution.
// Callers must drain the returned channel to avoid goroutine leaks.
func (uc *Usecase) materializeAndRunBatch(jobID, templateID string, templateVersion, pilotCount int, assetIDs []string) <-chan error {
	errCh := make(chan error, 1)
	ctx := context.Background()
	job, err := uc.repo.FindJobByID(ctx, jobID)
	if err != nil {
		slog.Warn("materializeAndRunBatch: FindJobByID failed", "jobID", jobID, "err", err)
	}
	uc.ensureBatchParentRun(ctx, job)
	targetID := targetIDFromBackfillJob(job)
	items, err := uc.repo.FindItemsByJobID(ctx, jobID)
	if err != nil {
		slog.Error("materializeAndRunBatch: FindItemsByJobID failed", "jobID", jobID, "err", err)
		select {
		case errCh <- fmt.Errorf("find items: %w", err):
		default:
		}
		_ = uc.repo.UpdateJobStatus(ctx, jobID, "failed")
		close(errCh)
		return errCh
	}
	if len(items) == 0 && len(assetIDs) > 0 {
		slog.Warn("materializeAndRunBatch: no persisted items found", "jobID", jobID, "assetCount", len(assetIDs))
	}
	if len(items) != len(assetIDs) {
		slog.Warn("materializeAndRunBatch: persisted item count differs from submitted asset count",
			"jobID", jobID, "itemCount", len(items), "assetCount", len(assetIDs))
	}

	if uc.pipelineUC != nil {
		for i := range items {
			if items[i].PipelineRunID != nil && strings.TrimSpace(*items[i].PipelineRunID) != "" {
				continue
			}
			if strings.TrimSpace(items[i].Status) != "pending" {
				continue
			}
			runID, workflowName, err := uc.pipelineUC.UpsertBatchSubtaskRun(ctx, pipelineUC.BatchSubtaskRunInput{
				TemplateID:      templateID,
				TemplateVersion: templateVersion,
				TargetID:        targetID,
				BatchJobID:      jobID,
				AssetID:         items[i].AssetID,
				Status:          "Pending",
			})
			if err != nil {
				slog.Warn("materializeAndRunBatch: UpsertBatchSubtaskRun failed, skipping item",
					"jobID", jobID, "assetID", items[i].AssetID, "err", err)
				continue
			}
			items[i].PipelineRunID = &runID
			if wf := strings.TrimSpace(workflowName); wf != "" {
				items[i].WorkflowName = &wf
			}
			if err := uc.repo.UpdateItemPipelineRun(ctx, items[i].ID, runID, workflowName, "pending"); err != nil {
				slog.Warn("materializeAndRunBatch: UpdateItemPipelineRun failed",
					"jobID", jobID, "itemID", items[i].ID, "err", err)
			}
		}
	}

	status := "running"
	itemsToRun := items
	if pilotCount > 0 && pilotCount < len(items) {
		status = "pilot_running"
		itemsToRun = items[:pilotCount]
	}
	if err := uc.repo.UpdateJobStatus(ctx, jobID, status); err != nil {
		slog.Error("materializeAndRunBatch: UpdateJobStatus failed", "jobID", jobID, "status", status, "err", err)
		select {
		case errCh <- fmt.Errorf("update job status: %w", err):
		default:
		}
		close(errCh)
		return errCh
	}
	uc.ensureBatchParentRunByID(ctx, jobID)
	if uc.pipelineUC != nil && len(itemsToRun) > 0 {
		go func() {
			slog.Info("materializeAndRunBatch: starting runItems", "jobID", jobID, "templateID", templateID)
			uc.runItems(context.Background(), jobID, templateID, templateVersion, itemsToRun, "pending")
		}()
	}
	close(errCh)
	return errCh
}

func (uc *Usecase) runItems(ctx context.Context, jobID, templateID string, templateVersion int, items []models.BackfillItem, allowed ...string) {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, status := range allowed {
		allowedSet[status] = struct{}{}
	}

	work := make(chan models.BackfillItem)
	var wg sync.WaitGroup
	for range maxConcurrentBatchItems {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range work {
				if uc.isJobPaused(ctx, jobID) {
					continue
				}
				if err := uc.executeItem(ctx, item, templateID, templateVersion, jobID); err != nil {
					slog.Warn("runItems: executeItem failed",
						"jobID", jobID, "itemID", item.ID, "assetID", item.AssetID, "err", err)
				}
			}
		}()
	}

enqueue:
	for _, item := range items {
		if len(allowedSet) > 0 {
			if _, ok := allowedSet[item.Status]; !ok {
				continue
			}
		}
		if uc.isJobPaused(ctx, jobID) {
			break enqueue
		}
		work <- item
	}
	close(work)
	wg.Wait()
	_ = uc.syncJobProgress(ctx, jobID)
}

func (uc *Usecase) isJobPaused(ctx context.Context, jobID string) bool {
	job, err := uc.repo.FindJobByID(ctx, jobID)
	return err != nil || job == nil || job.Status == "paused"
}

// executeItem deploys the pipeline for a single backfill item and tracks status.
func (uc *Usecase) executeItem(ctx context.Context, item models.BackfillItem, templateID string, templateVersion int, jobID string) error {
	job, err := uc.repo.FindJobByID(ctx, item.JobID)
	if err != nil || job == nil || job.Status == "paused" {
		return nil
	}

	if fresh, err := uc.repo.FindItemByID(ctx, item.ID); err == nil && fresh != nil {
		item = *fresh
	}

	runID := ""
	if item.PipelineRunID != nil {
		runID = strings.TrimSpace(*item.PipelineRunID)
	}
	workflowName := ""
	if item.WorkflowName != nil {
		workflowName = strings.TrimSpace(*item.WorkflowName)
	}
	targetID := targetIDFromBackfillJob(job)

	if runID == "" && uc.pipelineUC != nil {
		var initErr error
		runID, workflowName, initErr = uc.pipelineUC.UpsertBatchSubtaskRun(ctx, pipelineUC.BatchSubtaskRunInput{
			TemplateID:      templateID,
			TemplateVersion: templateVersion,
			TargetID:        targetID,
			BatchJobID:      jobID,
			AssetID:         item.AssetID,
			Status:          "Pending",
		})
		if initErr == nil {
			_ = uc.repo.UpdateItemPipelineRun(ctx, item.ID, runID, workflowName, "pending")
		}
	}

	deployOpts := pipelineUC.DeployOptions{
		BatchJobID:         jobID,
		TemplateVersion:    templateVersion,
		TargetID:           targetID,
		AllowUnknownAssets: true,
		PreallocatedRunID:  runID,
	}
	if job != nil && job.FilterJSON != nil {
		deployOpts.TargetID = stringFromBackfillFilter(job.FilterJSON, "targetId", "target_id", "executionTargetId", "execution_target_id")
		if configRaw, ok := job.FilterJSON["configSelection"]; ok {
			if selection := decodeRuntimeConfigSelection(configRaw); selection != nil {
				deployOpts.ConfigSelection = selection
			}
		}
	}
	dep, err := uc.pipelineUC.DeployByTemplateID(
		ctx,
		templateID,
		"",
		[]string{item.AssetID},
		deployOpts,
	)
	if err != nil {
		if uc.pipelineUC != nil {
			errMsg := err.Error()
			if runID == "" {
				runID, workflowName, _ = uc.pipelineUC.RecordBatchSubtaskFailure(ctx, pipelineUC.BatchSubtaskRunInput{
					TemplateID:      templateID,
					TemplateVersion: templateVersion,
					TargetID:        targetID,
					BatchJobID:      jobID,
					AssetID:         item.AssetID,
					Status:          "Failed",
					Message:         errMsg,
					WorkflowName:    workflowName,
				})
			} else {
				_, workflowName, _ = uc.pipelineUC.RecordBatchSubtaskFailure(ctx, pipelineUC.BatchSubtaskRunInput{
					TemplateID:      templateID,
					TemplateVersion: templateVersion,
					TargetID:        targetID,
					BatchJobID:      jobID,
					AssetID:         item.AssetID,
					RunID:           runID,
					Status:          "Failed",
					Message:         errMsg,
					WorkflowName:    workflowName,
				})
			}
			if runID != "" {
				_ = uc.repo.UpdateItemPipelineRun(ctx, item.ID, runID, workflowName, "failed")
			} else {
				_ = uc.repo.UpdateItemStatus(ctx, item.ID, "failed", workflowName, errMsg)
			}
		} else {
			_ = uc.repo.UpdateItemStatus(ctx, item.ID, "failed", "", err.Error())
		}
		_ = uc.syncJobProgress(ctx, jobID)
		return err
	}

	_ = uc.repo.UpdateItemPipelineRun(ctx, item.ID, dep.ID, dep.WorkflowName, "running")
	if bindErr := uc.pipelineUC.CommitBatchSubtaskDeploy(ctx, runID, dep); bindErr != nil {
		slog.Warn("executeItem: commit batch subtask deploy failed",
			"jobID", jobID, "assetID", item.AssetID, "runID", runID, "err", bindErr)
	}
	_ = uc.syncJobProgress(ctx, jobID)
	return nil
}

func stringFromBackfillFilter(values map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		raw, ok := values[key]
		if !ok {
			continue
		}
		if value := strings.TrimSpace(fmt.Sprint(raw)); value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}

func decodeRuntimeConfigSelection(raw interface{}) *pipelineUC.RuntimeConfigSelection {
	payload, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var selection pipelineUC.RuntimeConfigSelection
	if err := json.Unmarshal(payload, &selection); err != nil {
		return nil
	}
	if strings.TrimSpace(selection.Mode) == "" {
		return nil
	}
	return &selection
}

// ReconcileSubtaskRuns ensures backfill items missing ledger rows get one.
func (uc *Usecase) ReconcileSubtaskRuns(ctx context.Context, jobID string) error {
	return uc.reconcileMissingRuns(ctx, jobID)
}

// SyncBatchView reconciles batch subtasks, refreshes the current list page from
// Argo when needed, and updates backfill job counters. Call after reading runs
// for a batch list/detail view — not for global pipeline lists.
func (uc *Usecase) SyncBatchView(ctx context.Context, jobID string, runs []models.PipelineRun) error {
	if err := uc.ReconcileSubtaskRuns(ctx, jobID); err != nil {
		return err
	}
	if uc.pipelineUC != nil {
		for i := range runs {
			uc.pipelineUC.RefreshRunForList(ctx, &runs[i])
		}
	}
	return uc.syncJobProgress(ctx, jobID)
}

// ReconcileItemByID materializes or repairs the ledger row for a single backfill item.
func (uc *Usecase) ReconcileItemByID(ctx context.Context, itemID string) (string, error) {
	item, err := uc.repo.FindItemByID(ctx, itemID)
	if err != nil {
		return "", err
	}
	if item == nil {
		item, err = uc.repo.FindItemByPipelineRunID(ctx, itemID)
		if err != nil {
			return "", err
		}
	}
	if item == nil {
		return "", ErrNotFound
	}
	itemID = item.ID
	job, err := uc.repo.FindJobByID(ctx, item.JobID)
	if err != nil {
		return "", err
	}
	if job == nil {
		return "", ErrNotFound
	}
	if err := uc.reconcileItemRun(ctx, job, *item); err != nil {
		return "", err
	}
	fresh, err := uc.repo.FindItemByID(ctx, itemID)
	if err != nil {
		return "", err
	}
	if fresh == nil || fresh.PipelineRunID == nil {
		return "", nil
	}
	return strings.TrimSpace(*fresh.PipelineRunID), nil
}

func (uc *Usecase) reconcileMissingRuns(ctx context.Context, jobID string) error {
	if uc.pipelineUC == nil {
		return nil
	}
	job, err := uc.repo.FindJobByID(ctx, jobID)
	if err != nil || job == nil {
		return err
	}
	items, err := uc.repo.FindItemsMissingPipelineRun(ctx, jobID)
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := uc.reconcileItemRun(ctx, job, item); err != nil {
			return err
		}
	}
	return nil
}

func (uc *Usecase) reconcileItemRun(ctx context.Context, job *models.BackfillJob, item models.BackfillItem) error {
	runID := ""
	if item.PipelineRunID != nil {
		runID = strings.TrimSpace(*item.PipelineRunID)
	}
	if runID != "" {
		run, err := uc.pipelineUC.GetRun(ctx, runID)
		if err == nil && run != nil {
			workflowName := strings.TrimSpace(run.WorkflowName)
			itemWorkflowName := ""
			if item.WorkflowName != nil {
				itemWorkflowName = strings.TrimSpace(*item.WorkflowName)
			}
			if workflowName != "" && (itemWorkflowName == "" ||
				(strings.Contains(itemWorkflowName, "-batch-") && !strings.Contains(workflowName, "-batch-"))) {
				_ = uc.repo.UpdateItemPipelineRun(ctx, item.ID, runID, workflowName, item.Status)
			}
			return nil
		}
	}

	workflowName := ""
	if item.WorkflowName != nil {
		workflowName = strings.TrimSpace(*item.WorkflowName)
	}
	status, message := backfillItemLedgerStatus(item)
	newRunID, newWorkflowName, err := uc.pipelineUC.UpsertBatchSubtaskRun(ctx, pipelineUC.BatchSubtaskRunInput{
		TemplateID:      job.TemplateID,
		TemplateVersion: job.TemplateVersion,
		TargetID:        targetIDFromBackfillJob(job),
		BatchJobID:      job.ID,
		AssetID:         item.AssetID,
		RunID:           runID,
		Status:          status,
		Message:         message,
		WorkflowName:    workflowName,
	})
	if err != nil {
		return err
	}
	if newWorkflowName != "" {
		workflowName = newWorkflowName
	}
	_ = uc.repo.UpdateItemPipelineRun(ctx, item.ID, newRunID, workflowName, item.Status)
	return nil
}

func backfillItemLedgerStatus(item models.BackfillItem) (status, message string) {
	switch item.Status {
	case "failed", "cancelled":
		status = "Failed"
		if item.ErrorMessage != nil {
			message = strings.TrimSpace(*item.ErrorMessage)
		}
	case "completed":
		status = "Succeeded"
	case "running":
		status = "Running"
	default:
		status = "Pending"
	}
	return status, message
}

// ListJobs returns all backfill jobs from the database.
// Listing must stay read-only: syncJobProgress (per-job DB + optional GetRun/Argo)
// belongs on GetJob, ReconcileSubtaskRuns, and background runners — not on list.
func (uc *Usecase) ListJobs(ctx context.Context) ([]models.BackfillJob, error) {
	return uc.repo.FindAllJobs(ctx)
}

// GetJob returns a backfill job without loading all items.
func (uc *Usecase) GetJob(ctx context.Context, id string) (*models.BackfillJob, error) {
	job, err := uc.repo.FindJobByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, nil
	}
	_ = uc.syncJobProgress(ctx, id)
	_ = uc.ReconcileSubtaskRuns(ctx, id)
	return uc.repo.FindJobByID(ctx, id)
}

// PauseJobOptions controls optional pause side effects.
type PauseJobOptions struct {
	StopRunning bool
}

// PauseJobResult is returned after pausing a batch job.
type PauseJobResult struct {
	Status          string `json:"status"`
	StoppedCount    int    `json:"stoppedCount,omitempty"`
	StopFailedCount int    `json:"stopFailedCount,omitempty"`
}

// PauseJob pauses a running backfill job.
func (uc *Usecase) PauseJob(ctx context.Context, id string, opts PauseJobOptions) (*PauseJobResult, error) {
	if err := uc.repo.UpdateJobStatus(ctx, id, "paused"); err != nil {
		return nil, err
	}
	result := &PauseJobResult{Status: "paused"}
	if opts.StopRunning && uc.pipelineUC != nil {
		items, err := uc.repo.FindItemsByJobIDWithStatuses(ctx, id, []string{"running"})
		if err != nil {
			return result, err
		}
		for _, item := range items {
			runID := ""
			if item.PipelineRunID != nil {
				runID = strings.TrimSpace(*item.PipelineRunID)
			}
			if runID == "" {
				continue
			}
			if err := uc.pipelineUC.StopRun(ctx, runID); err != nil {
				result.StopFailedCount++
				slog.Warn("PauseJob: stop run failed", "jobID", id, "runID", runID, "err", err)
				continue
			}
			result.StoppedCount++
		}
	}
	_ = uc.syncJobProgress(ctx, id)
	return result, nil
}

// ResumeJob resumes a paused backfill job and re-schedules pending items.
func (uc *Usecase) ResumeJob(ctx context.Context, id string) error {
	job, err := uc.repo.FindJobByID(ctx, id)
	if err != nil {
		return err
	}
	if job == nil {
		return ErrNotFound
	}
	if err := uc.repo.UpdateJobStatus(ctx, id, "running"); err != nil {
		return err
	}
	uc.ensureBatchParentRunByID(ctx, id)
	items, err := uc.repo.FindItemsByJobID(ctx, id)
	if err != nil {
		return err
	}
	if uc.pipelineUC != nil {
		go func() {
			slog.Info("ResumeJob: starting runItems", "jobID", id)
			uc.runItems(context.Background(), id, job.TemplateID, job.TemplateVersion, items, "pending")
		}()
	}
	return nil
}

// RetryFailed retries all failed items for a backfill job.
func (uc *Usecase) RetryFailed(ctx context.Context, jobID string) error {
	_, err := uc.Rerun(ctx, jobID, RerunRequest{Scope: "failed"})
	return err
}

func (uc *Usecase) Rerun(ctx context.Context, jobID string, req RerunRequest) (*RerunResult, error) {
	job, err := uc.repo.FindJobByID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, ErrNotFound
	}

	filter, err := rerunFilter(jobID, req)
	if err != nil {
		return nil, err
	}
	items, err := uc.repo.FindItemsByScope(ctx, filter)
	if err != nil {
		return nil, err
	}
	templateID := strings.TrimSpace(req.TemplateID)
	if templateID == "" {
		templateID = job.TemplateID
	}
	templateVersion := req.TemplateVersion
	if templateVersion <= 0 {
		templateVersion = job.TemplateVersion
	}
	if templateVersion <= 0 {
		templateVersion = uc.resolveTemplateVersion(ctx, templateID)
	}
	targetID := targetIDFromBackfillJob(job)
	result := &RerunResult{
		Status:          "accepted",
		DryRun:          req.DryRun,
		MatchedCount:    len(items),
		TemplateID:      templateID,
		TemplateVersion: templateVersion,
		Skipped:         []RerunSkippedItem{},
	}
	runnable := make([]models.BackfillItem, 0, len(items))
	itemIDs := make([]string, 0, len(items))
	for _, item := range items {
		if item.Status == "running" {
			result.Skipped = append(result.Skipped, RerunSkippedItem{ItemID: item.ID, Reason: "already_running"})
			continue
		}
		runnable = append(runnable, item)
		itemIDs = append(itemIDs, item.ID)
	}
	if req.DryRun {
		return result, nil
	}
	retriedCount := 0
	if err := uc.repo.PrepareItemsForRerun(ctx, itemIDs); err != nil {
		return nil, err
	}
	scheduled := make([]models.BackfillItem, 0, len(runnable))
	for i := range runnable {
		if uc.pipelineUC == nil {
			continue
		}
		runID, workflowName, err := uc.pipelineUC.UpsertBatchSubtaskRun(ctx, pipelineUC.BatchSubtaskRunInput{
			TemplateID:      templateID,
			TemplateVersion: templateVersion,
			TargetID:        targetID,
			BatchJobID:      jobID,
			AssetID:         runnable[i].AssetID,
			Status:          "Pending",
			ForceNewAttempt: true,
		})
		if err != nil {
			result.Skipped = append(result.Skipped, RerunSkippedItem{ItemID: runnable[i].ID, Reason: err.Error()})
			continue
		}
		retriedCount++
		runnable[i].PipelineRunID = &runID
		runnable[i].Status = "pending"
		runnable[i].ErrorMessage = nil
		runnable[i].StartedAt = nil
		runnable[i].FinishedAt = nil
		if wf := strings.TrimSpace(workflowName); wf != "" {
			runnable[i].WorkflowName = &wf
		}
		_ = uc.repo.UpdateItemPipelineRun(ctx, runnable[i].ID, runID, workflowName, "pending")
		scheduled = append(scheduled, runnable[i])
	}
	if len(scheduled) > 0 {
		if err := uc.repo.UpdateJobStatus(ctx, jobID, "running"); err != nil {
			return nil, err
		}
		uc.ensureBatchParentRunByID(ctx, jobID)
		if uc.pipelineUC != nil {
			go func() {
				slog.Info("Rerun: starting runItems", "jobID", jobID, "templateID", templateID)
				uc.runItems(context.Background(), jobID, templateID, templateVersion, scheduled, "pending")
			}()
		}
	}
	result.RetriedCount = retriedCount
	switch {
	case result.RetriedCount == 0:
		if len(result.Skipped) > 0 {
			result.Status = "failed"
		} else {
			result.Status = "noop"
		}
	case len(result.Skipped) == 0:
		result.Status = "succeeded"
	case result.RetriedCount < result.MatchedCount:
		result.Status = "partial_success"
	default:
		result.Status = "accepted"
	}
	return result, nil
}

func rerunFilter(jobID string, req RerunRequest) (repository.BackfillRerunItemFilter, error) {
	scope := strings.ToLower(strings.TrimSpace(req.Scope))
	if scope == "" {
		scope = "failed"
	}
	filter := repository.BackfillRerunItemFilter{JobID: jobID}
	switch scope {
	case "failed":
		filter.Statuses = []string{"failed", "cancelled"}
	case "pending":
		filter.Statuses = []string{"pending"}
	case "incomplete":
		filter.Statuses = []string{"pending", "failed", "cancelled"}
	case "completed":
		filter.Statuses = []string{"completed"}
	case "custom":
		filter.ItemIDs = req.ItemIDs
		filter.AssetIDs = req.AssetIDs
		if len(filter.ItemIDs) == 0 && len(filter.AssetIDs) == 0 {
			return filter, fmt.Errorf("%w: custom requires itemIds or assetIds", ErrInvalidRerunScope)
		}
	case "node_failed":
		filter.PipelineNodeID = strings.TrimSpace(req.PipelineNodeID)
		filter.NodeStatuses = []string{"Failed", "Error"}
		if filter.PipelineNodeID == "" {
			return filter, fmt.Errorf("%w: node_failed requires pipelineNodeId", ErrInvalidRerunScope)
		}
	default:
		return filter, fmt.Errorf("%w: %s", ErrInvalidRerunScope, scope)
	}
	return filter, nil
}

func (uc *Usecase) GetBatchNodeSummary(ctx context.Context, jobID string) (*models.BatchNodeSummary, error) {
	job, err := uc.repo.FindJobByID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, ErrNotFound
	}
	uc.refreshBatchReadModel(ctx, jobID)
	if fresh, err := uc.repo.FindJobByID(ctx, jobID); err == nil && fresh != nil {
		job = fresh
	}
	summary, err := uc.repo.SummarizeItemStatuses(ctx, jobID)
	if err != nil {
		return nil, err
	}
	aggregates, err := uc.repo.AggregateNodeStatusByBatchJobID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	runsTotal := job.TotalCount
	if runsTotal <= 0 {
		runsTotal, _ = uc.repo.CountPipelineRunsByBatchJobID(ctx, jobID)
	}
	runsWithNodeRows, _ := uc.repo.CountRunsWithNodeRowsByBatchJobID(ctx, jobID)
	nodeOrder := uc.resolveBatchNodeOrder(ctx, job.TemplateID, job.TemplateVersion)
	nodesByID := map[string]*models.BatchNodeSummaryNode{}
	order := 1
	for _, aggregate := range aggregates {
		node := nodesByID[aggregate.PipelineNodeID]
		if node == nil {
			dagOrder := order
			if configuredOrder := nodeOrder[normalizeBatchPipelineNodeID(aggregate.PipelineNodeID)]; configuredOrder > 0 {
				dagOrder = configuredOrder
			} else if len(nodeOrder) > 0 {
				dagOrder = len(nodeOrder) + order
			}
			node = &models.BatchNodeSummaryNode{
				PipelineNodeID: aggregate.PipelineNodeID,
				DisplayName:    aggregate.DisplayName,
				DagOrder:       dagOrder,
				Counts: map[string]int{
					"Pending":   0,
					"Running":   0,
					"Succeeded": 0,
					"Failed":    0,
					"Error":     0,
					"Skipped":   0,
					"Omitted":   0,
				},
			}
			nodesByID[aggregate.PipelineNodeID] = node
			order++
		}
		status := normalizeNodeStatus(aggregate.Status)
		node.Counts[status] += aggregate.Count
		node.Attempted += aggregate.Count
	}
	nodes := make([]models.BatchNodeSummaryNode, 0, len(nodesByID))
	for _, node := range nodesByID {
		if missing := job.TotalCount - node.Attempted; missing > 0 {
			node.Counts["Pending"] += missing
		}
		failures := node.Counts["Failed"] + node.Counts["Error"]
		if node.Attempted > 0 {
			node.FailureRate = float64(failures) / float64(node.Attempted)
		}
		nodes = append(nodes, *node)
	}
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].DagOrder == nodes[j].DagOrder {
			return nodes[i].PipelineNodeID < nodes[j].PipelineNodeID
		}
		return nodes[i].DagOrder < nodes[j].DagOrder
	})
	return &models.BatchNodeSummary{
		BatchJobID:      job.ID,
		TemplateID:      job.TemplateID,
		TemplateVersion: job.TemplateVersion,
		Subtasks: models.BatchNodeSubtaskCounts{
			Total:     job.TotalCount,
			Completed: summary.Completed,
			Failed:    summary.Failed,
			Running:   summary.Running,
			Pending:   summary.Pending,
			Paused:    job.Status == "paused",
		},
		Nodes: nodes,
		DataCoverage: models.BatchNodeDataCoverage{
			RunsWithNodeRows: runsWithNodeRows,
			RunsTotal:        runsTotal,
			Complete:         runsTotal == 0 || runsWithNodeRows >= runsTotal,
		},
		GeneratedAt: time.Now().UTC(),
	}, nil
}

func (uc *Usecase) resolveBatchNodeOrder(ctx context.Context, templateID string, templateVersion int) map[string]int {
	if uc == nil || uc.pipelineUC == nil || strings.TrimSpace(templateID) == "" {
		return nil
	}
	tmpl, err := uc.pipelineUC.GetTemplateVersion(ctx, templateID, templateVersion)
	if err != nil || tmpl == nil {
		return nil
	}
	return batchNodeOrderFromPipeline(tmpl.Pipeline)
}

func batchNodeOrderFromPipeline(pipeline map[string]interface{}) map[string]int {
	rawNodes, _ := pipeline["nodes"].([]interface{})
	if len(rawNodes) == 0 {
		return nil
	}
	nodeKeys := make([]string, 0, len(rawNodes))
	nodeSet := make(map[string]struct{}, len(rawNodes))
	for _, raw := range rawNodes {
		node, _ := raw.(map[string]interface{})
		id, _ := node["id"].(string)
		key := normalizeBatchPipelineNodeID(id)
		if key == "" {
			continue
		}
		if _, exists := nodeSet[key]; !exists {
			nodeSet[key] = struct{}{}
			nodeKeys = append(nodeKeys, key)
		}
	}
	if len(nodeKeys) == 0 {
		return nil
	}
	if order := batchNodeOrderFromEdges(pipeline, nodeKeys, nodeSet); len(order) > 0 {
		return order
	}
	out := make(map[string]int, len(nodeKeys))
	for i, key := range nodeKeys {
		out[key] = i + 1
	}
	return out
}

func batchNodeOrderFromEdges(pipeline map[string]interface{}, nodeKeys []string, nodeSet map[string]struct{}) map[string]int {
	rawEdges, _ := pipeline["edges"].([]interface{})
	if len(rawEdges) == 0 {
		return nil
	}
	adjacency := make(map[string][]string, len(nodeKeys))
	indegree := make(map[string]int, len(nodeKeys))
	for _, key := range nodeKeys {
		indegree[key] = 0
	}
	seenEdges := map[string]struct{}{}
	edgeCount := 0
	for _, raw := range rawEdges {
		edge, _ := raw.(map[string]interface{})
		source := normalizeBatchPipelineNodeID(edgeEndpointNodeID(fmt.Sprint(edge["source"])))
		target := normalizeBatchPipelineNodeID(edgeEndpointNodeID(fmt.Sprint(edge["target"])))
		if source == "" || target == "" || source == target {
			continue
		}
		if _, ok := nodeSet[source]; !ok {
			continue
		}
		if _, ok := nodeSet[target]; !ok {
			continue
		}
		edgeKey := source + "\x00" + target
		if _, exists := seenEdges[edgeKey]; exists {
			continue
		}
		seenEdges[edgeKey] = struct{}{}
		adjacency[source] = append(adjacency[source], target)
		indegree[target]++
		edgeCount++
	}
	if edgeCount == 0 {
		return nil
	}
	queue := make([]string, 0, len(nodeKeys))
	for _, key := range nodeKeys {
		if indegree[key] == 0 {
			queue = append(queue, key)
		}
	}
	ordered := make([]string, 0, len(nodeKeys))
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		ordered = append(ordered, current)
		for _, next := range adjacency[current] {
			indegree[next]--
			if indegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}
	if len(ordered) != len(nodeKeys) {
		return nil
	}
	out := make(map[string]int, len(ordered))
	for i, key := range ordered {
		out[key] = i + 1
	}
	return out
}

func edgeEndpointNodeID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if before, _, ok := strings.Cut(value, "."); ok {
		return before
	}
	return value
}

func normalizeBatchPipelineNodeID(id string) string {
	id = strings.ToLower(strings.TrimSpace(id))
	id = strings.TrimPrefix(id, "step-")
	id = strings.ReplaceAll(id, "-", "_")
	return id
}

func normalizeNodeStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "succeeded", "success", "completed":
		return "Succeeded"
	case "failed":
		return "Failed"
	case "error":
		return "Error"
	case "running":
		return "Running"
	case "skipped":
		return "Skipped"
	case "omitted":
		return "Omitted"
	default:
		return "Pending"
	}
}

func (uc *Usecase) ListNodeFailures(ctx context.Context, jobID string, filter repository.BatchNodeFailureFilter) (*models.BatchNodeFailureListResult, error) {
	if job, err := uc.repo.FindJobByID(ctx, jobID); err != nil {
		return nil, err
	} else if job == nil {
		return nil, ErrNotFound
	}
	uc.refreshBatchReadModel(ctx, jobID)
	filter.JobID = jobID
	return uc.repo.ListNodeFailures(ctx, filter)
}

func (uc *Usecase) refreshBatchReadModel(ctx context.Context, jobID string) {
	if uc == nil {
		return
	}
	if err := uc.SyncBatchView(ctx, jobID, nil); err != nil {
		slog.Warn("refreshBatchReadModel: sync batch view failed", "jobID", jobID, "err", err)
	}
}

func (uc *Usecase) ContinueFull(ctx context.Context, jobID string) error {
	job, err := uc.repo.FindJobByID(ctx, jobID)
	if err != nil {
		return err
	}
	if job == nil {
		return ErrNotFound
	}
	if job.PilotPhase != "review" {
		return fmt.Errorf("%w: pilot is not awaiting review", ErrInvalidRerunScope)
	}
	if err := uc.repo.UpdateJobPilotPhase(ctx, jobID, "running", "done"); err != nil {
		return err
	}
	uc.ensureBatchParentRunByID(ctx, jobID)
	items, err := uc.repo.FindItemsByJobIDWithStatuses(ctx, jobID, []string{"pending"})
	if err != nil {
		return err
	}
	if uc.pipelineUC != nil {
		go func() {
			slog.Info("ContinueFull: starting runItems", "jobID", jobID)
			uc.runItems(context.Background(), jobID, job.TemplateID, job.TemplateVersion, items, "pending")
		}()
	}
	return nil
}

func (uc *Usecase) syncJobProgress(ctx context.Context, jobID string) error {
	job, err := uc.repo.FindJobByID(ctx, jobID)
	if err != nil || job == nil {
		return err
	}

	if uc.pipelineUC != nil {
		ledgerItems, err := uc.repo.FindItemsByJobIDWithStatuses(ctx, jobID, []string{"running", "pending"})
		if err != nil {
			return err
		}
		for _, item := range ledgerItems {
			if item.PipelineRunID == nil || strings.TrimSpace(*item.PipelineRunID) == "" {
				continue
			}
			run, err := uc.pipelineUC.GetRun(ctx, *item.PipelineRunID)
			if err != nil || run == nil {
				continue
			}
			mapped := mapRunStatusToItem(run.Status)
			if mapped == "completed" {
				mapped = uc.resolveCompletionStatus(ctx, job, item)
			}
			if mapped != item.Status {
				wf := run.WorkflowName
				errMsg := ""
				if mapped == "failed" {
					errMsg = strings.TrimSpace(run.Message)
				}
				_ = uc.repo.UpdateItemStatus(ctx, item.ID, mapped, wf, errMsg)
			}
		}
	}

	summary, err := uc.repo.SummarizeItemStatuses(ctx, jobID)
	if err != nil {
		return err
	}
	latestJob, err := uc.repo.FindJobByID(ctx, jobID)
	if err != nil {
		return err
	}
	if latestJob == nil {
		return nil
	}
	if latestJob.Status == "paused" {
		return uc.updateJobProgressAndParentRun(ctx, jobID, summary.Completed, summary.Failed, "paused")
	}
	if latestJob.PilotPhase == "running" && latestJob.PilotCount > 0 {
		attemptedPilot := summary.Completed + summary.Failed
		if attemptedPilot >= latestJob.PilotCount && summary.Pending > 0 {
			if err := uc.repo.UpdateJobPilotPhase(ctx, jobID, "pilot_review", "review"); err != nil {
				return err
			}
			uc.ensureBatchParentRunByID(ctx, jobID)
			return nil
		}
	}
	jobStatus := deriveJobStatus(summary, latestJob.TotalCount)
	if latestJob.PilotPhase == "review" && jobStatus == "running" {
		jobStatus = "pilot_review"
	}
	if latestJob.PilotPhase == "running" && jobStatus == "completed" {
		_ = uc.repo.UpdateJobPilotPhase(ctx, jobID, jobStatus, "done")
		uc.ensureBatchParentRunByID(ctx, jobID)
		return nil
	}
	return uc.updateJobProgressAndParentRun(ctx, jobID, summary.Completed, summary.Failed, jobStatus)
}

func (uc *Usecase) updateJobProgressAndParentRun(ctx context.Context, jobID string, completed, failed int, status string) error {
	if err := uc.repo.UpdateJobProgress(ctx, jobID, completed, failed, status); err != nil {
		return err
	}
	uc.ensureBatchParentRunByID(ctx, jobID)
	return nil
}

func (uc *Usecase) ensureBatchParentRunByID(ctx context.Context, jobID string) {
	job, err := uc.repo.FindJobByID(ctx, jobID)
	if err != nil {
		slog.Warn("ensureBatchParentRunByID: FindJobByID failed", "jobID", jobID, "err", err)
		return
	}
	uc.ensureBatchParentRun(ctx, job)
}

func (uc *Usecase) ensureBatchParentRun(ctx context.Context, job *models.BackfillJob) {
	if uc == nil || uc.pipelineUC == nil || job == nil {
		return
	}
	if err := uc.pipelineUC.UpsertBatchParentRun(ctx, pipelineUC.BatchParentRunInput{
		ID:              job.ID,
		Name:            job.Name,
		TemplateID:      job.TemplateID,
		TemplateVersion: job.TemplateVersion,
		TargetID:        targetIDFromBackfillJob(job),
		Status:          job.Status,
		AssetCount:      job.TotalCount,
	}); err != nil {
		slog.Warn("ensureBatchParentRun: upsert parent run failed", "jobID", job.ID, "err", err)
	}
}

func deriveJobStatus(summary repository.BackfillItemStatusSummary, totalCount int) string {
	switch {
	case summary.Pending > 0 || summary.Running > 0:
		return "running"
	case summary.Failed > 0 && summary.Completed+summary.Failed == totalCount:
		return "failed"
	case summary.Completed == totalCount:
		return "completed"
	default:
		return "running"
	}
}

func mapRunStatusToItem(runStatus string) string {
	switch strings.ToLower(runStatus) {
	case "succeeded", "success":
		return "completed"
	case "failed", "error":
		return "failed"
	case "pending":
		return "pending"
	case "running":
		return "running"
	default:
		return "running"
	}
}

// GetItemAttempts returns all pipeline runs (attempts) for one logical backfill item.
func (uc *Usecase) GetItemAttempts(ctx context.Context, jobID, itemID, assetID string) (*models.BackfillItemAttemptsResult, error) {
	job, err := uc.repo.FindJobByID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, ErrNotFound
	}
	item, err := uc.resolveBackfillItem(ctx, jobID, itemID, assetID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, ErrNotFound
	}
	if uc.pipelineUC == nil {
		return &models.BackfillItemAttemptsResult{
			ItemID:   item.ID,
			AssetID:  item.AssetID,
			Attempts: []models.BackfillItemAttempt{},
		}, nil
	}
	runs, err := uc.pipelineUC.ListBatchAssetRuns(ctx, jobID, item.AssetID)
	if err != nil {
		return nil, err
	}
	currentRunID := ""
	if item.PipelineRunID != nil {
		currentRunID = strings.TrimSpace(*item.PipelineRunID)
	}
	runIDs := make([]string, 0, len(runs))
	for _, run := range runs {
		runIDs = append(runIDs, run.ID)
	}
	var nodeRows []models.PipelineRunAssetNode
	if len(runIDs) > 0 {
		nodeRows, err = uc.pipelineUC.ListAssetNodesByRunIDs(ctx, runIDs)
		if err != nil {
			return nil, err
		}
	}
	progressByRun := batchprogress.ByRunID(nodeRows, runs)
	attempts := make([]models.BackfillItemAttempt, 0, len(runs))
	for i, run := range runs {
		version := 0
		if run.TemplateVersion != nil {
			version = *run.TemplateVersion
		}
		attempts = append(attempts, models.BackfillItemAttempt{
			RunID:           run.ID,
			AttemptNo:       i + 1,
			Status:          run.Status,
			TemplateVersion: version,
			WorkflowName:    run.WorkflowName,
			Message:         strings.TrimSpace(run.Message),
			NodeProgress:    progressByRun[run.ID],
			IsCurrent:       run.ID == currentRunID,
			StartedAt:       run.StartedAt,
			FinishedAt:      run.FinishedAt,
			CreatedAt:       run.CreatedAt,
		})
	}
	sort.Slice(attempts, func(i, j int) bool {
		return attempts[i].CreatedAt.After(attempts[j].CreatedAt)
	})
	for i := range attempts {
		attempts[i].AttemptNo = len(attempts) - i
	}
	return &models.BackfillItemAttemptsResult{
		ItemID:       item.ID,
		AssetID:      item.AssetID,
		CurrentRunID: currentRunID,
		Attempts:     attempts,
	}, nil
}

// ValidateAssets checks which submitted asset IDs exist in the catalog.
func (uc *Usecase) ValidateAssets(ctx context.Context, assetIDs []string) (*models.ValidateBackfillAssetsResult, error) {
	normalized, err := assetvalidation.NormalizeAssetIDs("assetIds", assetIDs)
	if err != nil {
		return nil, err
	}
	result := &models.ValidateBackfillAssetsResult{
		Registered: []string{},
		Unknown:    []string{},
	}
	if len(normalized) == 0 {
		return result, nil
	}
	if uc.assetRepo == nil {
		result.Unknown = append(result.Unknown, normalized...)
		return result, nil
	}
	existing, err := uc.assetRepo.FindExistingIDs(ctx, normalized)
	if err != nil {
		return nil, err
	}
	for _, id := range normalized {
		if _, ok := existing[id]; ok {
			result.Registered = append(result.Registered, id)
		} else {
			result.Unknown = append(result.Unknown, id)
		}
	}
	return result, nil
}

func (uc *Usecase) resolveBackfillItem(ctx context.Context, jobID, itemID, assetID string) (*models.BackfillItem, error) {
	itemID = strings.TrimSpace(itemID)
	assetID = strings.TrimSpace(assetID)
	if itemID != "" {
		item, err := uc.repo.FindItemByID(ctx, itemID)
		if err != nil {
			return nil, err
		}
		if item == nil || item.JobID != jobID {
			return nil, ErrNotFound
		}
		return item, nil
	}
	item, err := uc.repo.FindItemByJobAndAssetID(ctx, jobID, assetID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, ErrNotFound
	}
	return item, nil
}
