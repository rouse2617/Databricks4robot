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
		got.SuspendedCount != want.SuspendedCount {
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
