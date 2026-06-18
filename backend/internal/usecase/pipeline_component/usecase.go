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
	repo        repository.PipelineComponentRepository
	releaseRepo repository.PipelineComponentReleaseRepository
}

// New creates a Usecase.
func New(repo repository.PipelineComponentRepository) *Usecase {
	uc := &Usecase{repo: repo}
	if releaseRepo, ok := repo.(repository.PipelineComponentReleaseRepository); ok {
		uc.releaseRepo = releaseRepo
	}
	return uc
}

// NewWithReleaseRepo creates a Usecase with independently wired release storage.
func NewWithReleaseRepo(repo repository.PipelineComponentRepository, releaseRepo repository.PipelineComponentReleaseRepository) *Usecase {
	return &Usecase{repo: repo, releaseRepo: releaseRepo}
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

// SyncReleases validates and persists generated component release records.
func (uc *Usecase) SyncReleases(ctx context.Context, releases []models.PipelineComponentRelease) ([]models.PipelineComponentRelease, error) {
	return uc.SyncReleaseManifest(ctx, models.ComponentReleaseIngestManifest{Items: releases})
}

// SyncReleaseManifest validates and persists a CI-generated component release
// manifest. Source fields are applied as defaults to each item before
// validation so CI can publish shared build context once per batch.
func (uc *Usecase) SyncReleaseManifest(ctx context.Context, manifest models.ComponentReleaseIngestManifest) ([]models.PipelineComponentRelease, error) {
	if uc.releaseRepo == nil {
		return nil, errors.New("component release repository is not configured")
	}
	out := make([]models.PipelineComponentRelease, 0, len(manifest.Items))
	for i := range manifest.Items {
		release := manifest.Items[i]
		applyIngestSource(&release, manifest.Source)
		if err := normalizeComponentRelease(&release); err != nil {
			return nil, err
		}
		if err := uc.releaseRepo.UpsertRelease(ctx, &release); err != nil {
			return nil, fmt.Errorf("sync component release %q/%q: %w", release.ComponentID, release.ReleaseLabel, err)
		}
		out = append(out, release)
	}
	return out, nil
}

// ListReleases returns generated component releases for component selection UI.
func (uc *Usecase) ListReleases(ctx context.Context, filter repository.ComponentReleaseFilter) ([]models.PipelineComponentRelease, error) {
	if uc.releaseRepo == nil {
		return nil, errors.New("component release repository is not configured")
	}
	items, err := uc.releaseRepo.FindReleases(ctx, &filter)
	if err != nil {
		return nil, err
	}
	normalized := make([]models.PipelineComponentRelease, 0, len(items))
	for i := range items {
		release := items[i]
		if err := normalizeComponentRelease(&release); err != nil {
			return nil, err
		}
		if filter.Selectable != nil && release.Selectable != *filter.Selectable {
			continue
		}
		normalized = append(normalized, release)
	}
	return normalized, nil
}

// GetRelease returns one generated component release by ID.
func (uc *Usecase) GetRelease(ctx context.Context, id string) (*models.PipelineComponentRelease, error) {
	if uc.releaseRepo == nil {
		return nil, errors.New("component release repository is not configured")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("id is required")
	}
	release, err := uc.releaseRepo.FindReleaseByID(ctx, id)
	if err != nil || release == nil {
		return release, err
	}
	if err := normalizeComponentRelease(release); err != nil {
		return nil, err
	}
	return release, nil
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

func normalizeComponentRelease(release *models.PipelineComponentRelease) error {
	if release == nil {
		return errors.New("component release is required")
	}
	release.ID = strings.TrimSpace(release.ID)
	release.ImageUID = strings.TrimSpace(release.ImageUID)
	release.ComponentID = strings.TrimSpace(release.ComponentID)
	release.TaskName = strings.TrimSpace(release.TaskName)
	release.TaskPath = strings.TrimSpace(release.TaskPath)
	release.DisplayName = strings.TrimSpace(release.DisplayName)
	release.Owner = strings.TrimSpace(release.Owner)
	release.ReleaseLabel = strings.TrimSpace(release.ReleaseLabel)
	release.Channel = strings.ToLower(strings.TrimSpace(release.Channel))
	release.SourceRepo = strings.TrimSpace(release.SourceRepo)
	release.SourceRef = strings.TrimSpace(release.SourceRef)
	release.SourceRefType = normalizeSourceRefType(release.SourceRefType)
	release.SourceCommit = normalizeStoredCommit(strings.TrimSpace(release.SourceCommit))
	release.BuildID = strings.TrimSpace(release.BuildID)
	release.ImageRepo = strings.TrimSpace(release.ImageRepo)
	release.ImageTag = strings.TrimSpace(release.ImageTag)
	release.ImageDigest = normalizeDigest(strings.TrimSpace(release.ImageDigest))
	release.RuntimeImage = strings.TrimSpace(release.RuntimeImage)
	release.Status = strings.ToLower(strings.TrimSpace(release.Status))
	release.ValidationStatus = strings.ToLower(strings.TrimSpace(release.ValidationStatus))

	if release.ComponentID == "" {
		release.ComponentID = release.TaskName
	}
	if release.TaskName == "" {
		release.TaskName = release.ComponentID
	}
	if release.DisplayName == "" {
		release.DisplayName = release.TaskName
	}
	if release.Owner == "" {
		release.Owner = "platform"
	}
	if release.SourceRefType == "" {
		release.SourceRefType = deriveSourceRefType(release.SourceRef, release.ReleaseLabel, release.SourceCommit)
	}
	if release.ReleaseLabel == "" {
		release.ReleaseLabel = deriveDefaultReleaseLabel(release.SourceRefType, release.SourceRef, release.SourceCommit, release.ImageTag)
	}
	if release.Channel == "" {
		release.Channel = deriveReleaseChannel(release.ReleaseLabel, release.SourceRefType)
	}
	if release.ID == "" && release.ComponentID != "" && release.ReleaseLabel != "" {
		release.ID = uuid.NewSHA1(uuid.NameSpaceURL, []byte(release.ComponentID+":"+release.ReleaseLabel)).String()
	}
	if release.RuntimeSnapshot.Image == "" {
		release.RuntimeSnapshot.Image = release.RuntimeImage
	}
	if release.RuntimeImage == "" {
		release.RuntimeImage = release.RuntimeSnapshot.Image
	}
	if release.ImageDigest == "" {
		release.ImageDigest = digestFromImage(release.RuntimeImage)
	}
	if release.RuntimeImage == "" && release.ImageRepo != "" && release.ImageDigest != "" {
		release.RuntimeImage = release.ImageRepo + "@" + release.ImageDigest
		release.RuntimeSnapshot.Image = release.RuntimeImage
	}
	if release.RuntimeSnapshot.Image == "" {
		release.RuntimeSnapshot.Image = release.RuntimeImage
	}
	if release.ImageUID == "" {
		release.ImageUID = models.ShortImageUID(firstNonEmpty(release.ImageDigest, release.RuntimeImage))
	}
	if release.RuntimeSnapshot.InputPorts == nil || len(release.RuntimeSnapshot.InputPorts) == 0 {
		release.RuntimeSnapshot.InputPorts = []models.PortDef{{Name: "input", Type: "asset"}}
	}
	if release.RuntimeSnapshot.OutputPorts == nil || len(release.RuntimeSnapshot.OutputPorts) == 0 {
		release.RuntimeSnapshot.OutputPorts = []models.PortDef{{Name: "output", Type: "asset"}}
	}
	if release.RuntimeSnapshot.Resources == nil {
		release.RuntimeSnapshot.Resources = map[string]interface{}{}
	}
	if release.TechnicalMetadata == nil {
		release.TechnicalMetadata = map[string]interface{}{}
	}
	if release.SourceRefType != "" {
		release.TechnicalMetadata["sourceRefType"] = release.SourceRefType
	}

	var validationErrors []string
	if release.ComponentID == "" {
		validationErrors = append(validationErrors, "componentId is required")
	}
	if release.TaskName == "" {
		validationErrors = append(validationErrors, "taskName is required")
	}
	if release.TaskPath != "" && !strings.HasPrefix(release.TaskPath, "tasks/") {
		validationErrors = append(validationErrors, "taskPath must be under tasks/")
	}
	if release.ReleaseLabel == "" {
		validationErrors = append(validationErrors, "releaseLabel is required")
	}
	runtimeDigest := digestFromImage(release.RuntimeImage)
	if release.RuntimeImage == "" {
		validationErrors = append(validationErrors, "runtimeImage is required")
	} else if !isSHA256Digest(runtimeDigest) {
		validationErrors = append(validationErrors, "runtimeImage must be digest pinned as image@sha256:<64 hex>")
	}
	if !isSHA256Digest(release.ImageDigest) {
		validationErrors = append(validationErrors, "imageDigest is required and must be sha256:<64 hex>")
	} else if runtimeDigest != "" && runtimeDigest != release.ImageDigest {
		validationErrors = append(validationErrors, "runtimeImage digest must match imageDigest")
	}
	if len(release.RuntimeSnapshot.Command) == 0 {
		validationErrors = append(validationErrors, "runtimeSnapshot.command is required")
	}
	if len(release.RuntimeSnapshot.Resources) == 0 {
		validationErrors = append(validationErrors, "runtimeSnapshot.resources is required")
	}

	if len(validationErrors) == 0 {
		release.ValidationStatus = "passed"
	} else {
		release.ValidationStatus = "failed"
	}
	release.ValidationErrors = validationErrors

	if release.Status == "" {
		if release.ValidationStatus == "passed" {
			release.Status = "ready"
		} else {
			release.Status = "failed"
		}
	}
	release.Selectable = release.ValidationStatus == "passed" && release.Status == "ready"
	return nil
}

func applyIngestSource(release *models.PipelineComponentRelease, source models.ComponentReleaseIngestSource) {
	source.Provider = strings.TrimSpace(source.Provider)
	source.Repo = strings.TrimSpace(source.Repo)
	source.Ref = strings.TrimSpace(source.Ref)
	source.RefType = normalizeSourceRefType(source.RefType)
	source.Commit = strings.TrimSpace(source.Commit)
	source.BuildID = strings.TrimSpace(source.BuildID)
	source.Trigger = strings.TrimSpace(source.Trigger)

	if release.SourceRepo == "" {
		release.SourceRepo = source.Repo
	}
	if release.SourceRef == "" {
		release.SourceRef = source.Ref
	}
	if release.SourceRefType == "" {
		release.SourceRefType = source.RefType
	}
	if release.SourceCommit == "" {
		release.SourceCommit = source.Commit
	}
	if release.BuildID == "" {
		release.BuildID = source.BuildID
	}
	if release.TechnicalMetadata == nil {
		release.TechnicalMetadata = map[string]interface{}{}
	}
	if source.Provider != "" {
		release.TechnicalMetadata["sourceProvider"] = source.Provider
	}
	if source.RefType != "" {
		release.TechnicalMetadata["sourceRefType"] = source.RefType
	}
	if source.Trigger != "" {
		release.TechnicalMetadata["sourceTrigger"] = source.Trigger
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func shortCommit(commit string) string {
	commit = normalizeStoredCommit(commit)
	if len(commit) <= 7 {
		return commit
	}
	return commit[:7]
}

func normalizeStoredCommit(commit string) string {
	commit = strings.ToLower(strings.TrimSpace(commit))
	if commit == "" {
		return ""
	}
	if len(commit) >= 40 && isCommitLike(commit) {
		trimmed := strings.TrimRight(commit, "0")
		if len(trimmed) >= 4 && len(trimmed) <= 12 {
			return trimmed
		}
	}
	return commit
}

func deriveReleaseChannel(label string, refType string) string {
	label = strings.ToLower(strings.TrimSpace(label))
	refType = normalizeSourceRefType(refType)
	if refType == "tag" {
		return "prod"
	}
	switch {
	case strings.HasPrefix(label, "pr-"):
		return "preview"
	case strings.Contains(label, "-rc."):
		return "rc"
	case strings.HasPrefix(label, "main-"):
		return "candidate"
	case strings.HasPrefix(label, "v"):
		return "prod"
	default:
		return "dev"
	}
}

func normalizeSourceRefType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "tag", "commit", "branch", "pr":
		return value
	default:
		return ""
	}
}

func deriveSourceRefType(ref string, label string, commit string) string {
	ref = strings.TrimSpace(ref)
	label = strings.TrimSpace(label)
	commit = strings.TrimSpace(commit)
	lowerRef := strings.ToLower(ref)
	lowerLabel := strings.ToLower(label)

	switch {
	case strings.HasPrefix(lowerRef, "refs/tags/"):
		return "tag"
	case strings.HasPrefix(lowerRef, "refs/pull/"), strings.HasPrefix(lowerLabel, "pr-"):
		return "pr"
	case strings.HasPrefix(lowerRef, "refs/heads/"):
		return "branch"
	case isCommitLike(ref):
		return "commit"
	case commit != "" && (label == shortCommit(commit) || lowerLabel == "commit-"+strings.ToLower(shortCommit(commit))):
		return "commit"
	case lowerRef != "":
		return "branch"
	default:
		return ""
	}
}

func deriveDefaultReleaseLabel(refType string, ref string, commit string, imageTag string) string {
	refType = normalizeSourceRefType(refType)
	refName := sourceRefName(ref)
	short := shortCommit(commit)
	switch refType {
	case "tag":
		return firstNonEmpty(refName, imageTag, short)
	case "commit":
		return firstNonEmpty(short, imageTag)
	case "pr":
		if refName != "" && short != "" {
			return refName + "-" + short
		}
		return firstNonEmpty(refName, short, imageTag)
	case "branch":
		if refName != "" && short != "" {
			return refName + "-" + short
		}
		return firstNonEmpty(imageTag, refName, short)
	default:
		return firstNonEmpty(imageTag, short, refName)
	}
}

func sourceRefName(ref string) string {
	ref = strings.TrimSpace(ref)
	switch {
	case strings.HasPrefix(ref, "refs/heads/"):
		return sanitizeReleasePart(strings.TrimPrefix(ref, "refs/heads/"))
	case strings.HasPrefix(ref, "refs/tags/"):
		return strings.TrimPrefix(ref, "refs/tags/")
	case strings.HasPrefix(ref, "refs/pull/"):
		parts := strings.Split(ref, "/")
		if len(parts) >= 3 {
			return "pr-" + parts[2]
		}
	}
	return sanitizeReleasePart(ref)
}

func sanitizeReleasePart(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, "_", "-")
	return strings.Trim(value, "-")
}

func isCommitLike(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) < 7 || len(value) > 64 {
		return false
	}
	for _, char := range value {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}
	return true
}

func normalizeDigest(digest string) string {
	digest = strings.TrimSpace(digest)
	if digest == "" {
		return ""
	}
	if strings.HasPrefix(digest, "sha256:") {
		return digest
	}
	if strings.HasPrefix(digest, "@sha256:") {
		return strings.TrimPrefix(digest, "@")
	}
	return digest
}

func digestFromImage(image string) string {
	parts := strings.Split(image, "@")
	if len(parts) != 2 {
		return ""
	}
	return normalizeDigest(parts[1])
}

func isSHA256Digest(digest string) bool {
	const prefix = "sha256:"
	if len(digest) != len(prefix)+64 || !strings.HasPrefix(digest, prefix) {
		return false
	}
	for _, char := range digest[len(prefix):] {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}
	return true
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
		Command:     []string{"sh", "-c", "echo pass"},
		InputPorts:  []models.PortDef{{Name: "input", Type: "asset", Desc: "输入资产"}},
		OutputPorts: []models.PortDef{{Name: "output", Type: "asset", Desc: "输出资产"}},
	},
}

// patchSystemComponent backfills missing fields on seeded system components.
func patchSystemComponent(existing, seed *models.PipelineComponent) bool {
	if existing == nil || seed == nil {
		return false
	}
	changed := false
	if len(existing.Command) == 0 && len(seed.Command) > 0 {
		existing.Command = append([]string(nil), seed.Command...)
		changed = true
	}
	return changed
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
			if patchSystemComponent(existing, &sc) {
				if err := normalizeComponent(existing, true); err != nil {
					return fmt.Errorf("patch system component %q: %w", sc.ID, err)
				}
				existing.UpdatedAt = time.Now().UTC()
				if err := uc.repo.Update(ctx, existing); err != nil {
					return fmt.Errorf("patch system component %q: %w", sc.ID, err)
				}
			}
			continue
		}
		now := time.Now().UTC()
		sc.CreatedAt = now
		sc.UpdatedAt = now
		if err := normalizeComponent(&sc, false); err != nil {
			return fmt.Errorf("seed system component %q: %w", sc.ID, err)
		}
		if err := uc.repo.Save(ctx, &sc); err != nil {
			return fmt.Errorf("seed system component %q: %w", sc.ID, err)
		}
	}
	return nil
}
