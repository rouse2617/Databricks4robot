package admin

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
)

type statsEventRepo struct {
	counts map[string]int64
	err    error
}

func (s *statsEventRepo) Append(context.Context, repository.AssetEventAppendInput) error { return nil }
func (s *statsEventRepo) ListPending(context.Context, int) ([]*models.AssetEvent, error) {
	return nil, nil
}
func (s *statsEventRepo) ListPendingSafe(context.Context, time.Duration, int) ([]*models.AssetEvent, error) {
	return nil, nil
}
func (s *statsEventRepo) ListByAsset(context.Context, string, repository.AssetEventListOptions) ([]*models.AssetEvent, error) {
	return nil, nil
}

func (m *statsEventRepo) ListGlobal(_ context.Context, opts repository.AssetEventListOptions) ([]*models.AssetEvent, error) {
	return nil, nil
}

func (s *statsEventRepo) MarkPublished(context.Context, []int64) error    { return nil }
func (s *statsEventRepo) MarkFailed(context.Context, int64, string) error { return nil }
func (s *statsEventRepo) CountPending(context.Context) (int64, error)     { return 0, nil }
func (s *statsEventRepo) CountPendingClaimable(context.Context, time.Duration) (int64, error) {
	return 0, nil
}
func (s *statsEventRepo) CountProcessing(context.Context) (int64, error)    { return 0, nil }
func (s *statsEventRepo) OldestPendingAge(context.Context) (float64, error) { return 0, nil }
func (s *statsEventRepo) ListBetweenSeq(context.Context, int64, int64, int) ([]*models.AssetEvent, error) {
	return nil, nil
}
func (s *statsEventRepo) PublishStateCounts(context.Context) (map[string]int64, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.counts, nil
}

type stubDLQ struct {
	n int64
}

func (s *stubDLQ) MoveToDLQ(context.Context, int) (int64, error) { return 0, nil }
func (s *stubDLQ) Count(context.Context) (int64, error)          { return s.n, nil }

func TestSearchOutboxStats_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := New(nil, nil, nil, nil, nil, nil, &statsEventRepo{
		counts: map[string]int64{"pending": 2, "published": 100, "dlq": 1},
	}, &stubDLQ{n: 5}, nil)

	r := gin.New()
	r.GET("/outbox-stats", h.SearchOutboxStats)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/outbox-stats", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var got SearchOutboxStatsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.OutboxDLQRows != 5 {
		t.Fatalf("dlq rows: %+v", got)
	}
	if got.PublishStateCounts["pending"] != 2 {
		t.Fatalf("counts: %+v", got.PublishStateCounts)
	}
}

func TestSearchOutboxStats_NoEventsRepo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := New(nil, nil, nil, nil, nil, nil, nil, nil, nil)
	r := gin.New()
	r.GET("/x", h.SearchOutboxStats)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}
