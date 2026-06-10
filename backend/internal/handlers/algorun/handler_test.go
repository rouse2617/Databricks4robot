package algorun

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	algorunUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/algorun"
)

type mockAlgoRunRepo struct {
	listFn func(ctx context.Context, filter repository.AlgoRunListFilter) ([]*models.AlgoRun, int64, error)
}

func (m *mockAlgoRunRepo) Insert(context.Context, *models.AlgoRun) error { return nil }
func (m *mockAlgoRunRepo) Get(context.Context, string) (*models.AlgoRun, error) {
	return nil, nil
}
func (m *mockAlgoRunRepo) Exists(context.Context, string) (bool, error)   { return false, nil }
func (m *mockAlgoRunRepo) Start(context.Context, string, time.Time) error { return nil }
func (m *mockAlgoRunRepo) Finish(context.Context, string, repository.AlgoRunFinishPatch) error {
	return nil
}
func (m *mockAlgoRunRepo) List(ctx context.Context, filter repository.AlgoRunListFilter) ([]*models.AlgoRun, int64, error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return []*models.AlgoRun{}, 0, nil
}
func (m *mockAlgoRunRepo) Cancel(context.Context, string, string, time.Time) error {
	return nil
}
func (m *mockAlgoRunRepo) GetAffectedAssets(context.Context, string) ([]*repository.AffectedAsset, error) {
	return nil, nil
}

func TestList_ReturnsNormalizedPaginationMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &mockAlgoRunRepo{}
	h := New(algorunUC.New(repo))
	r := gin.New()
	r.GET("/algo-runs", h.List)

	req := httptest.NewRequest(http.MethodGet, "/algo-runs?page=-1&page_size=999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Page     int `json:"page"`
		PageSize int `json:"page_size"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Page != 1 || body.PageSize != 50 {
		t.Fatalf("expected normalized page/page_size 1/50, got %d/%d", body.Page, body.PageSize)
	}
}
