package asset

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"data-platform/internal/config"
	"data-platform/internal/middleware"
	"data-platform/internal/models"
	"data-platform/internal/repository"
)

var (
	ErrNotFound           = errors.New("asset not found")
	ErrInvalidRange       = errors.New("end_timestamp_ns must be greater than start_timestamp_ns")
	ErrMcapFileIDRequired = errors.New("mcap_file_id is required")
	ErrInvalidTag         = errors.New("invalid tag")
)

// defaultLifecycleMeta returns the lifecycle governance defaults for cf_meta.
// These fields are reserved at ingest time for future storage governance.
func defaultLifecycleMeta() map[string]interface{} {
	return map[string]interface{}{
		"retention_tier":     "standard",
		"archive_after_days": 90,
		"delete_after_days":  365,
		"total_size_bytes":   0,
		"last_accessed_at":   nil,
	}
}

type Usecase struct {
	repo         repository.AssetRepository
	tagRegistry  *config.TagRegistry
	algoRegistry *config.AlgoRegistry
	tx           repository.TxRunner
	tagRepo      repository.AssetTagRepository
	eventRepo    repository.AssetEventRepository
}

func New(repo repository.AssetRepository) *Usecase {
	return &Usecase{repo: repo}
}

// NewWithTagRegistry creates a Usecase with tag validation support.
func NewWithTagRegistry(repo repository.AssetRepository, tagReg *config.TagRegistry) *Usecase {
	return &Usecase{repo: repo, tagRegistry: tagReg}
}

// NewFull creates a Usecase with tag validation and algo registry support.
// The algo registry is used to initialize algorithm states on asset creation.
func NewFull(repo repository.AssetRepository, tagReg *config.TagRegistry, algoReg *config.AlgoRegistry) *Usecase {
	return &Usecase{repo: repo, tagRegistry: tagReg, algoRegistry: algoReg}
}

// NewWithProjections wires the asset usecase with the projection and outbox
// repositories so asset mutations can update current-state tables and append
// business events in the same transaction.
func NewWithProjections(
	tx repository.TxRunner,
	repo repository.AssetRepository,
	tagRepo repository.AssetTagRepository,
	eventRepo repository.AssetEventRepository,
	tagReg *config.TagRegistry,
	algoReg *config.AlgoRegistry,
) *Usecase {
	return &Usecase{
		repo:         repo,
		tagRegistry:  tagReg,
		algoRegistry: algoReg,
		tx:           tx,
		tagRepo:      tagRepo,
		eventRepo:    eventRepo,
	}
}

func (u *Usecase) withMutationTx(ctx context.Context, fn func(context.Context) error) error {
	if u.tx == nil {
		return fn(ctx)
	}
	return u.tx.WithTx(ctx, fn)
}

func (u *Usecase) tagTypeFor(key string) string {
	if u.tagRegistry == nil {
		return "string"
	}
	if def, ok := u.tagRegistry.GetAllTags()[key]; ok && def.Type != "" {
		return def.Type
	}
	return "string"
}

func jsonPayload(v map[string]any) []byte {
	if v == nil {
		return []byte(`{}`)
	}
	b, err := json.Marshal(v)
	if err != nil {
		return []byte(`{}`)
	}
	return b
}

func (u *Usecase) appendAssetEvent(ctx context.Context, eventType string, a *models.Asset, payload map[string]any) error {
	if u.eventRepo == nil || a == nil {
		return nil
	}
	return u.eventRepo.Append(ctx, repository.AssetEventAppendInput{
		EventType:     eventType,
		AssetID:       a.AssetID,
		McapFileID:    a.McapFileID,
		TenantID:      a.TenantID,
		ProjectID:     a.ProjectID,
		EventSource:   "backend",
		RequestID:     middleware.RequestIDFromContext(ctx),
		EventPayload:  jsonPayload(payload),
		AggregateType: "asset",
	})
}

func (u *Usecase) upsertTagProjection(ctx context.Context, a *models.Asset, tags map[string]string, sourceType string) error {
	if u.tagRepo == nil || a == nil {
		return nil
	}
	for k, v := range tags {
		tagType := u.tagTypeFor(k)
		if err := u.tagRepo.Upsert(ctx, a.AssetID, k, v, tagType, sourceType); err != nil {
			return err
		}
		if err := u.appendAssetEvent(ctx, "tag_upserted", a, map[string]any{
			"tag_key":     k,
			"tag_value":   v,
			"tag_type":    tagType,
			"source_type": sourceType,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (u *Usecase) persistNewAsset(ctx context.Context, a *models.Asset, tags map[string]string) error {
	if err := u.withMutationTx(ctx, func(txCtx context.Context) error {
		if err := u.repo.Set(txCtx, a); err != nil {
			return err
		}
		if err := u.appendAssetEvent(txCtx, "asset_created", a, map[string]any{
			"asset_id":        a.AssetID,
			"mcap_file_id":    a.McapFileID,
			"segment_locator": a.SegmentLocator,
			"lifecycle_state": a.LifecycleState,
			"asset_type":      a.AssetType,
			"owner":           a.Owner,
			"reviewer":        a.Reviewer,
		}); err != nil {
			return err
		}
		return u.upsertTagProjection(txCtx, a, tags, "manual")
	}); err != nil {
		return err
	}
	// Secondary index write is best-effort for now.
	return u.repo.WriteSegmentIndex(ctx, a)
}

type CreateInput struct {
	McapFileID       string
	StartTimestampNs int64
	EndTimestampNs   int64
	Reviewer         string
	Owner            string
	SegType          string
	Env              string
	Task             string
	Tags             map[string]string
}

type UpdateInput struct {
	Status   *string
	Reviewer *string
	Owner    *string
	Tags     map[string]string
}

type CommitSegmentsInput struct {
	McapFileID string
	Ranges     [][2]int64
	Reviewer   string
	Owner      string
}

func (u *Usecase) Get(ctx context.Context, assetID string) (*models.Asset, error) {
	a, err := u.repo.Get(ctx, assetID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrNotFound
	}
	return a, nil
}

func (u *Usecase) ListByMcapFile(ctx context.Context, mcapFileID string) ([]*models.Asset, error) {
	if mcapFileID == "" {
		return nil, ErrMcapFileIDRequired
	}
	return u.repo.ListByMcapFile(ctx, mcapFileID)
}

// ListWithFilters queries assets using a parameterized WHERE clause with pagination and sorting.
func (u *Usecase) ListWithFilters(ctx context.Context, whereSQL string, args []interface{},
	page, pageSize int, orderBy string) ([]*models.Asset, int64, error) {
	return u.repo.ListWithFilters(ctx, whereSQL, args, page, pageSize, orderBy)
}

func (u *Usecase) Create(ctx context.Context, in CreateInput) (*models.Asset, error) {
	if in.McapFileID == "" {
		return nil, ErrMcapFileIDRequired
	}
	if in.EndTimestampNs <= in.StartTimestampNs {
		return nil, ErrInvalidRange
	}
	tags := in.Tags
	if tags == nil {
		tags = map[string]string{}
	}
	// Validate tags if registry is available.
	if u.tagRegistry != nil {
		for k, v := range tags {
			if err := u.tagRegistry.Validate(k, v); err != nil {
				return nil, fmt.Errorf("%w: %s", ErrInvalidTag, err.Error())
			}
		}
	}
	a := &models.Asset{
		AssetID:          uuid.NewString(),
		McapFileID:       in.McapFileID,
		StartTimestampNs: in.StartTimestampNs,
		EndTimestampNs:   in.EndTimestampNs,
		DurationSec:      float64(in.EndTimestampNs-in.StartTimestampNs) / 1e9,
		Reviewer:         in.Reviewer,
		Status:           models.AssetStatusApproved,
		Owner:            in.Owner,
		SegType:          in.SegType,
		Env:              in.Env,
		Task:             in.Task,
		Tags:             tags,
		AlgoResults:      map[string]string{},
		Files:            map[string]string{},
		LifecycleMeta:    defaultLifecycleMeta(),
		CreatedAt:        time.Now(),
	}
	// Initialize algorithm states from algo_registry if available.
	if u.algoRegistry != nil {
		initAlgoStates(a, u.algoRegistry)
		// Write raw_mcap reference to cf_files.
		a.Files["raw_mcap"] = in.McapFileID
	}
	if err := u.persistNewAsset(ctx, a, tags); err != nil {
		return nil, err
	}
	return a, nil
}

func (u *Usecase) Update(ctx context.Context, assetID string, in UpdateInput) (*models.Asset, error) {
	a, err := u.repo.Get(ctx, assetID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrNotFound
	}
	prevStatus := string(a.Status)
	prevLifecycle := a.LifecycleState
	if in.Status != nil {
		a.Status = models.AssetStatus(*in.Status)
	}
	if in.Reviewer != nil {
		a.Reviewer = *in.Reviewer
	}
	if in.Owner != nil {
		a.Owner = *in.Owner
	}
	// Validate tags if registry is available.
	if u.tagRegistry != nil {
		for k, v := range in.Tags {
			if err := u.tagRegistry.Validate(k, v); err != nil {
				return nil, fmt.Errorf("%w: %s", ErrInvalidTag, err.Error())
			}
		}
	}
	for k, v := range in.Tags {
		if a.Tags == nil {
			a.Tags = map[string]string{}
		}
		a.Tags[k] = v
	}

	if err := u.withMutationTx(ctx, func(txCtx context.Context) error {
		if err := u.repo.Set(txCtx, a); err != nil {
			return err
		}
		if err := u.appendAssetEvent(txCtx, "asset_updated", a, map[string]any{
			"asset_id":        a.AssetID,
			"lifecycle_state": a.LifecycleState,
			"owner":           a.Owner,
			"reviewer":        a.Reviewer,
		}); err != nil {
			return err
		}
		if in.Status != nil && (prevStatus != string(a.Status) || prevLifecycle != a.LifecycleState) {
			if err := u.appendAssetEvent(txCtx, "asset_lifecycle_changed", a, map[string]any{
				"prev_status":          prevStatus,
				"new_status":           string(a.Status),
				"prev_lifecycle_state": prevLifecycle,
				"new_lifecycle_state":  a.LifecycleState,
			}); err != nil {
				return err
			}
		}
		return u.upsertTagProjection(txCtx, a, in.Tags, "manual")
	}); err != nil {
		return nil, err
	}
	return a, nil
}

func (u *Usecase) Delete(ctx context.Context, assetID string) error {
	a, err := u.repo.Get(ctx, assetID)
	if err != nil {
		return err
	}
	if a == nil {
		return ErrNotFound
	}
	prevStatus := string(a.Status)
	prevLifecycle := a.LifecycleState
	return u.withMutationTx(ctx, func(txCtx context.Context) error {
		if err := u.repo.SoftDelete(txCtx, assetID); err != nil {
			return err
		}
		return u.appendAssetEvent(txCtx, "asset_lifecycle_changed", a, map[string]any{
			"prev_status":          prevStatus,
			"new_status":           string(models.AssetStatusArchived),
			"prev_lifecycle_state": prevLifecycle,
			"new_lifecycle_state":  "archived",
		})
	})
}

func (u *Usecase) CommitSegments(ctx context.Context, in CommitSegmentsInput) ([]string, error) {
	if in.McapFileID == "" {
		return nil, ErrMcapFileIDRequired
	}
	var created []string
	for _, r := range in.Ranges {
		startNs, endNs := r[0], r[1]
		if endNs <= startNs {
			return created, ErrInvalidRange
		}
		a := &models.Asset{
			AssetID:          uuid.NewString(),
			McapFileID:       in.McapFileID,
			StartTimestampNs: startNs,
			EndTimestampNs:   endNs,
			DurationSec:      float64(endNs-startNs) / 1e9,
			Reviewer:         in.Reviewer,
			Status:           models.AssetStatusApproved,
			Owner:            in.Owner,
			AlgoResults:      map[string]string{},
			Tags:             map[string]string{},
			Files:            map[string]string{},
			LifecycleMeta:    defaultLifecycleMeta(),
			CreatedAt:        time.Now(),
		}
		// Initialize algorithm states from algo_registry if available.
		if u.algoRegistry != nil {
			initAlgoStates(a, u.algoRegistry)
			a.Files["raw_mcap"] = in.McapFileID
		}
		if err := u.persistNewAsset(ctx, a, a.Tags); err != nil {
			return created, err
		}
		created = append(created, a.AssetID)
	}
	return created, nil
}

// initAlgoStates initializes algorithm statuses on a new asset based on the algo registry.
// Algorithms with no dependencies get status "pending"; those with dependencies get "blocked".
func initAlgoStates(a *models.Asset, reg *config.AlgoRegistry) {
	allAlgos := reg.GetAllAlgorithms()
	for algoName, def := range allAlgos {
		for _, ver := range def.Versions {
			algoKey := algoName + "@" + ver
			statusKey := algoKey + ":" + models.AlgoFieldStatus
			if len(def.DependsOn) == 0 {
				a.AlgoResults[statusKey] = string(models.AlgoStatusPending)
			} else {
				a.AlgoResults[statusKey] = string(models.AlgoStatusBlocked)
			}
		}
	}
}
