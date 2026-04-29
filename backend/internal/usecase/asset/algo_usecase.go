// Package asset hosts asset-domain business logic. AlgoUsecase implements the
// per-algorithm lifecycle on an asset.
//
// Source-of-truth model:
//
//   - asset_algo_latest is the projection table that holds the current
//     per-algorithm state. Primary key (asset_id, algo_name) plus a
//     monotonic guard on algo_version make concurrent finishes from
//     different algorithm versions safe without taking any lock on
//     `assets`. This is the architectural fix for the OCC contention
//     described in data-platform-design.md §5.3.1.
//   - asset_events is the outbox: every state transition appends one event
//     in the SAME transaction as the projection write, so ES / Iceberg /
//     audit consumers can replay deterministically.
//   - assets.version is intentionally NOT bumped on algo state changes.
//     It is reserved for changes to fields owned by the assets table
//     (lifecycle_state, owner, manual edits, etc.).
package asset

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"data-platform/internal/config"
	"data-platform/internal/models"
	"data-platform/internal/repository"
)

// Custom errors for algorithm lifecycle operations.
var (
	ErrInvalidAlgoKey         = errors.New("invalid or unregistered algo_key")
	ErrAssetNotFound          = errors.New("asset not found")
	ErrAlgoAlreadyRunning     = errors.New("algorithm is already running")
	ErrInvalidStateTransition = errors.New("invalid state transition")
	ErrConcurrentConflict     = errors.New("concurrent conflict after retries")
	ErrMissingRequiredField   = errors.New("missing required field")
	ErrMissingReason          = errors.New("missing reason for failed status")
)

// Event types appended to asset_events. Names align with
// data-platform-design.md §5.2.7 typical event types.
const (
	eventAlgoStarted   = "algo_started"
	eventAlgoFinished  = "algo_finished" // status = ok
	eventAlgoFailed    = "algo_failed"
	eventAlgoReset     = "algo_reset"
	eventAlgoUnblocked = "algo_unblocked"
)

// algoEventTypes lists every event_type produced by AlgoUsecase. Used by
// ListAlgoEvents to filter asset_events down to the algo-lifecycle subset.
var algoEventTypes = []string{
	eventAlgoStarted,
	eventAlgoFinished,
	eventAlgoFailed,
	eventAlgoReset,
	eventAlgoUnblocked,
}

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

// AlgoUsecase manages algorithm lifecycle business logic. It depends only on
// projection / outbox repos, NOT on AssetRepository — algo state is owned by
// asset_algo_latest, not by the assets row.
//
// The assetRepo is held only as an existence-check helper; it is read-only
// here and never written to.
type AlgoUsecase struct {
	tx             repository.TxRunner
	existenceRepo  repository.AssetRepository
	algoLatestRepo repository.AssetAlgoLatestRepository
	eventRepo      repository.AssetEventRepository
	registry       *config.AlgoRegistry
}

// NewAlgoUsecase wires AlgoUsecase. existenceRepo is used for "asset must
// exist" guards on read paths; it is never written to.
func NewAlgoUsecase(
	tx repository.TxRunner,
	existenceRepo repository.AssetRepository,
	algoLatestRepo repository.AssetAlgoLatestRepository,
	eventRepo repository.AssetEventRepository,
	registry *config.AlgoRegistry,
) *AlgoUsecase {
	return &AlgoUsecase{
		tx:             tx,
		existenceRepo:  existenceRepo,
		algoLatestRepo: algoLatestRepo,
		eventRepo:      eventRepo,
		registry:       registry,
	}
}

// parseAlgoKey splits "name@version" into its components. Returns ok=false on
// malformed input.
func parseAlgoKey(algoKey string) (name, version string, ok bool) {
	for i := len(algoKey) - 1; i >= 0; i-- {
		if algoKey[i] == '@' && i > 0 && i < len(algoKey)-1 {
			return algoKey[:i], algoKey[i+1:], true
		}
	}
	return "", "", false
}

// currentStatus returns the persisted status for (assetID, algoKey) or "" when
// no row exists. The empty string is the sentinel for "algorithm has never
// run on this asset" and is treated as a legal starting state.
func (u *AlgoUsecase) currentStatus(ctx context.Context, assetID, algoKey string) (models.AlgoStatus, *models.AssetAlgoLatest, error) {
	algoName, _, ok := parseAlgoKey(algoKey)
	if !ok {
		return "", nil, fmt.Errorf("%w: %q", ErrInvalidAlgoKey, algoKey)
	}
	row, err := u.algoLatestRepo.GetByAlgo(ctx, assetID, algoName)
	if err != nil {
		return "", nil, err
	}
	if row == nil {
		return "", nil, nil
	}
	return models.AlgoStatus(row.Status), row, nil
}

// requireAssetExists returns ErrAssetNotFound when the asset is missing.
func (u *AlgoUsecase) requireAssetExists(ctx context.Context, assetID string) error {
	a, err := u.existenceRepo.Get(ctx, assetID)
	if err != nil {
		return err
	}
	if a == nil {
		return ErrAssetNotFound
	}
	return nil
}

// runIDPtr is a small helper for events.
func runIDPtr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// reasonPtr is a small helper for events.
func reasonPtr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// makePayload builds the canonical JSON payload for algo_* events.
//
// Stable schema (`payload_schema_version=v1`):
//
//	{ "algo_key": "name@version",
//	  "algo_name": "name",
//	  "algo_version": "version",
//	  "prev_status": "...",
//	  "new_status": "...",
//	  "run_id":  "..." (omitted when empty),
//	  "reason": "..." (omitted when empty)
//	}
func makePayload(algoKey, algoName, algoVersion, prev, next, runID, reason string) []byte {
	m := map[string]interface{}{
		"algo_key":     algoKey,
		"algo_name":    algoName,
		"algo_version": algoVersion,
		"prev_status":  prev,
		"new_status":   next,
	}
	if runID != "" {
		m["run_id"] = runID
	}
	if reason != "" {
		m["reason"] = reason
	}
	b, _ := json.Marshal(m)
	return b
}

// ────────────────────────────────────────────────────────────────────────────
// StartAlgo
// ────────────────────────────────────────────────────────────────────────────

// StartAlgo marks an algorithm as running on the given asset.
//
// Legal prior states: empty (no prior row) or pending. Running, blocked, ok
// and failed are rejected per the documented state machine; callers must
// reset before re-starting a finished algo.
func (u *AlgoUsecase) StartAlgo(ctx context.Context, assetID, algoKey string, input StartAlgoInput) error {
	if err := u.registry.Validate(algoKey); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidAlgoKey, err.Error())
	}
	algoName, algoVersion, _ := parseAlgoKey(algoKey)

	if err := u.requireAssetExists(ctx, assetID); err != nil {
		return err
	}

	now := time.Now().UTC()
	runID := runIDPtr(input.RunID)

	return u.tx.WithTx(ctx, func(txCtx context.Context) error {
		curStatus, _, err := u.currentStatus(txCtx, assetID, algoKey)
		if err != nil {
			return err
		}
		switch curStatus {
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
			return fmt.Errorf("%w: unexpected current status %q", ErrInvalidStateTransition, curStatus)
		}

		row := &models.AssetAlgoLatest{
			AssetID:     assetID,
			AlgoName:    algoName,
			AlgoVersion: algoVersion,
			Status:      string(models.AlgoStatusRunning),
			RunID:       runID,
			Method:      input.Method,
			StartedAt:   &now,
		}
		if err := u.algoLatestRepo.Upsert(txCtx, row); err != nil {
			return err
		}

		return u.eventRepo.Append(txCtx, repository.AssetEventAppendInput{
			EventType:    eventAlgoStarted,
			AssetID:      assetID,
			RunID:        runID,
			EventPayload: makePayload(algoKey, algoName, algoVersion, string(curStatus), string(models.AlgoStatusRunning), runID, ""),
		})
	})
}

// ────────────────────────────────────────────────────────────────────────────
// FinishAlgo
// ────────────────────────────────────────────────────────────────────────────

// FinishAlgo transitions a running algorithm to ok or failed.
//
// Idempotency: if the algorithm is already ok and the supplied run_id matches
// the persisted run_id, the call is a no-op. This handles retries from the
// caller side.
//
// Concurrency: distinct algo versions are safe to finish concurrently — the
// monotonic guard on algo_version in Upsert prevents an older version from
// overwriting a newer one. Same-version concurrent finishes converge under
// last-writer-wins; both events are emitted (at-least-once).
func (u *AlgoUsecase) FinishAlgo(ctx context.Context, assetID, algoKey string, input FinishAlgoInput) error {
	if err := u.registry.Validate(algoKey); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidAlgoKey, err.Error())
	}
	algoName, algoVersion, _ := parseAlgoKey(algoKey)

	finishStatus := models.AlgoStatus(input.Status)
	if finishStatus != models.AlgoStatusOk && finishStatus != models.AlgoStatusFailed {
		return fmt.Errorf("%w: finish status must be ok or failed", ErrInvalidStateTransition)
	}
	if finishStatus == models.AlgoStatusOk {
		if err := u.validateOkPayload(algoKey, input); err != nil {
			return err
		}
	}
	if finishStatus == models.AlgoStatusFailed && (input.Reason == nil || *input.Reason == "") {
		return ErrMissingReason
	}

	if err := u.requireAssetExists(ctx, assetID); err != nil {
		return err
	}

	runID := runIDPtr(input.RunID)
	reason := reasonPtr(input.Reason)
	now := time.Now().UTC()

	err := u.tx.WithTx(ctx, func(txCtx context.Context) error {
		curStatus, prev, err := u.currentStatus(txCtx, assetID, algoKey)
		if err != nil {
			return err
		}

		// Idempotency: already-ok with same run_id → no-op.
		if curStatus == models.AlgoStatusOk && runID != "" && prev != nil && prev.RunID == runID {
			return nil
		}
		if curStatus != models.AlgoStatusRunning {
			return fmt.Errorf("%w: current status is %q, only running can be finished", ErrInvalidStateTransition, curStatus)
		}

		row := &models.AssetAlgoLatest{
			AssetID:     assetID,
			AlgoName:    algoName,
			AlgoVersion: algoVersion,
			Status:      string(finishStatus),
			RunID:       runID,
			FinishedAt:  &now,
		}
		if finishStatus == models.AlgoStatusOk {
			if input.OutputURI != nil {
				row.OutputURI = *input.OutputURI
			}
			if input.ExtraFields != nil {
				row.ResultSummary = input.ExtraFields
			}
			if input.ResultSizeBytes != nil {
				if row.ResultSummary == nil {
					row.ResultSummary = map[string]interface{}{}
				}
				row.ResultSummary["result_size_bytes"] = *input.ResultSizeBytes
			}
		} else {
			row.ErrorMessage = reason
		}

		if err := u.algoLatestRepo.Upsert(txCtx, row); err != nil {
			return err
		}

		eventType := eventAlgoFinished
		if finishStatus == models.AlgoStatusFailed {
			eventType = eventAlgoFailed
		}
		if err := u.eventRepo.Append(txCtx, repository.AssetEventAppendInput{
			EventType:    eventType,
			AssetID:      assetID,
			RunID:        runID,
			EventPayload: makePayload(algoKey, algoName, algoVersion, string(curStatus), string(finishStatus), runID, reason),
		}); err != nil {
			return err
		}

		// Cascade unblock for ok finishes — same transaction.
		if finishStatus == models.AlgoStatusOk {
			return u.tryUnblockDownstream(txCtx, assetID, algoKey)
		}
		return nil
	})
	return err
}

// validateOkPayload checks that the finish-ok payload satisfies registry
// requirements (output_uri / extra fields / result_size_bytes).
func (u *AlgoUsecase) validateOkPayload(algoKey string, input FinishAlgoInput) error {
	requiredFields, uriRequired, err := u.registry.GetRequiredFields(algoKey)
	if err != nil {
		return err
	}
	if uriRequired && (input.OutputURI == nil || *input.OutputURI == "") {
		return fmt.Errorf("%w: output_uri is required", ErrMissingRequiredField)
	}
	for _, field := range requiredFields {
		if field == "output_uri" {
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
	algoName, _, _ := parseAlgoKey(algoKey)
	if def, ok := u.registry.GetDefinition(algoName); ok && def.Output.ReportSize {
		if input.ResultSizeBytes == nil {
			return fmt.Errorf("%w: result_size_bytes is required when report_size=true", ErrMissingRequiredField)
		}
	}
	return nil
}

// ────────────────────────────────────────────────────────────────────────────
// ResetAlgo
// ────────────────────────────────────────────────────────────────────────────

// ResetAlgo transitions a finished (ok/failed) algorithm back to pending so
// that StartAlgo can be invoked again. Clears run_id, output_uri, error
// fields and timestamps.
func (u *AlgoUsecase) ResetAlgo(ctx context.Context, assetID, algoKey string) error {
	if err := u.registry.Validate(algoKey); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidAlgoKey, err.Error())
	}
	algoName, algoVersion, _ := parseAlgoKey(algoKey)

	if err := u.requireAssetExists(ctx, assetID); err != nil {
		return err
	}

	return u.tx.WithTx(ctx, func(txCtx context.Context) error {
		curStatus, _, err := u.currentStatus(txCtx, assetID, algoKey)
		if err != nil {
			return err
		}
		if curStatus != models.AlgoStatusFailed && curStatus != models.AlgoStatusOk {
			return fmt.Errorf("%w: current status is %q, only failed or ok can be reset", ErrInvalidStateTransition, curStatus)
		}

		row := &models.AssetAlgoLatest{
			AssetID:     assetID,
			AlgoName:    algoName,
			AlgoVersion: algoVersion,
			Status:      string(models.AlgoStatusPending),
		}
		if err := u.algoLatestRepo.Upsert(txCtx, row); err != nil {
			return err
		}
		return u.eventRepo.Append(txCtx, repository.AssetEventAppendInput{
			EventType:    eventAlgoReset,
			AssetID:      assetID,
			EventPayload: makePayload(algoKey, algoName, algoVersion, string(curStatus), string(models.AlgoStatusPending), "", ""),
		})
	})
}

// ────────────────────────────────────────────────────────────────────────────
// tryUnblockDownstream — called from inside FinishAlgo's tx
// ────────────────────────────────────────────────────────────────────────────

// tryUnblockDownstream walks the registry for algorithms that depend on the
// just-completed algo and transitions any that are currently `blocked` AND
// have all dependencies satisfied (status=ok) to `pending`. Each unblock
// writes its own event row.
//
// Registry convention: `depends_on` entries are *versioned* algo keys
// ("name@version"); both the completed-algo match and the dependency
// satisfaction check therefore compare on the (name, version) pair.
//
// Runs inside the caller-provided transaction so unblocked downstream
// transitions and the original finish event commit atomically.
func (u *AlgoUsecase) tryUnblockDownstream(ctx context.Context, assetID, completedAlgoKey string) error {
	completedName, completedVersion, ok := parseAlgoKey(completedAlgoKey)
	if !ok {
		return fmt.Errorf("%w: %q", ErrInvalidAlgoKey, completedAlgoKey)
	}
	all := u.registry.GetAllAlgorithms()

	for downName, def := range all {
		if !dependsOnKey(def.DependsOn, completedName, completedVersion) {
			continue
		}
		for _, ver := range def.Versions {
			downKey := downName + "@" + ver

			downStatus, _, err := u.currentStatus(ctx, assetID, downKey)
			if err != nil {
				return err
			}
			if downStatus != models.AlgoStatusBlocked {
				continue
			}

			depsOk, err := u.allDepsOk(ctx, assetID, def.DependsOn)
			if err != nil {
				return err
			}
			if !depsOk {
				continue
			}

			row := &models.AssetAlgoLatest{
				AssetID:     assetID,
				AlgoName:    downName,
				AlgoVersion: ver,
				Status:      string(models.AlgoStatusPending),
			}
			if err := u.algoLatestRepo.Upsert(ctx, row); err != nil {
				return err
			}
			if err := u.eventRepo.Append(ctx, repository.AssetEventAppendInput{
				EventType:    eventAlgoUnblocked,
				AssetID:      assetID,
				EventPayload: makePayload(downKey, downName, ver, string(models.AlgoStatusBlocked), string(models.AlgoStatusPending), "", ""),
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

// dependsOnKey reports whether the (name, version) pair appears in deps.
// Each entry in deps is itself a "name@version" string (registry convention).
func dependsOnKey(deps []string, name, version string) bool {
	want := name + "@" + version
	for _, d := range deps {
		if d == want {
			return true
		}
	}
	return false
}

// allDepsOk reports whether every dependency in deps is satisfied — i.e. the
// projection row for that (algo_name, algo_version) exists with status=ok.
// A missing row, version mismatch or any non-ok status counts as "not ok".
func (u *AlgoUsecase) allDepsOk(ctx context.Context, assetID string, deps []string) (bool, error) {
	for _, dep := range deps {
		depName, depVer, ok := parseAlgoKey(dep)
		if !ok {
			return false, fmt.Errorf("invalid registry depends_on entry %q: must be name@version", dep)
		}
		row, err := u.algoLatestRepo.GetByAlgo(ctx, assetID, depName)
		if err != nil {
			return false, err
		}
		if row == nil || row.AlgoVersion != depVer || row.Status != string(models.AlgoStatusOk) {
			return false, nil
		}
	}
	return true, nil
}

// ────────────────────────────────────────────────────────────────────────────
// ListAlgoEvents
// ────────────────────────────────────────────────────────────────────────────

// algoEventEnvelope is the canonical payload shape for algo_* events.
// Mirrors makePayload above.
type algoEventEnvelope struct {
	AlgoKey     string `json:"algo_key"`
	AlgoName    string `json:"algo_name"`
	AlgoVersion string `json:"algo_version"`
	PrevStatus  string `json:"prev_status"`
	NewStatus   string `json:"new_status"`
	RunID       string `json:"run_id,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

// ListAlgoEvents returns algo lifecycle events for an asset, optionally
// filtered to a single algo_key. Sourced from the asset_events outbox table
// (the legacy `algo_events` table has been retired).
//
// Newest event first.
func (u *AlgoUsecase) ListAlgoEvents(ctx context.Context, assetID string, algoKey *string) ([]*models.AlgoEvent, error) {
	if err := u.requireAssetExists(ctx, assetID); err != nil {
		return nil, err
	}

	const queryLimit = 200
	rows, err := u.eventRepo.ListByAsset(ctx, assetID, algoEventTypes, queryLimit)
	if err != nil {
		return nil, err
	}

	out := make([]*models.AlgoEvent, 0, len(rows))
	for _, r := range rows {
		var env algoEventEnvelope
		if len(r.EventPayload) > 0 {
			if err := json.Unmarshal(r.EventPayload, &env); err != nil {
				continue // skip malformed payloads rather than failing the whole list
			}
		}
		if algoKey != nil && env.AlgoKey != *algoKey {
			continue
		}

		ev := &models.AlgoEvent{
			EventID:   r.EventID,
			AssetID:   r.AssetID,
			AlgoKey:   env.AlgoKey,
			NewStatus: env.NewStatus,
			CreatedAt: r.CreatedAt,
		}
		if env.PrevStatus != "" {
			prev := env.PrevStatus
			ev.PrevStatus = &prev
		}
		if env.RunID != "" {
			runID := env.RunID
			ev.RunID = &runID
		}
		if env.Reason != "" {
			reason := env.Reason
			ev.Reason = &reason
		}
		out = append(out, ev)
	}
	return out, nil
}

// ListCurrentStates returns every current algorithm projection row for an asset.
func (u *AlgoUsecase) ListCurrentStates(ctx context.Context, assetID string) ([]*models.AssetAlgoLatest, error) {
	if err := u.requireAssetExists(ctx, assetID); err != nil {
		return nil, err
	}
	rows, err := u.algoLatestRepo.ListByAsset(ctx, assetID)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return []*models.AssetAlgoLatest{}, nil
	}
	return rows, nil
}
