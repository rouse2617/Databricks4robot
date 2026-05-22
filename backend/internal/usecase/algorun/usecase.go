package algorun

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/id"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
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
	"processing":  {},
	"split":       {},
	"qa":          {},
	"enrichment":  {},
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
	Limit         int
	Cursor        string
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
	repo repository.AlgoRunRepository
}

func New(repo repository.AlgoRunRepository) *Usecase {
	return &Usecase{repo: repo}
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
		if errors.Is(err, repository.ErrDuplicateRunID) {
			existing, gerr := u.repo.Get(ctx, runID)
			if gerr != nil {
				return nil, gerr
			}
			if existing != nil {
				return existing, nil
			}
		}
		return nil, err
	}
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
func (u *Usecase) List(ctx context.Context, f ListFilter) ([]*models.AlgoRun, error) {
	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = 50
	}
	return u.repo.List(ctx, repository.AlgoRunListFilter{
		AlgoName:      strings.TrimSpace(f.AlgoName),
		Status:        strings.TrimSpace(f.Status),
		StartedAfter:  f.StartedAfter,
		StartedBefore: f.StartedBefore,
		Limit:         f.Limit,
		Cursor:        strings.TrimSpace(f.Cursor),
	})
}

// Cancel transitions a pending/running run to cancelled.
func (u *Usecase) Cancel(ctx context.Context, runID, reason string) (*models.AlgoRun, error) {
	if !id.ValidateRunID(runID) {
		return nil, ErrInvalidRunID
	}
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("%w: cancel reason", ErrMissingField)
	}
	if err := u.repo.Cancel(ctx, runID, reason, time.Now().UTC()); err != nil {
		return nil, err
	}
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
