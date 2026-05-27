package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// PipelineTemplateRepo persists rows to the pipeline_templates table.
type PipelineTemplateRepo struct {
	c *Client
}

// NewPipelineTemplateRepo creates a PipelineTemplateRepo bound to c.
func NewPipelineTemplateRepo(c *Client) *PipelineTemplateRepo { return &PipelineTemplateRepo{c: c} }

var _ repository.PipelineTemplateRepository = (*PipelineTemplateRepo)(nil)

const pipelineTemplateSelectCols = `id, name, version, pipeline, node_count, created_at, updated_at`

func scanPipelineTemplate(rs rowScanner) (*models.PipelineTemplate, error) {
	var (
		t            models.PipelineTemplate
		pipelineJSON []byte
	)
	if err := rs.Scan(
		&t.ID, &t.Name, &t.Version, &pipelineJSON, &t.NodeCount, &t.CreatedAt, &t.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if len(pipelineJSON) > 0 {
		_ = json.Unmarshal(pipelineJSON, &t.Pipeline)
	}
	if t.Pipeline == nil {
		t.Pipeline = map[string]interface{}{}
	}
	return &t, nil
}

// Save inserts or updates a pipeline template. When ID is empty a new UUID is
// assigned. The Pipeline JSONB is marshalled from the struct field.
func (r *PipelineTemplateRepo) Save(ctx context.Context, t *models.PipelineTemplate) error {
	if t == nil {
		return errors.New("postgres PipelineTemplateRepo.Save: nil template")
	}
	now := time.Now().UTC()
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now

	pipelineJSON, err := json.Marshal(t.Pipeline)
	if err != nil {
		return fmt.Errorf("postgres PipelineTemplateRepo.Save: marshal pipeline: %w", err)
	}
	if len(pipelineJSON) == 0 {
		pipelineJSON = []byte(`{}`)
	}

	const q = `
INSERT INTO pipeline_templates (id, name, pipeline, node_count, created_at, updated_at)
VALUES ($1, $2, $3::jsonb, $4, $5, $6)
ON CONFLICT (id) DO UPDATE SET
    name       = EXCLUDED.name,
    pipeline   = EXCLUDED.pipeline,
    node_count = EXCLUDED.node_count,
    updated_at = EXCLUDED.updated_at`

	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q, t.ID, t.Name, pipelineJSON, t.NodeCount, t.CreatedAt, t.UpdatedAt); err != nil {
		return fmt.Errorf("postgres PipelineTemplateRepo.Save: %w", err)
	}
	return nil
}

// FindAll returns all pipeline templates ordered by created_at DESC.
func (r *PipelineTemplateRepo) FindAll(ctx context.Context) ([]models.PipelineTemplate, error) {
	q := `SELECT ` + pipelineTemplateSelectCols + `
FROM pipeline_templates
ORDER BY created_at DESC`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("postgres PipelineTemplateRepo.FindAll: %w", err)
	}
	defer rows.Close()
	var out []models.PipelineTemplate
	for rows.Next() {
		t, err := scanPipelineTemplate(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres PipelineTemplateRepo.FindAll scan: %w", err)
		}
		out = append(out, *t)
	}
	return out, nil
}

// FindByID returns a pipeline template by id, or (nil, nil) when not found.
func (r *PipelineTemplateRepo) FindByID(ctx context.Context, id string) (*models.PipelineTemplate, error) {
	q := `SELECT ` + pipelineTemplateSelectCols + `
FROM pipeline_templates
WHERE id = $1`
	db := dbFromCtx(ctx, r.c.db)
	t, err := scanPipelineTemplate(db.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres PipelineTemplateRepo.FindByID: %w", err)
	}
	return t, nil
}

// Delete removes a pipeline template by id. It is a no-op when the row does
// not exist.
func (r *PipelineTemplateRepo) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM pipeline_templates WHERE id = $1`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q, id); err != nil {
		return fmt.Errorf("postgres PipelineTemplateRepo.Delete: %w", err)
	}
	return nil
}

// FindVersionsByName returns all versions of a named pipeline template
// ordered by version DESC.
func (r *PipelineTemplateRepo) FindVersionsByName(ctx context.Context, name string) ([]models.PipelineTemplate, error) {
	q := `SELECT ` + pipelineTemplateSelectCols + `
	FROM pipeline_templates
	WHERE name = $1
	ORDER BY version DESC`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q, name)
	if err != nil {
		return nil, fmt.Errorf("postgres PipelineTemplateRepo.FindVersionsByName: %w", err)
	}
	defer rows.Close()
	var out []models.PipelineTemplate
	for rows.Next() {
		t, err := scanPipelineTemplate(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres PipelineTemplateRepo.FindVersionsByName scan: %w", err)
		}
		out = append(out, *t)
	}
	return out, nil
}

// GetNextVersion returns the next version number for a template name.
func (r *PipelineTemplateRepo) GetNextVersion(ctx context.Context, name string) (int, error) {
	const q = `SELECT COALESCE(MAX(version), 0) + 1 FROM pipeline_templates WHERE name = $1`
	db := dbFromCtx(ctx, r.c.db)
	var v int
	if err := db.QueryRow(ctx, q, name).Scan(&v); err != nil {
		return 0, fmt.Errorf("postgres PipelineTemplateRepo.GetNextVersion: %w", err)
	}
	return v, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// PipelineDeploymentRepo
// ──────────────────────────────────────────────────────────────────────────────

// PipelineDeploymentRepo persists rows to the pipeline_deployments table.
type PipelineDeploymentRepo struct {
	c *Client
}

// NewPipelineDeploymentRepo creates a PipelineDeploymentRepo bound to c.
func NewPipelineDeploymentRepo(c *Client) *PipelineDeploymentRepo {
	return &PipelineDeploymentRepo{c: c}
}

var _ repository.PipelineDeploymentRepository = (*PipelineDeploymentRepo)(nil)

const pipelineDeploymentSelectCols = `id, template_id, pipeline_name, workflow_name,
  status, node_count, manifest, pipeline_json, created_at, updated_at, finished_at`

func scanPipelineDeployment(rs rowScanner) (*models.PipelineDeployment, error) {
	var (
		d            models.PipelineDeployment
		manifest     *string
		pipelineJSON []byte
	)
	if err := rs.Scan(
		&d.ID, &d.TemplateID, &d.PipelineName, &d.WorkflowName,
		&d.Status, &d.NodeCount, &manifest, &pipelineJSON, &d.CreatedAt, &d.UpdatedAt, &d.FinishedAt,
	); err != nil {
		return nil, err
	}
	if manifest != nil {
		d.Manifest = manifest
	}
	if len(pipelineJSON) > 0 {
		_ = json.Unmarshal(pipelineJSON, &d.PipelineJSON)
	}
	return &d, nil
}

// Save inserts or updates a pipeline deployment. When ID is empty a new UUID is
// assigned. The PipelineJSON is marshalled from the struct field.
func (r *PipelineDeploymentRepo) Save(ctx context.Context, d *models.PipelineDeployment) error {
	if d == nil {
		return errors.New("postgres PipelineDeploymentRepo.Save: nil deployment")
	}
	now := time.Now().UTC()
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}

	var pipelineJSON []byte
	if d.PipelineJSON != nil {
		var err error
		pipelineJSON, err = json.Marshal(d.PipelineJSON)
		if err != nil {
			return fmt.Errorf("postgres PipelineDeploymentRepo.Save: marshal pipeline_json: %w", err)
		}
	}

	var manifest interface{}
	if d.Manifest != nil {
		manifest = *d.Manifest
	}

	const q = `
INSERT INTO pipeline_deployments (id, template_id, pipeline_name, workflow_name, status, node_count, manifest, pipeline_json, created_at, updated_at, finished_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10, $11)
ON CONFLICT (id) DO UPDATE SET
    template_id   = EXCLUDED.template_id,
    pipeline_name = EXCLUDED.pipeline_name,
    workflow_name = EXCLUDED.workflow_name,
    status        = EXCLUDED.status,
    node_count    = EXCLUDED.node_count,
    manifest      = EXCLUDED.manifest,
    pipeline_json = EXCLUDED.pipeline_json,
    updated_at    = EXCLUDED.updated_at,
    finished_at   = EXCLUDED.finished_at`

	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q,
		d.ID, d.TemplateID, d.PipelineName, d.WorkflowName, d.Status, d.NodeCount,
		manifest, pipelineJSON, d.CreatedAt, now, d.FinishedAt,
	); err != nil {
		return fmt.Errorf("postgres PipelineDeploymentRepo.Save: %w", err)
	}
	return nil
}

// FindAll returns all pipeline deployments ordered by created_at DESC.
func (r *PipelineDeploymentRepo) FindAll(ctx context.Context) ([]models.PipelineDeployment, error) {
	q := `SELECT ` + pipelineDeploymentSelectCols + `
FROM pipeline_deployments
ORDER BY created_at DESC`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("postgres PipelineDeploymentRepo.FindAll: %w", err)
	}
	defer rows.Close()
	var out []models.PipelineDeployment
	for rows.Next() {
		d, err := scanPipelineDeployment(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres PipelineDeploymentRepo.FindAll scan: %w", err)
		}
		out = append(out, *d)
	}
	return out, nil
}

// FindByID returns a pipeline deployment by id, or (nil, nil) when not found.
func (r *PipelineDeploymentRepo) FindByID(ctx context.Context, id string) (*models.PipelineDeployment, error) {
	q := `SELECT ` + pipelineDeploymentSelectCols + `
FROM pipeline_deployments
WHERE id = $1`
	db := dbFromCtx(ctx, r.c.db)
	d, err := scanPipelineDeployment(db.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres PipelineDeploymentRepo.FindByID: %w", err)
	}
	return d, nil
}

// Delete removes a pipeline deployment by id. It is a no-op when the row does
// not exist.
func (r *PipelineDeploymentRepo) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM pipeline_deployments WHERE id = $1`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q, id); err != nil {
		return fmt.Errorf("postgres PipelineDeploymentRepo.Delete: %w", err)
	}
	return nil
}

// UpdateStatus sets the status for a pipeline deployment. It is a no-op when
// the row does not exist.
func (r *PipelineDeploymentRepo) UpdateStatus(ctx context.Context, id, status string) error {
	const q = `UPDATE pipeline_deployments SET status = $2 WHERE id = $1`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q, id, status); err != nil {
		return fmt.Errorf("postgres PipelineDeploymentRepo.UpdateStatus: %w", err)
	}
	return nil
}
