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

const pipelineComponentSelectCols = `id, name, description, image, tag, source,
  input_ports, output_ports, resources, env_vars, created_at, updated_at`

func scanPipelineComponent(rs rowScanner) (*models.PipelineComponent, error) {
	var (
		pc         models.PipelineComponent
		inPorts    []byte
		outPorts   []byte
		resources  []byte
		envVars    []byte
	)
	if err := rs.Scan(
		&pc.ID, &pc.Name, &pc.Description, &pc.Image, &pc.Tag, &pc.Source,
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
	if pc.InputPorts == nil {
		pc.InputPorts = []models.PortDef{}
	}
	if pc.OutputPorts == nil {
		pc.OutputPorts = []models.PortDef{}
	}
	return &pc, nil
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
	INSERT INTO pipeline_components (id, name, description, image, tag, source,
	  input_ports, output_ports, resources, env_vars, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb, $9::jsonb, $10::jsonb, $11, $12)`

	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q,
		pc.ID, pc.Name, pc.Description, pc.Image, pc.Tag, pc.Source,
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
	  name = $2, description = $3, image = $4, tag = $5, source = $6,
	  input_ports = $7::jsonb, output_ports = $8::jsonb,
	  resources = $9::jsonb, env_vars = $10::jsonb, updated_at = $11
	WHERE id = $1`

	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q,
		pc.ID, pc.Name, pc.Description, pc.Image, pc.Tag, pc.Source,
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
