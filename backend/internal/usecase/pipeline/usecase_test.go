package pipeline

import (
	"context"
	"errors"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// ── Mocks ─────────────────────────────────────────────────────────────────

type mockTemplateRepo struct {
	saved  []*models.PipelineTemplate
	byID   map[string]*models.PipelineTemplate
	byName map[string][]models.PipelineTemplate
	ver    int
}

func (m *mockTemplateRepo) Save(_ context.Context, t *models.PipelineTemplate) error {
	m.saved = append(m.saved, t)
	return nil
}
func (m *mockTemplateRepo) FindAll(_ context.Context) ([]models.PipelineTemplate, error) { return nil, nil }
func (m *mockTemplateRepo) FindByID(_ context.Context, id string) (*models.PipelineTemplate, error) {
	return m.byID[id], nil
}
func (m *mockTemplateRepo) FindVersionsByName(_ context.Context, _ string) ([]models.PipelineTemplate, error) {
	return nil, nil
}
func (m *mockTemplateRepo) GetNextVersion(_ context.Context, _ string) (int, error) { m.ver++; return m.ver, nil }
func (m *mockTemplateRepo) Delete(_ context.Context, _ string) error { return nil }

type mockDeploymentRepo struct {
	saved []*models.PipelineDeployment
}

func (m *mockDeploymentRepo) Save(_ context.Context, d *models.PipelineDeployment) error {
	m.saved = append(m.saved, d)
	return nil
}
func (m *mockDeploymentRepo) FindAll(_ context.Context) ([]models.PipelineDeployment, error) { return nil, nil }
func (m *mockDeploymentRepo) FindByID(_ context.Context, _ string) (*models.PipelineDeployment, error) {
	return nil, nil
}
func (m *mockDeploymentRepo) Delete(_ context.Context, _ string) error { return nil }
func (m *mockDeploymentRepo) UpdateStatus(_ context.Context, _, _ string) error { return nil }

type mockAssetRepo struct {
	assets map[string]*models.Asset
}

func (m *mockAssetRepo) Get(_ context.Context, assetID string) (*models.Asset, error) {
	a, ok := m.assets[assetID]
	if !ok {
		return nil, nil
	}
	return a, nil
}
func (m *mockAssetRepo) GetAll(_ context.Context, _ string) (*models.Asset, error)       { return nil, nil }
func (m *mockAssetRepo) InsertNew(_ context.Context, _ *models.Asset) error               { return nil }
func (m *mockAssetRepo) Set(_ context.Context, _ *models.Asset) error                      { return nil }
func (m *mockAssetRepo) SoftDelete(_ context.Context, _ string) error                      { return nil }
func (m *mockAssetRepo) ListByMcapFile(_ context.Context, _ string) ([]*models.Asset, error) { return nil, nil }
func (m *mockAssetRepo) ListByLogicalAssetID(_ context.Context, _ string) ([]*models.Asset, error) {
	return nil, nil
}
func (m *mockAssetRepo) WriteSegmentIndex(_ context.Context, _ *models.Asset) error       { return nil }
func (m *mockAssetRepo) ListWithFilters(_ context.Context, _ string, _ []interface{}, _ int, _ int, _ filter.OrderByClause) ([]*models.Asset, int64, error) {
	return nil, 0, nil
}
func (m *mockAssetRepo) ListDescendants(_ context.Context, _ string) ([]*models.Asset, error) {
	return nil, nil
}

func newMockAssetRepo() *mockAssetRepo {
	return &mockAssetRepo{assets: make(map[string]*models.Asset)}
}

func newUsecase(assetRepo *mockAssetRepo) *Usecase {
	return &Usecase{
		templateRepo:   &mockTemplateRepo{byID: make(map[string]*models.PipelineTemplate)},
		deploymentRepo: &mockDeploymentRepo{},
		assetRepo:      assetRepo,
	}
}

// ── Tests ─────────────────────────────────────────────────────────────────

func TestDeploy_ValidatesAssetExistence(t *testing.T) {
	ctx := context.Background()
	pipe := map[string]interface{}{
		"name": "test-pipe",
		"nodes": []interface{}{
			map[string]interface{}{
				"id": "step-1",
				"component": map[string]interface{}{
					"name": "test", "image": "busybox",
				},
			},
		},
		"edges": []interface{}{},
	}

	t.Run("missing asset returns ErrAssetNotFound", func(t *testing.T) {
		repo := newMockAssetRepo()
		uc := newUsecase(repo)
		// No assets in repo — asset "nonexistent" will not be found.
		_, err := uc.Deploy(ctx, pipe, "", []string{"nonexistent"})
		if err == nil {
			t.Fatal("expected error for missing asset, got nil")
		}
		if !errors.Is(err, ErrAssetNotFound) {
			t.Fatalf("expected ErrAssetNotFound, got: %v", err)
		}
	})

	t.Run("valid asset passes validation", func(t *testing.T) {
		repo := newMockAssetRepo()
		repo.assets["asset-1"] = &models.Asset{
			AssetID:    "asset-1",
			AssetType:  "dataset",
			StorageURI: "gs://bucket/data",
		}
		repo.assets["asset-2"] = &models.Asset{
			AssetID:    "asset-2",
			AssetType:  "model",
			StorageURI: "gs://bucket/model",
		}
		uc := newUsecase(repo)

		dep, err := uc.Deploy(ctx, pipe, "", []string{"asset-1", "asset-2"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dep == nil {
			t.Fatal("expected deployment, got nil")
		}
		if dep.PipelineName != "test-pipe" {
			t.Fatalf("expected pipeline name 'test-pipe', got %q", dep.PipelineName)
		}
	})

	t.Run("no asset IDs skips validation", func(t *testing.T) {
		repo := newMockAssetRepo()
		uc := newUsecase(repo)

		dep, err := uc.Deploy(ctx, pipe, "", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dep == nil {
			t.Fatal("expected deployment, got nil")
		}
	})

	t.Run("first invalid asset among many triggers error", func(t *testing.T) {
		repo := newMockAssetRepo()
		repo.assets["valid-1"] = &models.Asset{AssetID: "valid-1"}
		uc := newUsecase(repo)

		_, err := uc.Deploy(ctx, pipe, "", []string{"valid-1", "missing-2"})
		if err == nil {
			t.Fatal("expected error for missing asset, got nil")
		}
		if !errors.Is(err, ErrAssetNotFound) {
			t.Fatalf("expected ErrAssetNotFound, got: %v", err)
		}
	})
}
