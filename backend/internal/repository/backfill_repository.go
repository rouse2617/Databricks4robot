package repository

import (
	"context"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// BackfillRepository defines persistence operations for backfill jobs and items.
type BackfillRepository interface {
	// Job operations.
	SaveJob(ctx context.Context, j *models.BackfillJob) error
	FindAllJobs(ctx context.Context) ([]models.BackfillJob, error)
	FindJobByID(ctx context.Context, id string) (*models.BackfillJob, error)
	UpdateJobStatus(ctx context.Context, id, status string) error
	UpdateJobPilotPhase(ctx context.Context, id, status, pilotPhase string) error
	IncrementCompleted(ctx context.Context, id string) error
	IncrementFailed(ctx context.Context, id string) error

	// Item operations.
	SaveItem(ctx context.Context, item *models.BackfillItem) error
	SaveItems(ctx context.Context, items []models.BackfillItem) error
	FindItemsByJobID(ctx context.Context, jobID string) ([]models.BackfillItem, error)
	FindItemByID(ctx context.Context, id string) (*models.BackfillItem, error)
	FindItemByPipelineRunID(ctx context.Context, pipelineRunID string) (*models.BackfillItem, error)
	UpdateItemStatus(ctx context.Context, id, status, workflowName, errorMsg string) error
	UpdateItemPipelineRun(ctx context.Context, id, pipelineRunID, workflowName, status string) error
	UpdateJobProgress(ctx context.Context, id string, completed, failed int, status string) error
	CountItemsByStatus(ctx context.Context, jobID, status string) (int, error)
	SummarizeItemStatuses(ctx context.Context, jobID string) (BackfillItemStatusSummary, error)
	FindItemsByJobIDWithStatuses(ctx context.Context, jobID string, statuses []string) ([]models.BackfillItem, error)
	FindItemsMissingPipelineRun(ctx context.Context, jobID string) ([]models.BackfillItem, error)
	FindItemsByScope(ctx context.Context, filter BackfillRerunItemFilter) ([]models.BackfillItem, error)
	PrepareItemsForRerun(ctx context.Context, itemIDs []string) error
	AggregateNodeStatusByBatchJobID(ctx context.Context, jobID string) ([]BatchNodeStatusAggregate, error)
	ListNodeFailures(ctx context.Context, filter BatchNodeFailureFilter) (*models.BatchNodeFailureListResult, error)
	CountPipelineRunsByBatchJobID(ctx context.Context, jobID string) (int, error)
	CountRunsWithNodeRowsByBatchJobID(ctx context.Context, jobID string) (int, error)
	FindItemsByAssetID(ctx context.Context, assetID string) ([]models.BackfillItem, error)
	FindItemByJobAndAssetID(ctx context.Context, jobID, assetID string) (*models.BackfillItem, error)
}

// BackfillItemStatusSummary aggregates item counts by coarse status bucket.
type BackfillItemStatusSummary struct {
	Completed int
	Failed    int
	Pending   int
	Running   int
}

type BackfillRerunItemFilter struct {
	JobID          string
	Statuses       []string
	ItemIDs        []string
	AssetIDs       []string
	PipelineNodeID string
	NodeStatuses   []string
}

type BatchNodeStatusAggregate struct {
	PipelineNodeID string
	DisplayName    string
	Status         string
	Count          int
}

type BatchNodeFailureFilter struct {
	JobID          string
	PipelineNodeID string
	Statuses       []string
	Page           int
	PageSize       int
	Query          string
}
