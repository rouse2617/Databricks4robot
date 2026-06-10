package algorun

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/id"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// Event types for algo_run outbox events (aggregate_type="algo_run").
const (
	eventAlgoRunCreated   = "algo_run_created"
	eventAlgoRunStarted   = "algo_run_started"
	eventAlgoRunFinished  = "algo_run_finished"
	eventAlgoRunCancelled = "algo_run_cancelled"
)

var (
	ErrInvalidRunID    = errors.New("run_id must be 16 alphanumeric characters")
	ErrInvalidAlgoKind = errors.New("algo_kind must be processing, split, qa, or enrichment")
	ErrInvalidStatus   = errors.New("finish status must be ok or failed")
	ErrMissingField    = errors.New("required field missing")
	ErrRunNotFound     = repository.ErrAlgoRunNotFound
	ErrBadTransition   = repository.ErrAlgoRunBadState
)

var allowedAlgoKinds = map[string]struct{}{
	"processing": {},
	"split":      {},
	"qa":         {},
	"enrichment": {},
}

// CreateInput is the body for POST /algo-runs.
type CreateInput struct {
	RunID           string                 `json:"run_id"`
	AlgoName        string                 `json:"algo_name"`
	AlgoVersion     string                 `json:"algo_version"`
	AlgoKind        string                 `json:"algo_kind"`
	TriggeredBy     string                 `json:"triggered_by"`
	InputFilter     map[string]interface{} `json:"input_filter"`
	InputAssetIDs   []string               `json:"input_asset_ids"`
	Params          map[string]interface{} `json:"params"`
	CodeCommit      string                 `json:"code_commit"`
	ImageDigest     string                 `json:"image_digest"`
	PipelineName    string                 `json:"pipeline_name"`
	PipelineVersion string                 `json:"pipeline_version"`
	TenantID        string                 `json:"tenant_id"`
	ProjectID       string                 `json:"project_id"`
}

// ListFilter carries query params for the list endpoint.
type ListFilter struct {
	AlgoName      string
	Status        string
	StartedAfter  *time.Time
	StartedBefore *time.Time
	Page          int
	PageSize      int
}

// FinishInput is the body for POST /algo-runs/{id}/finish.
type FinishInput struct {
	Status          string                 `json:"status"`
	AssetsProcessed *int                   `json:"assets_processed"`
	AssetsSucceeded *int                   `json:"assets_succeeded"`
	AssetsFailed    *int                   `json:"assets_failed"`
	ActionsCreated  *int                   `json:"actions_created"`
	MetricsWritten  *int                   `json:"metrics_written"`
	Outputs         map[string]interface{} `json:"outputs"`
	CPUSeconds      *int64                 `json:"cpu_seconds"`
	GPUSeconds      *int64                 `json:"gpu_seconds"`
	CostUSDMicros   *int64                 `json:"cost_usd_micros"`
	ErrorClass      string                 `json:"error_class"`
	ErrorMessage    string                 `json:"error_message"`
}

// Usecase implements algo_runs lifecycle.
type Usecase struct {
	repo      repository.AlgoRunRepository
	eventRepo repository.AssetEventRepository // optional; nil disables outbox events
}

func New(repo repository.AlgoRunRepository) *Usecase {
	return &Usecase{repo: repo}
}

// SetEventRepo enables outbox event emission for algo_run state transitions.
func (u *Usecase) SetEventRepo(r repository.AssetEventRepository) {
	u.eventRepo = r
}

// emitAlgoRunEvent appends an algo_run outbox event. Errors are logged but
// not returned so they never block the primary state mutation.
func (u *Usecase) emitAlgoRunEvent(ctx context.Context, eventType, runID string, payload map[string]any) {
	if u.eventRepo == nil {
		return
	}
	data, _ := json.Marshal(payload)
	if err := u.eventRepo.Append(ctx, repository.AssetEventAppendInput{
		EventType:     eventType,
		AggregateType: "algo_run",
		AssetID:       runID, // run_id carried in the asset_id column for routing
		EventPayload:  data,
	}); err != nil {
		// Best-effort: log but don't fail the state transition.
		_ = err
	}
}

func (u *Usecase) Create(ctx context.Context, in CreateInput) (*models.AlgoRun, error) {
	runID := strings.TrimSpace(in.RunID)
	if runID == "" {
		generated, err := id.GenerateRunID()
		if err != nil {
			return nil, err
		}
		runID = generated
	}
	if !id.ValidateRunID(runID) {
		return nil, ErrInvalidRunID
	}
	algoName := strings.TrimSpace(in.AlgoName)
	algoVersion := strings.TrimSpace(in.AlgoVersion)
	triggeredBy := strings.TrimSpace(in.TriggeredBy)
	if algoName == "" || algoVersion == "" || triggeredBy == "" {
		return nil, fmt.Errorf("%w: algo_name, algo_version, and triggered_by", ErrMissingField)
	}
	kind := strings.TrimSpace(in.AlgoKind)
	if kind == "" {
		kind = "processing"
	}
	if _, ok := allowedAlgoKinds[kind]; !ok {
		return nil, ErrInvalidAlgoKind
	}

	run := &models.AlgoRun{
		RunID:           runID,
		AlgoName:        algoName,
		AlgoVersion:     algoVersion,
		AlgoKind:        kind,
		TriggeredBy:     triggeredBy,
		Status:          models.AlgoRunStatusPending,
		InputFilter:     in.InputFilter,
		InputAssetIDs:   in.InputAssetIDs,
		Params:          in.Params,
		CodeCommit:      strings.TrimSpace(in.CodeCommit),
		ImageDigest:     strings.TrimSpace(in.ImageDigest),
		PipelineName:    strings.TrimSpace(in.PipelineName),
		PipelineVersion: strings.TrimSpace(in.PipelineVersion),
		TenantID:        strings.TrimSpace(in.TenantID),
		ProjectID:       strings.TrimSpace(in.ProjectID),
		RowVersion:      1,
	}
	if err := u.repo.Insert(ctx, run); err != nil {
		return nil, err
	}
	u.emitAlgoRunEvent(ctx, eventAlgoRunCreated, runID, map[string]any{
		"run_id":    runID,
		"algo_name": algoName,
		"status":    models.AlgoRunStatusPending,
	})
	return u.repo.Get(ctx, runID)
}

func (u *Usecase) Get(ctx context.Context, runID string) (*models.AlgoRun, error) {
	if !id.ValidateRunID(runID) {
		return nil, ErrInvalidRunID
	}
	run, err := u.repo.Get(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, ErrRunNotFound
	}
	return run, nil
}

func (u *Usecase) Start(ctx context.Context, runID string) (*models.AlgoRun, error) {
	if !id.ValidateRunID(runID) {
		return nil, ErrInvalidRunID
	}
	now := time.Now().UTC()
	if err := u.repo.Start(ctx, runID, now); err != nil {
		return nil, err
	}
	u.emitAlgoRunEvent(ctx, eventAlgoRunStarted, runID, map[string]any{
		"run_id":     runID,
		"status":     models.AlgoRunStatusRunning,
		"started_at": now.Format(time.RFC3339Nano),
	})
	return u.Get(ctx, runID)
}

func (u *Usecase) Finish(ctx context.Context, runID string, in FinishInput) (*models.AlgoRun, error) {
	if !id.ValidateRunID(runID) {
		return nil, ErrInvalidRunID
	}
	status := strings.TrimSpace(in.Status)
	if status != models.AlgoRunStatusOK && status != models.AlgoRunStatusFailed {
		return nil, ErrInvalidStatus
	}
	if status == models.AlgoRunStatusFailed && strings.TrimSpace(in.ErrorMessage) == "" {
		return nil, fmt.Errorf("%w: error_message required for failed status", ErrMissingField)
	}
	now := time.Now().UTC()
	patch := repository.AlgoRunFinishPatch{
		Status:          status,
		AssetsProcessed: in.AssetsProcessed,
		AssetsSucceeded: in.AssetsSucceeded,
		AssetsFailed:    in.AssetsFailed,
		ActionsCreated:  in.ActionsCreated,
		MetricsWritten:  in.MetricsWritten,
		Outputs:         in.Outputs,
		CPUSeconds:      in.CPUSeconds,
		GPUSeconds:      in.GPUSeconds,
		CostUSDMicros:   in.CostUSDMicros,
		ErrorClass:      strings.TrimSpace(in.ErrorClass),
		ErrorMessage:    strings.TrimSpace(in.ErrorMessage),
		FinishedAt:      now,
	}
	if err := u.repo.Finish(ctx, runID, patch); err != nil {
		return nil, err
	}
	u.emitAlgoRunEvent(ctx, eventAlgoRunFinished, runID, map[string]any{
		"run_id":        runID,
		"status":        status,
		"finished_at":   now.Format(time.RFC3339Nano),
		"error_class":   patch.ErrorClass,
		"error_message": patch.ErrorMessage,
	})
	return u.Get(ctx, runID)
}

func (u *Usecase) Exists(ctx context.Context, runID string) (bool, error) {
	if runID == "" {
		return false, nil
	}
	if !id.ValidateRunID(runID) {
		return false, ErrInvalidRunID
	}
	return u.repo.Exists(ctx, runID)
}

// List returns algo_runs matching the given filter.
func (u *Usecase) List(ctx context.Context, f ListFilter) ([]*models.AlgoRun, int64, error) {
	f = NormalizeListFilter(f)
	return u.repo.List(ctx, repository.AlgoRunListFilter{
		AlgoName:      strings.TrimSpace(f.AlgoName),
		Status:        strings.TrimSpace(f.Status),
		StartedAfter:  f.StartedAfter,
		StartedBefore: f.StartedBefore,
		Page:          f.Page,
		PageSize:      f.PageSize,
	})
}

func NormalizeListFilter(f ListFilter) ListFilter {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize <= 0 || f.PageSize > 200 {
		f.PageSize = 50
	}
	return f
}

// Cancel transitions a pending/running run to cancelled.
func (u *Usecase) Cancel(ctx context.Context, runID, reason string) (*models.AlgoRun, error) {
	if !id.ValidateRunID(runID) {
		return nil, ErrInvalidRunID
	}
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("%w: cancel reason", ErrMissingField)
	}
	now := time.Now().UTC()
	if err := u.repo.Cancel(ctx, runID, reason, now); err != nil {
		return nil, err
	}
	u.emitAlgoRunEvent(ctx, eventAlgoRunCancelled, runID, map[string]any{
		"run_id":        runID,
		"status":        models.AlgoRunStatusCancelled,
		"finished_at":   now.Format(time.RFC3339Nano),
		"error_message": reason,
	})
	return u.Get(ctx, runID)
}

// GetAffectedAssets returns assets processed by this run.
func (u *Usecase) GetAffectedAssets(ctx context.Context, runID string) ([]*repository.AffectedAsset, error) {
	if !id.ValidateRunID(runID) {
		return nil, ErrInvalidRunID
	}
	// Verify the run exists.
	exists, err := u.Exists(ctx, runID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrRunNotFound
	}
	return u.repo.GetAffectedAssets(ctx, runID)
}
