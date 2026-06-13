package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

const pipelineTemplateSelectCols = `id, name, version, pipeline, node_count, active_version, scope, owner, created_at, updated_at`

func scanPipelineTemplate(rs rowScanner) (*models.PipelineTemplate, error) {
	var (
		t            models.PipelineTemplate
		pipelineJSON []byte
	)
	if err := rs.Scan(
		&t.ID, &t.Name, &t.Version, &pipelineJSON, &t.NodeCount, &t.ActiveVersion, &t.Scope, &t.Owner, &t.CreatedAt, &t.UpdatedAt,
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
INSERT INTO pipeline_templates (id, name, version, pipeline, node_count, active_version, scope, owner, created_at, updated_at)
VALUES ($1, $2, $7, $3::jsonb, $4, $8, $9, $10, $5, $6)
ON CONFLICT (id) DO UPDATE SET
    name           = EXCLUDED.name,
    version        = EXCLUDED.version,
    pipeline       = EXCLUDED.pipeline,
    node_count     = EXCLUDED.node_count,
    active_version = EXCLUDED.active_version,
    scope          = EXCLUDED.scope,
    owner          = EXCLUDED.owner,
    updated_at     = EXCLUDED.updated_at`

	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q, t.ID, t.Name, pipelineJSON, t.NodeCount, t.CreatedAt, t.UpdatedAt, t.Version, t.ActiveVersion, t.Scope, t.Owner); err != nil {
		return fmt.Errorf("postgres PipelineTemplateRepo.Save: %w", err)
	}
	return nil
}

// FindAll returns the latest version of each pipeline template ordered by
// updated_at DESC.
func (r *PipelineTemplateRepo) FindAll(ctx context.Context) ([]models.PipelineTemplate, error) {
	q := `SELECT ` + pipelineTemplateSelectCols + `
       , version_count
FROM (
	SELECT ` + pipelineTemplateSelectCols + `,
	       ROW_NUMBER() OVER (PARTITION BY name ORDER BY version DESC, updated_at DESC) AS rn,
	       COUNT(*) OVER (PARTITION BY name) AS version_count
	FROM pipeline_templates
) latest
WHERE rn = 1
ORDER BY updated_at DESC`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("postgres PipelineTemplateRepo.FindAll: %w", err)
	}
	defer rows.Close()
	var out []models.PipelineTemplate
	for rows.Next() {
		var (
			t            models.PipelineTemplate
			pipelineJSON []byte
		)
		if err := rows.Scan(
			&t.ID, &t.Name, &t.Version, &pipelineJSON, &t.NodeCount, &t.ActiveVersion, &t.Scope, &t.Owner, &t.CreatedAt, &t.UpdatedAt, &t.VersionCount,
		); err != nil {
			return nil, fmt.Errorf("postgres PipelineTemplateRepo.FindAll scan: %w", err)
		}
		if len(pipelineJSON) > 0 {
			_ = json.Unmarshal(pipelineJSON, &t.Pipeline)
		}
		if t.Pipeline == nil {
			t.Pipeline = map[string]interface{}{}
		}
		out = append(out, t)
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

// FindByNameAndVersion returns a pipeline template snapshot by name and
// version, or (nil, nil) when not found.
func (r *PipelineTemplateRepo) FindByNameAndVersion(ctx context.Context, name string, version int) (*models.PipelineTemplate, error) {
	q := `SELECT ` + pipelineTemplateSelectCols + `
FROM pipeline_templates
WHERE name = $1 AND version = $2`
	db := dbFromCtx(ctx, r.c.db)
	t, err := scanPipelineTemplate(db.QueryRow(ctx, q, name, version))
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres PipelineTemplateRepo.FindByNameAndVersion: %w", err)
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
	for i := range out {
		out[i].VersionCount = len(out)
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

// SetActiveVersion updates the active_version on all rows for a named pipeline.
func (r *PipelineTemplateRepo) SetActiveVersion(ctx context.Context, name string, version int) error {
	const q = `UPDATE pipeline_templates SET active_version = $2, updated_at = NOW() WHERE name = $1`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q, name, version); err != nil {
		return fmt.Errorf("postgres PipelineTemplateRepo.SetActiveVersion: %w", err)
	}
	return nil
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
  status, node_count, scope, owner, manifest, pipeline_json, created_at, updated_at, finished_at`

func scanPipelineDeployment(rs rowScanner) (*models.PipelineDeployment, error) {
	var (
		d            models.PipelineDeployment
		templateID   *string
		manifest     *string
		pipelineJSON []byte
	)
	if err := rs.Scan(
		&d.ID, &templateID, &d.PipelineName, &d.WorkflowName,
		&d.Status, &d.NodeCount, &d.Scope, &d.Owner, &manifest, &pipelineJSON, &d.CreatedAt, &d.UpdatedAt, &d.FinishedAt,
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
INSERT INTO pipeline_deployments (id, template_id, pipeline_name, workflow_name, status, node_count, scope, owner, manifest, pipeline_json, created_at, updated_at, finished_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11, $12, $13)
ON CONFLICT (id) DO UPDATE SET
    template_id   = EXCLUDED.template_id,
    pipeline_name = EXCLUDED.pipeline_name,
    workflow_name = EXCLUDED.workflow_name,
    status        = EXCLUDED.status,
    node_count    = EXCLUDED.node_count,
    scope         = EXCLUDED.scope,
    owner         = EXCLUDED.owner,
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
		d.ID, templateID, d.PipelineName, d.WorkflowName, d.Status, d.NodeCount, d.Scope, d.Owner,
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
  manifest, pipeline_json, argo_namespace, argo_workflow_uid, message, scope, owner, batch_job_id,
  created_at, updated_at, started_at, finished_at`

const pipelineRunSummarySelectCols = `id, template_id, pipeline_name, template_version, workflow_name,
  execution_target_id, status, node_count, asset_ids, asset_count, no_asset_run,
  argo_namespace, argo_workflow_uid, message, scope, owner, batch_job_id,
  created_at, updated_at, started_at, finished_at`

func scanPipelineRunSummary(rs rowScanner) (*models.PipelineRun, error) {
	var (
		r            models.PipelineRun
		templateID   *string
		templateVer  *int
		assetIDs     []string
		batchJobID   *string
	)
	if err := rs.Scan(
		&r.ID, &templateID, &r.PipelineName, &templateVer, &r.WorkflowName,
		&r.ExecutionTargetID, &r.Status, &r.NodeCount, &assetIDs, &r.AssetCount, &r.NoAssetRun,
		&r.ArgoNamespace, &r.ArgoWorkflowUID, &r.Message,
		&r.Scope, &r.Owner, &batchJobID,
		&r.CreatedAt, &r.UpdatedAt, &r.StartedAt, &r.FinishedAt,
	); err != nil {
		return nil, err
	}
	r.TemplateID = templateID
	r.TemplateVersion = templateVer
	r.AssetIDs = assetIDs
	r.BatchJobID = batchJobID
	return &r, nil
}

func scanPipelineRun(rs rowScanner) (*models.PipelineRun, error) {
	var (
		r              models.PipelineRun
		templateID     *string
		templateVer    *int
		targetSnapshot []byte
		assetIDs       []string
		manifest       *string
		pipelineJSON   []byte
		batchJobID     *string
	)
	if err := rs.Scan(
		&r.ID, &templateID, &r.PipelineName, &templateVer, &r.WorkflowName,
		&r.ExecutionTargetID, &targetSnapshot, &r.Status, &r.NodeCount, &assetIDs, &r.AssetCount, &r.NoAssetRun,
		&manifest, &pipelineJSON, &r.ArgoNamespace, &r.ArgoWorkflowUID, &r.Message,
		&r.Scope, &r.Owner, &batchJobID,
		&r.CreatedAt, &r.UpdatedAt, &r.StartedAt, &r.FinishedAt,
	); err != nil {
		return nil, err
	}
	r.TemplateID = templateID
	r.TemplateVersion = templateVer
	r.AssetIDs = assetIDs
	r.BatchJobID = batchJobID
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
  manifest, pipeline_json, argo_namespace, argo_workflow_uid, message, scope, owner, batch_job_id,
  created_at, updated_at, started_at, finished_at
) VALUES (
  $1, $2, $3, $4, $5,
  $6, $7::jsonb, $8, $9, $10::text[], $11, $12,
  $13, $14::jsonb, $15, $16, $17, $18, $19, $20,
  $21, $22, $23, $24
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
  scope = EXCLUDED.scope,
  owner = EXCLUDED.owner,
  batch_job_id = EXCLUDED.batch_job_id,
  updated_at = EXCLUDED.updated_at,
  started_at = EXCLUDED.started_at,
  finished_at = EXCLUDED.finished_at`

	db := dbFromCtx(ctx, r.c.db)
	assetIDs := pgtype.FlatArray[string](run.AssetIDs)
	if err := db.Exec(ctx, q,
		run.ID, templateID, run.PipelineName, templateVersion, run.WorkflowName,
		run.ExecutionTargetID, targetSnapshot, run.Status, run.NodeCount, assetIDs, run.AssetCount, run.NoAssetRun,
		manifest, pipelineJSON, run.ArgoNamespace, run.ArgoWorkflowUID, run.Message, run.Scope, run.Owner, run.BatchJobID,
		run.CreatedAt, run.UpdatedAt, run.StartedAt, run.FinishedAt,
	); err != nil {
		return fmt.Errorf("postgres PipelineRunRepo.Save: %w", err)
	}
	return nil
}

// FindAll returns pipeline runs ordered by created_at DESC.
func (r *PipelineRunRepo) FindAll(ctx context.Context) ([]models.PipelineRun, error) {
	return r.findAllPipelineRuns(ctx, pipelineRunSelectCols, scanPipelineRun, "FindAll")
}

// FindAllSummaries returns lightweight pipeline runs for list endpoints.
func (r *PipelineRunRepo) FindAllSummaries(ctx context.Context) ([]models.PipelineRun, error) {
	return r.findAllPipelineRuns(ctx, pipelineRunSummarySelectCols, scanPipelineRunSummary, "FindAllSummaries")
}

// ListSummaries returns filtered/paginated summary rows.
func (r *PipelineRunRepo) ListSummaries(ctx context.Context, filter models.PipelineRunListFilter) ([]models.PipelineRun, int, error) {
	var (
		conds  []string
		args   []any
		argPos = 1
	)
	if filter.BatchJobID != "" {
		conds = append(conds, fmt.Sprintf("batch_job_id = $%d", argPos))
		args = append(args, filter.BatchJobID)
		argPos++
	}
	if filter.ExcludeBatch {
		conds = append(conds, "batch_job_id IS NULL")
	}
	if filter.Status != "" {
		conds = append(conds, fmt.Sprintf("status = $%d", argPos))
		args = append(args, filter.Status)
		argPos++
	}

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	countQ := `SELECT COUNT(*) FROM pipeline_runs ` + where
	db := dbFromCtx(ctx, r.c.db)
	var total int
	if err := db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres PipelineRunRepo.ListSummaries count: %w", err)
	}

	listQ := `SELECT ` + pipelineRunSummarySelectCols + `
FROM pipeline_runs
` + where + `
ORDER BY created_at DESC`

	if filter.Page > 0 || filter.PageSize > 0 {
		page := filter.Page
		if page < 1 {
			page = 1
		}
		pageSize := filter.PageSize
		if pageSize < 1 {
			pageSize = 20
		}
		if pageSize > 200 {
			pageSize = 200
		}
		offset := (page - 1) * pageSize
		limitClause := fmt.Sprintf(" LIMIT $%d OFFSET $%d", argPos, argPos+1)
		listArgs := append(append([]any{}, args...), pageSize, offset)
		rows, err := db.Query(ctx, listQ+limitClause, listArgs...)
		if err != nil {
			return nil, 0, fmt.Errorf("postgres PipelineRunRepo.ListSummaries: %w", err)
		}
		defer rows.Close()
		var out []models.PipelineRun
		for rows.Next() {
			run, err := scanPipelineRunSummary(rows)
			if err != nil {
				return nil, 0, fmt.Errorf("postgres PipelineRunRepo.ListSummaries scan: %w", err)
			}
			out = append(out, *run)
		}
		return out, total, nil
	}

	rows, err := db.Query(ctx, listQ, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres PipelineRunRepo.ListSummaries: %w", err)
	}
	defer rows.Close()
	var out []models.PipelineRun
	for rows.Next() {
		run, err := scanPipelineRunSummary(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("postgres PipelineRunRepo.ListSummaries scan: %w", err)
		}
		out = append(out, *run)
	}
	return out, total, nil
}

func (r *PipelineRunRepo) findAllPipelineRuns(
	ctx context.Context,
	cols string,
	scan func(rowScanner) (*models.PipelineRun, error),
	op string,
) ([]models.PipelineRun, error) {
	q := `SELECT ` + cols + `
FROM pipeline_runs
ORDER BY created_at DESC`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("postgres PipelineRunRepo.%s: %w", op, err)
	}
	defer rows.Close()
	var out []models.PipelineRun
	for rows.Next() {
		run, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres PipelineRunRepo.%s scan: %w", op, err)
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

func (r *PipelineRunRepo) UpdateLedgerState(ctx context.Context, id, ledgerState string) error {
	const q = `
	UPDATE pipeline_runs
	SET ledger_state = $2, updated_at = NOW()
	WHERE id = $1`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q, id, ledgerState); err != nil {
		return fmt.Errorf("postgres PipelineRunRepo.UpdateLedgerState: %w", err)
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
  inputs, outputs, resources_duration, resource_summary, log_ref, estimated_cost_usd,
  started_at, finished_at, created_at, updated_at`

func scanPipelineRunNode(rs rowScanner) (*models.PipelineRunNode, error) {
	var (
		n                 models.PipelineRunNode
		children          []string
		inputs            []byte
		outputs           []byte
		resourcesDuration []byte
		resourceSummary   []byte
		estimatedCost     *float64
	)
	if err := rs.Scan(
		&n.ID, &n.RunID, &n.PipelineNodeID, &n.ArgoNodeID, &n.ArgoNodeName,
		&n.DisplayName, &n.TemplateName, &n.Type, &n.Phase, &n.Message, &n.PodName, &n.HostNodeName, &children,
		&inputs, &outputs, &resourcesDuration, &resourceSummary, &n.LogRef, &estimatedCost,
		&n.StartedAt, &n.FinishedAt, &n.CreatedAt, &n.UpdatedAt,
	); err != nil {
		return nil, err
	}
	n.Children = children
	n.Inputs = mapFromJSON(inputs)
	n.Outputs = mapFromJSON(outputs)
	n.ResourcesDuration = mapFromJSON(resourcesDuration)
	n.ResourceSummary = mapFromJSON(resourceSummary)
	n.EstimatedCostUSD = estimatedCost
	return &n, nil
}

// ReplaceByRunID replaces a run's node snapshot.
func (r *PipelineRunNodeRepo) ReplaceByRunID(ctx context.Context, runID string, nodes []models.PipelineRunNode) error {
	const q = `
INSERT INTO pipeline_run_nodes (
  id, run_id, pipeline_node_id, argo_node_id, argo_node_name,
  display_name, template_name, type, phase, message, pod_name, host_node_name, children,
  inputs, outputs, resources_duration, resource_summary, log_ref, estimated_cost_usd,
  started_at, finished_at, created_at, updated_at
) VALUES (
  $1, $2, $3, $4, $5,
  $6, $7, $8, $9, $10, $11, $12, $13,
  $14::jsonb, $15::jsonb, $16::jsonb, $17::jsonb, $18, $19,
  $20, $21, $22, $23
)`
	return r.c.WithTx(ctx, func(txCtx context.Context) error {
		db := dbFromCtx(txCtx, r.c.db)
		if err := db.Exec(txCtx, `DELETE FROM pipeline_run_nodes WHERE run_id = $1`, runID); err != nil {
			return fmt.Errorf("postgres PipelineRunNodeRepo.ReplaceByRunID delete: %w", err)
		}
		if len(nodes) == 0 {
			return nil
		}
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
			if n.Children == nil {
				n.Children = []string{}
			}
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
			if err := db.Exec(txCtx, q,
				n.ID, n.RunID, n.PipelineNodeID, n.ArgoNodeID, n.ArgoNodeName,
				n.DisplayName, n.TemplateName, n.Type, n.Phase, n.Message, n.PodName, n.HostNodeName, n.Children,
				inputs, outputs, resourcesDuration, resourceSummary, n.LogRef, n.EstimatedCostUSD,
				n.StartedAt, n.FinishedAt, n.CreatedAt, n.UpdatedAt,
			); err != nil {
				return fmt.Errorf("postgres PipelineRunNodeRepo.ReplaceByRunID insert: %w", err)
			}
		}
		return nil
	})
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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres PipelineRunNodeRepo.FindByRunID rows: %w", err)
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

// ──────────────────────────────────────────────────────────────────────────────
// PipelineRunEventRepo
// ──────────────────────────────────────────────────────────────────────────────

// PipelineRunEventRepo persists durable timeline events for pipeline runs.
type PipelineRunEventRepo struct {
	c *Client
}

// NewPipelineRunEventRepo creates a PipelineRunEventRepo bound to c.
func NewPipelineRunEventRepo(c *Client) *PipelineRunEventRepo { return &PipelineRunEventRepo{c: c} }

var _ repository.PipelineRunEventRepository = (*PipelineRunEventRepo)(nil)

const pipelineRunEventSelectCols = `id, run_id, workflow_name, event_type, subject_type, subject_id,
  status, message, reason, payload, idempotency_key, sequence, occurred_at, observed_at, created_at`

func scanPipelineRunEvent(rs rowScanner) (*models.PipelineRunEvent, error) {
	var (
		e       models.PipelineRunEvent
		payload []byte
	)
	if err := rs.Scan(
		&e.ID, &e.RunID, &e.WorkflowName, &e.EventType, &e.SubjectType, &e.SubjectID,
		&e.Status, &e.Message, &e.Reason, &payload, &e.IdempotencyKey, &e.Sequence,
		&e.OccurredAt, &e.ObservedAt, &e.CreatedAt,
	); err != nil {
		return nil, err
	}
	e.Payload = mapFromJSON(payload)
	return &e, nil
}

// Append inserts an event unless its idempotency key already exists for the run.
func (r *PipelineRunEventRepo) Append(ctx context.Context, event *models.PipelineRunEvent) error {
	if event == nil {
		return errors.New("postgres PipelineRunEventRepo.Append: nil event")
	}
	now := time.Now().UTC()
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = now
	}
	if event.ObservedAt.IsZero() {
		event.ObservedAt = now
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = now
	}
	if event.SubjectID == "" {
		event.SubjectID = event.RunID
	}
	if event.IdempotencyKey == "" {
		event.IdempotencyKey = strings.Join([]string{
			event.EventType,
			event.SubjectType,
			event.SubjectID,
			event.Status,
			event.OccurredAt.UTC().Format(time.RFC3339Nano),
		}, ":")
	}
	payload, err := marshalMapForJSONB(event.Payload)
	if err != nil {
		return fmt.Errorf("postgres PipelineRunEventRepo.Append marshal payload: %w", err)
	}
	const q = `
INSERT INTO pipeline_run_events (
  id, run_id, workflow_name, event_type, subject_type, subject_id,
  status, message, reason, payload, idempotency_key,
  occurred_at, observed_at, created_at
) VALUES (
  $1, $2, $3, $4, $5, $6,
  $7, $8, $9, $10::jsonb, $11,
  $12, $13, $14
)
ON CONFLICT (run_id, idempotency_key) DO NOTHING`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q,
		event.ID, event.RunID, event.WorkflowName, event.EventType, event.SubjectType, event.SubjectID,
		event.Status, event.Message, event.Reason, payload, event.IdempotencyKey,
		event.OccurredAt, event.ObservedAt, event.CreatedAt,
	); err != nil {
		return fmt.Errorf("postgres PipelineRunEventRepo.Append: %w", err)
	}
	return nil
}

// ListByRunID returns chronological run events with cursor pagination.
func (r *PipelineRunEventRepo) ListByRunID(ctx context.Context, runID string, opts models.PipelineRunEventListOptions) (*models.PipelineRunEventListResult, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	args := []any{runID, limit + 1}
	clauses := []string{"run_id = $1"}
	if opts.Cursor > 0 {
		args = append(args, opts.Cursor)
		clauses = append(clauses, fmt.Sprintf("sequence > $%d", len(args)))
	}
	if strings.TrimSpace(opts.SubjectType) != "" {
		args = append(args, strings.TrimSpace(opts.SubjectType))
		clauses = append(clauses, fmt.Sprintf("subject_type = $%d", len(args)))
	}
	if strings.TrimSpace(opts.EventType) != "" {
		args = append(args, strings.TrimSpace(opts.EventType))
		clauses = append(clauses, fmt.Sprintf("event_type = $%d", len(args)))
	}
	if strings.TrimSpace(opts.Status) != "" {
		args = append(args, strings.TrimSpace(opts.Status))
		clauses = append(clauses, fmt.Sprintf("status = $%d", len(args)))
	}
	if strings.TrimSpace(opts.Query) != "" {
		args = append(args, "%"+strings.TrimSpace(opts.Query)+"%")
		clauses = append(clauses, fmt.Sprintf("(message ILIKE $%d OR subject_id ILIKE $%d OR event_type ILIKE $%d)", len(args), len(args), len(args)))
	}
	if opts.From != nil {
		args = append(args, *opts.From)
		clauses = append(clauses, fmt.Sprintf("occurred_at >= $%d", len(args)))
	}
	if opts.To != nil {
		args = append(args, *opts.To)
		clauses = append(clauses, fmt.Sprintf("occurred_at <= $%d", len(args)))
	}
	q := `SELECT ` + pipelineRunEventSelectCols + `
FROM pipeline_run_events
WHERE ` + strings.Join(clauses, " AND ") + `
ORDER BY sequence ASC
LIMIT $2`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres PipelineRunEventRepo.ListByRunID: %w", err)
	}
	defer rows.Close()
	items := make([]models.PipelineRunEvent, 0, limit)
	var nextCursor *int64
	for rows.Next() {
		event, err := scanPipelineRunEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres PipelineRunEventRepo.ListByRunID scan: %w", err)
		}
		if len(items) >= limit {
			cursor := event.Sequence - 1
			nextCursor = &cursor
			break
		}
		items = append(items, *event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres PipelineRunEventRepo.ListByRunID rows: %w", err)
	}
	return &models.PipelineRunEventListResult{
		Items:      items,
		NextCursor: nextCursor,
		Total:      len(items),
	}, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// PipelineRunAssetNodeRepo
// ──────────────────────────────────────────────────────────────────────────────

type PipelineRunAssetNodeRepo struct {
	c *Client
}

func NewPipelineRunAssetNodeRepo(c *Client) *PipelineRunAssetNodeRepo {
	return &PipelineRunAssetNodeRepo{c: c}
}

var _ repository.PipelineRunAssetNodeRepository = (*PipelineRunAssetNodeRepo)(nil)

const pipelineRunAssetNodeSelectCols = `id, run_id, asset_id, pipeline_node_id, argo_node_id,
  display_name, status, message, pod_name, log_ref, estimated_cost_usd, cost_source,
  started_at, finished_at, updated_at`

func scanPipelineRunAssetNode(rs rowScanner) (*models.PipelineRunAssetNode, error) {
	var row models.PipelineRunAssetNode
	if err := rs.Scan(
		&row.ID, &row.RunID, &row.AssetID, &row.PipelineNodeID, &row.ArgoNodeID,
		&row.DisplayName, &row.Status, &row.Message, &row.PodName, &row.LogRef,
		&row.EstimatedCostUSD, &row.CostSource, &row.StartedAt, &row.FinishedAt, &row.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *PipelineRunAssetNodeRepo) ReplaceByRunID(ctx context.Context, runID string, rows []models.PipelineRunAssetNode) error {
	const q = `
INSERT INTO pipeline_run_asset_nodes (
  id, run_id, asset_id, pipeline_node_id, argo_node_id, display_name, status,
  message, pod_name, log_ref, estimated_cost_usd, cost_source,
  started_at, finished_at, updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7,
  $8, $9, $10, $11, $12,
  $13, $14, $15
)`
	return r.c.WithTx(ctx, func(txCtx context.Context) error {
		db := dbFromCtx(txCtx, r.c.db)
		if err := db.Exec(txCtx, `DELETE FROM pipeline_run_asset_nodes WHERE run_id = $1`, runID); err != nil {
			return fmt.Errorf("postgres PipelineRunAssetNodeRepo.ReplaceByRunID delete: %w", err)
		}
		if len(rows) == 0 {
			return nil
		}
		now := time.Now().UTC()
		for i := range rows {
			row := &rows[i]
			if row.ID == "" {
				row.ID = uuid.New().String()
			}
			if row.RunID == "" {
				row.RunID = runID
			}
			if row.CostSource == "" {
				row.CostSource = "not_available"
				if row.EstimatedCostUSD != nil {
					row.CostSource = "estimated_resource_duration"
				}
			}
			if row.UpdatedAt.IsZero() {
				row.UpdatedAt = now
			}
			if err := db.Exec(txCtx, q,
				row.ID, row.RunID, row.AssetID, row.PipelineNodeID, row.ArgoNodeID, row.DisplayName, row.Status,
				row.Message, row.PodName, row.LogRef, row.EstimatedCostUSD, row.CostSource,
				row.StartedAt, row.FinishedAt, row.UpdatedAt,
			); err != nil {
				return fmt.Errorf("postgres PipelineRunAssetNodeRepo.ReplaceByRunID insert: %w", err)
			}
		}
		return nil
	})
}

func (r *PipelineRunAssetNodeRepo) ListByRunID(ctx context.Context, runID string, opts models.PipelineRunAssetNodeListOptions) (*models.PipelineRunAssetNodeListResult, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	args := []any{runID, limit + 1}
	clauses := []string{"run_id = $1"}
	if strings.TrimSpace(opts.Cursor) != "" {
		args = append(args, strings.TrimSpace(opts.Cursor))
		clauses = append(clauses, fmt.Sprintf("id > $%d", len(args)))
	}
	if strings.TrimSpace(opts.AssetID) != "" {
		args = append(args, strings.TrimSpace(opts.AssetID))
		clauses = append(clauses, fmt.Sprintf("asset_id = $%d", len(args)))
	}
	if strings.TrimSpace(opts.NodeID) != "" {
		args = append(args, strings.TrimSpace(opts.NodeID))
		clauses = append(clauses, fmt.Sprintf("(pipeline_node_id = $%d OR argo_node_id = $%d)", len(args), len(args)))
	}
	if strings.TrimSpace(opts.Status) != "" {
		args = append(args, strings.TrimSpace(opts.Status))
		clauses = append(clauses, fmt.Sprintf("status = $%d", len(args)))
	}
	orderBy := "asset_id ASC, pipeline_node_id ASC"
	switch strings.TrimSpace(opts.OrderBy) {
	case "cost":
		orderBy = "estimated_cost_usd DESC NULLS LAST, asset_id ASC"
	case "duration":
		orderBy = "finished_at - started_at DESC NULLS LAST, asset_id ASC"
	case "status":
		orderBy = "status ASC, asset_id ASC"
	}
	q := `SELECT ` + pipelineRunAssetNodeSelectCols + `
FROM pipeline_run_asset_nodes
WHERE ` + strings.Join(clauses, " AND ") + `
ORDER BY ` + orderBy + `
LIMIT $2`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres PipelineRunAssetNodeRepo.ListByRunID: %w", err)
	}
	defer rows.Close()
	items := make([]models.PipelineRunAssetNode, 0, limit)
	var nextCursor *string
	statuses := map[string]int{}
	assets := map[string]bool{}
	nodes := map[string]bool{}
	var totalCost float64
	hasCost := false
	for rows.Next() {
		row, err := scanPipelineRunAssetNode(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres PipelineRunAssetNodeRepo.ListByRunID scan: %w", err)
		}
		if len(items) >= limit {
			cursor := row.ID
			nextCursor = &cursor
			break
		}
		items = append(items, *row)
		statuses[row.Status]++
		assets[row.AssetID] = true
		nodes[row.PipelineNodeID] = true
		if row.EstimatedCostUSD != nil {
			totalCost += *row.EstimatedCostUSD
			hasCost = true
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres PipelineRunAssetNodeRepo.ListByRunID rows: %w", err)
	}
	var totalCostPtr *float64
	costSource := "not_available"
	if hasCost {
		totalCostPtr = &totalCost
		costSource = "estimated_resource_duration"
	}
	return &models.PipelineRunAssetNodeListResult{
		Items:      items,
		NextCursor: nextCursor,
		Total:      len(items),
		Summary: models.PipelineRunAssetNodeSummary{
			AssetCount:            len(assets),
			NodeCount:             len(nodes),
			Statuses:              statuses,
			TotalEstimatedCostUSD: totalCostPtr,
			CostSource:            costSource,
		},
	}, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// PipelineRunNotificationRepo
// ──────────────────────────────────────────────────────────────────────────────

type PipelineRunNotificationRepo struct {
	c *Client
}

func NewPipelineRunNotificationRepo(c *Client) *PipelineRunNotificationRepo {
	return &PipelineRunNotificationRepo{c: c}
}

var _ repository.PipelineRunNotificationRepository = (*PipelineRunNotificationRepo)(nil)

func (r *PipelineRunNotificationRepo) AppendCandidate(ctx context.Context, candidate *models.PipelineRunNotificationCandidate) error {
	if candidate == nil {
		return errors.New("postgres PipelineRunNotificationRepo.AppendCandidate: nil candidate")
	}
	now := time.Now().UTC()
	if candidate.ID == "" {
		candidate.ID = uuid.New().String()
	}
	if candidate.CreatedAt.IsZero() {
		candidate.CreatedAt = now
	}
	if candidate.SinkType == "" {
		candidate.SinkType = "candidate"
	}
	if candidate.DeliveryStatus == "" {
		candidate.DeliveryStatus = "pending"
	}
	if candidate.IdempotencyKey == "" {
		candidate.IdempotencyKey = fmt.Sprintf("notify:%s:%s", candidate.RunID, candidate.EventID)
	}
	const q = `
INSERT INTO pipeline_run_notification_candidates (
  id, run_id, event_id, event_type, subject_type, subject_id, status, message,
  sink_type, delivery_status, idempotency_key, created_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8,
  $9, $10, $11, $12
)
ON CONFLICT (idempotency_key) DO NOTHING`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q,
		candidate.ID, candidate.RunID, candidate.EventID, candidate.EventType, candidate.SubjectType, candidate.SubjectID,
		candidate.Status, candidate.Message, candidate.SinkType, candidate.DeliveryStatus, candidate.IdempotencyKey, candidate.CreatedAt,
	); err != nil {
		return fmt.Errorf("postgres PipelineRunNotificationRepo.AppendCandidate: %w", err)
	}
	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// PipelineRunWatcherStateRepo
// ──────────────────────────────────────────────────────────────────────────────

type PipelineRunWatcherStateRepo struct {
	c *Client
}

func NewPipelineRunWatcherStateRepo(c *Client) *PipelineRunWatcherStateRepo {
	return &PipelineRunWatcherStateRepo{c: c}
}

var _ repository.PipelineRunWatcherStateRepository = (*PipelineRunWatcherStateRepo)(nil)

func (r *PipelineRunWatcherStateRepo) Save(ctx context.Context, state *models.PipelineRunWatcherState) error {
	if state == nil {
		return errors.New("postgres PipelineRunWatcherStateRepo.Save: nil state")
	}
	if state.ID == "" {
		state.ID = "default"
	}
	if state.ActiveScanLimit <= 0 {
		state.ActiveScanLimit = 100
	}
	state.UpdatedAt = time.Now().UTC()
	const q = `
INSERT INTO pipeline_run_watcher_state (
  id, last_synced_at, last_scan_started_at, last_scan_finished_at,
  last_success_at, last_error_at, active_scan_limit, last_synced_run_count,
  consecutive_failures, total_scans, total_errors, scan_lag_seconds,
  last_error, updated_at
)
VALUES (
  $1, $2, $3, $4,
  $5, $6, $7, $8,
  $9, $10, $11, $12,
  $13, $14
)
ON CONFLICT (id) DO UPDATE SET
  last_synced_at = EXCLUDED.last_synced_at,
  last_scan_started_at = EXCLUDED.last_scan_started_at,
  last_scan_finished_at = EXCLUDED.last_scan_finished_at,
  last_success_at = EXCLUDED.last_success_at,
  last_error_at = EXCLUDED.last_error_at,
  active_scan_limit = EXCLUDED.active_scan_limit,
  last_synced_run_count = EXCLUDED.last_synced_run_count,
  consecutive_failures = EXCLUDED.consecutive_failures,
  total_scans = EXCLUDED.total_scans,
  total_errors = EXCLUDED.total_errors,
  scan_lag_seconds = EXCLUDED.scan_lag_seconds,
  last_error = EXCLUDED.last_error,
  updated_at = EXCLUDED.updated_at`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q,
		state.ID, state.LastSyncedAt, state.LastScanStartedAt, state.LastScanFinishedAt,
		state.LastSuccessAt, state.LastErrorAt, state.ActiveScanLimit, state.LastSyncedRunCount,
		state.ConsecutiveFailures, state.TotalScans, state.TotalErrors, state.ScanLagSeconds,
		state.LastError, state.UpdatedAt,
	); err != nil {
		return fmt.Errorf("postgres PipelineRunWatcherStateRepo.Save: %w", err)
	}
	return nil
}

func (r *PipelineRunWatcherStateRepo) FindByID(ctx context.Context, id string) (*models.PipelineRunWatcherState, error) {
	if id == "" {
		id = "default"
	}
	const q = `
SELECT id, last_synced_at, last_scan_started_at, last_scan_finished_at,
  last_success_at, last_error_at, active_scan_limit, last_synced_run_count,
  consecutive_failures, total_scans, total_errors, scan_lag_seconds,
  last_error, updated_at
FROM pipeline_run_watcher_state
WHERE id = $1`
	db := dbFromCtx(ctx, r.c.db)
	var state models.PipelineRunWatcherState
	if err := db.QueryRow(ctx, q, id).Scan(
		&state.ID, &state.LastSyncedAt, &state.LastScanStartedAt, &state.LastScanFinishedAt,
		&state.LastSuccessAt, &state.LastErrorAt, &state.ActiveScanLimit, &state.LastSyncedRunCount,
		&state.ConsecutiveFailures, &state.TotalScans, &state.TotalErrors, &state.ScanLagSeconds,
		&state.LastError, &state.UpdatedAt,
	); err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres PipelineRunWatcherStateRepo.FindByID: %w", err)
	}
	return &state, nil
}
