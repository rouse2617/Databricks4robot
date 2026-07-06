package backfill

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	backfillUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/backfill"
)

type missingJobRepo struct{}

func (missingJobRepo) SaveJob(_ context.Context, _ *models.BackfillJob) error { return nil }
func (missingJobRepo) FindAllJobs(_ context.Context) ([]models.BackfillJob, error) {
	return nil, nil
}
func (missingJobRepo) FindJobByID(_ context.Context, _ string) (*models.BackfillJob, error) {
	return nil, nil
}
func (missingJobRepo) UpdateJobStatus(_ context.Context, _, _ string) error        { return nil }
func (missingJobRepo) UpdateJobPilotPhase(_ context.Context, _, _, _ string) error { return nil }
func (missingJobRepo) ClaimJobNotification(_ context.Context, _ string) (bool, error) {
	return false, nil
}
func (missingJobRepo) IncrementCompleted(_ context.Context, _ string) error { return nil }
func (missingJobRepo) IncrementFailed(_ context.Context, _ string) error    { return nil }
func (missingJobRepo) SaveItem(_ context.Context, _ *models.BackfillItem) error {
	return nil
}
func (missingJobRepo) SaveItems(_ context.Context, _ []models.BackfillItem) error { return nil }
func (missingJobRepo) FindItemsByJobID(_ context.Context, _ string) ([]models.BackfillItem, error) {
	return nil, nil
}
func (missingJobRepo) FindItemByID(_ context.Context, _ string) (*models.BackfillItem, error) {
	return nil, nil
}
func (missingJobRepo) FindItemByPipelineRunID(_ context.Context, _ string) (*models.BackfillItem, error) {
	return nil, nil
}
func (missingJobRepo) FindItemByJobAndAssetID(_ context.Context, _, _ string) (*models.BackfillItem, error) {
	return nil, nil
}
func (missingJobRepo) UpdateItemStatus(_ context.Context, _, _, _, _ string) error      { return nil }
func (missingJobRepo) UpdateItemPipelineRun(_ context.Context, _, _, _, _ string) error { return nil }
func (missingJobRepo) UpdateJobProgress(_ context.Context, _ string, _, _ int, _ string) error {
	return nil
}
func (missingJobRepo) CountItemsByStatus(_ context.Context, _, _ string) (int, error) {
	return 0, nil
}
func (missingJobRepo) SummarizeItemStatuses(_ context.Context, _ string) (repository.BackfillItemStatusSummary, error) {
	return repository.BackfillItemStatusSummary{}, nil
}
func (missingJobRepo) FindItemsByJobIDWithStatuses(_ context.Context, _ string, _ []string) ([]models.BackfillItem, error) {
	return nil, nil
}
func (missingJobRepo) FindItemsMissingPipelineRun(_ context.Context, _ string) ([]models.BackfillItem, error) {
	return nil, nil
}
func (missingJobRepo) FindItemsByScope(_ context.Context, _ repository.BackfillRerunItemFilter) ([]models.BackfillItem, error) {
	return nil, nil
}
func (missingJobRepo) PrepareItemsForRerun(_ context.Context, _ []string) error { return nil }
func (missingJobRepo) AggregateNodeStatusByBatchJobID(_ context.Context, _ string) ([]repository.BatchNodeStatusAggregate, error) {
	return nil, nil
}
func (missingJobRepo) ListNodeFailures(_ context.Context, _ repository.BatchNodeFailureFilter) (*models.BatchNodeFailureListResult, error) {
	return &models.BatchNodeFailureListResult{}, nil
}
func (missingJobRepo) CountPipelineRunsByBatchJobID(_ context.Context, _ string) (int, error) {
	return 0, nil
}
func (missingJobRepo) CountRunsWithNodeRowsByBatchJobID(_ context.Context, _ string) (int, error) {
	return 0, nil
}
func (missingJobRepo) FindItemsByAssetID(_ context.Context, _ string) ([]models.BackfillItem, error) {
	return nil, nil
}
func (missingJobRepo) ClaimNextItem(_ context.Context, _ string) (*models.BackfillItem, error) {
	return nil, nil
}
func (missingJobRepo) ResetStaleItems(_ context.Context, _, _ int) (int, error) {
	return 0, nil
}
func (missingJobRepo) FindIncompleteJobs(_ context.Context) ([]models.BackfillJob, error) {
	return nil, nil
}

func TestRetryFailed_NotFoundHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := New(backfillUC.New(missingJobRepo{}, nil))
	r := gin.New()
	r.POST("/backfill/:id/retry-failed", h.RetryFailed)

	req := httptest.NewRequest(http.MethodPost, "/backfill/missing/retry-failed", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMapError_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	mapBackfillError(c, backfillUC.ErrNotFound)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestMapError_Other(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	mapBackfillError(c, errors.New("boom"))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestCreateJob_TooManyAssetsHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := New(backfillUC.New(missingJobRepo{}, nil))
	r := gin.New()
	r.POST("/backfill", h.CreateJob)

	assetIDs := make([]string, backfillUC.MaxBackfillAssetCount+1)
	for i := range assetIDs {
		assetIDs[i] = fmt.Sprintf("asset-%d", i)
	}
	var idsJSON strings.Builder
	idsJSON.WriteString("[")
	for i, id := range assetIDs {
		if i > 0 {
			idsJSON.WriteString(",")
		}
		idsJSON.WriteString(`"` + id + `"`)
	}
	idsJSON.WriteString("]")
	payload := `{"name":"too-big","templateId":"tpl-1","assetIds":` + idsJSON.String() + `}`

	req := httptest.NewRequest(http.MethodPost, "/backfill", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
