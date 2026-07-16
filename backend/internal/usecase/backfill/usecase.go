package backfill

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/CyberOrigin2077/cyber-databrew/internal/batchprogress"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	"github.com/CyberOrigin2077/cyber-databrew/internal/transpiler"
	"github.com/CyberOrigin2077/cyber-databrew/internal/usecase/assetvalidation"
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
)

// maxConcurrentBatchItems controls how many goroutines submit batch items
// to Argo in parallel. Set via BACKFILL_CONCURRENCY env var (default 5).
var maxConcurrentBatchItems = func() int {
	if v := os.Getenv("BACKFILL_CONCURRENCY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 5
}()

// perJobSubmitBatch caps how many pending items one job dispatches per submit
// cycle (submitterInterval, 15s). It is the per-job dispatch ceiling:
// perJobSubmitBatch * (60/15) items/min. Raised from the original hard-coded 32
// and made tunable (BACKFILL_SUBMIT_BATCH) because — once Argo parallelism was
// lifted — DISPATCH, not execution, was the throughput bottleneck (CYB-3491
// load test measured dispatch pinned at exactly 32*4 = 128/min). 128 feeds a
// parallelism=200 pool comfortably; maxConcurrentBatchItems still bounds the
// in-flight concurrency of each cycle's submits.
var perJobSubmitBatch = func() int {
	if v := os.Getenv("BACKFILL_SUBMIT_BATCH"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 128
}()

// deployTimeout caps how long a single executeItem call may take
// before the worker gives up. Without this, a hanging Argo API call
// holds the worker goroutine forever, blocking wg.Wait() and
// preventing the batch job from ever reaching a terminal state.
const deployTimeout = 60 * time.Second
const backfillItemChunkSize = 500

var ErrNotFound = errors.New("backfill job not found")
var ErrInvalidRerunScope = errors.New("invalid backfill rerun scope")

// Usecase orchestrates backfill job operations.
// maxFailedReconcilePerSync bounds how many already-failed items a single
// syncJobProgress pass re-reconciles against runtime truth. Failed items are
// otherwise never re-examined (the reconcile working set is running/pending
// only), so a transient/misclassified failure stays stuck forever. Bounded so
// large batches converge over successive syncs without flooding the Argo API.
const maxFailedReconcilePerSync = 50

type Usecase struct {
	repo       repository.BackfillRepository
	resultRepo repository.BackfillResultRepository
	assetRepo  repository.AssetRepository
	pipelineUC *pipelineUC.Usecase
	pgClient   any // *postgres.Client — set via NewWithPostgres

	lastSync   map[string]time.Time
	lastSyncMu sync.Mutex

	reconcileStop chan struct{}
	reconcileWg   sync.WaitGroup

	// CYB-3491 submitter: the periodic, idempotent, resumable dispatch loop
	// that replaced the claim/reaper/worker-pool execution queue. submitQueue
	// is the persistence surface (nil disables the submitter, e.g. in tests
	// that don't exercise dispatch); submitKick lets materialize/resume/rerun
	// trigger an immediate cycle instead of waiting for the ticker.
	submitQueue repository.SubmitQueue
	deployer    subtaskDeployer
	submitStop  chan struct{}
	submitWg    sync.WaitGroup
	submitKick  chan struct{}

	// Batch job completion Feishu notification (CYB-3071). notifier nil
	// disables the feature entirely (no claim, no send).
	notifier        Notifier
	frontendBaseURL string
}

// Notifier is the minimal capability this package needs to deliver a
// completion message. It is declared here (not imported from a notification
// provider package) so backfill has no compile-time dependency on any
// specific delivery mechanism; *feishu.Client satisfies this structurally.
type Notifier interface {
	SendText(ctx context.Context, text string) error
}

// SetNotifier configures the sender used for batch-job completion
// notifications and the base URL used to build the job detail link in the
// message. Passing a nil sender disables the feature (matches the zero-value
// default, so this call is optional).
func (uc *Usecase) SetNotifier(sender Notifier, frontendBaseURL string) {
	uc.notifier = sender
	uc.frontendBaseURL = strings.TrimSpace(frontendBaseURL)
}

// New creates a Usecase without transaction support.
func New(repo repository.BackfillRepository, pipelineUC *pipelineUC.Usecase) *Usecase {
	uc := &Usecase{repo: repo, pipelineUC: pipelineUC}
	if pipelineUC != nil {
		uc.deployer = pipelineUC
	}
	return uc
}

// NewWithPostgres creates a Usecase with transaction support via the postgres client.
func NewWithPostgres(repo repository.BackfillRepository, pipelineUC *pipelineUC.Usecase, pgClient any) *Usecase {
	uc := New(repo, pipelineUC)
	uc.pgClient = pgClient
	return uc
}

// SyncJob force-syncs a batch job's progress from its child runs, flipping it to
// a terminal status and sending the once-only completion notification if all
// children have finished. Safe to call repeatedly (notification is claimed
// atomically). Used by the run-status webhook cascade (CYB-3078, fast path).
func (uc *Usecase) SyncJob(ctx context.Context, jobID string) error {
	return uc.syncJobProgressForce(ctx, jobID)
}

// StartJobReconciler launches the reconcile backstop (CYB-3078): every interval
// it force-syncs each non-terminal batch job, so a job whose children finished
// in the background is finalized + notified without a user opening its page and
// without depending on the exit hook firing. Runs until StopJobReconciler.
func (uc *Usecase) StartJobReconciler(interval time.Duration, scanLimit int) {
	if interval <= 0 {
		interval = 60 * time.Second
	}
	if scanLimit <= 0 {
		scanLimit = 200
	}
	uc.reconcileStop = make(chan struct{})
	uc.reconcileWg.Add(1)
	go func() {
		defer uc.reconcileWg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-uc.reconcileStop:
				return
			case <-ticker.C:
				uc.reconcileActiveJobs(context.Background(), scanLimit)
			}
		}
	}()
}

// StopJobReconciler signals the reconcile goroutine to stop and waits for it.
func (uc *Usecase) StopJobReconciler() {
	if uc.reconcileStop != nil {
		close(uc.reconcileStop)
	}
	uc.reconcileWg.Wait()
}

// reconcileActiveJobs force-syncs every non-terminal batch job once. Best-effort:
// a per-job failure is logged and does not stop the sweep.
//
// CYB-3490: this loop is the single background home for batch-ledger
// convergence — missing-run repair (formerly done on GetJob/node-summary
// reads) plus counter/settle sync. Reads never do this work anymore.
func (uc *Usecase) reconcileActiveJobs(ctx context.Context, scanLimit int) {
	jobs, err := uc.repo.FindActiveJobs(ctx, scanLimit)
	if err != nil {
		slog.Warn("job reconciler: find active jobs failed", "err", err)
		return
	}
	for _, job := range jobs {
		if err := uc.reconcileMissingRuns(ctx, job.ID); err != nil {
			slog.Warn("job reconciler: missing-run repair failed", "jobID", job.ID, "err", err)
		}
		if err := uc.syncJobProgressForce(ctx, job.ID); err != nil {
			slog.Warn("job reconciler: sync failed", "jobID", job.ID, "err", err)
		}
	}
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
	Owner           string
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
		CreatedBy:       options.Owner,
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
	// CYB-3491: dispatch is owned by the periodic submitter — kick it for an
	// immediate cycle instead of spawning a per-job worker pool. Pilot gating
	// (itemsToRun) is enforced by the submitter's per-job quota.
	_ = itemsToRun
	uc.KickSubmitter()
	close(errCh)
	return errCh
}

func (uc *Usecase) isJobPaused(ctx context.Context, jobID string) bool {
	job, err := uc.repo.FindJobByID(ctx, jobID)
	return err != nil || job == nil || job.Status == "paused"
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
//
// CYB-3490: pure ledger read. Reconcile/sync moved off the request path —
// the webhook cascade (SyncJob) plus the background job reconciler own
// convergence; a read must not fan out to Argo (latency scaled with failed
// count, measured 12.7s at failed=76).
func (uc *Usecase) GetJob(ctx context.Context, id string) (*models.BackfillJob, error) {
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
		items, err := uc.repo.FindItemsByJobIDWithStatuses(ctx, id, []string{"running", "submitted"})
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
			// Stopping removes the workflow from Argo, so the item is no longer
			// running. Reset it to "pending" and drop its run link so a later
			// resume re-submits it cleanly. Leaving it "running" with a stale run
			// both hides it from ResumeJob (ClaimNextItem only claims "pending")
			// and trips executeItem's dedup guard (the run still looks submitted)
			// — that combination is what stranded items after pause→resume.
			if err := uc.repo.UpdateItemPipelineRun(ctx, item.ID, "", "", "pending"); err != nil {
				slog.Warn("PauseJob: reset stopped item to pending failed", "jobID", id, "itemID", item.ID, "err", err)
			}
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
	// CYB-3491: the periodic submitter owns dispatch; kick it so resume takes
	// effect immediately.
	uc.KickSubmitter()
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
		if item.Status == "running" || item.Status == "submitted" {
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
		// CYB-3491: rerun resets items to pending; the submitter dispatches.
		uc.KickSubmitter()
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
	// CYB-3490: pure ledger read — no reconcile on the request path. The
	// background reconciler + webhook cascade keep the ledger converged.
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
		nodes = append(nodes, *node)
	}
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].DagOrder == nodes[j].DagOrder {
			return nodes[i].PipelineNodeID < nodes[j].PipelineNodeID
		}
		return nodes[i].DagOrder < nodes[j].DagOrder
	})
	// Fill the runs that have no projected asset-node row yet for a node.
	// CYB-3491: node-level phase is only near-real-time (workflow-level
	// projection), so an in-flight run usually has NO asset-node rows at all.
	// Attributing every such run to Pending made the node overview report
	// "节点运行中=0 / 节点排队=N" while the subtask list showed N running — a
	// self-contradiction the moment a batch was mid-flight. Split the fill by
	// subtask truth instead: runs that never started (item 'pending') are
	// queued; the unprojected in-flight runs are running, and their execution
	// frontier is the first (lowest DagOrder) node with a gap — attribute them
	// there so the node overview and the subtask "运行中" count agree.
	notStarted := summary.Pending
	inFlightRemaining := summary.Running
	frontierTaken := false
	for i := range nodes {
		if missing := job.TotalCount - nodes[i].Attempted; missing > 0 {
			pendingFill := missing
			if !frontierTaken {
				runningFill := missing - notStarted
				if runningFill < 0 {
					runningFill = 0
				}
				if runningFill > inFlightRemaining {
					runningFill = inFlightRemaining
				}
				nodes[i].Counts["Running"] += runningFill
				inFlightRemaining -= runningFill
				pendingFill = missing - runningFill
				frontierTaken = true
			}
			nodes[i].Counts["Pending"] += pendingFill
		}
		failures := nodes[i].Counts["Failed"] + nodes[i].Counts["Error"]
		if nodes[i].Attempted > 0 {
			nodes[i].FailureRate = float64(failures) / float64(nodes[i].Attempted)
		}
	}
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
	type nodeMeta struct{ id, component string }
	nodeKeys := make([]string, 0, len(rawNodes))
	nodeSet := make(map[string]struct{}, len(rawNodes))
	metaByKey := make(map[string]nodeMeta, len(rawNodes))
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
			metaByKey[key] = nodeMeta{id: id, component: batchNodeComponentName(node)}
		}
	}
	if len(nodeKeys) == 0 {
		return nil
	}
	var out map[string]int
	if order := batchNodeOrderFromEdges(pipeline, nodeKeys, nodeSet); len(order) > 0 {
		out = order
	} else {
		out = make(map[string]int, len(nodeKeys))
		for i, key := range nodeKeys {
			out[key] = i + 1
		}
	}
	// Alias the readable template names (step-<component>[-<uuid8>]) to the same
	// order so node-summary ordering works whether a node was stored under the
	// legacy step-node-<uuid> name or the readable name (CYB-3076). No migration.
	for key, meta := range metaByKey {
		order, ok := out[key]
		if !ok {
			continue
		}
		for _, cand := range transpiler.StepCandidateKeys(meta.component, meta.id) {
			nk := normalizeBatchPipelineNodeID(cand)
			if nk == "" {
				continue
			}
			if _, exists := out[nk]; !exists {
				out[nk] = order
			}
		}
	}
	return out
}

func batchNodeComponentName(node map[string]interface{}) string {
	comp, _ := node["component"].(map[string]interface{})
	if comp == nil {
		return ""
	}
	name, _ := comp["name"].(string)
	return name
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
	// CYB-3490: pure ledger read — background reconciler owns convergence.
	filter.JobID = jobID
	return uc.repo.ListNodeFailures(ctx, filter)
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
	// CYB-3491: pilot released to full — remaining pending items are picked
	// up by the submitter (its pilot quota gate no longer applies once the
	// job status is back to running).
	uc.KickSubmitter()
	return nil
}

// syncProgressMinInterval throttles non-forced syncJobProgress calls (e.g.
// per-item post-execution sync) so overlapping background triggers don't
// stack full syncs. Since CYB-3490, reads never trigger sync — callers are
// the webhook cascade (SyncJob, forced) and the job reconciler (forced).
const syncProgressMinInterval = 30 * time.Second

func (uc *Usecase) shouldSyncProgress(jobID string) bool {
	uc.lastSyncMu.Lock()
	defer uc.lastSyncMu.Unlock()
	if uc.lastSync == nil {
		return true
	}
	last, ok := uc.lastSync[jobID]
	if !ok {
		return true
	}
	return time.Since(last) >= syncProgressMinInterval
}

func (uc *Usecase) markSyncProgressDone(jobID string) {
	uc.lastSyncMu.Lock()
	defer uc.lastSyncMu.Unlock()
	if uc.lastSync == nil {
		uc.lastSync = make(map[string]time.Time)
	}
	uc.lastSync[jobID] = time.Now()
}

func (uc *Usecase) syncJobProgress(ctx context.Context, jobID string) error {
	return uc.syncJobProgressInternal(ctx, jobID, false)
}

func (uc *Usecase) syncJobProgressForce(ctx context.Context, jobID string) error {
	return uc.syncJobProgressInternal(ctx, jobID, true)
}

func (uc *Usecase) syncJobProgressInternal(ctx context.Context, jobID string, force bool) error {
	if !force && !uc.shouldSyncProgress(jobID) {
		return nil
	}
	uc.markSyncProgressDone(jobID)
	job, err := uc.repo.FindJobByID(ctx, jobID)
	if err != nil || job == nil {
		return err
	}

	if uc.pipelineUC != nil {
		ledgerItems, err := uc.repo.FindItemsByJobIDWithStatuses(ctx, jobID, []string{"running", "pending", "submitted"})
		if err != nil {
			return err
		}
		if len(ledgerItems) > 0 {
			// Batch-fetch all runs for this batch job instead of N individual GetRun calls.
			runsByID := uc.fetchBatchRunsByIDMap(ctx, jobID, ledgerItems)
			for _, item := range ledgerItems {
				if item.PipelineRunID == nil || strings.TrimSpace(*item.PipelineRunID) == "" {
					continue
				}
				run, ok := runsByID[*item.PipelineRunID]
				if !ok || run == nil {
					continue
				}
				mapped := mapRunStatusToItem(run.Status, run.ArgoWorkflowUID)
				if mapped == "completed" {
					mapped = uc.resolveCompletionStatus(ctx, job, item)
				}
				if mapped != "" && mapped != item.Status {
					wf := run.WorkflowName
					errMsg := ""
					if mapped == "failed" {
						errMsg = strings.TrimSpace(run.Message)
					}
					_ = uc.repo.UpdateItemStatus(ctx, item.ID, mapped, wf, errMsg)
				}
			}
		}
	}

	// Re-reconcile already-failed items against runtime truth. They are not in
	// the running/pending working set above, so a transient/misclassified
	// 'failed' — e.g. a run momentarily reconciled to Failed during a scheduling
	// backlog while its Argo workflow was actually queued and later Succeeded —
	// would otherwise stay stuck forever, inflating failedCount and blocking the
	// job from settling to 'completed'. GetRun runs the existing
	// reconcileMisclassifiedRunFromArgo path, whose guards preserve genuine
	// pre-submission failures (isDefinitiveTerminalFailure). Bounded per sync.
	if uc.pipelineUC != nil {
		failedItems, err := uc.repo.FindItemsByJobIDWithStatuses(ctx, jobID, []string{"failed"})
		if err != nil {
			return err
		}
		reconciled := 0
		for _, item := range failedItems {
			if reconciled >= maxFailedReconcilePerSync {
				slog.Info("syncJobProgress: failed-item reconcile capped, deferring remainder to next sync",
					"jobID", jobID, "processed", reconciled, "deferred", len(failedItems)-reconciled)
				break
			}
			if item.PipelineRunID == nil || strings.TrimSpace(*item.PipelineRunID) == "" {
				continue
			}
			reconciled++
			run, err := uc.pipelineUC.GetRun(ctx, *item.PipelineRunID)
			if err != nil || run == nil {
				continue
			}
			mapped := mapRunStatusToItem(run.Status, run.ArgoWorkflowUID)
			if mapped == "completed" {
				mapped = uc.resolveCompletionStatus(ctx, job, item)
			}
			if mapped == "" || mapped == item.Status {
				continue
			}
			errMsg := ""
			if mapped == "failed" {
				errMsg = strings.TrimSpace(run.Message)
			}
			_ = uc.repo.UpdateItemStatus(ctx, item.ID, mapped, run.WorkflowName, errMsg)
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
		uc.notifyJobTerminalIfNeeded(ctx, latestJob, jobStatus, summary)
		return nil
	}
	if err := uc.updateJobProgressAndParentRun(ctx, jobID, summary.Completed, summary.Failed, jobStatus); err != nil {
		return err
	}
	uc.notifyJobTerminalIfNeeded(ctx, latestJob, jobStatus, summary)
	return nil
}

// isTerminalBackfillStatus reports whether a batch job status is one that
// deriveJobStatus can settle on and that the watcher will stop re-scanning
// (see FindIncompleteJobs, which only selects status='running' jobs) — i.e.
// this transition will not be observed again.
func isTerminalBackfillStatus(status string) bool {
	return status == "completed" || status == "failed"
}

// notifyJobTerminalIfNeeded sends a completion notification exactly once when
// a job crosses from non-terminal into a terminal status. previousJob must
// reflect the job's status *before* this sync pass; newStatus is the status
// about to be (or just) persisted. Safe to call unconditionally: it no-ops
// when no notifier is configured, when the job was already terminal, when
// the new status is not terminal, or when a concurrent caller already
// claimed the notification for this job.
func (uc *Usecase) notifyJobTerminalIfNeeded(ctx context.Context, previousJob *models.BackfillJob, newStatus string, summary repository.BackfillItemStatusSummary) {
	if uc.notifier == nil || previousJob == nil {
		return
	}
	if isTerminalBackfillStatus(previousJob.Status) || !isTerminalBackfillStatus(newStatus) {
		return
	}
	claimed, err := uc.repo.ClaimJobNotification(ctx, previousJob.ID)
	if err != nil {
		slog.Warn("backfill: claim job notification failed", "jobID", previousJob.ID, "err", err)
		return
	}
	if !claimed {
		return
	}
	text := formatBatchJobNotificationText(previousJob, newStatus, summary, uc.frontendBaseURL)
	if err := uc.notifier.SendText(ctx, text); err != nil {
		slog.Warn("backfill: send completion notification failed", "jobID", previousJob.ID, "err", err)
	}
}

// formatBatchJobNotificationText builds the Feishu message body for a batch
// job reaching a terminal status. Kept in this package (not the notify
// provider package) since it is specific to what a batch job is.
func formatBatchJobNotificationText(job *models.BackfillJob, status string, summary repository.BackfillItemStatusSummary, frontendBaseURL string) string {
	name := job.Name
	if strings.TrimSpace(name) == "" {
		name = job.ID
	}
	text := fmt.Sprintf(
		"【批量任务完成】%s\n状态：%s\n总数：%d　成功：%d　失败：%d",
		name, status, job.TotalCount, summary.Completed, summary.Failed,
	)
	if job.CreatedBy != "" {
		text += fmt.Sprintf("\n创建人：%s", job.CreatedBy)
	}
	if !job.CreatedAt.IsZero() {
		text += fmt.Sprintf("\n耗时：%s", time.Since(job.CreatedAt).Round(time.Second))
	}
	if frontendBaseURL != "" {
		text += fmt.Sprintf("\n链接：%s/pipeline/batch/%s", strings.TrimRight(frontendBaseURL, "/"), job.ID)
	}
	return text
}

// fetchBatchRunsByIDMap retrieves runs for a batch job in a single query
// and indexes them by ID for O(1) lookup during item sync. Falls back to
// individual GetRun calls if ListSummaries returns no rows (e.g. in tests
// where the mock only implements the older interface).
func (uc *Usecase) fetchBatchRunsByIDMap(ctx context.Context, jobID string, items []models.BackfillItem) map[string]*models.PipelineRun {
	m := make(map[string]*models.PipelineRun)
	if uc.pipelineUC == nil {
		return m
	}
	all, _, err := uc.pipelineUC.ListRunSummaries(ctx, models.PipelineRunListFilter{
		BatchJobID: jobID,
		PageSize:   max(len(items), 1),
	})
	if err != nil || len(all) == 0 {
		if err != nil {
			slog.Warn("syncJobProgress: batch listing runs failed, falling back to per-item", "jobID", jobID, "err", err)
		}
		// Fall back to per-item GetRun for backwards compatibility with test mocks.
		for _, item := range items {
			if item.PipelineRunID == nil || *item.PipelineRunID == "" {
				continue
			}
			run, err := uc.pipelineUC.GetRun(ctx, *item.PipelineRunID)
			if err != nil || run == nil {
				continue
			}
			m[run.ID] = run
		}
		return m
	}
	for i := range all {
		m[all[i].ID] = &all[i]
	}
	return m
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
		BatchJobID:      job.ID,
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

// mapRunStatusToItem projects a run's status onto its backfill item.
// CYB-3491 semantics: terminal maps to terminal; anything in-flight maps to
// "submitted" — but ONLY when the run has a persisted Argo UID. A uid-less
// run is an unsubmitted placeholder (invariant ②): the item must stay
// "pending" so the submitter re-submits it, and it must NEVER be surfaced as
// failed. The empty return means "no transition".
func mapRunStatusToItem(runStatus, argoWorkflowUID string) string {
	switch strings.ToLower(runStatus) {
	case "succeeded", "success":
		return "completed"
	case "failed", "error":
		return "failed"
	default:
		if strings.TrimSpace(argoWorkflowUID) == "" {
			return ""
		}
		return "submitted"
	}
}

// isTerminalRunStatus reports whether a pipeline run has reached a final state.
// Queued ("Pending") and "Running" are explicitly non-terminal.
func isTerminalRunStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "succeeded", "success", "failed", "error":
		return true
	default:
		return false
	}
}

// runAlreadySubmitted reports whether a pipeline run has already been dispatched
// to Argo and is still in flight, so executeItem must not submit a duplicate.
// A run counts as submitted once Argo has assigned a workflow UID (present even
// while the workflow is queued/Pending) or once it has a StartedAt. An unsubmitted
// placeholder (UpsertBatchSubtaskRun) has neither, so first-time deploys proceed.
// Terminal runs return false so retries can re-deploy.
func runAlreadySubmitted(run *models.PipelineRun) bool {
	if run == nil {
		return false
	}
	submitted := strings.TrimSpace(run.ArgoWorkflowUID) != "" || run.StartedAt != nil
	return submitted && !isTerminalRunStatus(run.Status)
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
