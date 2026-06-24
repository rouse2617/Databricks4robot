package repository

import (
	"context"
	"errors"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// ErrPipelineConfigNotFound is returned when a config mutation targets a missing config.
var ErrPipelineConfigNotFound = errors.New("pipeline config not found")

// ErrPipelineConfigNameExists is returned when creating/updating a config
// with a name that already belongs to another config.
var ErrPipelineConfigNameExists = errors.New("pipeline config name already exists")

// PipelineConfigFilter scopes config library list queries.
type PipelineConfigFilter struct {
	Query     string
	Owner     string
	Scope     string
	Lifecycle string
}

// PipelineConfigRepository persists standalone config files and immutable versions.
type PipelineConfigRepository interface {
	Create(ctx context.Context, config *models.PipelineConfig, version *models.PipelineConfigVersion) error
	FindAll(ctx context.Context, filter *PipelineConfigFilter) ([]models.PipelineConfig, error)
	FindByID(ctx context.Context, id string) (*models.PipelineConfig, error)
	UpdateMetadata(ctx context.Context, config *models.PipelineConfig) error
	CreateVersion(ctx context.Context, configID string, version *models.PipelineConfigVersion) error
	UpdateVersionStatus(ctx context.Context, configID string, version int, status string) (*models.PipelineConfigVersion, error)
	UpdateLifecycle(ctx context.Context, configID string, lifecycle string) error
	FindVersion(ctx context.Context, configID string, version int) (*models.PipelineConfigVersion, error)
	FindVersions(ctx context.Context, configID string) ([]models.PipelineConfigVersion, error)
	Deprecate(ctx context.Context, id string) error
}
