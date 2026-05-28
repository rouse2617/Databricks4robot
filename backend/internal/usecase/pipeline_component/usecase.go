package pipeline_component

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// Usecase orchestrates pipeline component registry operations.
type Usecase struct {
	repo repository.PipelineComponentRepository
}

// New creates a Usecase.
func New(repo repository.PipelineComponentRepository) *Usecase {
	return &Usecase{repo: repo}
}

// Create persists a new component.
func (uc *Usecase) Create(ctx context.Context, pc *models.PipelineComponent) (*models.PipelineComponent, error) {
	pc.ID = uuid.New().String()
	pc.CreatedAt = time.Now().UTC()
	pc.UpdatedAt = pc.CreatedAt
	if pc.Source == "" {
		pc.Source = "custom"
	}
	if pc.InputPorts == nil {
		pc.InputPorts = []models.PortDef{}
	}
	if pc.OutputPorts == nil {
		pc.OutputPorts = []models.PortDef{}
	}
	if err := uc.repo.Save(ctx, pc); err != nil {
		return nil, fmt.Errorf("create component: %w", err)
	}
	return pc, nil
}

// List returns all components, with optional search/filter.
// When query is non-empty, filters by name (ILIKE).
// When source is non-empty, filters by source.
func (uc *Usecase) List(ctx context.Context, query, source string) ([]models.PipelineComponent, error) {
	var filter *repository.ComponentFilter
	if query != "" || source != "" {
		filter = &repository.ComponentFilter{
			Query:  query,
			Source: source,
		}
	}
	return uc.repo.FindAll(ctx, filter)
}

// Get returns a component by id.
func (uc *Usecase) Get(ctx context.Context, id string) (*models.PipelineComponent, error) {
	return uc.repo.FindByID(ctx, id)
}

// Update modifies an existing component.
func (uc *Usecase) Update(ctx context.Context, pc *models.PipelineComponent) error {
	existing, err := uc.repo.FindByID(ctx, pc.ID)
	if err != nil {
		return fmt.Errorf("find component: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("component not found: %s", pc.ID)
	}
	pc.CreatedAt = existing.CreatedAt
	return uc.repo.Update(ctx, pc)
}

// Delete removes a component.
func (uc *Usecase) Delete(ctx context.Context, id string) error {
	return uc.repo.Delete(ctx, id)
}

// systemComponents are built-in components seeded at startup.
var systemComponents = []models.PipelineComponent{
	{
		ID:          "sys-pass-through",
		Name:        "Pass Through",
		Description: "透传输入到输出，用于测试 DAG 连线",
		Image:       "busybox:latest",
		Tag:         "latest",
		Source:      "system",
		InputPorts:  []models.PortDef{{Name: "input", Type: "asset", Desc: "输入资产"}},
		OutputPorts: []models.PortDef{{Name: "output", Type: "asset", Desc: "输出资产"}},
	},
}

// SeedSystemComponents ensures built-in system components exist in the database.
// Idempotent — skips components that already exist.
func (uc *Usecase) SeedSystemComponents(ctx context.Context) error {
	for _, sc := range systemComponents {
		existing, err := uc.repo.FindByID(ctx, sc.ID)
		if err != nil {
			return fmt.Errorf("seed system component %q: %w", sc.ID, err)
		}
		if existing != nil {
			continue
		}
		now := time.Now().UTC()
		sc.CreatedAt = now
		sc.UpdatedAt = now
		if err := uc.repo.Save(ctx, &sc); err != nil {
			return fmt.Errorf("seed system component %q: %w", sc.ID, err)
		}
	}
	return nil
}
