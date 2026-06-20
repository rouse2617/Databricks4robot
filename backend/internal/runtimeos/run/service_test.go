package run

import (
	"context"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

type fakePipelineService struct {
	PipelineUsecase
	refreshActive bool
	calls         []string
}

func (f *fakePipelineService) ListRuns(_ context.Context, refreshActive bool) ([]models.PipelineRun, error) {
	f.refreshActive = refreshActive
	return []models.PipelineRun{{ID: "run-1", Status: "Running"}}, nil
}

func (f *fakePipelineService) RuntimeRetryRun(_ context.Context, id string) (*models.PipelineRun, error) {
	f.calls = append(f.calls, "runtime_retry:"+id)
	return &models.PipelineRun{ID: id, Status: "Failed"}, nil
}

func (f *fakePipelineService) StopRun(_ context.Context, id string) error {
	f.calls = append(f.calls, "stop:"+id)
	return nil
}

func (f *fakePipelineService) RerunRun(_ context.Context, id string) (*models.PipelineRun, error) {
	f.calls = append(f.calls, "rerun:"+id)
	return &models.PipelineRun{ID: id + "-rerun", Status: "Pending"}, nil
}

func (f *fakePipelineService) SuspendRun(_ context.Context, id string) error {
	f.calls = append(f.calls, "suspend:"+id)
	return nil
}

func (f *fakePipelineService) ResumeRun(_ context.Context, id string) error {
	f.calls = append(f.calls, "resume:"+id)
	return nil
}

func (f *fakePipelineService) TerminateRun(_ context.Context, id string) error {
	f.calls = append(f.calls, "terminate:"+id)
	return nil
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

func TestServiceControlsRunLifecycle(t *testing.T) {
	fake := &fakePipelineService{}
	svc := NewService(fake)

	run, err := svc.ControlRun(context.Background(), "run-1", LifecycleRuntimeRetry)
	if err != nil {
		t.Fatalf("ControlRun retry returned error: %v", err)
	}
	if run == nil || run.ID != "run-1" {
		t.Fatalf("unexpected retry run: %#v", run)
	}
	for _, op := range []LifecycleOperation{LifecycleStop, LifecycleSuspend, LifecycleResume, LifecycleTerminate} {
		if _, err := svc.ControlRun(context.Background(), "run-1", op); err != nil {
			t.Fatalf("ControlRun %s returned error: %v", op, err)
		}
	}
	want := []string{
		"runtime_retry:run-1",
		"stop:run-1",
		"suspend:run-1",
		"resume:run-1",
		"terminate:run-1",
	}
	if len(fake.calls) != len(want) {
		t.Fatalf("unexpected calls: %#v", fake.calls)
	}
	for i := range want {
		if fake.calls[i] != want[i] {
			t.Fatalf("unexpected calls: got %#v want %#v", fake.calls, want)
		}
	}

	if _, err := svc.ControlRun(context.Background(), "run-1", LifecycleOperation("unknown")); err == nil {
		t.Fatal("expected unsupported lifecycle operation error")
	}
}
