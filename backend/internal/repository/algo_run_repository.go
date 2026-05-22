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

// AlgoRunRepository persists algo_runs rows.
type AlgoRunRepository interface {
	Insert(ctx context.Context, run *models.AlgoRun) error
	Get(ctx context.Context, runID string) (*models.AlgoRun, error)
	Exists(ctx context.Context, runID string) (bool, error)
	Start(ctx context.Context, runID string, startedAt time.Time) error
	Finish(ctx context.Context, runID string, patch AlgoRunFinishPatch) error
}
