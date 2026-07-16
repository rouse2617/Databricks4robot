package pipeline

import (
	"testing"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

	"github.com/CyberOrigin2077/cyber-databrew/internal/transpiler"
)

// CYB-3491: Argo's status.progress counts the injected databrew-exit-notify
// onExit hook as a pod, so a single-step pipeline reports "1/2"/"2/2".
// workflowStepProgress must exclude that hook and count only real step pods.
func TestWorkflowStepProgress_ExcludesExitNotifyHook(t *testing.T) {
	dag := wfv1.NodeStatus{ID: "wf", Name: "wf", Type: wfv1.NodeTypeDAG, Phase: wfv1.NodeSucceeded, TemplateName: "dag"}
	step := func(phase wfv1.NodePhase) wfv1.NodeStatus {
		return wfv1.NodeStatus{ID: "s1", Name: "step-1", Type: wfv1.NodeTypePod, Phase: phase, TemplateName: "step-1"}
	}
	step2 := func(phase wfv1.NodePhase) wfv1.NodeStatus {
		return wfv1.NodeStatus{ID: "s2", Name: "step-2", Type: wfv1.NodeTypePod, Phase: phase, TemplateName: "step-2"}
	}
	hook := func(phase wfv1.NodePhase) wfv1.NodeStatus {
		return wfv1.NodeStatus{ID: "x", Name: "wf.onExit", Type: wfv1.NodeTypePod, Phase: phase, TemplateName: transpiler.ExitNotifyTemplateName}
	}
	wfWith := func(nodes ...wfv1.NodeStatus) *wfv1.Workflow {
		m := map[string]wfv1.NodeStatus{}
		for _, n := range nodes {
			m[n.ID] = n
		}
		return &wfv1.Workflow{Status: wfv1.WorkflowStatus{Nodes: m}}
	}

	cases := []struct {
		name string
		wf   *wfv1.Workflow
		want string
	}{
		{
			// The reported bug: step done, exit hook also done. Argo says "2/2".
			name: "single step done, hook done -> 1/1",
			wf:   wfWith(dag, step(wfv1.NodeSucceeded), hook(wfv1.NodeSucceeded)),
			want: "1/1",
		},
		{
			// The "1/2 运行中 but actually finished" window: step done, hook still running.
			name: "single step done, hook running -> 1/1",
			wf:   wfWith(dag, step(wfv1.NodeSucceeded), hook(wfv1.NodeRunning)),
			want: "1/1",
		},
		{
			name: "single step running -> 0/1",
			wf:   wfWith(dag, step(wfv1.NodeRunning)),
			want: "0/1",
		},
		{
			name: "two steps, one done one running -> 1/2",
			wf:   wfWith(dag, step(wfv1.NodeSucceeded), step2(wfv1.NodeRunning), hook(wfv1.NodePending)),
			want: "1/2",
		},
		{
			// Nothing but the DAG container observed yet -> fall back to raw.
			name: "no step pods yet -> empty",
			wf:   wfWith(dag),
			want: "",
		},
		{
			name: "nil workflow -> empty",
			wf:   nil,
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := workflowStepProgress(tc.wf); got != tc.want {
				t.Fatalf("workflowStepProgress = %q, want %q", got, tc.want)
			}
		})
	}
}
