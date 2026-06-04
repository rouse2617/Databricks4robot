package pipeline_component

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	if err := normalizeComponent(pc, false); err != nil {
		return nil, err
	}
	pc.ID = uuid.New().String()
	pc.CreatedAt = time.Now().UTC()
	pc.UpdatedAt = pc.CreatedAt
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
	if err := normalizeComponent(pc, true); err != nil {
		return err
	}
	existing, err := uc.repo.FindByID(ctx, pc.ID)
	if err != nil {
		return fmt.Errorf("find component: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("component not found: %s", pc.ID)
	}
	if existing.Source == "system" && pc.Source != "system" {
		return errors.New("system components cannot be converted to custom components")
	}
	pc.CreatedAt = existing.CreatedAt
	return uc.repo.Update(ctx, pc)
}

// Delete removes a component.
func (uc *Usecase) Delete(ctx context.Context, id string) error {
	existing, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("find component: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("component not found: %s", id)
	}
	if existing.Source == "system" {
		return errors.New("system components cannot be deleted")
	}
	return uc.repo.Delete(ctx, id)
}

var validComponentTypes = map[string]struct{}{
	"container": {},
	"script":    {},
	"resource":  {},
	"suspend":   {},
}

func normalizeComponent(pc *models.PipelineComponent, preserveID bool) error {
	if pc == nil {
		return errors.New("component is required")
	}
	if preserveID {
		pc.ID = strings.TrimSpace(pc.ID)
	}
	pc.Name = strings.TrimSpace(pc.Name)
	pc.Type = strings.TrimSpace(pc.Type)
	pc.Description = strings.TrimSpace(pc.Description)
	pc.Image = strings.TrimSpace(pc.Image)
	pc.Tag = strings.TrimSpace(pc.Tag)
	pc.Source = strings.TrimSpace(pc.Source)
	if pc.Name == "" {
		return errors.New("name is required")
	}
	if pc.Image == "" {
		return errors.New("image is required")
	}
	if pc.Type == "" {
		pc.Type = readResourceString(pc.Resources, "type")
	}
	if pc.Type == "" {
		return errors.New("type is required")
	}
	if _, ok := validComponentTypes[pc.Type]; !ok {
		return fmt.Errorf("type must be one of: container, script, resource, suspend")
	}
	if pc.Tag == "" {
		pc.Tag = "latest"
	}
	if pc.Scope == "" {
		pc.Scope = "dev"
	}
	if pc.Source == "" {
		pc.Source = "custom"
	}
	if pc.InputPorts == nil {
		pc.InputPorts = []models.PortDef{}
	}
	if pc.OutputPorts == nil {
		pc.OutputPorts = []models.PortDef{}
	}
	if pc.Env == nil && len(pc.EnvVars) > 0 {
		pc.Env = make(map[string]string, len(pc.EnvVars))
		for _, item := range pc.EnvVars {
			name := strings.TrimSpace(item.Name)
			if name != "" {
				pc.Env[name] = item.Value
			}
		}
	}
	if pc.Resources == nil {
		pc.Resources = map[string]interface{}{}
	}
	pc.Resources["type"] = pc.Type
	if len(pc.Command) > 0 {
		pc.Resources["command"] = pc.Command
	} else if command := readResourceStringSlice(pc.Resources, "command"); len(command) > 0 {
		pc.Command = command
	}
	if len(pc.Args) > 0 {
		pc.Resources["args"] = pc.Args
	} else if args := readResourceStringSlice(pc.Resources, "args"); len(args) > 0 {
		pc.Args = args
	}
	if pc.Env != nil {
		pc.Resources["env"] = pc.Env
		pc.EnvVars = envMapToDefs(pc.Env)
	}
	return nil
}

func readResourceString(resources map[string]interface{}, key string) string {
	if resources == nil {
		return ""
	}
	value, ok := resources[key].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func readResourceStringSlice(resources map[string]interface{}, key string) []string {
	if resources == nil {
		return nil
	}
	switch value := resources[key].(type) {
	case []string:
		return value
	case []interface{}:
		out := make([]string, 0, len(value))
		for _, item := range value {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func envMapToDefs(env map[string]string) []models.EnvVarDef {
	if env == nil {
		return nil
	}
	out := make([]models.EnvVarDef, 0, len(env))
	for name, value := range env {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		out = append(out, models.EnvVarDef{Name: name, Value: value})
	}
	return out
}

// systemComponents are built-in components seeded at startup.
var systemComponents = []models.PipelineComponent{
	{
		ID:          "sys-pass-through",
		Name:        "Pass Through",
		Type:        "container",
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
