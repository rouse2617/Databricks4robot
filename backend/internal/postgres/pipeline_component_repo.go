package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// PipelineComponentRepo persists rows to the pipeline_components table.
type PipelineComponentRepo struct {
	c *Client
}

// NewPipelineComponentRepo creates a PipelineComponentRepo bound to c.
func NewPipelineComponentRepo(c *Client) *PipelineComponentRepo {
	return &PipelineComponentRepo{c: c}
}

var _ repository.PipelineComponentRepository = (*PipelineComponentRepo)(nil)
var _ repository.PipelineComponentReleaseRepository = (*PipelineComponentRepo)(nil)

const pipelineComponentSelectCols = `id, name, description, image, tag, source, scope, owner,
  input_ports, output_ports, resources, env_vars, created_at, updated_at`

const pipelineComponentReleaseSelectCols = `id, component_id, task_name, task_path, display_name, owner,
  release_label, channel, source_repo, source_ref, source_commit, build_id, image_repo, image_tag,
  image_digest, runtime_image, status, selectable, validation_status, validation_errors,
  runtime_snapshot, technical_metadata, created_at, updated_at, last_synced_at`

func scanPipelineComponent(rs rowScanner) (*models.PipelineComponent, error) {
	var (
		pc        models.PipelineComponent
		inPorts   []byte
		outPorts  []byte
		resources []byte
		envVars   []byte
	)
	if err := rs.Scan(
		&pc.ID, &pc.Name, &pc.Description, &pc.Image, &pc.Tag, &pc.Source, &pc.Scope, &pc.Owner,
		&inPorts, &outPorts, &resources, &envVars, &pc.CreatedAt, &pc.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if len(inPorts) > 0 {
		_ = json.Unmarshal(inPorts, &pc.InputPorts)
	}
	if len(outPorts) > 0 {
		_ = json.Unmarshal(outPorts, &pc.OutputPorts)
	}
	if len(resources) > 0 {
		_ = json.Unmarshal(resources, &pc.Resources)
	}
	if len(envVars) > 0 {
		_ = json.Unmarshal(envVars, &pc.EnvVars)
	}
	hydrateComponentDerivedFields(&pc)
	if pc.InputPorts == nil {
		pc.InputPorts = []models.PortDef{}
	}
	if pc.OutputPorts == nil {
		pc.OutputPorts = []models.PortDef{}
	}
	return &pc, nil
}

func scanPipelineComponentRelease(rs rowScanner) (*models.PipelineComponentRelease, error) {
	var (
		release           models.PipelineComponentRelease
		validationErrors  []byte
		runtimeSnapshot   []byte
		technicalMetadata []byte
	)
	if err := rs.Scan(
		&release.ID, &release.ComponentID, &release.TaskName, &release.TaskPath, &release.DisplayName, &release.Owner,
		&release.ReleaseLabel, &release.Channel, &release.SourceRepo, &release.SourceRef, &release.SourceCommit,
		&release.BuildID, &release.ImageRepo, &release.ImageTag, &release.ImageDigest, &release.RuntimeImage,
		&release.Status, &release.Selectable, &release.ValidationStatus, &validationErrors, &runtimeSnapshot,
		&technicalMetadata, &release.CreatedAt, &release.UpdatedAt, &release.LastSyncedAt,
	); err != nil {
		return nil, err
	}
	if len(validationErrors) > 0 {
		_ = json.Unmarshal(validationErrors, &release.ValidationErrors)
	}
	if len(runtimeSnapshot) > 0 {
		_ = json.Unmarshal(runtimeSnapshot, &release.RuntimeSnapshot)
	}
	if len(technicalMetadata) > 0 {
		_ = json.Unmarshal(technicalMetadata, &release.TechnicalMetadata)
	}
	if release.ValidationErrors == nil {
		release.ValidationErrors = []string{}
	}
	if release.RuntimeSnapshot.InputPorts == nil {
		release.RuntimeSnapshot.InputPorts = []models.PortDef{}
	}
	if release.RuntimeSnapshot.OutputPorts == nil {
		release.RuntimeSnapshot.OutputPorts = []models.PortDef{}
	}
	if release.RuntimeSnapshot.Resources == nil {
		release.RuntimeSnapshot.Resources = map[string]interface{}{}
	}
	if release.TechnicalMetadata == nil {
		release.TechnicalMetadata = map[string]interface{}{}
	}
	return &release, nil
}

func hydrateComponentDerivedFields(pc *models.PipelineComponent) {
	if pc.Resources == nil {
		pc.Resources = map[string]interface{}{}
	}
	if pc.Type == "" {
		if value, ok := pc.Resources["type"].(string); ok {
			pc.Type = value
		}
	}
	if pc.Type == "" {
		pc.Type = "container"
		pc.Resources["type"] = pc.Type
	}
	if len(pc.Command) == 0 {
		pc.Command = stringSliceFromJSONValue(pc.Resources["command"])
	}
	if len(pc.Args) == 0 {
		pc.Args = stringSliceFromJSONValue(pc.Resources["args"])
	}
	if pc.Env == nil {
		pc.Env = map[string]string{}
		if raw, ok := pc.Resources["env"].(map[string]interface{}); ok {
			for name, value := range raw {
				if s, ok := value.(string); ok {
					pc.Env[name] = s
				}
			}
		}
		for _, item := range pc.EnvVars {
			if item.Name != "" {
				pc.Env[item.Name] = item.Value
			}
		}
	}
}

func stringSliceFromJSONValue(value interface{}) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// Save inserts a pipeline component.
func (r *PipelineComponentRepo) Save(ctx context.Context, pc *models.PipelineComponent) error {
	if pc == nil {
		return errors.New("postgres PipelineComponentRepo.Save: nil component")
	}
	now := time.Now().UTC()
	if pc.ID == "" {
		pc.ID = uuid.New().String()
	}
	if pc.CreatedAt.IsZero() {
		pc.CreatedAt = now
	}
	pc.UpdatedAt = now

	inPorts, _ := json.Marshal(pc.InputPorts)
	outPorts, _ := json.Marshal(pc.OutputPorts)
	resources, _ := json.Marshal(pc.Resources)
	envVars, _ := json.Marshal(pc.EnvVars)

	const q = `
	INSERT INTO pipeline_components (id, name, description, image, tag, source, scope, owner,
	  input_ports, output_ports, resources, env_vars, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10::jsonb, $11::jsonb, $12::jsonb, $13, $14)`

	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q,
		pc.ID, pc.Name, pc.Description, pc.Image, pc.Tag, pc.Source, pc.Scope, pc.Owner,
		inPorts, outPorts, resources, envVars, pc.CreatedAt, pc.UpdatedAt,
	); err != nil {
		return fmt.Errorf("postgres PipelineComponentRepo.Save: %w", err)
	}
	return nil
}

// FindAll returns all pipeline components ordered by name, with optional search/filter.
func (r *PipelineComponentRepo) FindAll(ctx context.Context, filter *repository.ComponentFilter) ([]models.PipelineComponent, error) {
	q := `SELECT ` + pipelineComponentSelectCols + `
	FROM pipeline_components`
	var args []any
	var conditions []string
	argIdx := 0
	if filter != nil {
		if filter.Query != "" {
			argIdx++
			conditions = append(conditions, fmt.Sprintf(`name ILIKE $%d`, argIdx))
			args = append(args, "%"+filter.Query+"%")
		}
		if filter.Source != "" {
			argIdx++
			conditions = append(conditions, fmt.Sprintf(`source = $%d`, argIdx))
			args = append(args, filter.Source)
		}
	}
	if len(conditions) > 0 {
		q += ` WHERE ` + strings.Join(conditions, ` AND `)
	}
	q += ` ORDER BY name ASC`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres PipelineComponentRepo.FindAll: %w", err)
	}
	defer rows.Close()
	var out []models.PipelineComponent
	for rows.Next() {
		pc, err := scanPipelineComponent(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres PipelineComponentRepo.FindAll scan: %w", err)
		}
		out = append(out, *pc)
	}
	return out, nil
}

// FindByID returns a pipeline component by id, or (nil, nil) when not found.
func (r *PipelineComponentRepo) FindByID(ctx context.Context, id string) (*models.PipelineComponent, error) {
	q := `SELECT ` + pipelineComponentSelectCols + `
	FROM pipeline_components
	WHERE id = $1`
	db := dbFromCtx(ctx, r.c.db)
	pc, err := scanPipelineComponent(db.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres PipelineComponentRepo.FindByID: %w", err)
	}
	return pc, nil
}

// Update updates an existing pipeline component.
func (r *PipelineComponentRepo) Update(ctx context.Context, pc *models.PipelineComponent) error {
	if pc == nil {
		return errors.New("postgres PipelineComponentRepo.Update: nil component")
	}
	pc.UpdatedAt = time.Now().UTC()

	inPorts, _ := json.Marshal(pc.InputPorts)
	outPorts, _ := json.Marshal(pc.OutputPorts)
	resources, _ := json.Marshal(pc.Resources)
	envVars, _ := json.Marshal(pc.EnvVars)

	const q = `UPDATE pipeline_components SET
	  name = $2, description = $3, image = $4, tag = $5, source = $6, scope = $7, owner = $8,
	  input_ports = $9::jsonb, output_ports = $10::jsonb,
	  resources = $11::jsonb, env_vars = $12::jsonb, updated_at = $13
	WHERE id = $1`

	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q,
		pc.ID, pc.Name, pc.Description, pc.Image, pc.Tag, pc.Source, pc.Scope, pc.Owner,
		inPorts, outPorts, resources, envVars, pc.UpdatedAt,
	); err != nil {
		return fmt.Errorf("postgres PipelineComponentRepo.Update: %w", err)
	}
	return nil
}

// Delete removes a pipeline component by id.
func (r *PipelineComponentRepo) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM pipeline_components WHERE id = $1`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q, id); err != nil {
		return fmt.Errorf("postgres PipelineComponentRepo.Delete: %w", err)
	}
	return nil
}

// UpsertRelease stores a generated component release by component/version label.
func (r *PipelineComponentRepo) UpsertRelease(ctx context.Context, release *models.PipelineComponentRelease) error {
	if release == nil {
		return errors.New("postgres PipelineComponentRepo.UpsertRelease: nil release")
	}
	now := time.Now().UTC()
	if release.ID == "" {
		release.ID = uuid.NewString()
	}
	if release.CreatedAt.IsZero() {
		release.CreatedAt = now
	}
	release.UpdatedAt = now
	if release.LastSyncedAt == nil {
		release.LastSyncedAt = &now
	}

	validationErrors, _ := json.Marshal(release.ValidationErrors)
	runtimeSnapshot, _ := json.Marshal(release.RuntimeSnapshot)
	technicalMetadata, _ := json.Marshal(release.TechnicalMetadata)

	const q = `
	INSERT INTO pipeline_component_releases (
	  id, component_id, task_name, task_path, display_name, owner, release_label, channel,
	  source_repo, source_ref, source_commit, build_id, image_repo, image_tag, image_digest,
	  runtime_image, status, selectable, validation_status, validation_errors, runtime_snapshot,
	  technical_metadata, created_at, updated_at, last_synced_at
	) VALUES (
	  $1, $2, $3, $4, $5, $6, $7, $8,
	  $9, $10, $11, $12, $13, $14, $15,
	  $16, $17, $18, $19, $20::jsonb, $21::jsonb,
	  $22::jsonb, $23, $24, $25
	)
	ON CONFLICT (component_id, release_label) DO UPDATE SET
	  task_name = EXCLUDED.task_name,
	  task_path = EXCLUDED.task_path,
	  display_name = EXCLUDED.display_name,
	  owner = EXCLUDED.owner,
	  channel = EXCLUDED.channel,
	  source_repo = EXCLUDED.source_repo,
	  source_ref = EXCLUDED.source_ref,
	  source_commit = EXCLUDED.source_commit,
	  build_id = EXCLUDED.build_id,
	  image_repo = EXCLUDED.image_repo,
	  image_tag = EXCLUDED.image_tag,
	  image_digest = EXCLUDED.image_digest,
	  runtime_image = EXCLUDED.runtime_image,
	  status = EXCLUDED.status,
	  selectable = EXCLUDED.selectable,
	  validation_status = EXCLUDED.validation_status,
	  validation_errors = EXCLUDED.validation_errors,
	  runtime_snapshot = EXCLUDED.runtime_snapshot,
	  technical_metadata = EXCLUDED.technical_metadata,
	  updated_at = EXCLUDED.updated_at,
	  last_synced_at = EXCLUDED.last_synced_at
	RETURNING id, created_at`

	db := dbFromCtx(ctx, r.c.db)
	if err := db.QueryRow(ctx, q,
		release.ID, release.ComponentID, release.TaskName, release.TaskPath, release.DisplayName, release.Owner,
		release.ReleaseLabel, release.Channel, release.SourceRepo, release.SourceRef, release.SourceCommit,
		release.BuildID, release.ImageRepo, release.ImageTag, release.ImageDigest, release.RuntimeImage,
		release.Status, release.Selectable, release.ValidationStatus, validationErrors, runtimeSnapshot,
		technicalMetadata, release.CreatedAt, release.UpdatedAt, release.LastSyncedAt,
	).Scan(&release.ID, &release.CreatedAt); err != nil {
		return fmt.Errorf("postgres PipelineComponentRepo.UpsertRelease: %w", err)
	}
	return nil
}

// FindReleases returns generated component releases ordered for selection UI.
func (r *PipelineComponentRepo) FindReleases(ctx context.Context, filter *repository.ComponentReleaseFilter) ([]models.PipelineComponentRelease, error) {
	q := `SELECT ` + pipelineComponentReleaseSelectCols + `
	FROM pipeline_component_releases`
	var args []any
	var conditions []string
	argIdx := 0
	if filter != nil {
		if filter.Query != "" {
			argIdx++
			conditions = append(conditions, fmt.Sprintf(`(
				component_id ILIKE $%d OR task_name ILIKE $%d OR display_name ILIKE $%d OR release_label ILIKE $%d
			)`, argIdx, argIdx, argIdx, argIdx))
			args = append(args, "%"+filter.Query+"%")
		}
		if filter.ComponentID != "" {
			argIdx++
			conditions = append(conditions, fmt.Sprintf(`component_id = $%d`, argIdx))
			args = append(args, filter.ComponentID)
		}
		if filter.TaskName != "" {
			argIdx++
			conditions = append(conditions, fmt.Sprintf(`task_name = $%d`, argIdx))
			args = append(args, filter.TaskName)
		}
		if filter.Status != "" {
			argIdx++
			conditions = append(conditions, fmt.Sprintf(`status = $%d`, argIdx))
			args = append(args, filter.Status)
		}
		if filter.Channel != "" {
			argIdx++
			conditions = append(conditions, fmt.Sprintf(`channel = $%d`, argIdx))
			args = append(args, filter.Channel)
		}
		if filter.Selectable != nil {
			argIdx++
			conditions = append(conditions, fmt.Sprintf(`selectable = $%d`, argIdx))
			args = append(args, *filter.Selectable)
		}
	}
	if len(conditions) > 0 {
		q += ` WHERE ` + strings.Join(conditions, ` AND `)
	}
	q += ` ORDER BY selectable DESC, updated_at DESC, component_id ASC, release_label DESC`

	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres PipelineComponentRepo.FindReleases: %w", err)
	}
	defer rows.Close()
	var out []models.PipelineComponentRelease
	for rows.Next() {
		release, err := scanPipelineComponentRelease(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres PipelineComponentRepo.FindReleases scan: %w", err)
		}
		out = append(out, *release)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres PipelineComponentRepo.FindReleases rows: %w", err)
	}
	return out, nil
}

// FindReleaseByID returns one generated component release.
func (r *PipelineComponentRepo) FindReleaseByID(ctx context.Context, id string) (*models.PipelineComponentRelease, error) {
	q := `SELECT ` + pipelineComponentReleaseSelectCols + `
	FROM pipeline_component_releases
	WHERE id = $1`
	db := dbFromCtx(ctx, r.c.db)
	release, err := scanPipelineComponentRelease(db.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres PipelineComponentRepo.FindReleaseByID: %w", err)
	}
	return release, nil
}
