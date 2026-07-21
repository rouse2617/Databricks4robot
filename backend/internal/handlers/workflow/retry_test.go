package workflow

import (
	"testing"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func TestWorkflowCanRetry(t *testing.T) {
	failedNode := wfv1.NodeStatus{
		ID:    "task-1",
		Name:  "task-1",
		Type:  wfv1.NodeTypePod,
		Phase: wfv1.NodeFailed,
	}
	pendingNode := wfv1.NodeStatus{
		ID:    "task-2",
		Name:  "task-2",
		Type:  wfv1.NodeTypePod,
		Phase: wfv1.NodePending,
	}

	tests := []struct {
		name string
		wf   *wfv1.Workflow
		want bool
	}{
		{
			name: "failed node",
			wf: &wfv1.Workflow{
				Status: wfv1.WorkflowStatus{
					Phase: wfv1.WorkflowFailed,
					Nodes: wfv1.Nodes{"task-1": failedNode},
				},
			},
			want: true,
		},
		{
			name: "stopped workflow",
			wf: &wfv1.Workflow{
				Status: wfv1.WorkflowStatus{
					Phase:   wfv1.WorkflowFailed,
					Message: "Stopped",
					Nodes:   wfv1.Nodes{"task-2": pendingNode},
				},
			},
			want: false,
		},
		{
			name: "failed workflow without failed nodes",
			wf: &wfv1.Workflow{
				Status: wfv1.WorkflowStatus{
					Phase: wfv1.WorkflowFailed,
					Nodes: wfv1.Nodes{"task-2": pendingNode},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := workflowCanRetry(tt.wf); got != tt.want {
				t.Fatalf("workflowCanRetry() = %v, want %v", got, tt.want)
			}
		})
	}
}
