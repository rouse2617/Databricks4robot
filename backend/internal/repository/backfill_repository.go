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
	IncrementCompleted(ctx context.Context, id string) error
	IncrementFailed(ctx context.Context, id string) error

	// Item operations.
	SaveItem(ctx context.Context, item *models.BackfillItem) error
	SaveItems(ctx context.Context, items []models.BackfillItem) error
	FindItemsByJobID(ctx context.Context, jobID string) ([]models.BackfillItem, error)
	FindItemByID(ctx context.Context, id string) (*models.BackfillItem, error)
	UpdateItemStatus(ctx context.Context, id, status, workflowName, errorMsg string) error
	CountItemsByStatus(ctx context.Context, jobID, status string) (int, error)
}
