package asset

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"data-platform/internal/models"
	"data-platform/internal/repository"
)

var (
	ErrNotFound            = errors.New("asset not found")
	ErrInvalidRange        = errors.New("end_timestamp_ns must be greater than start_timestamp_ns")
	ErrMcapFileIDRequired  = errors.New("mcap_file_id is required")
)

type Usecase struct {
	repo repository.AssetRepository
}

func New(repo repository.AssetRepository) *Usecase {
	return &Usecase{repo: repo}
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
		CreatedAt:        time.Now(),
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
			CreatedAt:        time.Now(),
		}
		if err := u.repo.Set(ctx, a); err != nil {
			return created, err
		}
		_ = u.repo.WriteSegmentIndex(ctx, a)
		created = append(created, a.AssetID)
	}
	return created, nil
}

