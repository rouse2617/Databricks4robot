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

// PipelineComponentRepository defines persistence for pipeline components.
type PipelineComponentRepository interface {
	Save(ctx context.Context, c *models.PipelineComponent) error
	FindAll(ctx context.Context, filter *ComponentFilter) ([]models.PipelineComponent, error)
	FindByID(ctx context.Context, id string) (*models.PipelineComponent, error)
	Update(ctx context.Context, c *models.PipelineComponent) error
	Delete(ctx context.Context, id string) error
}
