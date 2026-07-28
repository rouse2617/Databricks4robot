package assetvalidation

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

type stubAssetRepo struct {
	existing map[string]struct{}
	calls    int
	lastIDs  []string
	err      error
}

func (s *stubAssetRepo) FindExistingIDs(_ context.Context, assetIDs []string) (map[string]struct{}, error) {
	s.calls++
	s.lastIDs = append([]string(nil), assetIDs...)
	if s.err != nil {
		return nil, s.err
	}
	return s.existing, nil
}
func (s *stubAssetRepo) Get(context.Context, string) (*models.Asset, error)    { return nil, nil }
func (s *stubAssetRepo) GetAll(context.Context, string) (*models.Asset, error) { return nil, nil }
func (s *stubAssetRepo) InsertNew(context.Context, *models.Asset) error        { return nil }
func (s *stubAssetRepo) Set(context.Context, *models.Asset) error              { return nil }
func (s *stubAssetRepo) SoftDelete(context.Context, string) error              { return nil }
func (s *stubAssetRepo) ListByMcapFile(context.Context, string) ([]*models.Asset, error) {
	return nil, nil
}
func (s *stubAssetRepo) ListByLogicalAssetID(context.Context, string) ([]*models.Asset, error) {
	return nil, nil
}
func (s *stubAssetRepo) WriteSegmentIndex(context.Context, *models.Asset) error { return nil }
func (s *stubAssetRepo) ListWithFilters(context.Context, string, []interface{}, int, int, filter.OrderByClause) ([]*models.Asset, int64, error) {
	return nil, 0, nil
}
func (s *stubAssetRepo) ListDescendants(context.Context, string) ([]*models.Asset, error) {
	return nil, nil
}

func TestValidate_EmptySkipsLookup(t *testing.T) {
	repo := &stubAssetRepo{}
	got, err := Validate(context.Background(), repo, "asset_ids", nil)
	if err != nil {
		t.Fatalf("Validate err=%v", err)
	}
	if got != nil || repo.calls != 0 {
		t.Fatalf("expected no IDs and no lookup, got ids=%v calls=%d", got, repo.calls)
	}
}

func TestValidate_NormalizesAndBatchLooksUpUniqueIDs(t *testing.T) {
	repo := &stubAssetRepo{existing: map[string]struct{}{"a1": {}, "a2": {}}}
	got, err := Validate(context.Background(), repo, "asset_ids", []string{" a1 ", "a2"})
	if err != nil {
		t.Fatalf("Validate err=%v", err)
	}
	if !reflect.DeepEqual(got, []string{"a1", "a2"}) {
		t.Fatalf("normalized ids=%v", got)
	}
	if repo.calls != 1 || !reflect.DeepEqual(repo.lastIDs, []string{"a1", "a2"}) {
		t.Fatalf("expected one batch lookup with normalized ids, calls=%d ids=%v", repo.calls, repo.lastIDs)
	}
}

func TestValidate_ReportsMissingInvalidAndDuplicateIDs(t *testing.T) {
	repo := &stubAssetRepo{existing: map[string]struct{}{"a1": {}}}
	_, err := Validate(context.Background(), repo, "input_asset_ids", []string{"a1", " missing ", " ", "a1", "missing"})
	if err == nil {
		t.Fatal("expected validation error")
	}
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T %[1]v", err)
	}
	if !reflect.DeepEqual(validationErr.MissingIDs, []string{"missing"}) {
		t.Fatalf("missing=%v", validationErr.MissingIDs)
	}
	if !reflect.DeepEqual(validationErr.InvalidIDs, []string{""}) {
		t.Fatalf("invalid=%v", validationErr.InvalidIDs)
	}
	if !reflect.DeepEqual(validationErr.DuplicateIDs, []string{"a1", "missing"}) {
		t.Fatalf("duplicates=%v", validationErr.DuplicateIDs)
	}
	wantDetails := map[string]any{
		"field":               "input_asset_ids",
		"missing_asset_ids":   []string{"missing"},
		"invalid_asset_ids":   []string{""},
		"duplicate_asset_ids": []string{"a1", "missing"},
	}
	if !reflect.DeepEqual(validationErr.Details(), wantDetails) {
		t.Fatalf("details=%#v", validationErr.Details())
	}
}

func TestValidate_PropagatesLookupErrors(t *testing.T) {
	repo := &stubAssetRepo{err: errors.New("db down")}
	if _, err := Validate(context.Background(), repo, "asset_ids", []string{"a1"}); err == nil {
		t.Fatal("expected lookup error")
	}
}

func TestValidate_AllowsUnknownAssetsWhenRepoNil(t *testing.T) {
	got, err := Validate(context.Background(), nil, "asset_ids", []string{" custom-asset-1 ", "custom-asset-2"})
	if err != nil {
		t.Fatalf("Validate err=%v", err)
	}
	if !reflect.DeepEqual(got, []string{"custom-asset-1", "custom-asset-2"}) {
		t.Fatalf("normalized ids=%v", got)
	}
}

func TestNormalizeAssetIDs_DedupesAndTrims(t *testing.T) {
	got, err := NormalizeAssetIDs("asset_ids", []string{" a ", "b", "a", "  c  "})
	if err != nil {
		t.Fatalf("NormalizeAssetIDs err=%v", err)
	}
	if !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Fatalf("normalized ids=%v", got)
	}
}

func (s *stubAssetRepo) LookupCosts(context.Context, []string, time.Time, time.Time, bool) ([]repository.AssetCostRow, error) {
	return nil, nil
}

func (s *stubAssetRepo) LookupDurations(context.Context, []string, int64, int64) ([]repository.DurationRow, error) {
	return nil, nil
}

func (s *stubAssetRepo) LookupLineage(context.Context, []string) ([]repository.LineageRow, error) {
	return nil, nil
}
