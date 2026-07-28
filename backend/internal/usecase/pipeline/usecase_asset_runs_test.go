package pipeline

import (
	"context"
	"errors"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// CYB-4297: reverse "asset → pipeline runs" lookup.
//
// Coverage:
//  - looksLikeGraceVideoID shape heuristic
//  - empty id → ErrInvalidArgument
//  - grace_video_id passthrough (no assetRepo hop)
//  - short asset_id resolves through assetRepo → grace_video_id
//  - short asset_id missing in assets keeps raw input (repo returns 0 rows)

func TestLooksLikeGraceVideoID(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"grace uuid v7", "019f8e88-f67e-7680-a565-84e2f4262c12", true},
		{"grace uuid v4", "12345678-1234-1234-1234-123456789012", true},
		{"short asset_id", "uYN6qys6", false},
		{"empty", "", false},
		{"no hyphens 32-hex", "019f8e88f67e7680a56584e2f4262c12", false},
		{"36 chars but wrong hyphen positions", "01234567X8901X2345X6789X012345678901", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := looksLikeGraceVideoID(tc.in); got != tc.want {
				t.Fatalf("looksLikeGraceVideoID(%q)=%v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestListRunsByAsset_EmptyID(t *testing.T) {
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, newMockAssetRepo(), &mockWorkflowClient{}, "default")
	_, _, err := uc.ListRunsByAsset(context.Background(), "   ", models.PipelineRunListFilter{})
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("expected ErrInvalidArgument, got %v", err)
	}
}

func TestListRunsByAsset_GraceIDPassthrough(t *testing.T) {
	grace := "019f8e88-f67e-7680-a565-84e2f4262c12"
	runRepo := &mockRunRepo{byID: map[string]*models.PipelineRun{}}
	assetRepo := newMockAssetRepo()
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, assetRepo, &mockWorkflowClient{}, "default")
	uc.SetRunRepositories(nil, runRepo, nil)

	if _, _, err := uc.ListRunsByAsset(context.Background(), grace, models.PipelineRunListFilter{PageSize: 5}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runRepo.listFilters) != 1 {
		t.Fatalf("want 1 ListSummaries call, got %d", len(runRepo.listFilters))
	}
	if runRepo.listFilters[0].AssetID != grace {
		t.Fatalf("filter.AssetID = %q, want %q", runRepo.listFilters[0].AssetID, grace)
	}
	// The heuristic short-circuits — assetRepo.Get MUST NOT be called on a
	// grace-shaped id (extra DB hop otherwise). newMockAssetRepo's Get is
	// benign, so instead we verify the filter kept the input verbatim.
}

func TestListRunsByAsset_ShortAssetIDResolvesToGrace(t *testing.T) {
	shortID := "uYN6qys6"
	grace := "019f8e88-f67e-7680-a565-84e2f4262c12"
	runRepo := &mockRunRepo{byID: map[string]*models.PipelineRun{}}
	assetRepo := newMockAssetRepo()
	assetRepo.assets[shortID] = &models.Asset{AssetID: shortID, GraceVideoID: grace}

	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, assetRepo, &mockWorkflowClient{}, "default")
	uc.SetRunRepositories(nil, runRepo, nil)

	if _, _, err := uc.ListRunsByAsset(context.Background(), shortID, models.PipelineRunListFilter{PageSize: 5}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runRepo.listFilters) != 1 {
		t.Fatalf("want 1 ListSummaries call, got %d", len(runRepo.listFilters))
	}
	if got := runRepo.listFilters[0].AssetID; got != grace {
		t.Fatalf("filter.AssetID = %q, want resolved grace %q", got, grace)
	}
}

func TestListRunsByAsset_ShortAssetIDMissingKeepsInput(t *testing.T) {
	// Short id not present in assets table: usecase keeps the raw id and lets
	// the repo return zero rows (legitimate empty result, not an error).
	shortID := "unknown7"
	runRepo := &mockRunRepo{byID: map[string]*models.PipelineRun{}}
	assetRepo := newMockAssetRepo()

	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, assetRepo, &mockWorkflowClient{}, "default")
	uc.SetRunRepositories(nil, runRepo, nil)

	if _, _, err := uc.ListRunsByAsset(context.Background(), shortID, models.PipelineRunListFilter{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runRepo.listFilters[0].AssetID; got != shortID {
		t.Fatalf("filter.AssetID = %q, want raw input %q", got, shortID)
	}
}
