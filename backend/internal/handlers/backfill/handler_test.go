package backfill

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
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
func (missingJobRepo) UpdateJobStatus(_ context.Context, _, _ string) error { return nil }
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
func (missingJobRepo) UpdateItemStatus(_ context.Context, _, _, _, _ string) error { return nil }
func (missingJobRepo) CountItemsByStatus(_ context.Context, _, _ string) (int, error) {
	return 0, nil
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
