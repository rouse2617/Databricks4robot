package asset

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/id"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

var (
	ErrLogicalAssetNotFound      = errors.New("logical asset not found")
	ErrLogicalAssetTypeMismatch  = errors.New("logical asset type mismatch")
	ErrLogicalRevisionOutOfOrder = errors.New("revision must be max+1")
)

func (u *Usecase) seedFirstVersion(ctx context.Context, a *models.Asset) error {
	if u.logicalRepo == nil {
		return nil
	}
	a.LogicalAssetID = a.AssetID
	a.Revision = 1
	a.IsCurrent = true
	la := &models.LogicalAsset{
		LogicalAssetID:  a.LogicalAssetID,
		AssetType:       a.AssetType,
		Owner:           a.Owner,
		CurrentRevision: 1,
		TotalRevisions:  1,
	}
	return u.logicalRepo.Insert(ctx, la)
}

type promoteVersionInput struct {
	LogicalAssetID string
	PriorAssetID   string
	NewRevision    int64
	PromoteReason  string
	RunID          string
}

func (u *Usecase) preparePromoteVersion(ctx context.Context, a *models.Asset, logicalAssetID string) (*promoteVersionInput, error) {
	if u.logicalRepo == nil {
		return nil, fmt.Errorf("logical asset repository not configured")
	}
	la, err := u.logicalRepo.Get(ctx, logicalAssetID)
	if err != nil {
		return nil, err
	}
	if la == nil {
		return nil, ErrLogicalAssetNotFound
	}
	if la.AssetType != a.AssetType {
		return nil, ErrLogicalAssetTypeMismatch
	}
	maxRev, err := u.logicalRepo.MaxRevision(ctx, logicalAssetID)
	if err != nil {
		return nil, err
	}
	next := maxRev + 1
	priorID, err := u.logicalRepo.CurrentAssetID(ctx, logicalAssetID)
	if err != nil {
		return nil, err
	}
	if priorID == "" && maxRev > 0 {
		return nil, fmt.Errorf("%w: no current revision row", ErrLogicalRevisionOutOfOrder)
	}
	a.LogicalAssetID = logicalAssetID
	a.Revision = next
	a.IsCurrent = true
	return &promoteVersionInput{
		LogicalAssetID: logicalAssetID,
		PriorAssetID:   priorID,
		NewRevision:    next,
		PromoteReason:  a.SplitReason,
		RunID:          a.SplitRunID,
	}, nil
}

// PromoteInput holds parameters for the B-route promote endpoint.
type PromoteInput struct {
	LogicalAssetID string
	RevisionOf     string // optional: explicit prior asset id
	Owner          string
}

// Promote creates a new revision of an asset within a logical asset family (B-route).
// It clones the source asset's metadata into a new asset entry, increments the revision,
// demotes the prior current asset, and writes a version_promoted event.
func (u *Usecase) Promote(ctx context.Context, sourceAssetID string, in PromoteInput) (*models.Asset, error) {
	source, err := u.repo.Get(ctx, sourceAssetID)
	if err != nil {
		return nil, err
	}
	if source == nil {
		return nil, ErrNotFound
	}

	logicalID := in.LogicalAssetID
	if logicalID == "" {
		// Default to source's own logical_asset_id if not provided.
		logicalID = source.LogicalAssetID
	}
	if logicalID == "" {
		return nil, fmt.Errorf("logical_asset_id is required")
	}

	// Clone source asset into a new asset entry.
	newAsset := &models.Asset{
		McapFileID:          source.McapFileID,
		StartTimestampNs:    source.StartTimestampNs,
		EndTimestampNs:      source.EndTimestampNs,
		DurationMs:          source.DurationMs,
		AssetType:           source.AssetType,
		Owner:               in.Owner,
		Reviewer:            source.Reviewer,
		RetentionTier:       source.RetentionTier,
		ExpireAt:            source.ExpireAt,
		StorageURI:          source.StorageURI,
		ThumbURI:            source.ThumbURI,
		AssetLevel:          source.AssetLevel,
		ParentAssetID:       source.ParentAssetID,
		RootAssetID:         source.RootAssetID,
		TenantID:            source.TenantID,
		ProjectID:           source.ProjectID,
		Metadata:            source.Metadata,
		FilesJSON:           source.FilesJSON,
		AlgoInputsURIs:      source.AlgoInputsURIs,
		AnnotInputsURIs:     source.AnnotInputsURIs,
		SplitMethod:         source.SplitMethod,
		SplitAlgoName:       source.SplitAlgoName,
		SplitAlgoVersion:    source.SplitAlgoVersion,
		SegmentIndex:        source.SegmentIndex,
		ParentStartOffsetMs: source.ParentStartOffsetMs,
		ParentEndOffsetMs:   source.ParentEndOffsetMs,
		Tags:                map[string]string{},
		AlgoResults:         map[string]string{},
		Files:               map[string]string{},
		LifecycleMeta:       defaultLifecycleMeta(),
	}
	if newAsset.Owner == "" {
		newAsset.Owner = source.Owner
	}
	// Copy files.
	if source.Files != nil {
		for k, v := range source.Files {
			newAsset.Files[k] = v
		}
	}
	// Copy algo results.
	if source.AlgoResults != nil {
		for k, v := range source.AlgoResults {
			newAsset.AlgoResults[k] = v
		}
	}
	if newAsset.LifecycleState == "" {
		newAsset.LifecycleState = "ready"
	}

	// Allocate a new asset ID and persist via the versioning flow.
	var result *models.Asset
	err = u.withMutationTx(ctx, func(txCtx context.Context) error {
		promoteIn, err := u.preparePromoteVersion(txCtx, newAsset, logicalID)
		if err != nil {
			return err
		}
		// Override prior asset if revision_of is explicitly provided.
		if in.RevisionOf != "" {
			promoteIn.PriorAssetID = in.RevisionOf
		}

		// Allocate a new asset ID with retry on collision.
		for range maxAssetIDAllocationAttempts {
			gid, gidErr := id.GenerateAssetID()
			if gidErr != nil {
				return gidErr
			}
			newAsset.AssetID = gid
			newAsset.Version = 0
			newAsset.CreatedAt = time.Time{}

			finalizeErr := u.finalizePromoteVersion(txCtx, newAsset, promoteIn)
			if finalizeErr != nil {
				if errors.Is(finalizeErr, repository.ErrDuplicateAssetID) {
					continue
				}
				return finalizeErr
			}
			result = newAsset
			return nil
		}
		return fmt.Errorf("exhausted asset id allocation attempts (%d)", maxAssetIDAllocationAttempts)
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// CurrentAssetResponse holds the response for GET /logical-assets/{id}/current.
type CurrentAssetResponse struct {
	AssetID        string `json:"asset_id"`
	Revision       int64  `json:"revision"`
	LogicalAssetID string `json:"logical_asset_id"`
	AssetType      string `json:"asset_type"`
	Owner          string `json:"owner"`
	LifecycleState string `json:"lifecycle_state"`
}

// GetCurrentForLogical returns the current revision for a logical asset.
func (u *Usecase) GetCurrentForLogical(ctx context.Context, logicalAssetID string) (*CurrentAssetResponse, error) {
	if u.logicalRepo == nil {
		return nil, fmt.Errorf("logical asset repository not configured")
	}
	la, err := u.logicalRepo.Get(ctx, logicalAssetID)
	if err != nil {
		return nil, err
	}
	if la == nil {
		return nil, ErrLogicalAssetNotFound
	}
	assetID, err := u.logicalRepo.CurrentAssetID(ctx, logicalAssetID)
	if err != nil {
		return nil, err
	}
	if assetID == "" {
		return nil, ErrNotFound
	}
	a, err := u.repo.Get(ctx, assetID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrNotFound
	}
	return &CurrentAssetResponse{
		AssetID:        a.AssetID,
		Revision:       a.Revision,
		LogicalAssetID: logicalAssetID,
		AssetType:      a.AssetType,
		Owner:          a.Owner,
		LifecycleState: a.LifecycleState,
	}, nil
}

func (u *Usecase) finalizePromoteVersion(ctx context.Context, a *models.Asset, in *promoteVersionInput) error {
	if err := u.logicalRepo.ClearCurrentForLogical(ctx, in.LogicalAssetID); err != nil {
		return err
	}
	if err := u.repo.InsertNew(ctx, a); err != nil {
		return err
	}
	if err := u.logicalRepo.BumpRevision(ctx, in.LogicalAssetID, in.NewRevision); err != nil {
		return err
	}
	if in.PriorAssetID != "" {
		if relRepo, ok := u.repo.(repository.AssetRelationWriter); ok {
			if err := relRepo.InsertRevisionOf(ctx, a.AssetID, in.PriorAssetID, in.RunID); err != nil {
				return err
			}
		}
	}
	payload := map[string]any{
		"logical_asset_id": in.LogicalAssetID,
		"revision":         in.NewRevision,
		"prior_asset_id":   in.PriorAssetID,
		"new_asset_id":     a.AssetID,
	}
	if in.RunID != "" {
		payload["run_id"] = in.RunID
	}
	if in.PromoteReason != "" {
		payload["reason"] = in.PromoteReason
	}
	if err := u.appendAssetEvent(ctx, "version_promoted", a, payload); err != nil {
		return err
	}
	if in.PriorAssetID != "" {
		prior := &models.Asset{AssetID: in.PriorAssetID, LogicalAssetID: in.LogicalAssetID, IsCurrent: false}
		return u.appendAssetEvent(ctx, "asset_updated", prior, map[string]any{
			"reason":           "version_demoted",
			"is_current":       false,
			"logical_asset_id": in.LogicalAssetID,
		})
	}
	return nil
}
