package repository

import (
	"context"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// DatabrewRunRepository defines persistence for the unified runs table.
type DatabrewRunRepository interface {
	Save(ctx context.Context, run *models.DatabrewRun) error
	FindByID(ctx context.Context, id string) (*models.DatabrewRun, error)
	FindByRuntimeResource(ctx context.Context, namespace, resourceName string) (*models.DatabrewRun, error)
	List(ctx context.Context, filter models.DatabrewRunListFilter) ([]models.DatabrewRun, int, error)
}

// ComponentBuildRunRepository stores component build extension rows.
type ComponentBuildRunRepository interface {
	Save(ctx context.Context, row *models.ComponentBuildRun) error
	FindByRunID(ctx context.Context, runID string) (*models.ComponentBuildRun, error)
}

// RAGBuildRunRepository stores RAG build extension rows.
type RAGBuildRunRepository interface {
	Save(ctx context.Context, row *models.RAGBuildRun) error
	FindByRunID(ctx context.Context, runID string) (*models.RAGBuildRun, error)
}

// ComponentReleaseRepository stores built component releases.
type ComponentReleaseRepository interface {
	Save(ctx context.Context, release *models.ComponentRelease) error
	ListByComponentID(ctx context.Context, componentID string) ([]models.ComponentRelease, error)
}
