package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	espkg "github.com/CyberOrigin2077/cyber-databrew/internal/elasticsearch"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

type inMemoryReindexJobRepo struct {
	mu   sync.Mutex
	jobs map[string]*repository.SearchReindexJob
}

func (r *inMemoryReindexJobRepo) Create(_ context.Context, dryRun bool, pageSize int) (*repository.SearchReindexJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	job := &repository.SearchReindexJob{
		ID:        "rj_test_1",
		Status:    repository.SearchReindexJobStatusQueued,
		DryRun:    dryRun,
		PageSize:  pageSize,
		NextPage:  1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if r.jobs == nil {
		r.jobs = map[string]*repository.SearchReindexJob{}
	}
	r.jobs[job.ID] = job
	return cloneJob(job), nil
}

func (r *inMemoryReindexJobRepo) Get(_ context.Context, jobID string) (*repository.SearchReindexJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	job := r.jobs[jobID]
	if job == nil {
		return nil, nil
	}
	return cloneJob(job), nil
}

func (r *inMemoryReindexJobRepo) ListRecent(_ context.Context, _ int) ([]*repository.SearchReindexJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*repository.SearchReindexJob, 0, len(r.jobs))
	for _, job := range r.jobs {
		out = append(out, cloneJob(job))
	}
	return out, nil
}

func (r *inMemoryReindexJobRepo) ClaimForRun(_ context.Context, _ string) (*repository.SearchReindexJob, error) {
	return nil, nil
}

func (r *inMemoryReindexJobRepo) RequestStop(_ context.Context, jobID string) (*repository.SearchReindexJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	job := r.jobs[jobID]
	if job == nil {
		return nil, nil
	}
	job.StopRequested = true
	job.UpdatedAt = time.Now().UTC()
	return cloneJob(job), nil
}

func (r *inMemoryReindexJobRepo) Resume(_ context.Context, jobID string) (*repository.SearchReindexJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	job := r.jobs[jobID]
	if job == nil {
		return nil, nil
	}
	if job.Status != repository.SearchReindexJobStatusPaused && job.Status != repository.SearchReindexJobStatusFailed {
		return nil, nil
	}
	job.Status = repository.SearchReindexJobStatusQueued
	job.StopRequested = false
	job.UpdatedAt = time.Now().UTC()
	return cloneJob(job), nil
}

func (r *inMemoryReindexJobRepo) IsStopRequested(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func (r *inMemoryReindexJobRepo) UpdateProgress(_ context.Context, _ string, _ repository.SearchReindexJobProgress) error {
	return nil
}

func (r *inMemoryReindexJobRepo) MarkPaused(_ context.Context, jobID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	job := r.jobs[jobID]
	if job == nil {
		return nil
	}
	now := time.Now().UTC()
	job.Status = repository.SearchReindexJobStatusPaused
	job.FinishedAt = &now
	job.UpdatedAt = now
	return nil
}

func (r *inMemoryReindexJobRepo) MarkFailed(_ context.Context, _ string, _ string, _ []string) error {
	return nil
}

func (r *inMemoryReindexJobRepo) MarkSucceeded(_ context.Context, _ string, _ *int64) error {
	return nil
}

func (r *inMemoryReindexJobRepo) MarkAbandoned(_ context.Context, jobID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if j, ok := r.jobs[jobID]; ok {
		j.Status = repository.SearchReindexJobStatusAbandoned
		now := time.Now()
		j.FinishedAt = &now
		j.UpdatedAt = now
		return nil
	}
	return errors.New("not found")
}
func cloneJob(in *repository.SearchReindexJob) *repository.SearchReindexJob {
	if in == nil {
		return nil
	}
	out := *in
	if in.ErrorSamples != nil {
		out.ErrorSamples = append([]string(nil), in.ErrorSamples...)
	}
	return &out
}

func newDummyES(t *testing.T) *espkg.Client {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(s.Close)
	return espkg.New(s.URL, "assets", "", "")
}

func TestSearchReindexCreateJob(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &inMemoryReindexJobRepo{jobs: map[string]*repository.SearchReindexJob{}}
	h := New(nil, nil, nil, nil, nil, newDummyES(t), nil, nil, repo)

	r := gin.New()
	r.POST("/jobs", h.SearchReindexCreateJob)
	req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(`{"dry_run":true,"page_size":300}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", w.Code, w.Body.String())
	}
	var got SearchReindexJobView
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !got.DryRun || got.PageSize != 300 || got.Status != repository.SearchReindexJobStatusQueued {
		t.Fatalf("unexpected job: %+v", got)
	}
}

func TestSearchReindexStopAndResumeJob(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Now().UTC()
	repo := &inMemoryReindexJobRepo{
		jobs: map[string]*repository.SearchReindexJob{
			"rj_1": {
				ID:        "rj_1",
				Status:    repository.SearchReindexJobStatusQueued,
				PageSize:  200,
				NextPage:  3,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
	h := New(nil, nil, nil, nil, nil, newDummyES(t), nil, nil, repo)

	r := gin.New()
	r.POST("/jobs/:id/stop", h.SearchReindexStopJob)
	r.POST("/jobs/:id/resume", h.SearchReindexResumeJob)

	stopW := httptest.NewRecorder()
	r.ServeHTTP(stopW, httptest.NewRequest(http.MethodPost, "/jobs/rj_1/stop", nil))
	if stopW.Code != http.StatusAccepted {
		t.Fatalf("expected stop 202, got %d body=%s", stopW.Code, stopW.Body.String())
	}

	repo.mu.Lock()
	repo.jobs["rj_1"].Status = repository.SearchReindexJobStatusPaused
	repo.mu.Unlock()

	resumeW := httptest.NewRecorder()
	r.ServeHTTP(resumeW, httptest.NewRequest(http.MethodPost, "/jobs/rj_1/resume", nil))
	if resumeW.Code != http.StatusAccepted {
		t.Fatalf("expected resume 202, got %d body=%s", resumeW.Code, resumeW.Body.String())
	}
}

func TestSearchReindexGetJobForcePausesStaleStopRequestedRunningJob(t *testing.T) {
	gin.SetMode(gin.TestMode)
	old := time.Now().UTC().Add(-2 * time.Minute)
	repo := &inMemoryReindexJobRepo{
		jobs: map[string]*repository.SearchReindexJob{
			"rj_stale": {
				ID:            "rj_stale",
				Status:        repository.SearchReindexJobStatusRunning,
				DryRun:        true,
				PageSize:      200,
				NextPage:      33,
				StopRequested: true,
				TotalAssets:   1000,
				AssetsScanned: 6400,
				CreatedAt:     old.Add(-5 * time.Minute),
				UpdatedAt:     old,
				StartedAt:     &old,
			},
		},
	}
	h := New(nil, nil, nil, nil, nil, newDummyES(t), nil, nil, repo)
	r := gin.New()
	r.GET("/jobs/:id", h.SearchReindexGetJob)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/jobs/rj_stale", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var got SearchReindexJobView
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got.Status != repository.SearchReindexJobStatusPaused {
		t.Fatalf("expected paused, got %+v", got)
	}
}
