package backfill

import (
	"context"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
)

// BackfillTaskHandler encapsulates all business logic for processing backfill items.
// It reads job context, invokes deployment services, and manages state transitions.
// This handler is completely decoupled from the Dispatcher via the TaskHandler interface,
// eliminating circular dependencies and enabling horizontal extension to new runtimes.
type BackfillTaskHandler struct {
	repo      repository.BackfillRepository
	pipelineUC *pipelineUC.Usecase // Concrete impl for DeployByTemplateID
	cfg       DispatcherConfig
	logger    *slog.Logger
}

// NewBackfillTaskHandler constructs a business logic handler.
func NewBackfillTaskHandler(repo repository.BackfillRepository, pipelineUC *pipelineUC.Usecase, cfg DispatcherConfig) *BackfillTaskHandler {
	return &BackfillTaskHandler{
		repo:       repo,
		pipelineUC: pipelineUC,
		cfg:        cfg,
		logger:     slog.Default().With(slog.String("component", "backfill-task-handler")),
	}
}

// Handle processes a single claimed backfill item through the full workflow:
// 1. Refresh lease (heartbeat)
// 2. Resolve deterministic workflow name
// 3. Fetch job context and runtime parameters
// 4. Update runtime fields
// 5. Submit via the idempotency hinge (DeployByTemplateID)
// 6. Transition state to submitted/failed/dead based on outcome
func (h *BackfillTaskHandler) Handle(ctx context.Context, item *models.BackfillItem) error {
	logger := h.logger.With(slog.String("item_id", item.ID))

	// 1. Refresh lease + state=submitting (heartbeat before slow submit).
	if err := h.repo.MarkDispatchSubmitting(ctx, item.ID, h.cfg.LeaseSec); err != nil {
		logger.Warn("mark submitting failed", "err", err)
		return err // leave lease, reaper will reset
	}

	// 2. Resolve deterministic workflow name. Phase 1's batch
	// placeholder is already deterministic per-asset; if it's
	// missing (rare: a row mutated externally), derive a fresh one.
	wfName := ""
	if item.WorkflowNamePlanned != nil {
		wfName = strings.TrimSpace(*item.WorkflowNamePlanned)
	}
	if wfName == "" && item.PipelineRunID != nil {
		// Fallback: only happens if migration ran without our setter.
		wfName = deriveWfNameFallback(item)
	}

	// 3. Resolve runtime params from the job. templateID is REQUIRED —
	// items carry no template of their own, so without the job's
	// template the submit below resolves nothing and would dead-letter
	// the whole batch. targetId/version are best-effort.
	targetID := ""
	templateID := ""
	templateVersion := 0
	if item.JobID != "" {
		if job, _ := h.repo.FindJobByID(ctx, item.JobID); job != nil {
			templateID = strings.TrimSpace(job.TemplateID)
			templateVersion = job.TemplateVersion
			if f, ok := job.FilterJSON["targetId"].(string); ok {
				targetID = strings.TrimSpace(f)
			}
		}
	}
	if templateID == "" {
		// Cannot dispatch without a template. Treat as retryable (the job
		// row may be mid-write / transiently unreadable); exhausted rows
		// dead-letter via markFailure rather than looping forever.
		logger.Warn("no templateID resolved for item", "job_id", item.JobID)
		h.markFailure(ctx, item, "template not resolved for job")
		return nil
	}

	// 4. Update runtime fields on the row so we have a self-consistent
	// record before submit (helps debug if submit crashes the worker).
	_ = h.repo.UpdateItemDispatchFields(ctx, item.ID, "", templateVersion, targetID, h.cfg.LeaseSec)

	// 5. Submit via the Phase 1 idempotency hinge.
	preallocRunID := ""
	if item.PipelineRunID != nil {
		preallocRunID = strings.TrimSpace(*item.PipelineRunID)
	}
	dep, err := h.pipelineUC.DeployByTemplateID(ctx, templateID, "", []string{item.AssetID}, pipelineUC.DeployOptions{
		BatchJobID:               item.JobID,
		TemplateVersion:          templateVersion,
		TargetID:                 targetID,
		PreallocatedRunID:        preallocRunID,
		PreallocatedWorkflowName: wfName,
		AllowUnknownAssets:       true,
	})

	// 6. Outcome dispatch.
	switch {
	case err == nil:
		argoUID := ""
		argoName := wfName
		if dep != nil {
			argoUID = dep.ID
			if n := strings.TrimSpace(dep.WorkflowName); n != "" {
				argoName = n
			}
		}
		// Success: transition to submitted (absorbing state).
		_ = h.repo.MarkDispatched(ctx, item.ID, argoName, argoUID)
		logger.Info("item dispatched successfully", "workflow_name", argoName, "workflow_id", argoUID)
		return nil

	case isPermanentDeployError(err):
		// Permanent error (template not found, invalid args, etc.): dead-letter immediately.
		logger.Warn("permanent deploy error", "err", err)
		_ = h.repo.MarkDispatchDead(ctx, item.ID, item.Attempts+1, err.Error())
		return nil

	default:
		// Transient error (network, timeout, etc.): retry with exponential backoff.
		logger.Warn("transient deploy error", "err", err)
		h.markFailure(ctx, item, err.Error())
		return nil
	}
}

// markFailure transitions the item to 'failed' if attempts < MaxAttempts,
// or to 'dead' if the ceiling is hit.
func (h *BackfillTaskHandler) markFailure(ctx context.Context, item *models.BackfillItem, errMsg string) {
	nextAttempts := item.Attempts + 1
	if nextAttempts >= h.cfg.MaxAttempts {
		_ = h.repo.MarkDispatchDead(ctx, item.ID, nextAttempts, errMsg)
		h.logger.Warn("item dead-lettered", "item_id", item.ID, "attempts", nextAttempts)
	} else {
		backoff := calculateBackoff(nextAttempts, h.cfg.BackoffBase, h.cfg.BackoffMax)
		backoffSec := int(backoff.Seconds())
		_ = h.repo.MarkDispatchFailedRetryable(ctx, item.ID, nextAttempts, errMsg, backoffSec)
		h.logger.Info("item scheduled for retry", "item_id", item.ID, "attempts", nextAttempts, "backoff_sec", backoffSec)
	}
}

// Exposed backoff helper for external callers (tests, etc).
func (h *BackfillTaskHandler) BackoffForAttempt(attempt int) time.Duration {
	return calculateBackoff(attempt, h.cfg.BackoffBase, h.cfg.BackoffMax)
}

// isPermanentDeployError classifies errors as permanent (dead-letter) vs transient (retry).
func isPermanentDeployError(err error) bool {
	if err == nil {
		return false
	}
	// Classify pipeline layer errors as permanent or transient.
	return err == pipelineUC.ErrTemplateNotFound ||
		err == pipelineUC.ErrInvalidArgument ||
		err == pipelineUC.ErrWorkflowUnavailable
}

// calculateBackoff computes exponential backoff with jitter and cap.
// It uses the same logic as Dispatcher.backoffForAttempt but is moved here
// to keep all business logic in the TaskHandler.
func calculateBackoff(attempt int, baseSeconds time.Duration, maxSeconds time.Duration) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	// Exponential: 2^(attempt-1) * base
	scaled := baseSeconds * time.Duration(math.Pow(2, float64(attempt-1)))
	if scaled > maxSeconds {
		scaled = maxSeconds
	}
	// Add ±10% jitter
	rngMu.Lock()
	jitter := time.Duration(rng.Intn(int(scaled/5))) - scaled/10
	rngMu.Unlock()
	return scaled + jitter
}
