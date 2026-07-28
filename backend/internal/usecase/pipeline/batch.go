package pipeline

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// CreateBatchJob creates an async batch job that will create pipeline runs
// for each asset ID via the durable backfill submitter. Returns immediately
// with a batch ID.
//
// dedupPreservingOrder returns ids with duplicates (and blank entries)
// removed, keeping first-occurrence order.
func dedupPreservingOrder(ids []string) []string {
	seen := make(map[string]bool, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func (uc *Usecase) CreateBatchJob(ctx context.Context, templateID, name string, assetIDs []string, targetID string, templateVersion int, owner string) (*models.BackfillJob, error) {
	if uc.backfillRepo == nil {
		return nil, fmt.Errorf("%w: backfill repository is not configured", ErrInvalidArgument)
	}

	t, err := uc.templateRepo.FindByID(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("find template: %w", err)
	}
	if t == nil {
		return nil, ErrTemplateNotFound
	}

	resolvedVersion := t.Version
	if templateVersion > 0 && templateVersion != t.Version {
		resolvedVersion = templateVersion
	} else if templateVersion == 0 && t.ActiveVersion > 0 && t.ActiveVersion != t.Version {
		resolvedVersion = t.ActiveVersion
	}

	if len(assetIDs) == 0 {
		return nil, fmt.Errorf("%w: asset_ids is required", ErrInvalidArgument)
	}
	// Dedup asset ids, order-preserving (G1 load-test finding): duplicate ids
	// create duplicate items, but progress summaries dedup per asset — the
	// job's total_count then exceeds what the summary can ever reach and the
	// job stays "running" forever. One asset = one item.
	assetIDs = dedupPreservingOrder(assetIDs)

	batchID := "batch_" + uuid.New().String()
	now := time.Now().UTC()

	if name == "" {
		name = t.Name + "-" + time.Now().Format("2006-01-02")
	}

	// The job is born 'running' — the status the durable backfill submitter
	// selects — and this call only persists; submission is owned by the
	// submitter (crash-resumable, idempotent).
	job := &models.BackfillJob{
		ID:         batchID,
		TemplateID: templateID,
		Name:       name,
		Status:     "running",
		PilotPhase: "none",
		TotalCount: len(assetIDs),
		// TemplateVersion pins the batch to the version resolved at creation
		// time (P0: the submitter must never mix versions within one batch
		// when the template's active version moves mid-dispatch).
		TemplateVersion: resolvedVersion,
		CreatedAt:       now,
		UpdatedAt:       now,
		FilterJSON:      map[string]interface{}{},
		CreatedBy:       owner,
	}

	if targetID != "" && targetID != "default" {
		job.FilterJSON["target_id"] = targetID
	}
	if resolvedVersion > 0 {
		job.FilterJSON["template_version"] = resolvedVersion
	}

	if err := uc.backfillRepo.SaveJob(ctx, job); err != nil {
		return nil, fmt.Errorf("save batch job: %w", err)
	}

	items := make([]models.BackfillItem, len(assetIDs))
	for i, aid := range assetIDs {
		items[i] = models.BackfillItem{
			ID:        uuid.New().String(),
			JobID:     batchID,
			AssetID:   aid,
			Status:    "pending",
			CreatedAt: now,
		}
	}

	if err := uc.backfillRepo.SaveItems(ctx, items); err != nil {
		_ = uc.backfillRepo.UpdateJobStatus(ctx, batchID, "failed")
		return nil, fmt.Errorf("save batch items: %w", err)
	}

	// Submitter mode: persistence IS the dispatch. Kick the submitter so the
	// first cycle starts now instead of on the next 15s tick; durability
	// never depends on the kick (boot-eager + ticker re-list this job).
	if uc.batchSubmitKick != nil {
		uc.batchSubmitKick()
	}
	return job, nil
}

// GetBatchJobStatus returns the current status of a batch job.
func (uc *Usecase) GetBatchJobStatus(ctx context.Context, batchID string) (*models.BackfillJob, error) {
	if uc.backfillRepo == nil {
		return nil, fmt.Errorf("%w: backfill repository is not configured", ErrInvalidArgument)
	}
	job, err := uc.backfillRepo.FindJobByID(ctx, batchID)
	if err != nil {
		return nil, err
	}
	return job, nil
}
