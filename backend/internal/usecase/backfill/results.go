package backfill

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	"github.com/CyberOrigin2077/cyber-databrew/internal/usecase/assetvalidation"
)

var (
	ErrAssetNotFound         = errors.New("asset not found")
	ErrManifestMismatch      = errors.New("manifest asset_id mismatch")
	ErrReportManifestMissing = errors.New("report manifest not found")
	ErrPayloadTooLarge       = errors.New("result payload too large")
	ErrInvalidUploadRequest  = errors.New("invalid backfill result upload request")
)

type UploadResultInput struct {
	AssetID  string
	ReportID string
	Version  string
	Manifest map[string]any
	Result   map[string]any
}

type UploadResultOutput struct {
	AssetID  string `json:"assetId"`
	ReportID string `json:"reportId"`
	Version  string `json:"version"`
	AlgoKey  string `json:"algoKey"`
	Status   string `json:"status"`
}

func (uc *Usecase) UploadResult(ctx context.Context, in UploadResultInput) (*UploadResultOutput, error) {
	assetID := strings.TrimSpace(in.AssetID)
	reportID := strings.TrimSpace(in.ReportID)
	version := strings.TrimSpace(in.Version)
	if assetID == "" || reportID == "" || version == "" {
		return nil, fmt.Errorf("%w: assetId, reportId, and version are required", ErrInvalidUploadRequest)
	}
	if uc.resultRepo == nil || uc.assetRepo == nil {
		return nil, errors.New("backfill result upload is not configured")
	}
	if in.Result == nil {
		in.Result = map[string]any{}
	}
	payloadBytes, err := json.Marshal(in.Result)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid result payload", ErrInvalidUploadRequest)
	}
	if len(payloadBytes) > MaxBackfillResultPayloadBytes {
		return nil, fmt.Errorf("%w: max %d bytes", ErrPayloadTooLarge, MaxBackfillResultPayloadBytes)
	}
	if err := validateManifestAssetID(in.Manifest, assetID); err != nil {
		return nil, err
	}

	normalized, err := assetvalidation.NormalizeAssetIDs("assetId", []string{assetID})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidUploadRequest, err)
	}
	assetID = normalized[0]

	existing, err := uc.assetRepo.FindExistingIDs(ctx, []string{assetID})
	if err != nil {
		return nil, fmt.Errorf("lookup asset: %w", err)
	}
	if len(existing) == 0 {
		return nil, ErrAssetNotFound
	}

	manifest, err := uc.resultRepo.GetReportManifest(ctx, reportID)
	if err != nil {
		if errors.Is(err, repository.ErrReportManifestNotFound) {
			return nil, ErrReportManifestMissing
		}
		return nil, err
	}
	if manifest.Version != "" && manifest.Version != version {
		return nil, fmt.Errorf("%w: expected version %q", ErrInvalidUploadRequest, manifest.Version)
	}

	algoKey := reportID
	if err := uc.resultRepo.UpsertAlgoRunResult(ctx, repository.AlgoRunResultWriteInput{
		AssetID:       assetID,
		AlgoKey:       algoKey,
		Version:       version,
		ReportID:      reportID,
		ResultPayload: in.Result,
	}); err != nil {
		return nil, err
	}

	if err := uc.markItemsRegistered(ctx, assetID, reportID, version); err != nil {
		return nil, err
	}

	return &UploadResultOutput{
		AssetID:  assetID,
		ReportID: reportID,
		Version:  version,
		AlgoKey:  algoKey,
		Status:   "registered",
	}, nil
}

func validateManifestAssetID(manifest map[string]any, assetID string) error {
	if len(manifest) == 0 {
		return nil
	}
	raw, ok := manifest["asset_id"]
	if !ok {
		raw, ok = manifest["assetId"]
	}
	if !ok {
		return nil
	}
	manifestAssetID, ok := raw.(string)
	if !ok || strings.TrimSpace(manifestAssetID) == "" {
		return fmt.Errorf("%w: manifest asset_id must be a string", ErrInvalidUploadRequest)
	}
	if strings.TrimSpace(manifestAssetID) != assetID {
		return ErrManifestMismatch
	}
	return nil
}

func (uc *Usecase) markItemsRegistered(ctx context.Context, assetID, reportID, version string) error {
	items, err := uc.repo.FindItemsByAssetID(ctx, assetID)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.Status != "awaiting_result" {
			continue
		}
		if !itemExpectsReport(item, reportID, version) {
			continue
		}
		if err := uc.repo.UpdateItemStatus(ctx, item.ID, "completed", "", ""); err != nil {
			return err
		}
	}
	return nil
}
