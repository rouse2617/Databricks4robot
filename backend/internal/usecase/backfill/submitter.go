package backfill

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/argo"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
)

// CYB-3491 P2 — the submitter.
//
// The backend no longer runs an execution work-queue (claim / lease / reaper /
// worker pool). Argo owns queueing, parallelism, execution and transient
// retry; the backend only ENSURES SUBMISSION and projects status back. The
// submitter is that "ensure submission" loop:
//
//   - periodic (ticker) AND kickable (materialize/resume/rerun poke it), so
//     progress never depends on a single boot-time goroutine surviving —
//     this is what permanently fixes the deploy-mid-dispatch hangs (D).
//   - idempotent: workflow names are deterministic and job-scoped; a crash
//     mid-submit leaves the item pending, and the re-submission heals via
//     ErrAlreadyExists + UID backfill instead of creating a duplicate.
//   - forward-only: a successful submit moves the item pending → submitted.
//     No lease, no reclaim, no running→pending regression.
//   - cross-instance safe WITHOUT a held lock: candidates are read lock-free
//     and submitted with NO surrounding transaction (see submitOneCandidate —
//     holding a tx across the Argo call starved the connection pool). Two
//     instances racing the same item mint the same deterministic workflow and
//     converge via AlreadyExists; the item advances to submitted only once
//     the UID is persisted.
const (
	submitterInterval       = 15 * time.Second
	submittableJobsPerCycle = 50
)

// subtaskDeployer is the slice of the pipeline usecase the submitter needs.
// It exists as a seam: production wires the concrete *pipeline.Usecase (see
// New); submitter tests wire a fake and exercise every branch without a
// transpiler / Argo stack.
type subtaskDeployer interface {
	GetRun(ctx context.Context, id string) (*models.PipelineRun, error)
	UpsertBatchSubtaskRun(ctx context.Context, in pipelineUC.BatchSubtaskRunInput) (string, string, error)
	DeployByTemplateID(ctx context.Context, templateID, name string, assetIDs []string, opts ...pipelineUC.DeployOptions) (*models.PipelineDeployment, error)
	CommitBatchSubtaskDeploy(ctx context.Context, runID string, dep *models.PipelineDeployment) error
	RecordBatchSubtaskFailure(ctx context.Context, in pipelineUC.BatchSubtaskRunInput) (string, string, error)
	RefreshRunFromWorkflowByName(ctx context.Context, workflowName, workflowUID string) (*models.PipelineRun, error)
}

// SetSubmitQueue wires the submitter's persistence surface. In production
// this is the postgres BackfillRepo (which implements repository.SubmitQueue);
// nil leaves the submitter disabled.
func (uc *Usecase) SetSubmitQueue(q repository.SubmitQueue) {
	uc.submitQueue = q
}

// StartSubmitter launches the periodic submit loop. Idempotent-ish: calling
// twice is a no-op while running.
func (uc *Usecase) StartSubmitter() {
	if uc.submitQueue == nil || uc.submitStop != nil {
		return
	}
	uc.submitStop = make(chan struct{})
	uc.submitKick = make(chan struct{}, 1)
	uc.submitWg.Add(1)
	go func() {
		defer uc.submitWg.Done()
		ticker := time.NewTicker(submitterInterval)
		defer ticker.Stop()
		// One eager cycle on boot: this replaces ResumeIncompleteBatches —
		// anything left pending by a redeploy is picked up immediately.
		uc.runSubmitterCycle(context.Background())
		for {
			select {
			case <-uc.submitStop:
				return
			case <-ticker.C:
				uc.runSubmitterCycle(context.Background())
			case <-uc.submitKick:
				uc.runSubmitterCycle(context.Background())
			}
		}
	}()
}

// StopSubmitter signals the loop to stop and waits for it.
func (uc *Usecase) StopSubmitter() {
	if uc.submitStop == nil {
		return
	}
	close(uc.submitStop)
	uc.submitWg.Wait()
	uc.submitStop = nil
}

// KickSubmitter requests an immediate submit cycle (non-blocking; coalesces).
func (uc *Usecase) KickSubmitter() {
	if uc.submitKick == nil {
		return
	}
	select {
	case uc.submitKick <- struct{}{}:
	default:
	}
}

// runSubmitterCycle submits pending items for every submittable job. Jobs are
// processed sequentially (a cycle is cheap when there is nothing to do);
// items within a job are submitted by a small worker group.
func (uc *Usecase) runSubmitterCycle(ctx context.Context) {
	if uc.submitQueue == nil || uc.deployer == nil {
		return
	}
	jobs, err := uc.submitQueue.FindSubmittableJobs(ctx, submittableJobsPerCycle)
	if err != nil {
		slog.Warn("submitter: find submittable jobs failed", "err", err)
		return
	}
	for i := range jobs {
		submitted := uc.submitJobBatch(ctx, &jobs[i])
		if submitted > 0 {
			// Non-forced: throttled counter/settle refresh; the webhook
			// cascade and the job reconciler own authoritative convergence.
			_ = uc.syncJobProgress(ctx, jobs[i].ID)
		}
	}
}

// submitJobBatch submits up to one batch of pending items for a job and
// returns how many submissions were attempted. Pilot jobs only submit within
// the remaining pilot quota — a stronger gate than the legacy pool, which
// raced the pilot_review flip.
func (uc *Usecase) submitJobBatch(ctx context.Context, job *models.BackfillJob) int {
	limit := perJobSubmitBatch
	if job.Status == "pilot_running" && job.PilotCount > 0 {
		pending, err := uc.repo.CountItemsByStatus(ctx, job.ID, "pending")
		if err != nil {
			slog.Warn("submitter: count pending failed", "jobID", job.ID, "err", err)
			return 0
		}
		attempted := job.TotalCount - pending
		quota := job.PilotCount - attempted
		if quota <= 0 {
			return 0
		}
		if quota < limit {
			limit = quota
		}
	}
	ids, err := uc.submitQueue.ListSubmittableItemIDs(ctx, job.ID, limit)
	if err != nil {
		slog.Warn("submitter: list candidates failed", "jobID", job.ID, "err", err)
		return 0
	}
	if len(ids) == 0 {
		return 0
	}
	templateVersion := job.TemplateVersion
	if templateVersion <= 0 {
		templateVersion = uc.resolveTemplateVersion(ctx, job.TemplateID)
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrentBatchItems)
	attempts := 0
	for _, id := range ids {
		attempts++
		wg.Add(1)
		sem <- struct{}{}
		go func(itemID string) {
			defer wg.Done()
			defer func() { <-sem }()
			uc.submitOneCandidate(ctx, job, itemID, templateVersion)
		}(id)
	}
	wg.Wait()
	return attempts
}

// submitOneCandidate re-checks and submits a single pending item.
//
// CYB-3491 (perf, critical): this deliberately does NOT wrap the submit in a
// transaction. submitItem calls Argo (DeployByTemplateID / GetRun), and those
// internally open their own short transactions. Wrapping them in an outer tx
// held a pgx pool connection across the Argo HTTP round-trip AND forced a
// nested pool.Begin (a second connection) — under the 5-way submitter
// concurrency that starved the pool and left transactions "idle in
// transaction" for tens of seconds, stalling every request on the service.
//
// Correctness does not need the lock: submission idempotency is guaranteed by
// the deterministic job-scoped workflow name + AlreadyExists backfill, so even
// if two instances race the same pending item they mint the same workflow and
// converge. Every write below is individually idempotent, and the item only
// advances to `submitted` after the UID is persisted — a crash mid-submit
// leaves it `pending` for the next cycle, exactly as before.
func (uc *Usecase) submitOneCandidate(ctx context.Context, job *models.BackfillJob, itemID string, templateVersion int) {
	itemCtx, cancel := context.WithTimeout(ctx, deployTimeout)
	defer cancel()
	item, err := uc.repo.FindItemByID(itemCtx, itemID)
	if err != nil || item == nil || item.Status != "pending" {
		return // gone, already advanced, or read error — next cycle re-lists
	}
	if uc.isJobPaused(itemCtx, job.ID) {
		return // leave pending; resume re-kicks the submitter
	}
	if err := uc.submitItem(itemCtx, job, *item, templateVersion); err != nil {
		slog.Warn("submitter: item submission failed, will retry next cycle",
			"jobID", job.ID, "itemID", itemID, "err", err)
	}
}

// submitItem performs the actual submission for a row-locked pending
// item. Every DB write here rides the caller's transaction. Terminal
// decisions (submitted / failed) COMMIT; only infra errors return non-nil
// (→ rollback → still pending).
func (uc *Usecase) submitItem(ctx context.Context, job *models.BackfillJob, item models.BackfillItem, templateVersion int) error {
	runID := ""
	if item.PipelineRunID != nil {
		runID = strings.TrimSpace(*item.PipelineRunID)
	}
	workflowName := ""
	if item.WorkflowName != nil {
		workflowName = strings.TrimSpace(*item.WorkflowName)
	}
	targetID := targetIDFromBackfillJob(job)

	// Idempotency guard: the run may already be live in Argo (uid persisted)
	// while the item is still pending — e.g. a crash between the run commit
	// and the item update. Mark submitted, done.
	if runID != "" {
		if existing, getErr := uc.deployer.GetRun(ctx, runID); getErr == nil && runAlreadySubmitted(existing) {
			wf := strings.TrimSpace(existing.WorkflowName)
			if wf == "" {
				wf = workflowName
			}
			return uc.repo.UpdateItemPipelineRun(ctx, item.ID, runID, wf, "submitted")
		}
	}

	// Ensure the run ledger row exists (deterministic, job-scoped workflow
	// name is minted here).
	if runID == "" {
		var initErr error
		runID, workflowName, initErr = uc.deployer.UpsertBatchSubtaskRun(ctx, pipelineUC.BatchSubtaskRunInput{
			TemplateID:      job.TemplateID,
			TemplateVersion: templateVersion,
			TargetID:        targetID,
			BatchJobID:      job.ID,
			AssetID:         item.AssetID,
			Status:          "Pending",
		})
		if initErr != nil {
			return initErr // infra: rollback, retry next cycle
		}
		if err := uc.repo.UpdateItemPipelineRun(ctx, item.ID, runID, workflowName, "pending"); err != nil {
			return err
		}
	}

	deployOpts := pipelineUC.DeployOptions{
		BatchJobID:         job.ID,
		TemplateVersion:    templateVersion,
		TargetID:           targetID,
		AllowUnknownAssets: true,
		PreallocatedRunID:  runID,
		Owner:              job.CreatedBy,
	}
	if job.FilterJSON != nil {
		deployOpts.TargetID = stringFromBackfillFilter(job.FilterJSON, "targetId", "target_id", "executionTargetId", "execution_target_id")
		if configRaw, ok := job.FilterJSON["configSelection"]; ok {
			if selection := decodeRuntimeConfigSelection(configRaw); selection != nil {
				deployOpts.ConfigSelection = selection
			}
		}
	}

	dep, err := uc.deployer.DeployByTemplateID(ctx, job.TemplateID, "", []string{item.AssetID}, deployOpts)
	if err != nil {
		if isWorkflowAlreadyExists(err) {
			// Our own prior submission survived a crash/rollback: backfill
			// the UID from Argo truth (same path the webhook uses) and mark
			// the item submitted. NOT a failure, and attempts is untouched.
			if _, refreshErr := uc.deployer.RefreshRunFromWorkflowByName(ctx, workflowName, ""); refreshErr != nil {
				slog.Warn("submitter: already-exists uid backfill failed, will retry",
					"jobID", job.ID, "itemID", item.ID, "workflow", workflowName, "err", refreshErr)
				return refreshErr
			}
			return uc.repo.UpdateItemPipelineRun(ctx, item.ID, runID, workflowName, "submitted")
		}
		// Deterministic submission failure (bad template/asset/transpile):
		// surface it — parity with the legacy executeItem behaviour.
		errMsg := err.Error()
		_, workflowName, _ = uc.deployer.RecordBatchSubtaskFailure(ctx, pipelineUC.BatchSubtaskRunInput{
			TemplateID:      job.TemplateID,
			TemplateVersion: templateVersion,
			TargetID:        targetID,
			BatchJobID:      job.ID,
			AssetID:         item.AssetID,
			RunID:           runID,
			Status:          "Failed",
			Message:         errMsg,
			WorkflowName:    workflowName,
		})
		if runID != "" {
			return uc.repo.UpdateItemPipelineRun(ctx, item.ID, runID, workflowName, "failed")
		}
		return uc.repo.UpdateItemStatus(ctx, item.ID, "failed", workflowName, errMsg)
	}

	if err := uc.repo.UpdateItemPipelineRun(ctx, item.ID, dep.ID, dep.WorkflowName, "submitted"); err != nil {
		return err
	}
	if bindErr := uc.deployer.CommitBatchSubtaskDeploy(ctx, runID, dep); bindErr != nil {
		slog.Warn("submitter: commit batch subtask deploy failed",
			"jobID", job.ID, "assetID", item.AssetID, "runID", runID, "err", bindErr)
	}
	return nil
}

// isWorkflowAlreadyExists reports whether err (possibly wrapped) is Argo's
// AlreadyExists. The string fallback covers error chains that crossed a
// non-%w boundary.
func isWorkflowAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, argo.ErrAlreadyExists) {
		return true
	}
	return strings.Contains(err.Error(), "already exists")
}
