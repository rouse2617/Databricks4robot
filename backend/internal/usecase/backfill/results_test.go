package backfill

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

type stubResultRepo struct {
	manifest *repository.ReportManifest
	upserted []repository.AlgoRunResultWriteInput
}

func (s *stubResultRepo) GetReportManifest(_ context.Context, reportID string) (*repository.ReportManifest, error) {
	if s.manifest == nil || s.manifest.ReportID != reportID {
		return nil, repository.ErrReportManifestNotFound
	}
	return s.manifest, nil
}

func (s *stubResultRepo) UpsertAlgoRunResult(_ context.Context, in repository.AlgoRunResultWriteInput) error {
	s.upserted = append(s.upserted, in)
	return nil
}

func (s *stubResultRepo) HasAlgoRunResult(_ context.Context, _, _, _ string) (bool, error) {
	return false, nil
}

type stubAssetRepo struct {
	ids map[string]struct{}
}

func (s *stubAssetRepo) Get(context.Context, string) (*models.Asset, error) { return nil, nil }
func (s *stubAssetRepo) FindExistingIDs(_ context.Context, assetIDs []string) (map[string]struct{}, error) {
	out := make(map[string]struct{})
	for _, id := range assetIDs {
		if _, ok := s.ids[id]; ok {
			out[id] = struct{}{}
		}
	}
	return out, nil
}
func (s *stubAssetRepo) GetAll(context.Context, string) (*models.Asset, error) { return nil, nil }
func (s *stubAssetRepo) InsertNew(context.Context, *models.Asset) error        { return nil }
func (s *stubAssetRepo) Set(context.Context, *models.Asset) error                { return nil }
func (s *stubAssetRepo) SoftDelete(context.Context, string) error                  { return nil }
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

type stubBackfillRepo struct {
	items []models.BackfillItem
}

func (s *stubBackfillRepo) SaveJob(context.Context, *models.BackfillJob) error { return nil }
func (s *stubBackfillRepo) FindAllJobs(context.Context) ([]models.BackfillJob, error) {
	return nil, nil
}
func (s *stubBackfillRepo) FindJobByID(context.Context, string) (*models.BackfillJob, error) {
	return nil, nil
}
func (s *stubBackfillRepo) UpdateJobStatus(context.Context, string, string) error { return nil }
func (s *stubBackfillRepo) UpdateJobPilotPhase(context.Context, string, string, string) error {
	return nil
}
func (s *stubBackfillRepo) IncrementCompleted(context.Context, string) error { return nil }
func (s *stubBackfillRepo) IncrementFailed(context.Context, string) error    { return nil }
func (s *stubBackfillRepo) SaveItem(context.Context, *models.BackfillItem) error {
	return nil
}
func (s *stubBackfillRepo) SaveItems(context.Context, []models.BackfillItem) error { return nil }
func (s *stubBackfillRepo) FindItemsByJobID(context.Context, string) ([]models.BackfillItem, error) {
	return nil, nil
}
func (s *stubBackfillRepo) FindItemByID(context.Context, string) (*models.BackfillItem, error) {
	return nil, nil
}
func (s *stubBackfillRepo) FindItemByPipelineRunID(context.Context, string) (*models.BackfillItem, error) {
	return nil, nil
}
func (s *stubBackfillRepo) FindItemByJobAndAssetID(_ context.Context, jobID, assetID string) (*models.BackfillItem, error) {
	for i := range s.items {
		if s.items[i].JobID == jobID && s.items[i].AssetID == assetID {
			item := s.items[i]
			return &item, nil
		}
	}
	return nil, nil
}

func (s *stubBackfillRepo) ClaimNextItem(ctx context.Context, jobID string) (*models.BackfillItem, error) {
	return nil, nil
}

func (s *stubBackfillRepo) ResetStaleItems(ctx context.Context, leaseTimeoutSec int, maxAttempts int) (int, error) {
	return 0, nil
}

func (s *stubBackfillRepo) FindIncompleteJobs(ctx context.Context) ([]models.BackfillJob, error) {
	return nil, nil
}

func (s *stubBackfillRepo) UpdateItemStatus(_ context.Context, id, status, _, _ string) error {
	for i := range s.items {
		if s.items[i].ID == id {
			s.items[i].Status = status
		}
	}
	return nil
}
func (s *stubBackfillRepo) UpdateItemPipelineRun(context.Context, string, string, string, string) error {
	return nil
}
func (s *stubBackfillRepo) UpdateJobProgress(context.Context, string, int, int, string) error {
	return nil
}
func (s *stubBackfillRepo) CountItemsByStatus(context.Context, string, string) (int, error) {
	return 0, nil
}
func (s *stubBackfillRepo) SummarizeItemStatuses(context.Context, string) (repository.BackfillItemStatusSummary, error) {
	return repository.BackfillItemStatusSummary{}, nil
}
func (s *stubBackfillRepo) FindItemsByJobIDWithStatuses(context.Context, string, []string) ([]models.BackfillItem, error) {
	return nil, nil
}
func (s *stubBackfillRepo) FindItemsMissingPipelineRun(context.Context, string) ([]models.BackfillItem, error) {
	return nil, nil
}
func (s *stubBackfillRepo) FindItemsByScope(context.Context, repository.BackfillRerunItemFilter) ([]models.BackfillItem, error) {
	return nil, nil
}
func (s *stubBackfillRepo) PrepareItemsForRerun(context.Context, []string) error { return nil }
func (s *stubBackfillRepo) AggregateNodeStatusByBatchJobID(context.Context, string) ([]repository.BatchNodeStatusAggregate, error) {
	return nil, nil
}
func (s *stubBackfillRepo) ListNodeFailures(context.Context, repository.BatchNodeFailureFilter) (*models.BatchNodeFailureListResult, error) {
	return nil, nil
}
func (s *stubBackfillRepo) CountPipelineRunsByBatchJobID(context.Context, string) (int, error) {
	return 0, nil
}
func (s *stubBackfillRepo) CountRunsWithNodeRowsByBatchJobID(context.Context, string) (int, error) {
	return 0, nil
}
func (s *stubBackfillRepo) FindItemsByAssetID(_ context.Context, assetID string) ([]models.BackfillItem, error) {
	var out []models.BackfillItem
	for _, item := range s.items {
		if item.AssetID == assetID {
			out = append(out, item)
		}
	}
	return out, nil
}

func TestUploadResult_Success(t *testing.T) {
	resultRepo := &stubResultRepo{
		manifest: &repository.ReportManifest{
			ReportID: "report.project@1.0.0-backfill-left-eye-only",
			Version:  "1.0.0",
		},
	}
	backfillRepo := &stubBackfillRepo{
		items: []models.BackfillItem{
			{ID: "item-1", AssetID: "asset-1", Status: "awaiting_result"},
		},
	}
	uc := New(backfillRepo, nil)
	uc.SetResultRepositories(resultRepo, &stubAssetRepo{ids: map[string]struct{}{"asset-1": {}}})

	out, err := uc.UploadResult(context.Background(), UploadResultInput{
		AssetID:  "asset-1",
		ReportID: "report.project@1.0.0-backfill-left-eye-only",
		Version:  "1.0.0",
		Result:   map[string]any{"ok": true},
	})
	if err != nil {
		t.Fatalf("UploadResult: %v", err)
	}
	if out.Status != "registered" {
		t.Fatalf("status = %q", out.Status)
	}
	if len(resultRepo.upserted) != 1 {
		t.Fatalf("upserted = %d", len(resultRepo.upserted))
	}
	if backfillRepo.items[0].Status != "completed" {
		t.Fatalf("item status = %q", backfillRepo.items[0].Status)
	}
}

func TestUploadResult_ManifestMismatch(t *testing.T) {
	uc := New(&stubBackfillRepo{}, nil)
	uc.SetResultRepositories(
		&stubResultRepo{manifest: &repository.ReportManifest{ReportID: "report.project@1.0.0-backfill-left-eye-only", Version: "1.0.0"}},
		&stubAssetRepo{ids: map[string]struct{}{"asset-1": {}}},
	)
	_, err := uc.UploadResult(context.Background(), UploadResultInput{
		AssetID:  "asset-1",
		ReportID: "report.project@1.0.0-backfill-left-eye-only",
		Version:  "1.0.0",
		Manifest: map[string]any{"assetId": "other-asset"},
		Result:   map[string]any{},
	})
	if !errors.Is(err, ErrManifestMismatch) {
		t.Fatalf("expected ErrManifestMismatch, got %v", err)
	}
}

func TestUploadResult_PayloadTooLarge(t *testing.T) {
	uc := New(&stubBackfillRepo{}, nil)
	uc.SetResultRepositories(
		&stubResultRepo{manifest: &repository.ReportManifest{ReportID: "report.project@1.0.0-backfill-left-eye-only", Version: "1.0.0"}},
		&stubAssetRepo{ids: map[string]struct{}{"asset-1": {}}},
	)
	large := map[string]any{"blob": string(make([]byte, MaxBackfillResultPayloadBytes+1))}
	_, _ = json.Marshal(large)
	_, err := uc.UploadResult(context.Background(), UploadResultInput{
		AssetID:  "asset-1",
		ReportID: "report.project@1.0.0-backfill-left-eye-only",
		Version:  "1.0.0",
		Result:   large,
	})
	if !errors.Is(err, ErrPayloadTooLarge) {
		t.Fatalf("expected ErrPayloadTooLarge, got %v", err)
	}
}

func TestResolveCompletionStatus_AwaitingResult(t *testing.T) {
	resultRepo := &stubResultRepo{}
	uc := New(&stubBackfillRepo{}, nil)
	uc.SetResultRepositories(resultRepo, nil)
	job := &models.BackfillJob{
		FilterJSON: map[string]any{
			"expectedReportId":      "report.project@1.0.0-backfill-left-eye-only",
			"expectedReportVersion": "1.0.0",
		},
	}
	status := uc.resolveCompletionStatus(context.Background(), job, models.BackfillItem{AssetID: "asset-1"})
	if status != "awaiting_result" {
		t.Fatalf("status = %q", status)
	}
}
