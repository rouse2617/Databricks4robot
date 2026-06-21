package state

import (
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

func TestNormalizeRunStatus(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"":              StatusPending,
		"completed":     StatusSucceeded,
		"Succeeded":     StatusSucceeded,
		"pilot_running": StatusRunning,
		"paused":        StatusSuspended,
		"cancelled":     StatusCancelled,
		"Error":         StatusError,
		"Expired":       StatusExpired,
	}
	for input, want := range tests {
		if got := NormalizeRunStatus(input); got != want {
			t.Fatalf("NormalizeRunStatus(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestAggregateChildRuns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   []models.PipelineRun
		want models.RunChildSummary
	}{
		{
			name: "empty is pending",
			want: models.RunChildSummary{
				Statuses:        map[string]int{},
				AggregateStatus: StatusPending,
				HealthStatus:    "pending",
			},
		},
		{
			name: "all success",
			in: []models.PipelineRun{
				{ID: "run-1", Status: "Succeeded"},
				{ID: "run-2", Status: "completed"},
			},
			want: models.RunChildSummary{
				Total:           2,
				Statuses:        map[string]int{StatusSucceeded: 2},
				AggregateStatus: StatusSucceeded,
				HealthStatus:    "healthy",
				TerminalCount:   2,
				SucceededCount:  2,
			},
		},
		{
			name: "active children keep aggregate active",
			in: []models.PipelineRun{
				{ID: "run-1", Status: "Failed"},
				{ID: "run-2", Status: "Running"},
				{ID: "run-3", Status: "Pending"},
			},
			want: models.RunChildSummary{
				Total:           3,
				Statuses:        map[string]int{StatusFailed: 1, StatusRunning: 1, StatusPending: 1},
				AggregateStatus: StatusRunning,
				ActiveCount:     2,
				TerminalCount:   1,
				FailedCount:     1,
				RunningCount:    1,
				PendingCount:    1,
				HasFailures:     true,
				HasBlocking:     true,
				HealthStatus:    "degraded",
			},
		},
		{
			name: "error wins after terminal failure",
			in: []models.PipelineRun{
				{ID: "run-1", Status: "Failed"},
				{ID: "run-2", Status: "Error"},
			},
			want: models.RunChildSummary{
				Total:           2,
				Statuses:        map[string]int{StatusFailed: 1, StatusError: 1},
				AggregateStatus: StatusError,
				TerminalCount:   2,
				FailedCount:     2,
				HasFailures:     true,
				HealthStatus:    "failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := AggregateChildRuns(tt.in)
			assertSummary(t, got, tt.want)
		})
	}
}

func TestAggregateChildRunsTopFailureReasons(t *testing.T) {
	t.Parallel()

	children := []models.PipelineRun{
		{
			ID:           "run-unsched-1",
			Status:       "Error",
			Message:      "Unschedulable: 0/12 nodes are available: 2 Insufficient memory.",
			AssetIDs:     []string{"asset-1"},
			WorkflowName: "wf-1",
		},
		{
			ID:           "run-unsched-2",
			Status:       "Failed",
			Message:      "0/12 nodes are available: 1 Insufficient cpu.",
			AssetIDs:     []string{"asset-2"},
			WorkflowName: "wf-2",
		},
		{
			ID:           "run-image",
			Status:       "Error",
			Message:      "InvalidImageName: couldn't parse image name",
			AssetIDs:     []string{"asset-3"},
			WorkflowName: "wf-3",
		},
	}

	summary := AggregateChildRuns(children)
	if len(summary.TopFailureReasons) != 2 {
		t.Fatalf("top reasons = %+v, want 2", summary.TopFailureReasons)
	}
	if got := summary.TopFailureReasons[0]; got.Reason != "unschedulable" || got.Count != 2 || got.ExampleRunID != "run-unsched-1" || got.ExampleAsset != "asset-1" {
		t.Fatalf("top reason = %+v, want unschedulable count=2", got)
	}
	if got := summary.TopFailureReasons[1]; got.Reason != "image_startup" || got.Count != 1 {
		t.Fatalf("second reason = %+v, want image_startup count=1", got)
	}
	if !summary.HasFailures {
		t.Fatal("expected summary to report failures")
	}
}

func TestAggregateChildRunsGenericFailureWithoutMessage(t *testing.T) {
	t.Parallel()

	summary := AggregateChildRuns([]models.PipelineRun{
		{ID: "run-empty-message", Status: "Failed", WorkflowName: "wf"},
	})
	if !summary.HasFailures {
		t.Fatal("expected summary to report failures")
	}
	if len(summary.TopFailureReasons) != 1 {
		t.Fatalf("top reasons = %+v, want one generic failure", summary.TopFailureReasons)
	}
	if got := summary.TopFailureReasons[0]; got.Reason != "run_failed" || got.Count != 1 || got.Message == "" {
		t.Fatalf("top reason = %+v, want generic run_failed with default message", got)
	}
}

func TestAnnotateRunDiagnosticsPendingWithoutRuntime(t *testing.T) {
	t.Parallel()

	run := models.PipelineRun{ID: "run-pending", Status: "Pending"}
	AnnotateRunDiagnostics(&run)
	if run.BlockingReason != "runtime_not_submitted" {
		t.Fatalf("blocking reason = %q, want runtime_not_submitted", run.BlockingReason)
	}
	if run.FailureReason != "" {
		t.Fatalf("failure reason = %q, want empty", run.FailureReason)
	}
	if run.BlockingMessage == "" {
		t.Fatal("expected default blocking message")
	}
}

func TestAnnotateRunDiagnosticsFailure(t *testing.T) {
	t.Parallel()

	run := models.PipelineRun{
		ID:           "run-failed",
		Status:       "Error",
		Message:      "ImagePullBackOff: pull access denied",
		WorkflowName: "wf",
	}
	AnnotateRunDiagnostics(&run)
	if run.FailureReason != "image_startup" {
		t.Fatalf("failure reason = %q, want image_startup", run.FailureReason)
	}
	if run.BlockingReason != "" || run.BlockingMessage != "" {
		t.Fatalf("blocking fields = %q/%q, want empty", run.BlockingReason, run.BlockingMessage)
	}
}

func TestAnnotateRunDiagnosticsFailureWithoutTopLevelMessageUsesNodeMessage(t *testing.T) {
	t.Parallel()

	run := models.PipelineRun{
		ID:           "run-node-failed",
		Status:       "Error",
		WorkflowName: "wf",
		Nodes: []models.PipelineRunNode{
			{
				ID:      "node-1",
				Phase:   "Error",
				Message: "InvalidImageName: couldn't parse image name",
			},
		},
	}
	AnnotateRunDiagnostics(&run)
	if run.FailureReason != "image_startup" {
		t.Fatalf("failure reason = %q, want image_startup", run.FailureReason)
	}
	if run.Message != "" {
		t.Fatalf("run message = %q, want original empty message preserved", run.Message)
	}
}

func TestAnnotateRunDiagnosticsRuntimeConfigProjectionFailed(t *testing.T) {
	t.Parallel()

	run := models.PipelineRun{
		ID:      "run-config",
		Status:  "Error",
		Message: "create runtime config projection failed: configmap forbidden",
	}
	AnnotateRunDiagnostics(&run)
	if run.FailureReason != "runtime_config_projection_failed" {
		t.Fatalf("failure reason = %q, want runtime_config_projection_failed", run.FailureReason)
	}
}

func TestAnnotateRunDiagnosticsGenericFailureWithoutMessage(t *testing.T) {
	t.Parallel()

	run := models.PipelineRun{ID: "run-failed", Status: "Failed", WorkflowName: "wf"}
	AnnotateRunDiagnostics(&run)
	if run.FailureReason != "run_failed" {
		t.Fatalf("failure reason = %q, want run_failed", run.FailureReason)
	}
	if run.Message != "" {
		t.Fatalf("run message = %q, want original empty message preserved", run.Message)
	}
}

func TestAnnotateRunDiagnosticsResourceIncompatible(t *testing.T) {
	t.Parallel()

	run := models.PipelineRun{
		ID:      "run-resource",
		Status:  "Failed",
		Message: `invalid argument: 执行目标 "Default Argo target"不支持该资源规格：节点请求 cpu=14000m，最大可用 cpu=8。`,
	}
	AnnotateRunDiagnostics(&run)
	if run.FailureReason != "resource_incompatible" {
		t.Fatalf("failure reason = %q, want resource_incompatible", run.FailureReason)
	}
}

func TestBuildBatchChildRelations(t *testing.T) {
	t.Parallel()

	parentID := "batch-1"
	children := []models.PipelineRun{
		{ID: "run-1", BatchJobID: &parentID, AssetIDs: []string{"asset-1"}},
		{ID: "run-2", AssetIDs: []string{"asset-2"}},
		{ID: "batch-1", BatchJobID: &parentID},
	}

	relations := BuildBatchChildRelations(parentID, children)
	if len(relations) != 1 {
		t.Fatalf("relations = %d, want 1: %+v", len(relations), relations)
	}
	got := relations[0]
	if got.ParentRunID != parentID || got.ChildRunID != "run-1" || got.RelationType != "batch_child" || got.AssetID != "asset-1" {
		t.Fatalf("unexpected relation: %+v", got)
	}
}

func assertSummary(t *testing.T, got, want models.RunChildSummary) {
	t.Helper()
	if got.Total != want.Total ||
		got.AggregateStatus != want.AggregateStatus ||
		got.ActiveCount != want.ActiveCount ||
		got.TerminalCount != want.TerminalCount ||
		got.SucceededCount != want.SucceededCount ||
		got.FailedCount != want.FailedCount ||
		got.CancelledCount != want.CancelledCount ||
		got.PendingCount != want.PendingCount ||
		got.RunningCount != want.RunningCount ||
		got.SuspendedCount != want.SuspendedCount ||
		got.HasFailures != want.HasFailures ||
		got.HasBlocking != want.HasBlocking ||
		got.HealthStatus != want.HealthStatus {
		t.Fatalf("summary = %+v, want %+v", got, want)
	}
	if len(got.Statuses) != len(want.Statuses) {
		t.Fatalf("statuses = %+v, want %+v", got.Statuses, want.Statuses)
	}
	for status, wantCount := range want.Statuses {
		if got.Statuses[status] != wantCount {
			t.Fatalf("statuses[%q] = %d, want %d", status, got.Statuses[status], wantCount)
		}
	}
}
