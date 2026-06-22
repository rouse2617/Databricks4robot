package pipeline_config

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

const MaxConfigFileBytes = 1 << 20

var (
	ErrConfigNotFound   = errors.New("config not found")
	ErrInvalidConfig    = errors.New("invalid config")
	ErrConfigNameExists = errors.New("config name already exists")
)

// Usecase orchestrates standalone pipeline config file operations.
type Usecase struct {
	repo repository.PipelineConfigRepository
}

// New creates a config usecase.
func New(repo repository.PipelineConfigRepository) *Usecase {
	return &Usecase{repo: repo}
}

type CreateConfigInput struct {
	Name        string
	Description string
	Tags        []string
	FileType    string
	Lifecycle   string
	Content     string
	Summary     string
	Owner       string
	Scope       string
}

type UpdateConfigInput struct {
	Name        string
	Description string
	Tags        []string
	FileType    string
	Lifecycle   string
}

type CreateVersionInput struct {
	Status  string
	Content string
	Summary string
	Author  string
}

func (uc *Usecase) List(ctx context.Context, filter repository.PipelineConfigFilter) ([]models.PipelineConfig, error) {
	if filter.Scope == "" {
		filter.Scope = "dev"
	}
	return uc.repo.FindAll(ctx, &filter)
}

func (uc *Usecase) Get(ctx context.Context, id string) (*models.PipelineConfig, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: id is required", ErrInvalidConfig)
	}
	cfg, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrPipelineConfigNotFound) {
			return nil, ErrConfigNotFound
		}
		return nil, err
	}
	if cfg == nil {
		return nil, ErrConfigNotFound
	}
	return cfg, nil
}

func (uc *Usecase) Create(ctx context.Context, in CreateConfigInput) (*models.PipelineConfig, error) {
	lifecycle, err := normalizeLifecycle(in.Lifecycle, "draft")
	if err != nil {
		return nil, err
	}
	cfg := &models.PipelineConfig{
		ID:             uuid.NewString(),
		Name:           strings.TrimSpace(in.Name),
		Description:    strings.TrimSpace(in.Description),
		Owner:          strings.TrimSpace(in.Owner),
		Scope:          firstNonEmpty(strings.TrimSpace(in.Scope), "dev"),
		Tags:           normalizeTags(in.Tags),
		FileType:       normalizeFileType(in.FileType, in.Name),
		Lifecycle:      lifecycle,
		CurrentVersion: 1,
	}
	if err := validateConfigMetadata(cfg); err != nil {
		return nil, err
	}
	version := &models.PipelineConfigVersion{
		Status:  cfg.Lifecycle,
		Content: in.Content,
		Summary: strings.TrimSpace(in.Summary),
		Author:  cfg.Owner,
	}
	if err := validateVersion(version); err != nil {
		return nil, err
	}
	if err := uc.repo.Create(ctx, cfg, version); err != nil {
		if errors.Is(err, repository.ErrPipelineConfigNameExists) { return nil, ErrConfigNameExists }; return nil, err
	}
	cfg.VersionCount = 1
	cfg.Versions = []models.PipelineConfigVersion{withoutContent(*version)}
	return cfg, nil
}

func (uc *Usecase) Update(ctx context.Context, id string, in UpdateConfigInput) (*models.PipelineConfig, error) {
	existing, err := uc.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	existing.Name = strings.TrimSpace(in.Name)
	existing.Description = strings.TrimSpace(in.Description)
	existing.Tags = normalizeTags(in.Tags)
	existing.FileType = normalizeFileType(in.FileType, existing.Name)
	lifecycle, err := normalizeLifecycle(in.Lifecycle, existing.Lifecycle)
	if err != nil {
		return nil, err
	}
	existing.Lifecycle = lifecycle
	if err := validateConfigMetadata(existing); err != nil {
		return nil, err
	}
	if err := uc.repo.UpdateMetadata(ctx, existing); err != nil {
		if errors.Is(err, repository.ErrPipelineConfigNotFound) {
			return nil, ErrConfigNotFound
		}
		if errors.Is(err, repository.ErrPipelineConfigNameExists) { return nil, ErrConfigNameExists }; return nil, err
	}
	return uc.Get(ctx, id)
}

func (uc *Usecase) CreateVersion(ctx context.Context, configID string, in CreateVersionInput) (*models.PipelineConfigVersion, error) {
	cfg, err := uc.Get(ctx, configID)
	if err != nil {
		return nil, err
	}
	status, err := normalizeLifecycle(in.Status, "draft")
	if err != nil {
		return nil, err
	}
	version := &models.PipelineConfigVersion{
		Status:  status,
		Content: in.Content,
		Summary: strings.TrimSpace(in.Summary),
		Author:  firstNonEmpty(strings.TrimSpace(in.Author), cfg.Owner),
	}
	if err := validateVersion(version); err != nil {
		return nil, err
	}
	if err := uc.repo.CreateVersion(ctx, cfg.ID, version); err != nil {
		if errors.Is(err, repository.ErrPipelineConfigNotFound) {
			return nil, ErrConfigNotFound
		}
		if errors.Is(err, repository.ErrPipelineConfigNameExists) { return nil, ErrConfigNameExists }; return nil, err
	}
	return version, nil
}

func (uc *Usecase) GetVersion(ctx context.Context, configID string, version int) (*models.PipelineConfigVersion, error) {
	if strings.TrimSpace(configID) == "" || version <= 0 {
		return nil, fmt.Errorf("%w: config id and version are required", ErrInvalidConfig)
	}
	item, err := uc.repo.FindVersion(ctx, configID, version)
	if err != nil {
		if errors.Is(err, repository.ErrPipelineConfigNotFound) {
			return nil, ErrConfigNotFound
		}
		return nil, err
	}
	if item == nil {
		return nil, ErrConfigNotFound
	}
	return item, nil
}

func (uc *Usecase) Deprecate(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: id is required", ErrInvalidConfig)
	}
	if err := uc.repo.Deprecate(ctx, id); err != nil {
		if errors.Is(err, repository.ErrPipelineConfigNotFound) {
			return ErrConfigNotFound
		}
		return err
	}
	return nil
}

func normalizeTags(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, tag := range in {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}
	return out
}

func normalizeLifecycle(value, defaultValue string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		if defaultValue == "" {
			defaultValue = "draft"
		}
		return defaultValue, nil
	case "draft":
		return "draft", nil
	case "ready":
		return "ready", nil
	case "deprecated":
		return "deprecated", nil
	default:
		return "", fmt.Errorf("%w: lifecycle must be draft, ready, or deprecated", ErrInvalidConfig)
	}
}

func normalizeFileType(value, name string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "json":
		return "json"
	case "yaml", "yml":
		return "yaml"
	}
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(name)))
	if ext == ".json" {
		return "json"
	}
	return "yaml"
}

func validateConfigMetadata(cfg *models.PipelineConfig) error {
	if cfg == nil {
		return fmt.Errorf("%w: config is required", ErrInvalidConfig)
	}
	if strings.TrimSpace(cfg.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidConfig)
	}
	if cfg.FileType != "yaml" && cfg.FileType != "json" {
		return fmt.Errorf("%w: fileType must be yaml or json", ErrInvalidConfig)
	}
	if cfg.Lifecycle != "draft" && cfg.Lifecycle != "ready" && cfg.Lifecycle != "deprecated" {
		return fmt.Errorf("%w: lifecycle must be draft, ready, or deprecated", ErrInvalidConfig)
	}
	return nil
}

func validateVersion(version *models.PipelineConfigVersion) error {
	if version == nil {
		return fmt.Errorf("%w: version is required", ErrInvalidConfig)
	}
	if strings.TrimSpace(version.Content) == "" {
		return fmt.Errorf("%w: content is required", ErrInvalidConfig)
	}
	if len([]byte(version.Content)) > MaxConfigFileBytes {
		return fmt.Errorf("%w: content must be <= %d bytes", ErrInvalidConfig, MaxConfigFileBytes)
	}
	if version.Status != "draft" && version.Status != "ready" && version.Status != "deprecated" {
		return fmt.Errorf("%w: status must be draft, ready, or deprecated", ErrInvalidConfig)
	}
	return nil
}

func withoutContent(version models.PipelineConfigVersion) models.PipelineConfigVersion {
	version.Content = ""
	return version
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
