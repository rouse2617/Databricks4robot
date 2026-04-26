package asset

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"data-platform/internal/config"
	"data-platform/internal/models"
	"data-platform/internal/repository"
)

var (
	ErrNotFound            = errors.New("asset not found")
	ErrInvalidRange        = errors.New("end_timestamp_ns must be greater than start_timestamp_ns")
	ErrMcapFileIDRequired  = errors.New("mcap_file_id is required")
	ErrInvalidTag          = errors.New("invalid tag")
)

// defaultLifecycleMeta returns the lifecycle governance defaults for cf_meta.
// These fields are reserved at ingest time for future storage governance.
func defaultLifecycleMeta() map[string]interface{} {
	return map[string]interface{}{
		"retention_tier":    "standard",
		"archive_after_days": 90,
		"delete_after_days":  365,
		"total_size_bytes":   0,
		"last_accessed_at":   nil,
	}
}

type Usecase struct {
	repo        repository.AssetRepository
	tagRegistry *config.TagRegistry
	algoRegistry *config.AlgoRegistry
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
	if err := u.repo.Set(ctx, a); err != nil {
		return nil, err
	}
	// Secondary index write is best-effort for now.
	_ = u.repo.WriteSegmentIndex(ctx, a)
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
		a.Tags[k] = v
	}
	if err := u.repo.Set(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (u *Usecase) Delete(ctx context.Context, assetID string) error {
	return u.repo.SoftDelete(ctx, assetID)
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
		if err := u.repo.Set(ctx, a); err != nil {
			return created, err
		}
		_ = u.repo.WriteSegmentIndex(ctx, a)
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
