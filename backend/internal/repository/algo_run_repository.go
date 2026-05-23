package repository

import (
	"context"
	"errors"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

var (
	ErrDuplicateRunID     = errors.New("duplicate run id")
	ErrAlgoRunNotFound    = errors.New("algo run not found")
	ErrAlgoRunBadState    = errors.New("invalid algo run state transition")
	ErrAlgoRunOptimistic  = errors.New("algo run optimistic lock conflict")
)

// AlgoRunFinishPatch carries terminal-state fields for finish.
type AlgoRunFinishPatch struct {
	Status          string
	AssetsProcessed *int
	AssetsSucceeded *int
	AssetsFailed    *int
	ActionsCreated  *int
	MetricsWritten  *int
	Outputs         map[string]interface{}
	CPUSeconds      *int64
	GPUSeconds      *int64
	CostUSDMicros   *int64
	ErrorClass      string
	ErrorMessage    string
	FinishedAt      time.Time
}

// AlgoRunListFilter controls list query for algo_runs.
type AlgoRunListFilter struct {
	AlgoName      string
	Status        string
	StartedAfter  *time.Time
	StartedBefore *time.Time
	Page          int
	PageSize      int
}

// AffectedAsset represents a single asset processed by an algo run.
type AffectedAsset struct {
	AssetID     string  `json:"asset_id"`
	AlgoName    string  `json:"algo_name"`
	AlgoVersion string  `json:"algo_version"`
	Status      string  `json:"status"`
	ResultTag   string  `json:"result_tag,omitempty"`
	ResultScore *float64 `json:"result_score,omitempty"`
	RunID       string  `json:"run_id,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AlgoRunRepository persists algo_runs rows.
type AlgoRunRepository interface {
	Insert(ctx context.Context, run *models.AlgoRun) error
	Get(ctx context.Context, runID string) (*models.AlgoRun, error)
	Exists(ctx context.Context, runID string) (bool, error)
	Start(ctx context.Context, runID string, startedAt time.Time) error
	Finish(ctx context.Context, runID string, patch AlgoRunFinishPatch) error
	List(ctx context.Context, filter AlgoRunListFilter) ([]*models.AlgoRun, int64, error)
	Cancel(ctx context.Context, runID, reason string, finishedAt time.Time) error
	GetAffectedAssets(ctx context.Context, runID string) ([]*AffectedAsset, error)
}
