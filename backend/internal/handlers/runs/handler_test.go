package runs_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/handlers/runs"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	runsUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/runs"
)

type stubRunRepo struct {
	items []models.DatabrewRun
}

func (s *stubRunRepo) Save(_ context.Context, run *models.DatabrewRun) error {
	s.items = append(s.items, *run)
	return nil
}

func (s *stubRunRepo) FindByID(_ context.Context, id string) (*models.DatabrewRun, error) {
	for i := range s.items {
		if s.items[i].ID == id {
			copy := s.items[i]
			return &copy, nil
		}
	}
	return nil, nil
}

func (s *stubRunRepo) FindByRuntimeResource(_ context.Context, _, name string) (*models.DatabrewRun, error) {
	for i := range s.items {
		if s.items[i].RuntimeResourceName == name {
			copy := s.items[i]
			return &copy, nil
		}
	}
	return nil, nil
}

func (s *stubRunRepo) List(_ context.Context, filter models.DatabrewRunListFilter) ([]models.DatabrewRun, int, error) {
	filtered := make([]models.DatabrewRun, 0)
	for _, item := range s.items {
		if filter.Type != "" && item.Type != filter.Type {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered, len(filtered), nil
}

type stubComponentBuildRepo struct{}

func (s *stubComponentBuildRepo) Save(_ context.Context, _ *models.ComponentBuildRun) error {
	return nil
}

func (s *stubComponentBuildRepo) FindByRunID(_ context.Context, _ string) (*models.ComponentBuildRun, error) {
	return nil, nil
}

type stubRAGBuildRepo struct{}

func (s *stubRAGBuildRepo) Save(_ context.Context, _ *models.RAGBuildRun) error {
	return nil
}

func (s *stubRAGBuildRepo) FindByRunID(_ context.Context, _ string) (*models.RAGBuildRun, error) {
	return nil, nil
}

type stubComponentReleaseRepo struct{}

func (s *stubComponentReleaseRepo) Save(_ context.Context, _ *models.ComponentRelease) error {
	return nil
}

func (s *stubComponentReleaseRepo) ListByComponentID(_ context.Context, _ string) ([]models.ComponentRelease, error) {
	return nil, nil
}

func TestHandler_GetRun(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubRunRepo{
		items: []models.DatabrewRun{
			{
				ID:                  "run-1",
				Type:                models.RunTypePipeline,
				Name:                "demo",
				Status:              "Running",
				Runtime:             "argo",
				RuntimeNamespace:    "argo",
				RuntimeResourceName: "demo-wf",
				CreatedAt:           time.Now().UTC(),
				UpdatedAt:           time.Now().UTC(),
			},
		},
	}
	uc := runsUC.New(repo, &stubComponentBuildRepo{}, &stubRAGBuildRepo{}, &stubComponentReleaseRepo{}, nil, nil, nil, "argo")
	h := runs.New(uc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "runId", Value: "run-1"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/runs/run-1", nil)

	h.GetRun(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["id"] != "run-1" {
		t.Fatalf("unexpected id: %v", payload["id"])
	}
}

func TestHandler_ListRuns(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubRunRepo{
		items: []models.DatabrewRun{
			{
				ID:                  "run-2",
				Type:                models.RunTypeComponentBuild,
				Name:                "build-demo",
				Status:              "Pending",
				Runtime:             "argo",
				RuntimeNamespace:    "argo",
				RuntimeResourceName: "build-demo-1",
				CreatedAt:           time.Now().UTC(),
				UpdatedAt:           time.Now().UTC(),
			},
		},
	}
	uc := runsUC.New(repo, &stubComponentBuildRepo{}, &stubRAGBuildRepo{}, &stubComponentReleaseRepo{}, nil, nil, nil, "argo")
	h := runs.New(uc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/runs?type=component_build", nil)

	h.ListRuns(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "run-2") {
		t.Fatalf("expected run-2 in body: %s", w.Body.String())
	}
}
