package dashboard

import (
	"context"
	"errors"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

type fakeRepo struct {
	gotAssetType string
	resp         *models.DurationDistribution
	err          error
}

func (f *fakeRepo) DurationDistribution(_ context.Context, assetType string) (*models.DurationDistribution, error) {
	f.gotAssetType = assetType
	if f.err != nil {
		return nil, f.err
	}
	return f.resp, nil
}

func TestDurationDistributionTrimsAndEchoes(t *testing.T) {
	repo := &fakeRepo{resp: &models.DurationDistribution{}}
	uc := New(repo)

	dist, err := uc.DurationDistribution(context.Background(), "  raw_mcap  ")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if repo.gotAssetType != "raw_mcap" {
		t.Fatalf("repo got %q, want trimmed 'raw_mcap'", repo.gotAssetType)
	}
	if dist.AssetType == nil || *dist.AssetType != "raw_mcap" {
		t.Fatalf("response AssetType = %v, want *raw_mcap", dist.AssetType)
	}
}

func TestDurationDistributionEmptyAssetTypeStaysNil(t *testing.T) {
	repo := &fakeRepo{resp: &models.DurationDistribution{}}
	uc := New(repo)

	dist, err := uc.DurationDistribution(context.Background(), "   ")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if repo.gotAssetType != "" {
		t.Fatalf("repo got %q, want empty", repo.gotAssetType)
	}
	if dist.AssetType != nil {
		t.Fatalf("response AssetType = %v, want nil", dist.AssetType)
	}
}

func TestDurationDistributionWrapsRepoError(t *testing.T) {
	repo := &fakeRepo{err: errors.New("boom")}
	uc := New(repo)

	_, err := uc.DurationDistribution(context.Background(), "raw_mcap")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
