package pipeline

import (
	"testing"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestBuildPodResourceUsageReport(t *testing.T) {
	manifest := `apiVersion: argoproj.io/v1alpha1
kind: Workflow
spec:
  templates:
  - name: main
    container:
      resources:
        requests:
          cpu: "100m"
          memory: "128Mi"
        limits:
          cpu: "200m"
          memory: "256Mi"
`
	wf := &wfv1.Workflow{}
	wf.Status.Nodes = wfv1.Nodes{
		"pod-1": {
			ID:           "pod-1",
			Name:         "pod-1",
			Type:         wfv1.NodeTypePod,
			TemplateName: "main",
			HostNodeName: "node-a",
			ResourcesDuration: wfv1.ResourcesDuration{
				corev1.ResourceCPU:    wfv1.NewResourceDuration(30 * time.Second),
				corev1.ResourceMemory: wfv1.NewResourceDuration(45 * time.Second),
			},
		},
		"step-root": {
			ID:   "step-root",
			Name: "step-root",
			Type: wfv1.NodeTypeSteps,
		},
	}

	pods := buildPodResourceUsageReport(wf, &manifest, "2026-01-01T00:00:00Z", "")
	if len(pods) != 1 {
		t.Fatalf("expected 1 pod, got %d", len(pods))
	}
	if pods[0].PodName != "pod-1" || pods[0].NodeName != "node-a" {
		t.Fatalf("unexpected pod identity: %#v", pods[0])
	}
	if pods[0].CPUUsage != "30s" || pods[0].MemoryUsage != "45s" {
		t.Fatalf("unexpected usage: %#v", pods[0])
	}
	if pods[0].ResourceDuration.CPU != "30s" || pods[0].ResourceDuration.Memory != "45s" {
		t.Fatalf("unexpected resource duration: %#v", pods[0].ResourceDuration)
	}
	if pods[0].CPURequest != "100m" || pods[0].MemoryRequest != "128Mi" {
		t.Fatalf("unexpected requests: %#v", pods[0])
	}
	if pods[0].Requests.CPU != "100m" || pods[0].Requests.Memory != "128Mi" {
		t.Fatalf("unexpected structured requests: %#v", pods[0].Requests)
	}
	if pods[0].CPULimit != "200m" || pods[0].MemoryLimit != "256Mi" {
		t.Fatalf("unexpected limits: %#v", pods[0])
	}
	if pods[0].Limits.CPU != "200m" || pods[0].Limits.Memory != "256Mi" {
		t.Fatalf("unexpected structured limits: %#v", pods[0].Limits)
	}
	if pods[0].LiveMetricsAvailable {
		t.Fatal("expected live metrics unavailable by default")
	}
}

func TestBuildPodResourceUsageReportFiltersNode(t *testing.T) {
	wf := &wfv1.Workflow{}
	wf.Status.Nodes = wfv1.Nodes{
		"pod-1": {ID: "pod-1", Type: wfv1.NodeTypePod},
		"pod-2": {ID: "pod-2", Type: wfv1.NodeTypePod},
	}

	pods := buildPodResourceUsageReport(wf, nil, "2026-01-01T00:00:00Z", "pod-2")
	if len(pods) != 1 || pods[0].PodName != "pod-2" {
		t.Fatalf("unexpected filtered pods: %#v", pods)
	}
}

func TestBuildPodResourceUsageReportFallsBackToLiveWorkflowSpec(t *testing.T) {
	wf := &wfv1.Workflow{}
	wf.Spec.Templates = []wfv1.Template{
		{
			Name: "main",
			Container: &corev1.Container{
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("14"),
						corev1.ResourceMemory: resource.MustParse("55Gi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("14"),
						corev1.ResourceMemory: resource.MustParse("55Gi"),
					},
				},
			},
		},
	}
	wf.Status.Nodes = wfv1.Nodes{
		"pod-1": {ID: "pod-1", Type: wfv1.NodeTypePod, TemplateName: "main"},
	}

	pods := buildPodResourceUsageReport(wf, nil, "2026-01-01T00:00:00Z", "")
	if len(pods) != 1 {
		t.Fatalf("expected 1 pod, got %d", len(pods))
	}
	if pods[0].CPURequest != "14" || pods[0].MemoryRequest != "55Gi" {
		t.Fatalf("expected live spec requests, got %#v", pods[0])
	}
	if pods[0].CPULimit != "14" || pods[0].MemoryLimit != "55Gi" {
		t.Fatalf("expected live spec limits, got %#v", pods[0])
	}
}

func TestQuantityString(t *testing.T) {
	if quantityString(resource.MustParse("0")) != "" {
		t.Fatal("expected empty for zero quantity")
	}
	if got := quantityString(resource.MustParse("100m")); got != "100m" {
		t.Fatalf("expected 100m, got %q", got)
	}
}
