package repository

import (
	"context"
	"time"
)

type SearchReindexJobStatus string

const (
	SearchReindexJobStatusQueued    SearchReindexJobStatus = "queued"
	SearchReindexJobStatusRunning   SearchReindexJobStatus = "running"
	SearchReindexJobStatusPaused    SearchReindexJobStatus = "paused"
	SearchReindexJobStatusSucceeded SearchReindexJobStatus = "succeeded"
	SearchReindexJobStatusFailed    SearchReindexJobStatus = "failed"
	SearchReindexJobStatusAbandoned SearchReindexJobStatus = "abandoned"
)

type SearchReindexJob struct {
	ID                    string
	Status                SearchReindexJobStatus
	DryRun                bool
	PageSize              int
	NextPage              int
	StopRequested         bool
	TotalAssets           int64
	AssetsScanned         int64
	DocumentsIndexed      int64
	DocumentsDeleted      int64
	Failed                int64
	Error                 string
	ErrorSamples          []string
	ElasticsearchDocCount *int64
	IndexCleared          bool
	CreatedAt             time.Time
	UpdatedAt             time.Time
	StartedAt             *time.Time
	FinishedAt            *time.Time
}

type SearchReindexJobRepository interface {
	Create(ctx context.Context, dryRun bool, pageSize int) (*SearchReindexJob, error)
	Get(ctx context.Context, jobID string) (*SearchReindexJob, error)
	ListRecent(ctx context.Context, limit int) ([]*SearchReindexJob, error)
	ClaimForRun(ctx context.Context, jobID string) (*SearchReindexJob, error)
	RequestStop(ctx context.Context, jobID string) (*SearchReindexJob, error)
	Resume(ctx context.Context, jobID string) (*SearchReindexJob, error)
	IsStopRequested(ctx context.Context, jobID string) (bool, error)
	UpdateProgress(ctx context.Context, jobID string, progress SearchReindexJobProgress) error
	MarkPaused(ctx context.Context, jobID string) error
	MarkFailed(ctx context.Context, jobID string, errorMsg string, errorSamples []string) error
	MarkSucceeded(ctx context.Context, jobID string, esDocCount *int64) error
	MarkAbandoned(ctx context.Context, jobID string) error
}

type SearchReindexJobProgress struct {
	TotalAssets      int64
	NextPage         int
	AssetsScanned    int64
	DocumentsIndexed int64
	DocumentsDeleted int64
	Failed           int64
	ErrorSamples     []string
	IndexCleared     bool
}
