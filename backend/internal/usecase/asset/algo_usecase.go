package asset

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"data-platform/internal/config"
	"data-platform/internal/models"
	"data-platform/internal/repository"
)

// Custom errors for algorithm lifecycle operations.
var (
	ErrInvalidAlgoKey        = errors.New("invalid or unregistered algo_key")
	ErrAssetNotFound         = errors.New("asset not found")
	ErrAlgoAlreadyRunning    = errors.New("algorithm is already running")
	ErrInvalidStateTransition = errors.New("invalid state transition")
	ErrConcurrentConflict    = errors.New("concurrent conflict after retries")
	ErrMissingRequiredField  = errors.New("missing required field")
	ErrMissingReason         = errors.New("missing reason for failed status")
)

// StartAlgoInput holds the request body for starting an algorithm.
type StartAlgoInput struct {
	Method string  `json:"method"`
	RunID  *string `json:"run_id,omitempty"`
}

// FinishAlgoInput holds the request body for finishing an algorithm.
type FinishAlgoInput struct {
	Status          string                 `json:"status"`
	OutputURI       *string                `json:"output_uri,omitempty"`
	RunID           *string                `json:"run_id,omitempty"`
	Reason          *string                `json:"reason,omitempty"`
	ResultSizeBytes *int64                 `json:"result_size_bytes,omitempty"`
	ExtraFields     map[string]interface{} `json:"extra_fields,omitempty"`
}

// AlgoUsecase manages algorithm lifecycle business logic.
type AlgoUsecase struct {
	assetRepo repository.AssetRepository
	eventRepo repository.AlgoEventRepository
	registry  *config.AlgoRegistry
}

// NewAlgoUsecase creates a new AlgoUsecase.
func NewAlgoUsecase(
	assetRepo repository.AssetRepository,
	eventRepo repository.AlgoEventRepository,
	registry *config.AlgoRegistry,
) *AlgoUsecase {
	return &AlgoUsecase{
		assetRepo: assetRepo,
		eventRepo: eventRepo,
		registry:  registry,
	}
}

// getAlgoStatus extracts the current status of an algorithm from the asset's AlgoResults map.
// Returns empty string if no status is set (first time).
func getAlgoStatus(asset *models.Asset, algoKey string) models.AlgoStatus {
	key := algoKey + ":" + models.AlgoFieldStatus
	if v, ok := asset.AlgoResults[key]; ok && v != "" {
		return models.AlgoStatus(v)
	}
	return ""
}

// getAlgoField extracts a specific field value for an algorithm from the asset's AlgoResults map.
func getAlgoField(asset *models.Asset, algoKey, field string) string {
	key := algoKey + ":" + field
	return asset.AlgoResults[key]
}

// StartAlgo marks an algorithm as running on the given asset.
func (u *AlgoUsecase) StartAlgo(ctx context.Context, assetID, algoKey string, input StartAlgoInput) error {
	// Validate algoKey via registry.
	if err := u.registry.Validate(algoKey); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidAlgoKey, err.Error())
	}

	// Optimistic lock retry loop.
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		asset, err := u.assetRepo.Get(ctx, assetID)
		if err != nil {
			return err
		}
		if asset == nil {
			return ErrAssetNotFound
		}

		// Check state machine: current status must be pending or empty.
		currentStatus := getAlgoStatus(asset, algoKey)
		switch currentStatus {
		case models.AlgoStatusRunning:
			return ErrAlgoAlreadyRunning
		case models.AlgoStatusBlocked:
			return fmt.Errorf("%w: algorithm is blocked, dependencies not met", ErrInvalidStateTransition)
		case models.AlgoStatusOk:
			return fmt.Errorf("%w: algorithm already succeeded, must reset first", ErrInvalidStateTransition)
		case models.AlgoStatusFailed:
			return fmt.Errorf("%w: algorithm failed, must reset first", ErrInvalidStateTransition)
		case models.AlgoStatusPending, "":
			// allowed
		default:
			return fmt.Errorf("%w: unexpected current status %q", ErrInvalidStateTransition, currentStatus)
		}

		// Build algoKV.
		now := time.Now().UTC().Format(time.RFC3339)
		algoKV := map[string]interface{}{
			algoKey + ":" + models.AlgoFieldStatus:    string(models.AlgoStatusRunning),
			algoKey + ":" + models.AlgoFieldStartedAt: now,
			algoKey + ":" + models.AlgoFieldMethod:    input.Method,
		}
		if input.RunID != nil {
			algoKV[algoKey+":"+models.AlgoFieldRunID] = *input.RunID
		}

		filesKV := map[string]interface{}{}

		_, err = u.assetRepo.MergeCfAlgo(ctx, assetID, asset.Version, algoKV, filesKV)
		if err == nil {
			// Insert event record.
			prevStatus := stringPtr(string(currentStatus))
			if currentStatus == "" {
				prevStatus = nil
			}
			event := &models.AlgoEvent{
				EventID:    uuid.NewString(),
				AssetID:    assetID,
				AlgoKey:    algoKey,
				PrevStatus: prevStatus,
				NewStatus:  string(models.AlgoStatusRunning),
				RunID:      input.RunID,
				CreatedAt:  time.Now().UTC(),
			}
			return u.eventRepo.Insert(ctx, event)
		}
		if !errors.Is(err, repository.ErrOptimisticLock) {
			return err
		}
		lastErr = err
	}
	_ = lastErr
	return ErrConcurrentConflict
}

func stringPtr(s string) *string {
	return &s
}

// FinishAlgo marks an algorithm as ok or failed on the given asset.
func (u *AlgoUsecase) FinishAlgo(ctx context.Context, assetID, algoKey string, input FinishAlgoInput) error {
	// Validate algoKey via registry.
	if err := u.registry.Validate(algoKey); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidAlgoKey, err.Error())
	}

	finishStatus := models.AlgoStatus(input.Status)
	if finishStatus != models.AlgoStatusOk && finishStatus != models.AlgoStatusFailed {
		return fmt.Errorf("%w: finish status must be ok or failed", ErrInvalidStateTransition)
	}

	// Validate required fields based on finish status.
	if finishStatus == models.AlgoStatusOk {
		requiredFields, uriRequired, err := u.registry.GetRequiredFields(algoKey)
		if err != nil {
			return err
		}
		if uriRequired && (input.OutputURI == nil || *input.OutputURI == "") {
			return fmt.Errorf("%w: output_uri is required", ErrMissingRequiredField)
		}
		for _, field := range requiredFields {
			if field == "output_uri" {
				// Already checked above via uriRequired.
				if input.OutputURI == nil || *input.OutputURI == "" {
					return fmt.Errorf("%w: %s", ErrMissingRequiredField, field)
				}
				continue
			}
			if input.ExtraFields == nil {
				return fmt.Errorf("%w: %s", ErrMissingRequiredField, field)
			}
			if _, ok := input.ExtraFields[field]; !ok {
				return fmt.Errorf("%w: %s", ErrMissingRequiredField, field)
			}
		}

		// Check report_size requirement.
		algoName, _, _ := parseAlgoKeyParts(algoKey)
		if def, ok := u.registry.GetDefinition(algoName); ok && def.Output.ReportSize {
			if input.ResultSizeBytes == nil {
				return fmt.Errorf("%w: result_size_bytes is required when report_size=true", ErrMissingRequiredField)
			}
		}
	}

	if finishStatus == models.AlgoStatusFailed {
		if input.Reason == nil || *input.Reason == "" {
			return ErrMissingReason
		}
	}

	// Optimistic lock retry loop.
	for attempt := 0; attempt < 3; attempt++ {
		asset, err := u.assetRepo.Get(ctx, assetID)
		if err != nil {
			return err
		}
		if asset == nil {
			return ErrAssetNotFound
		}

		currentStatus := getAlgoStatus(asset, algoKey)

		// Idempotency check: if status already ok AND run_id matches, return nil.
		if currentStatus == models.AlgoStatusOk && input.RunID != nil {
			storedRunID := getAlgoField(asset, algoKey, models.AlgoFieldRunID)
			if storedRunID == *input.RunID {
				return nil
			}
		}

		// Current status must be running.
		if currentStatus != models.AlgoStatusRunning {
			return fmt.Errorf("%w: current status is %q, only running can be finished", ErrInvalidStateTransition, currentStatus)
		}

		// Build algoKV.
		now := time.Now().UTC().Format(time.RFC3339)
		algoKV := map[string]interface{}{
			algoKey + ":" + models.AlgoFieldStatus:     string(finishStatus),
			algoKey + ":" + models.AlgoFieldFinishedAt: now,
		}
		filesKV := map[string]interface{}{}

		if finishStatus == models.AlgoStatusOk {
			if input.OutputURI != nil {
				algoKV[algoKey+":"+models.AlgoFieldOutputURI] = *input.OutputURI
				filesKV[algoKey] = *input.OutputURI
			}
			if input.RunID != nil {
				algoKV[algoKey+":"+models.AlgoFieldRunID] = *input.RunID
			}
			// Store extra fields.
			for k, v := range input.ExtraFields {
				algoKV[algoKey+":"+k] = v
			}
			// Accumulate total_size_bytes if report_size=true.
			if input.ResultSizeBytes != nil {
				currentTotalStr := getAlgoField(asset, algoKey, "total_size_bytes")
				_ = currentTotalStr // total_size_bytes is in cf_meta, handled via algoKV for now
				// We store the result_size_bytes in the algo's own field space.
				algoKV[algoKey+":result_size_bytes"] = *input.ResultSizeBytes
			}
		} else {
			// failed
			if input.Reason != nil {
				algoKV[algoKey+":"+models.AlgoFieldReason] = *input.Reason
			}
			if input.RunID != nil {
				algoKV[algoKey+":"+models.AlgoFieldRunID] = *input.RunID
			}
			filesKV[algoKey] = nil
		}

		_, err = u.assetRepo.MergeCfAlgo(ctx, assetID, asset.Version, algoKV, filesKV)
		if err == nil {
			// Insert event record.
			prevStatus := stringPtr(string(currentStatus))
			event := &models.AlgoEvent{
				EventID:    uuid.NewString(),
				AssetID:    assetID,
				AlgoKey:    algoKey,
				PrevStatus: prevStatus,
				NewStatus:  string(finishStatus),
				RunID:      input.RunID,
				CreatedAt:  time.Now().UTC(),
			}
			if finishStatus == models.AlgoStatusFailed {
				event.Reason = input.Reason
			}
			if err := u.eventRepo.Insert(ctx, event); err != nil {
				return err
			}

			// Try to unblock downstream algorithms on success.
			if finishStatus == models.AlgoStatusOk {
				// Re-fetch asset to get updated state for downstream check.
				updatedAsset, err := u.assetRepo.Get(ctx, assetID)
				if err != nil {
					return err
				}
				if updatedAsset != nil {
					return u.tryUnblockDownstream(ctx, updatedAsset, algoKey)
				}
			}
			return nil
		}
		if !errors.Is(err, repository.ErrOptimisticLock) {
			return err
		}
	}
	return ErrConcurrentConflict
}

// parseAlgoKeyParts splits "name@version" into its components.
func parseAlgoKeyParts(algoKey string) (name, version string, ok bool) {
	for i := len(algoKey) - 1; i >= 0; i-- {
		if algoKey[i] == '@' && i > 0 && i < len(algoKey)-1 {
			return algoKey[:i], algoKey[i+1:], true
		}
	}
	return "", "", false
}

// ResetAlgo resets a failed or ok algorithm back to pending.
func (u *AlgoUsecase) ResetAlgo(ctx context.Context, assetID, algoKey string) error {
	// Validate algoKey via registry.
	if err := u.registry.Validate(algoKey); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidAlgoKey, err.Error())
	}

	// Optimistic lock retry loop.
	for attempt := 0; attempt < 3; attempt++ {
		asset, err := u.assetRepo.Get(ctx, assetID)
		if err != nil {
			return err
		}
		if asset == nil {
			return ErrAssetNotFound
		}

		currentStatus := getAlgoStatus(asset, algoKey)

		// Current status must be failed or ok.
		if currentStatus != models.AlgoStatusFailed && currentStatus != models.AlgoStatusOk {
			return fmt.Errorf("%w: current status is %q, only failed or ok can be reset", ErrInvalidStateTransition, currentStatus)
		}

		// Build algoKV: status=pending, clear reason/started_at/finished_at/output_uri.
		algoKV := map[string]interface{}{
			algoKey + ":" + models.AlgoFieldStatus:     string(models.AlgoStatusPending),
			algoKey + ":" + models.AlgoFieldReason:     nil,
			algoKey + ":" + models.AlgoFieldStartedAt:  nil,
			algoKey + ":" + models.AlgoFieldFinishedAt: nil,
			algoKey + ":" + models.AlgoFieldOutputURI:  nil,
		}

		// Build filesKV: cf_files[algo_key]=null.
		filesKV := map[string]interface{}{
			algoKey: nil,
		}

		_, err = u.assetRepo.MergeCfAlgo(ctx, assetID, asset.Version, algoKV, filesKV)
		if err == nil {
			// Insert event record.
			prevStatus := stringPtr(string(currentStatus))
			event := &models.AlgoEvent{
				EventID:    uuid.NewString(),
				AssetID:    assetID,
				AlgoKey:    algoKey,
				PrevStatus: prevStatus,
				NewStatus:  string(models.AlgoStatusPending),
				CreatedAt:  time.Now().UTC(),
			}
			return u.eventRepo.Insert(ctx, event)
		}
		if !errors.Is(err, repository.ErrOptimisticLock) {
			return err
		}
	}
	return ErrConcurrentConflict
}

// tryUnblockDownstream checks all algorithms in the registry for those that depend on
// completedAlgo. If a downstream algo is blocked and all its dependencies are ok,
// it transitions from blocked to pending.
func (u *AlgoUsecase) tryUnblockDownstream(ctx context.Context, asset *models.Asset, completedAlgo string) error {
	allAlgos := u.registry.GetAllAlgorithms()

	for algoName, def := range allAlgos {
		if len(def.DependsOn) == 0 {
			continue
		}

		// Check if completedAlgo is in this algorithm's depends_on list.
		dependsOnCompleted := false
		for _, dep := range def.DependsOn {
			if dep == completedAlgo {
				dependsOnCompleted = true
				break
			}
		}
		if !dependsOnCompleted {
			continue
		}

		// For each version of this downstream algorithm, check if it should be unblocked.
		for _, ver := range def.Versions {
			downstreamKey := algoName + "@" + ver
			currentStatus := getAlgoStatus(asset, downstreamKey)

			if currentStatus != models.AlgoStatusBlocked {
				continue
			}

			// Check if ALL dependencies are ok.
			allDepsOk := true
			for _, depKey := range def.DependsOn {
				depStatus := getAlgoStatus(asset, depKey)
				if depStatus != models.AlgoStatusOk {
					allDepsOk = false
					break
				}
			}

			if !allDepsOk {
				continue
			}

			// Unblock: blocked → pending.
			algoKV := map[string]interface{}{
				downstreamKey + ":" + models.AlgoFieldStatus: string(models.AlgoStatusPending),
			}
			filesKV := map[string]interface{}{}

			_, err := u.assetRepo.MergeCfAlgo(ctx, asset.AssetID, asset.Version, algoKV, filesKV)
			if err != nil {
				// Best-effort: if optimistic lock fails, skip (will be retried on next finish).
				if errors.Is(err, repository.ErrOptimisticLock) {
					continue
				}
				return err
			}

			// Insert event record for the unblock.
			prevStatus := stringPtr(string(models.AlgoStatusBlocked))
			event := &models.AlgoEvent{
				EventID:    uuid.NewString(),
				AssetID:    asset.AssetID,
				AlgoKey:    downstreamKey,
				PrevStatus: prevStatus,
				NewStatus:  string(models.AlgoStatusPending),
				CreatedAt:  time.Now().UTC(),
			}
			if err := u.eventRepo.Insert(ctx, event); err != nil {
				return err
			}

			// Re-fetch asset to get updated version for subsequent updates.
			updatedAsset, err := u.assetRepo.Get(ctx, asset.AssetID)
			if err != nil {
				return err
			}
			if updatedAsset != nil {
				asset = updatedAsset
			}
		}
	}
	return nil
}

// ListAlgoEvents returns algorithm events for the given asset, optionally filtered by algo_key.
func (u *AlgoUsecase) ListAlgoEvents(ctx context.Context, assetID string, algoKey *string) ([]*models.AlgoEvent, error) {
	// Verify asset exists.
	asset, err := u.assetRepo.Get(ctx, assetID)
	if err != nil {
		return nil, err
	}
	if asset == nil {
		return nil, ErrAssetNotFound
	}

	return u.eventRepo.ListByAsset(ctx, assetID, algoKey)
}
