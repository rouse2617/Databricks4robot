package repository

import (
	"context"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// PipelineTemplateRepository defines persistence operations for the
// pipeline_templates table.
type PipelineTemplateRepository interface {
	// Save inserts a pipeline template row. When ID is empty a new UUID is
	// assigned by the implementation.
	Save(ctx context.Context, t *models.PipelineTemplate) error

	// FindAll returns the latest version of each pipeline template ordered by
	// updated_at DESC.
	FindAll(ctx context.Context) ([]models.PipelineTemplate, error)

	// FindByID returns a single pipeline template by id, or (nil, nil) when
	// not found.
	FindByID(ctx context.Context, id string) (*models.PipelineTemplate, error)

	// FindVersionsByName returns all versions of a named pipeline template
	// ordered by version DESC.
	FindVersionsByName(ctx context.Context, name string) ([]models.PipelineTemplate, error)

	// GetNextVersion returns the next version number for a template name.
	GetNextVersion(ctx context.Context, name string) (int, error)

	// Delete removes a pipeline template by id. It is a no-op when the row
	// does not exist.
	Delete(ctx context.Context, id string) error
}

// PipelineDeploymentRepository defines persistence operations for the
// pipeline_deployments table.
type PipelineDeploymentRepository interface {
	// Save inserts or updates a pipeline deployment row. When ID is empty a new
	// UUID is assigned by the implementation.
	Save(ctx context.Context, d *models.PipelineDeployment) error

	// FindAll returns all pipeline deployments ordered by created_at DESC.
	FindAll(ctx context.Context) ([]models.PipelineDeployment, error)

	// FindByID returns a single pipeline deployment by id, or (nil, nil) when
	// not found.
	FindByID(ctx context.Context, id string) (*models.PipelineDeployment, error)

	// Delete removes a pipeline deployment by id. It is a no-op when the row
	// does not exist.
	Delete(ctx context.Context, id string) error

	// DeleteByTemplateID removes all pipeline deployments associated with a
	// template id. It is a no-op when no rows exist.
	DeleteByTemplateID(ctx context.Context, templateID string) error

	// UpdateStatus sets the status for a pipeline deployment. It is a no-op
	// when the row does not exist.
	UpdateStatus(ctx context.Context, id, status string) error
}
