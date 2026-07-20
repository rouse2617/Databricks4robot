package argo

import (
	"testing"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func wfWithNodes(name string, nodes map[string]wfv1.NodeStatus) *wfv1.Workflow {
	wf := &wfv1.Workflow{}
	wf.Name = name
	wf.Status.Nodes = nodes
	return wf
}

func TestPodNameForNode_V2NameDiffersFromNodeID(t *testing.T) {
	wf := wfWithNodes("wf-1", map[string]wfv1.NodeStatus{
		"wf-1-2": {ID: "wf-1-2", TemplateName: "transcode", Type: wfv1.NodeTypePod},
	})
	got, ok := PodNameForNode(wf, "wf-1-2")
	if !ok {
		t.Fatalf("expected resolvable pod node")
	}
	if want := "wf-1-transcode-2"; got != want {
		t.Errorf("pod name: want %q, got %q", want, got)
	}
	if got == "wf-1-2" {
		t.Errorf("pod name must not equal node ID (regression: CRD client used node.ID)")
	}
}

func TestPodNameForNode_SanitizesTemplateSegment(t *testing.T) {
	wf := wfWithNodes("wf-1", map[string]wfv1.NodeStatus{
		"wf-1-9": {ID: "wf-1-9", TemplateName: "Step X/Encode", Type: wfv1.NodeTypePod},
	})
	got, ok := PodNameForNode(wf, "wf-1-9")
	if !ok {
		t.Fatalf("expected ok")
	}
	if want := "wf-1-step-x-encode-9"; got != want {
		t.Errorf("sanitized pod name: want %q, got %q", want, got)
	}
}

func TestPodNameForNode_FallsBackToDisplayName(t *testing.T) {
	wf := wfWithNodes("wf-1", map[string]wfv1.NodeStatus{
		"wf-1-3": {ID: "wf-1-3", DisplayName: "cleanup", Type: wfv1.NodeTypePod},
	})
	got, ok := PodNameForNode(wf, "wf-1-3")
	if !ok {
		t.Fatalf("expected ok")
	}
	if want := "wf-1-cleanup-3"; got != want {
		t.Errorf("displayName fallback: want %q, got %q", want, got)
	}
}

func TestPodNameForNode_RootNodeFallsBackToID(t *testing.T) {
	// Single-node workflow: node ID equals the workflow name, so there is no
	// "<workflow>-" prefix to strip → argo names the pod after the node ID.
	wf := wfWithNodes("wf-1", map[string]wfv1.NodeStatus{
		"wf-1": {ID: "wf-1", TemplateName: "main", Type: wfv1.NodeTypePod},
	})
	got, ok := PodNameForNode(wf, "wf-1")
	if !ok {
		t.Fatalf("expected ok")
	}
	if got != "wf-1" {
		t.Errorf("root pod name: want %q, got %q", "wf-1", got)
	}
}

func TestPodNameForNode_NonPodNodeRejected(t *testing.T) {
	wf := wfWithNodes("wf-1", map[string]wfv1.NodeStatus{
		"wf-1-0": {ID: "wf-1-0", TemplateName: "dag", Type: wfv1.NodeTypeDAG},
	})
	if _, ok := PodNameForNode(wf, "wf-1-0"); ok {
		t.Errorf("non-pod node must not resolve to a pod name")
	}
}

func TestPodNameForNode_MissingInputs(t *testing.T) {
	if _, ok := PodNameForNode(nil, "wf-1-2"); ok {
		t.Errorf("nil workflow must return false")
	}
	wf := wfWithNodes("wf-1", map[string]wfv1.NodeStatus{
		"wf-1-2": {ID: "wf-1-2", TemplateName: "transcode", Type: wfv1.NodeTypePod},
	})
	if _, ok := PodNameForNode(wf, ""); ok {
		t.Errorf("empty node ID must return false")
	}
	if _, ok := PodNameForNode(wf, "missing"); ok {
		t.Errorf("unknown node ID must return false")
	}
}
