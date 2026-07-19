package backfill

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/argo"
	"github.com/CyberOrigin2077/cyber-databrew/internal/metrics"
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
	// ResolveTargetClusterID maps a target to its cluster ("" → "default"),
	// the sharding key for per-cluster dispatch channels (CYB-3678).
	ResolveTargetClusterID(ctx context.Context, targetID string) string
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
		// Boot jitter (CYB-3678 C17): multi-instance cold starts (rollout /
		// scale-out) de-align their eager cycles and lock probes instead of
		// stampeding the DB and clusters in the same instant.
		if uc.bootJitter != nil {
			select {
			case <-time.After(uc.bootJitter()):
			case <-uc.submitStop:
				return
			}
		}
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

// runSubmitterCycle shards one dispatch cycle by target cluster (CYB-3678):
// each cluster gets its own goroutine, governor, and per-cluster advisory
// lock, so a slow cluster only stalls its own channel. When any channel
// filled a whole batch (backlog remains), the cycle self-kicks instead of
// idling until the next tick — short-task/large-node clusters stay fed.
func (uc *Usecase) runSubmitterCycle(ctx context.Context) {
	if uc.submitQueue == nil || uc.deployer == nil {
		return
	}
	uc.refreshDispatcherConfigs(ctx) // CYB-3679: pick up online tuning each cycle
	jobs, err := uc.submitQueue.FindSubmittableJobs(ctx, submittableJobsPerCycle)
	if err != nil {
		slog.Warn("submitter: find submittable jobs failed", "err", err)
		return
	}
	groups := map[string][]*models.BackfillJob{}
	for i := range jobs {
		cluster := uc.deployer.ResolveTargetClusterID(ctx, targetIDFromBackfillJob(&jobs[i]))
		groups[cluster] = append(groups[cluster], &jobs[i])
	}
	var wg sync.WaitGroup
	var refillMu sync.Mutex
	refill := false
	for cluster, cjobs := range groups {
		wg.Add(1)
		go func(cluster string, cjobs []*models.BackfillJob) {
			defer wg.Done()
			if uc.runClusterChannel(ctx, cluster, cjobs) {
				refillMu.Lock()
				refill = true
				refillMu.Unlock()
			}
		}(cluster, cjobs)
	}
	wg.Wait()
	if refill {
		uc.KickSubmitter()
	}
}

// submitJobBatch submits up to one batch of pending items for a job under
// the cluster governor's bounds (token bucket rate + AIMD concurrency) and
// returns the attempt count plus per-item outcomes for the channel breaker.
// Pilot jobs only submit within the remaining pilot quota.
func (uc *Usecase) submitJobBatch(ctx context.Context, job *models.BackfillJob, gov *clusterGovernor) (int, []submitOutcome) {
	limit := perJobSubmitBatch
	if gov != nil {
		if sb := gov.submitBatchLimit(); sb > 0 {
			limit = sb // CYB-3679 per-cluster override
		}
	}
	if job.Status == "pilot_running" && job.PilotCount > 0 {
		pending, err := uc.repo.CountItemsByStatus(ctx, job.ID, "pending")
		if err != nil {
			slog.Warn("submitter: count pending failed", "jobID", job.ID, "err", err)
			return 0, nil
		}
		attempted := job.TotalCount - pending
		quota := job.PilotCount - attempted
		if quota <= 0 {
			return 0, nil
		}
		if quota < limit {
			limit = quota
		}
	}
	ids, err := uc.submitQueue.ListSubmittableItemIDs(ctx, job.ID, limit)
	if err != nil {
		slog.Warn("submitter: list candidates failed", "jobID", job.ID, "err", err)
		return 0, nil
	}
	if len(ids) == 0 {
		return 0, nil
	}
	// Version pinning (CYB-3677 P0): the template_version column is
	// authoritative — when set, the submitter must NEVER fall back to the
	// template's current active version, or one batch mixes versions when the
	// template moves mid-dispatch. Legacy rows (column NULL) pinned the
	// version in filter_json only; truly unpinned rows fall back to active
	// with a warning metric.
	templateVersion := job.TemplateVersion
	if templateVersion <= 0 {
		templateVersion = templateVersionFromBackfillFilter(job.FilterJSON)
	}
	if templateVersion <= 0 {
		templateVersion = uc.resolveTemplateVersion(ctx, job.TemplateID)
		metrics.DispatcherTemplateFallbackTotal.Inc()
		slog.Warn("submitter: template version pin missing, using active version",
			"jobID", job.ID, "templateID", job.TemplateID, "resolved", templateVersion)
	}

	slots := maxConcurrentBatchItems
	if gov != nil {
		slots = gov.slots()
	}
	var wg sync.WaitGroup
	sem := make(chan struct{}, slots)
	outcomes := make([]submitOutcome, len(ids))
	attempts := 0
	for i, id := range ids {
		// Token bucket (CYB-3678): the hard sustained-rate roof per cluster,
		// independent of the concurrency knob — sized to controller
		// consumption, not API-server acceptance.
		if gov != nil {
			if err := gov.limiter.Wait(ctx); err != nil {
				break // ctx cancelled — leave the rest pending
			}
		}
		attempts++
		wg.Add(1)
		sem <- struct{}{}
		go func(slot int, itemID string) {
			defer wg.Done()
			defer func() { <-sem }()
			start := time.Now()
			out := uc.submitOneCandidate(ctx, job, itemID, templateVersion)
			outcomes[slot] = out
			if gov != nil {
				gov.record(time.Since(start), out)
			}
			metrics.DispatcherSubmitDurationSeconds.Observe(time.Since(start).Seconds())
		}(i, id)
	}
	wg.Wait()
	return attempts, outcomes[:attempts]
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
func (uc *Usecase) submitOneCandidate(ctx context.Context, job *models.BackfillJob, itemID string, templateVersion int) submitOutcome {
	itemCtx, cancel := context.WithTimeout(ctx, deployTimeout)
	defer cancel()
	item, err := uc.repo.FindItemByID(itemCtx, itemID)
	if err != nil || item == nil || item.Status != "pending" {
		return outcomeSkip // gone, already advanced, or read error — next cycle re-lists
	}
	if uc.isJobPaused(itemCtx, job.ID) {
		return outcomeSkip // leave pending; resume re-kicks the submitter
	}
	out, err := uc.submitItem(itemCtx, job, *item, templateVersion)
	if err != nil {
		slog.Warn("submitter: item submission failed, will retry next cycle",
			"jobID", job.ID, "itemID", itemID, "err", err)
	}
	return out
}

// submitItem performs the actual submission for a row-locked pending
// item. Every DB write here rides the caller's transaction. Terminal
// decisions (submitted / failed) COMMIT; only infra errors return non-nil
// (→ rollback → still pending).
func (uc *Usecase) submitItem(ctx context.Context, job *models.BackfillJob, item models.BackfillItem, templateVersion int) (submitOutcome, error) {
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
			return outcomeOK, uc.repo.UpdateItemPipelineRun(ctx, item.ID, runID, wf, "submitted")
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
			return outcomeTransient, initErr // infra: retry next cycle
		}
		if err := uc.repo.UpdateItemPipelineRun(ctx, item.ID, runID, workflowName, "pending"); err != nil {
			return outcomeTransient, err
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
			refreshed, refreshErr := uc.deployer.RefreshRunFromWorkflowByName(ctx, workflowName, "")
			if refreshErr != nil {
				slog.Warn("submitter: already-exists uid backfill failed, will retry",
					"jobID", job.ID, "itemID", item.ID, "workflow", workflowName, "err", refreshErr)
				return outcomeTransient, refreshErr
			}
			// AlreadyExists but the CR isn't actually readable in Argo (name
			// reuse, or the CR was GC'd right after create) → the run still has
			// no uid. Advancing to "submitted" here would strand the item
			// forever (invariant ②: only pending items get re-listed). Leave it
			// pending so the next cycle re-submits.
			if refreshed == nil || !runAlreadySubmitted(refreshed) {
				slog.Warn("submitter: already-exists but run still has no uid, leaving pending",
					"jobID", job.ID, "itemID", item.ID, "workflow", workflowName)
				return outcomeTransient, fmt.Errorf("%w: already-exists without a readable workflow", pipelineUC.ErrWorkflowSubmitIncomplete)
			}
			return outcomeOK, uc.repo.UpdateItemPipelineRun(ctx, item.ID, runID, workflowName, "submitted")
		}
		// Retryable incomplete submit: the runtime accepted the workflow but no
		// Argo uid materialized (typically a rate-limited post-submit re-read).
		// The CR is very likely live, so DON'T fail the run — leave the item
		// pending and let the next cycle re-submit (deterministic name →
		// AlreadyExists → uid backfill). Distinct from the deterministic failure
		// below (bad template/asset), which does fail the item.
		if errors.Is(err, pipelineUC.ErrWorkflowSubmitIncomplete) {
			slog.Warn("submitter: submit incomplete (no uid yet), leaving pending for retry",
				"jobID", job.ID, "itemID", item.ID, "err", err)
			return outcomeTransient, err
		}
		// Error classification (CYB-3678, review P1-1 v2): only explicitly
		// permanent errors fail the item now; everything else is transient and
		// retried up to the attempt cap — misclassifying a network blip as
		// permanent would mass-fail innocent items.
		if classifySubmitError(err) == outcomeTransient {
			attempts, incErr := uc.repo.IncrementItemSubmitAttempts(ctx, item.ID)
			if incErr != nil {
				slog.Warn("submitter: attempt counter increment failed",
					"jobID", job.ID, "itemID", item.ID, "err", incErr)
				return outcomeTransient, err // stay pending; counter retries too
			}
			if attempts < maxSubmitAttempts {
				slog.Warn("submitter: transient submit failure, will retry",
					"jobID", job.ID, "itemID", item.ID, "attempt", attempts, "err", err)
				return outcomeTransient, err
			}
			// Poison item: cap reached → DLQ (failed with reason). Explicit
			// human retry via the DLQ API resets the counter.
			metrics.DispatcherDLQTotal.WithLabelValues("max_submit_attempts").Inc()
			errMsg := fmt.Sprintf("max submit attempts (%d) exceeded: %v", maxSubmitAttempts, err)
			if runID != "" {
				return outcomePermanent, uc.repo.UpdateItemPipelineRun(ctx, item.ID, runID, workflowName, "failed")
			}
			return outcomePermanent, uc.repo.UpdateItemStatus(ctx, item.ID, "failed", workflowName, errMsg)
		}
		// Deterministic submission failure (bad template/asset/transpile):
		// surface it — parity with the legacy executeItem behaviour.
		metrics.DispatcherDLQTotal.WithLabelValues("permanent").Inc()
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
			return outcomePermanent, uc.repo.UpdateItemPipelineRun(ctx, item.ID, runID, workflowName, "failed")
		}
		return outcomePermanent, uc.repo.UpdateItemStatus(ctx, item.ID, "failed", workflowName, errMsg)
	}

	if err := uc.repo.UpdateItemPipelineRun(ctx, item.ID, dep.ID, dep.WorkflowName, "submitted"); err != nil {
		return outcomeTransient, err
	}
	if bindErr := uc.deployer.CommitBatchSubtaskDeploy(ctx, runID, dep); bindErr != nil {
		slog.Warn("submitter: commit batch subtask deploy failed",
			"jobID", job.ID, "assetID", item.AssetID, "runID", runID, "err", bindErr)
	}
	return outcomeOK, nil
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
