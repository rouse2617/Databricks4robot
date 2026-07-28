package main

import (
	"context"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
	espkg "github.com/CyberOrigin2077/cyber-databrew/internal/elasticsearch"
	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	assetUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/asset"
)

type testAssetRepo struct {
	asset *models.Asset
}

func (r *testAssetRepo) Get(ctx context.Context, id string) (*models.Asset, error) {
	return r.GetAll(ctx, id)
}
func (r *testAssetRepo) GetAll(_ context.Context, _ string) (*models.Asset, error) {
	return r.asset, nil
}
func (r *testAssetRepo) FindExistingIDs(_ context.Context, assetIDs []string) (map[string]struct{}, error) {
	out := make(map[string]struct{})
	if r.asset == nil {
		return out, nil
	}
	for _, assetID := range assetIDs {
		if assetID == r.asset.AssetID {
			out[assetID] = struct{}{}
		}
	}
	return out, nil
}

func (r *testAssetRepo) InsertNew(_ context.Context, a *models.Asset) error {
	return r.Set(context.Background(), a)
}
func (r *testAssetRepo) Set(_ context.Context, a *models.Asset) error {
	r.asset = a
	return nil
}

func (r *testAssetRepo) SoftDelete(_ context.Context, _ string) error {
	return nil
}

func (r *testAssetRepo) ListByMcapFile(_ context.Context, _ string) ([]*models.Asset, error) {
	return nil, nil
}

func (r *testAssetRepo) ListByLogicalAssetID(_ context.Context, _ string) ([]*models.Asset, error) {
	return nil, nil
}

func (r *testAssetRepo) WriteSegmentIndex(_ context.Context, _ *models.Asset) error {
	return nil
}

func (r *testAssetRepo) ListDescendants(_ context.Context, _ string) ([]*models.Asset, error) {
	return nil, nil
}

func (r *testAssetRepo) ListWithFilters(_ context.Context, _ string, _ []interface{}, page, pageSize int, _ filter.OrderByClause) ([]*models.Asset, int64, error) {
	return nil, 0, nil
}

func (r *testAssetRepo) MergeCfAlgo(_ context.Context, _ string, _ int64, _ map[string]interface{}, _ map[string]interface{}) (int64, error) {
	return 0, nil
}

func (r *testAssetRepo) LookupDurations(context.Context, []string, int64, int64) ([]repository.DurationRow, error) {
	return nil, nil
}

func (r *testAssetRepo) LookupLineage(context.Context, []string) ([]repository.LineageRow, error) {
	return nil, nil
}

func (r *testAssetRepo) LookupCosts(context.Context, []string, time.Time, time.Time, bool) ([]repository.AssetCostRow, error) {
	return nil, nil
}

func buildTestAlgoRegistry(t *testing.T) *config.AlgoRegistry {
	t.Helper()
	reg, err := config.LoadAlgoRegistry("../../config/algo_registry.yaml")
	if err != nil {
		t.Fatalf("failed to load algo registry: %v", err)
	}
	return reg
}

func TestNewAssetUsecaseInitializesAlgoStateForAllBackends(t *testing.T) {
	repo := &testAssetRepo{}
	uc := newAssetUsecase(repo, nil, buildTestAlgoRegistry(t))

	asset, err := uc.Create(context.Background(), assetUC.CreateInput{
		McapFileID:       "mcap-001",
		StartTimestampNs: 1000,
		EndTimestampNs:   1001000,
		Reviewer:         "tester",
		Owner:            "owner",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if got := asset.AlgoResults["env_analysis@1.0.0:status"]; got != string(models.AlgoStatusPending) {
		t.Fatalf("env_analysis should initialize to pending, got %q", got)
	}
	if got := asset.AlgoResults["action_annotation@1.0.0:status"]; got != string(models.AlgoStatusBlocked) {
		t.Fatalf("action_annotation should initialize to blocked, got %q", got)
	}
	if got := asset.Files["raw_mcap"]; got != "mcap-001" {
		t.Fatalf("raw_mcap should be initialized, got %q", got)
	}
}

func TestBuildSearchSyncInfoReportsOutboxSubscriberMode(t *testing.T) {
	cfg := &config.Config{Env: "production"}
	info := buildSearchSyncInfo(cfg, espkg.New("http://example.invalid", "", "", ""), true, true)

	if info.SearchIndexMode != "outbox_es_subscriber" {
		t.Fatalf("expected outbox_es_subscriber mode, got %q", info.SearchIndexMode)
	}
	if !info.ElasticsearchOK || !info.OutboxESSubscriberEnabled {
		t.Fatalf("expected ES and outbox subscriber to be marked healthy: %+v", info)
	}
}

func TestBuildSearchSyncInfoReportsFallbackModes(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.Config
		es   *espkg.Client
		want string
	}{
		{
			name: "no elasticsearch",
			cfg:  &config.Config{Env: "production"},
			want: "unavailable",
		},
		{
			name: "dev local reconcile",
			cfg:  &config.Config{Env: "development"},
			es:   espkg.New("http://example.invalid", "", "", ""),
			want: "local_reconcile",
		},
		{
			name: "production manual",
			cfg:  &config.Config{Env: "production"},
			es:   espkg.New("http://example.invalid", "", "", ""),
			want: "manual",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := buildSearchSyncInfo(tt.cfg, tt.es, false, false)
			if info.SearchIndexMode != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, info.SearchIndexMode)
			}
		})
	}
}
