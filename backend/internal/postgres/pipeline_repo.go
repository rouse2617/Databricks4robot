package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

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
INSERT INTO pipeline_templates (id, name, version, pipeline, node_count, created_at, updated_at)
VALUES ($1, $2, $7, $3::jsonb, $4, $5, $6)
ON CONFLICT (id) DO UPDATE SET
    name       = EXCLUDED.name,
    version    = EXCLUDED.version,
    pipeline   = EXCLUDED.pipeline,
    node_count = EXCLUDED.node_count,
    updated_at = EXCLUDED.updated_at`

	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q, t.ID, t.Name, pipelineJSON, t.NodeCount, t.CreatedAt, t.UpdatedAt, t.Version); err != nil {
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
		templateID   *string
		manifest     *string
		pipelineJSON []byte
	)
	if err := rs.Scan(
		&d.ID, &templateID, &d.PipelineName, &d.WorkflowName,
		&d.Status, &d.NodeCount, &manifest, &pipelineJSON, &d.CreatedAt, &d.UpdatedAt, &d.FinishedAt,
	); err != nil {
		return nil, err
	}
	if manifest != nil {
		d.Manifest = manifest
	}
	if templateID != nil {
		d.TemplateID = templateID
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

	var templateID any
	if d.TemplateID != nil {
		templateID = *d.TemplateID
	}

	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q,
		d.ID, templateID, d.PipelineName, d.WorkflowName, d.Status, d.NodeCount,
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

// DeleteByTemplateID removes all pipeline deployments for a pipeline template.
func (r *PipelineDeploymentRepo) DeleteByTemplateID(ctx context.Context, templateID string) error {
	const q = `DELETE FROM pipeline_deployments WHERE template_id = $1`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q, templateID); err != nil {
		return fmt.Errorf("postgres PipelineDeploymentRepo.DeleteByTemplateID: %w", err)
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

// ──────────────────────────────────────────────────────────────────────────────
// ExecutionTargetRepo
// ──────────────────────────────────────────────────────────────────────────────

// ExecutionTargetRepo persists rows to the execution_targets table.
type ExecutionTargetRepo struct {
	c *Client
}

// NewExecutionTargetRepo creates an ExecutionTargetRepo bound to c.
func NewExecutionTargetRepo(c *Client) *ExecutionTargetRepo { return &ExecutionTargetRepo{c: c} }

var _ repository.ExecutionTargetRepository = (*ExecutionTargetRepo)(nil)

const executionTargetSelectCols = `id, name, description, cluster, namespace, service_account,
  argo_server_url, argo_auth_secret_ref, argo_insecure_skip_verify, argo_ca_cert_ref,
  enabled, status, is_default, resource_defaults, quota_policy, labels, created_at, updated_at`

func mapFromJSON(raw []byte) map[string]interface{} {
	if len(raw) == 0 {
		return map[string]interface{}{}
	}
	out := map[string]interface{}{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]interface{}{}
	}
	return out
}

func marshalMapForJSONB(v map[string]interface{}) ([]byte, error) {
	if v == nil {
		return []byte(`{}`), nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return []byte(`{}`), nil
	}
	return raw, nil
}

func scanExecutionTarget(rs rowScanner) (*models.ExecutionTarget, error) {
	var (
		t                models.ExecutionTarget
		resourceDefaults []byte
		quotaPolicy      []byte
		labels           []byte
	)
	if err := rs.Scan(
		&t.ID, &t.Name, &t.Description, &t.Cluster, &t.Namespace, &t.ServiceAccount,
		&t.ArgoServerURL, &t.ArgoAuthSecretRef, &t.ArgoInsecureSkipTLS, &t.ArgoCACertRef,
		&t.Enabled, &t.Status, &t.IsDefault, &resourceDefaults, &quotaPolicy, &labels,
		&t.CreatedAt, &t.UpdatedAt,
	); err != nil {
		return nil, err
	}
	t.ArgoServerConfigured = t.ArgoServerURL != "" || t.Status == "available"
	t.ResourceDefaults = mapFromJSON(resourceDefaults)
	t.QuotaPolicy = mapFromJSON(quotaPolicy)
	t.Labels = mapFromJSON(labels)
	return &t, nil
}

// Save inserts or updates an execution target.
func (r *ExecutionTargetRepo) Save(ctx context.Context, t *models.ExecutionTarget) error {
	if t == nil {
		return errors.New("postgres ExecutionTargetRepo.Save: nil target")
	}
	now := time.Now().UTC()
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now
	resourceDefaults, err := marshalMapForJSONB(t.ResourceDefaults)
	if err != nil {
		return fmt.Errorf("postgres ExecutionTargetRepo.Save: marshal resource_defaults: %w", err)
	}
	quotaPolicy, err := marshalMapForJSONB(t.QuotaPolicy)
	if err != nil {
		return fmt.Errorf("postgres ExecutionTargetRepo.Save: marshal quota_policy: %w", err)
	}
	labels, err := marshalMapForJSONB(t.Labels)
	if err != nil {
		return fmt.Errorf("postgres ExecutionTargetRepo.Save: marshal labels: %w", err)
	}

	const q = `
INSERT INTO execution_targets (
  id, name, description, cluster, namespace, service_account,
  argo_server_url, argo_auth_secret_ref, argo_insecure_skip_verify, argo_ca_cert_ref,
  enabled, status, is_default, resource_defaults, quota_policy, labels, created_at, updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6,
  $7, $8, $9, $10,
  $11, $12, $13, $14::jsonb, $15::jsonb, $16::jsonb, $17, $18
)
ON CONFLICT (id) DO UPDATE SET
  name = EXCLUDED.name,
  description = EXCLUDED.description,
  cluster = EXCLUDED.cluster,
  namespace = EXCLUDED.namespace,
  service_account = EXCLUDED.service_account,
  argo_server_url = EXCLUDED.argo_server_url,
  argo_auth_secret_ref = EXCLUDED.argo_auth_secret_ref, -- pragma: allowlist secret
  argo_insecure_skip_verify = EXCLUDED.argo_insecure_skip_verify,
  argo_ca_cert_ref = EXCLUDED.argo_ca_cert_ref,
  enabled = EXCLUDED.enabled,
  status = EXCLUDED.status,
  is_default = EXCLUDED.is_default,
  resource_defaults = EXCLUDED.resource_defaults,
  quota_policy = EXCLUDED.quota_policy,
  labels = EXCLUDED.labels,
  updated_at = EXCLUDED.updated_at`

	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q,
		t.ID, t.Name, t.Description, t.Cluster, t.Namespace, t.ServiceAccount,
		t.ArgoServerURL, t.ArgoAuthSecretRef, t.ArgoInsecureSkipTLS, t.ArgoCACertRef,
		t.Enabled, t.Status, t.IsDefault, resourceDefaults, quotaPolicy, labels,
		t.CreatedAt, t.UpdatedAt,
	); err != nil {
		return fmt.Errorf("postgres ExecutionTargetRepo.Save: %w", err)
	}
	return nil
}

// FindAll returns enabled and disabled execution targets ordered by default first.
func (r *ExecutionTargetRepo) FindAll(ctx context.Context) ([]models.ExecutionTarget, error) {
	q := `SELECT ` + executionTargetSelectCols + `
FROM execution_targets
ORDER BY is_default DESC, name ASC`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("postgres ExecutionTargetRepo.FindAll: %w", err)
	}
	defer rows.Close()
	var out []models.ExecutionTarget
	for rows.Next() {
		t, err := scanExecutionTarget(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres ExecutionTargetRepo.FindAll scan: %w", err)
		}
		out = append(out, *t)
	}
	return out, nil
}

// FindByID returns an execution target by id, or (nil, nil) when not found.
func (r *ExecutionTargetRepo) FindByID(ctx context.Context, id string) (*models.ExecutionTarget, error) {
	q := `SELECT ` + executionTargetSelectCols + `
FROM execution_targets
WHERE id = $1`
	db := dbFromCtx(ctx, r.c.db)
	t, err := scanExecutionTarget(db.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres ExecutionTargetRepo.FindByID: %w", err)
	}
	return t, nil
}

// FindDefault returns the default execution target, or (nil, nil) when none exists.
func (r *ExecutionTargetRepo) FindDefault(ctx context.Context) (*models.ExecutionTarget, error) {
	q := `SELECT ` + executionTargetSelectCols + `
FROM execution_targets
WHERE is_default
ORDER BY updated_at DESC
LIMIT 1`
	db := dbFromCtx(ctx, r.c.db)
	t, err := scanExecutionTarget(db.QueryRow(ctx, q))
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres ExecutionTargetRepo.FindDefault: %w", err)
	}
	return t, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// PipelineRunRepo
// ──────────────────────────────────────────────────────────────────────────────

// PipelineRunRepo persists rows to the pipeline_runs table.
type PipelineRunRepo struct {
	c *Client
}

// NewPipelineRunRepo creates a PipelineRunRepo bound to c.
func NewPipelineRunRepo(c *Client) *PipelineRunRepo { return &PipelineRunRepo{c: c} }

var _ repository.PipelineRunRepository = (*PipelineRunRepo)(nil)

const pipelineRunSelectCols = `id, template_id, pipeline_name, template_version, workflow_name,
  execution_target_id, target_snapshot, status, node_count, asset_ids, asset_count, no_asset_run,
  manifest, pipeline_json, argo_namespace, argo_workflow_uid, message,
  created_at, updated_at, started_at, finished_at`

func scanPipelineRun(rs rowScanner) (*models.PipelineRun, error) {
	var (
		r              models.PipelineRun
		templateID     *string
		templateVer    *int
		targetSnapshot []byte
		assetIDs       []string
		manifest       *string
		pipelineJSON   []byte
	)
	if err := rs.Scan(
		&r.ID, &templateID, &r.PipelineName, &templateVer, &r.WorkflowName,
		&r.ExecutionTargetID, &targetSnapshot, &r.Status, &r.NodeCount, &assetIDs, &r.AssetCount, &r.NoAssetRun,
		&manifest, &pipelineJSON, &r.ArgoNamespace, &r.ArgoWorkflowUID, &r.Message,
		&r.CreatedAt, &r.UpdatedAt, &r.StartedAt, &r.FinishedAt,
	); err != nil {
		return nil, err
	}
	r.TemplateID = templateID
	r.TemplateVersion = templateVer
	r.AssetIDs = assetIDs
	r.Manifest = manifest
	r.TargetSnapshot = mapFromJSON(targetSnapshot)
	r.PipelineJSON = mapFromJSON(pipelineJSON)
	return &r, nil
}

// Save inserts or updates a pipeline run.
func (r *PipelineRunRepo) Save(ctx context.Context, run *models.PipelineRun) error {
	if run == nil {
		return errors.New("postgres PipelineRunRepo.Save: nil run")
	}
	now := time.Now().UTC()
	if run.ID == "" {
		run.ID = uuid.New().String()
	}
	if run.CreatedAt.IsZero() {
		run.CreatedAt = now
	}
	run.UpdatedAt = now
	if run.AssetIDs == nil {
		run.AssetIDs = []string{}
	}
	if run.AssetCount == 0 {
		run.AssetCount = len(run.AssetIDs)
	}
	run.NoAssetRun = len(run.AssetIDs) == 0
	targetSnapshot, err := marshalMapForJSONB(run.TargetSnapshot)
	if err != nil {
		return fmt.Errorf("postgres PipelineRunRepo.Save: marshal target_snapshot: %w", err)
	}
	pipelineJSON, err := marshalMapForJSONB(run.PipelineJSON)
	if err != nil {
		return fmt.Errorf("postgres PipelineRunRepo.Save: marshal pipeline_json: %w", err)
	}
	var templateID any
	if run.TemplateID != nil {
		templateID = *run.TemplateID
	}
	var templateVersion any
	if run.TemplateVersion != nil {
		templateVersion = *run.TemplateVersion
	}
	var manifest any
	if run.Manifest != nil {
		manifest = *run.Manifest
	}

	const q = `
INSERT INTO pipeline_runs (
  id, template_id, pipeline_name, template_version, workflow_name,
  execution_target_id, target_snapshot, status, node_count, asset_ids, asset_count, no_asset_run,
  manifest, pipeline_json, argo_namespace, argo_workflow_uid, message,
  created_at, updated_at, started_at, finished_at
) VALUES (
  $1, $2, $3, $4, $5,
  $6, $7::jsonb, $8, $9, $10::text[], $11, $12,
  $13, $14::jsonb, $15, $16, $17,
  $18, $19, $20, $21
)
ON CONFLICT (id) DO UPDATE SET
  template_id = EXCLUDED.template_id,
  pipeline_name = EXCLUDED.pipeline_name,
  template_version = EXCLUDED.template_version,
  workflow_name = EXCLUDED.workflow_name,
  execution_target_id = EXCLUDED.execution_target_id,
  target_snapshot = EXCLUDED.target_snapshot,
  status = EXCLUDED.status,
  node_count = EXCLUDED.node_count,
  asset_ids = EXCLUDED.asset_ids,
  asset_count = EXCLUDED.asset_count,
  no_asset_run = EXCLUDED.no_asset_run,
  manifest = EXCLUDED.manifest,
  pipeline_json = EXCLUDED.pipeline_json,
  argo_namespace = EXCLUDED.argo_namespace,
  argo_workflow_uid = EXCLUDED.argo_workflow_uid,
  message = EXCLUDED.message,
  updated_at = EXCLUDED.updated_at,
  started_at = EXCLUDED.started_at,
  finished_at = EXCLUDED.finished_at`

	db := dbFromCtx(ctx, r.c.db)
	assetIDs := pgtype.FlatArray[string](run.AssetIDs)
	if err := db.Exec(ctx, q,
		run.ID, templateID, run.PipelineName, templateVersion, run.WorkflowName,
		run.ExecutionTargetID, targetSnapshot, run.Status, run.NodeCount, assetIDs, run.AssetCount, run.NoAssetRun,
		manifest, pipelineJSON, run.ArgoNamespace, run.ArgoWorkflowUID, run.Message,
		run.CreatedAt, run.UpdatedAt, run.StartedAt, run.FinishedAt,
	); err != nil {
		return fmt.Errorf("postgres PipelineRunRepo.Save: %w", err)
	}
	return nil
}

// FindAll returns pipeline runs ordered by created_at DESC.
func (r *PipelineRunRepo) FindAll(ctx context.Context) ([]models.PipelineRun, error) {
	q := `SELECT ` + pipelineRunSelectCols + `
FROM pipeline_runs
ORDER BY created_at DESC`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("postgres PipelineRunRepo.FindAll: %w", err)
	}
	defer rows.Close()
	var out []models.PipelineRun
	for rows.Next() {
		run, err := scanPipelineRun(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres PipelineRunRepo.FindAll scan: %w", err)
		}
		out = append(out, *run)
	}
	return out, nil
}

func (r *PipelineRunRepo) findOne(ctx context.Context, q, arg string, op string) (*models.PipelineRun, error) {
	db := dbFromCtx(ctx, r.c.db)
	run, err := scanPipelineRun(db.QueryRow(ctx, q, arg))
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres PipelineRunRepo.%s: %w", op, err)
	}
	return run, nil
}

// FindByID returns a pipeline run by id, or (nil, nil) when not found.
func (r *PipelineRunRepo) FindByID(ctx context.Context, id string) (*models.PipelineRun, error) {
	q := `SELECT ` + pipelineRunSelectCols + `
FROM pipeline_runs
WHERE id = $1`
	return r.findOne(ctx, q, id, "FindByID")
}

// FindByWorkflowName returns a pipeline run by Argo workflow name.
func (r *PipelineRunRepo) FindByWorkflowName(ctx context.Context, workflowName string) (*models.PipelineRun, error) {
	q := `SELECT ` + pipelineRunSelectCols + `
FROM pipeline_runs
WHERE workflow_name = $1`
	return r.findOne(ctx, q, workflowName, "FindByWorkflowName")
}

// Delete removes a pipeline run by id.
func (r *PipelineRunRepo) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM pipeline_runs WHERE id = $1`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q, id); err != nil {
		return fmt.Errorf("postgres PipelineRunRepo.Delete: %w", err)
	}
	return nil
}

// DeleteByTemplateID removes all pipeline runs associated with a template id.
func (r *PipelineRunRepo) DeleteByTemplateID(ctx context.Context, templateID string) error {
	const q = `DELETE FROM pipeline_runs WHERE template_id = $1`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q, templateID); err != nil {
		return fmt.Errorf("postgres PipelineRunRepo.DeleteByTemplateID: %w", err)
	}
	return nil
}

// UpdateStatus sets run status and optional finished_at timestamp.
func (r *PipelineRunRepo) UpdateStatus(ctx context.Context, id, status string, finishedAt *time.Time) error {
	const q = `
UPDATE pipeline_runs
SET status = $2, finished_at = COALESCE($3, finished_at), updated_at = NOW()
WHERE id = $1`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q, id, status, finishedAt); err != nil {
		return fmt.Errorf("postgres PipelineRunRepo.UpdateStatus: %w", err)
	}
	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// PipelineRunNodeRepo
// ──────────────────────────────────────────────────────────────────────────────

// PipelineRunNodeRepo persists rows to the pipeline_run_nodes table.
type PipelineRunNodeRepo struct {
	c *Client
}

// NewPipelineRunNodeRepo creates a PipelineRunNodeRepo bound to c.
func NewPipelineRunNodeRepo(c *Client) *PipelineRunNodeRepo { return &PipelineRunNodeRepo{c: c} }

var _ repository.PipelineRunNodeRepository = (*PipelineRunNodeRepo)(nil)

const pipelineRunNodeSelectCols = `id, run_id, pipeline_node_id, argo_node_id, argo_node_name,
  display_name, template_name, type, phase, message, pod_name, host_node_name, children,
  inputs, outputs, resources_duration, resource_summary, log_ref,
  started_at, finished_at, created_at, updated_at`

func scanPipelineRunNode(rs rowScanner) (*models.PipelineRunNode, error) {
	var (
		n                 models.PipelineRunNode
		children          []string
		inputs            []byte
		outputs           []byte
		resourcesDuration []byte
		resourceSummary   []byte
	)
	if err := rs.Scan(
		&n.ID, &n.RunID, &n.PipelineNodeID, &n.ArgoNodeID, &n.ArgoNodeName,
		&n.DisplayName, &n.TemplateName, &n.Type, &n.Phase, &n.Message, &n.PodName, &n.HostNodeName, &children,
		&inputs, &outputs, &resourcesDuration, &resourceSummary, &n.LogRef,
		&n.StartedAt, &n.FinishedAt, &n.CreatedAt, &n.UpdatedAt,
	); err != nil {
		return nil, err
	}
	n.Children = children
	n.Inputs = mapFromJSON(inputs)
	n.Outputs = mapFromJSON(outputs)
	n.ResourcesDuration = mapFromJSON(resourcesDuration)
	n.ResourceSummary = mapFromJSON(resourceSummary)
	return &n, nil
}

// ReplaceByRunID replaces a run's node snapshot.
func (r *PipelineRunNodeRepo) ReplaceByRunID(ctx context.Context, runID string, nodes []models.PipelineRunNode) error {
	if err := r.DeleteByRunID(ctx, runID); err != nil {
		return err
	}
	if len(nodes) == 0 {
		return nil
	}
	const q = `
INSERT INTO pipeline_run_nodes (
  id, run_id, pipeline_node_id, argo_node_id, argo_node_name,
  display_name, template_name, type, phase, message, pod_name, host_node_name, children,
  inputs, outputs, resources_duration, resource_summary, log_ref,
  started_at, finished_at, created_at, updated_at
) VALUES (
  $1, $2, $3, $4, $5,
  $6, $7, $8, $9, $10, $11, $12, $13,
  $14::jsonb, $15::jsonb, $16::jsonb, $17::jsonb, $18,
  $19, $20, $21, $22
)`
	db := dbFromCtx(ctx, r.c.db)
	now := time.Now().UTC()
	for i := range nodes {
		n := &nodes[i]
		if n.ID == "" {
			n.ID = uuid.New().String()
		}
		if n.RunID == "" {
			n.RunID = runID
		}
		if n.CreatedAt.IsZero() {
			n.CreatedAt = now
		}
		n.UpdatedAt = now
		inputs, err := marshalMapForJSONB(n.Inputs)
		if err != nil {
			return fmt.Errorf("postgres PipelineRunNodeRepo.ReplaceByRunID marshal inputs: %w", err)
		}
		outputs, err := marshalMapForJSONB(n.Outputs)
		if err != nil {
			return fmt.Errorf("postgres PipelineRunNodeRepo.ReplaceByRunID marshal outputs: %w", err)
		}
		resourcesDuration, err := marshalMapForJSONB(n.ResourcesDuration)
		if err != nil {
			return fmt.Errorf("postgres PipelineRunNodeRepo.ReplaceByRunID marshal resources_duration: %w", err)
		}
		resourceSummary, err := marshalMapForJSONB(n.ResourceSummary)
		if err != nil {
			return fmt.Errorf("postgres PipelineRunNodeRepo.ReplaceByRunID marshal resource_summary: %w", err)
		}
		if err := db.Exec(ctx, q,
			n.ID, n.RunID, n.PipelineNodeID, n.ArgoNodeID, n.ArgoNodeName,
			n.DisplayName, n.TemplateName, n.Type, n.Phase, n.Message, n.PodName, n.HostNodeName, n.Children,
			inputs, outputs, resourcesDuration, resourceSummary, n.LogRef,
			n.StartedAt, n.FinishedAt, n.CreatedAt, n.UpdatedAt,
		); err != nil {
			return fmt.Errorf("postgres PipelineRunNodeRepo.ReplaceByRunID insert: %w", err)
		}
	}
	return nil
}

// FindByRunID returns node snapshots for a run.
func (r *PipelineRunNodeRepo) FindByRunID(ctx context.Context, runID string) ([]models.PipelineRunNode, error) {
	q := `SELECT ` + pipelineRunNodeSelectCols + `
FROM pipeline_run_nodes
WHERE run_id = $1
ORDER BY created_at ASC`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q, runID)
	if err != nil {
		return nil, fmt.Errorf("postgres PipelineRunNodeRepo.FindByRunID: %w", err)
	}
	defer rows.Close()
	var out []models.PipelineRunNode
	for rows.Next() {
		n, err := scanPipelineRunNode(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres PipelineRunNodeRepo.FindByRunID scan: %w", err)
		}
		out = append(out, *n)
	}
	return out, nil
}

// DeleteByRunID removes node snapshots for a run.
func (r *PipelineRunNodeRepo) DeleteByRunID(ctx context.Context, runID string) error {
	const q = `DELETE FROM pipeline_run_nodes WHERE run_id = $1`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q, runID); err != nil {
		return fmt.Errorf("postgres PipelineRunNodeRepo.DeleteByRunID: %w", err)
	}
	return nil
}
