package asset

import (
	"context"
	"errors"
	"fmt"

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
	return u.appendAssetEvent(ctx, "version_promoted", a, payload)
}
