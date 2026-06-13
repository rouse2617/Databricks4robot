package assetvalidation

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

var ErrInvalidAssetIDs = errors.New("invalid asset ids")

type ValidationError struct {
	Field        string
	MissingIDs   []string
	InvalidIDs   []string
	DuplicateIDs []string
}

func (e *ValidationError) Error() string {
	return e.Message()
}

func (e *ValidationError) Unwrap() error {
	return ErrInvalidAssetIDs
}

func (e *ValidationError) Message() string {
	field := strings.TrimSpace(e.Field)
	if field == "" {
		field = "asset_ids"
	}
	switch {
	case len(e.MissingIDs) > 0:
		return field + " contain unknown assets"
	case len(e.DuplicateIDs) > 0:
		return field + " contain duplicate assets"
	default:
		return field + " contain invalid asset ids"
	}
}

func (e *ValidationError) Details() map[string]any {
	details := map[string]any{"field": e.Field}
	if len(e.MissingIDs) > 0 {
		details["missing_asset_ids"] = e.MissingIDs
	}
	if len(e.InvalidIDs) > 0 {
		details["invalid_asset_ids"] = e.InvalidIDs
	}
	if len(e.DuplicateIDs) > 0 {
		details["duplicate_asset_ids"] = e.DuplicateIDs
	}
	return details
}

// NormalizeAssetIDs trims, drops empty values, and deduplicates while preserving order.
// It does not check whether assets exist in the database.
func NormalizeAssetIDs(field string, assetIDs []string) ([]string, error) {
	if len(assetIDs) == 0 {
		return nil, nil
	}

	result := make([]string, 0, len(assetIDs))
	seen := make(map[string]struct{}, len(assetIDs))
	var invalid []string

	for _, raw := range assetIDs {
		assetID := strings.TrimSpace(raw)
		if assetID == "" {
			invalid = append(invalid, raw)
			continue
		}
		if _, ok := seen[assetID]; ok {
			continue
		}
		seen[assetID] = struct{}{}
		result = append(result, assetID)
	}

	if len(invalid) > 0 {
		return nil, &ValidationError{Field: field, InvalidIDs: invalid}
	}
	return result, nil
}

func Validate(ctx context.Context, repo repository.AssetRepository, field string, assetIDs []string) ([]string, error) {
	if len(assetIDs) == 0 {
		return nil, nil
	}

	normalized := make([]string, 0, len(assetIDs))
	unique := make([]string, 0, len(assetIDs))
	seen := make(map[string]struct{}, len(assetIDs))
	duplicateSeen := make(map[string]struct{})
	var invalid []string
	var duplicates []string

	for _, raw := range assetIDs {
		assetID := strings.TrimSpace(raw)
		normalized = append(normalized, assetID)
		if assetID == "" {
			invalid = append(invalid, assetID)
			continue
		}
		if _, ok := seen[assetID]; ok {
			if _, duplicateRecorded := duplicateSeen[assetID]; !duplicateRecorded {
				duplicates = append(duplicates, assetID)
				duplicateSeen[assetID] = struct{}{}
			}
			continue
		}
		seen[assetID] = struct{}{}
		unique = append(unique, assetID)
	}

	var missing []string
	if repo != nil && len(unique) > 0 {
		existing, err := repo.FindExistingIDs(ctx, unique)
		if err != nil {
			return nil, fmt.Errorf("validate %s: %w", field, err)
		}
		for _, assetID := range unique {
			if _, ok := existing[assetID]; !ok {
				missing = append(missing, assetID)
			}
		}
	}

	if len(invalid) > 0 || len(duplicates) > 0 || len(missing) > 0 {
		return nil, &ValidationError{
			Field:        field,
			MissingIDs:   missing,
			InvalidIDs:   invalid,
			DuplicateIDs: duplicates,
		}
	}
	return normalized, nil
}
