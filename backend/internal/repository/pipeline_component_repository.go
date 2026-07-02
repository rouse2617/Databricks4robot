package repository

import (
	"context"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// ComponentFilter holds optional search/filter parameters for FindAll.
type ComponentFilter struct {
	Query  string // name search, empty = no filter
	Source string // source filter, empty = no filter
}

// ComponentReleaseFilter holds optional release list filters.
type ComponentReleaseFilter struct {
	Query       string
	ComponentID string
	TaskName    string
	Status      string
	Channel     string
	Selectable  *bool
}

// PipelineComponentRepository defines persistence for pipeline components.
type PipelineComponentRepository interface {
	Save(ctx context.Context, c *models.PipelineComponent) error
	FindAll(ctx context.Context, filter *ComponentFilter) ([]models.PipelineComponent, error)
	FindByID(ctx context.Context, id string) (*models.PipelineComponent, error)
	Update(ctx context.Context, c *models.PipelineComponent) error
	Delete(ctx context.Context, id string) error
}

// PipelineComponentReleaseRepository defines persistence for generated
// component releases. It is separate from the legacy component CRUD interface
// so release ingestion can evolve independently of hand-authored components.
type PipelineComponentReleaseRepository interface {
	UpsertRelease(ctx context.Context, release *models.PipelineComponentRelease) error
	FindReleases(ctx context.Context, filter *ComponentReleaseFilter) ([]models.PipelineComponentRelease, error)
	FindReleaseByID(ctx context.Context, id string) (*models.PipelineComponentRelease, error)
}
