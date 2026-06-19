package run

import (
	"context"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

type fakePipelineService struct {
	PipelineUsecase
	refreshActive bool
}

func (f *fakePipelineService) ListRuns(_ context.Context, refreshActive bool) ([]models.PipelineRun, error) {
	f.refreshActive = refreshActive
	return []models.PipelineRun{{ID: "run-1", Status: "Running"}}, nil
}

func TestServiceDelegatesRunList(t *testing.T) {
	fake := &fakePipelineService{}
	svc := NewService(fake)

	items, err := svc.ListRuns(context.Background(), true)
	if err != nil {
		t.Fatalf("ListRuns returned error: %v", err)
	}
	if !fake.refreshActive {
		t.Fatalf("ListRuns did not delegate refreshActive")
	}
	if len(items) != 1 || items[0].ID != "run-1" {
		t.Fatalf("unexpected items: %#v", items)
	}
}
