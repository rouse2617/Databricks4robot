package pipeline

import (
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	sigsyaml "sigs.k8s.io/yaml"
)

type templateResourceSpec struct {
	CPURequest    string
	MemoryRequest string
	CPULimit      string
	MemoryLimit   string
}

func buildPodResourceUsageReport(wf *wfv1.Workflow, manifest *string, observedAt, onlyNodeID string) []PodResourceUsage {
	if wf == nil {
		return []PodResourceUsage{}
	}
	reqs := templateResourcesFromManifest(manifest)
	pods := make([]PodResourceUsage, 0)
	for _, node := range wf.Status.Nodes {
		if node.Type != wfv1.NodeTypePod {
			continue
		}
		if onlyNodeID != "" && node.ID != onlyNodeID {
			continue
		}
		podName := node.ID
		if podName == "" {
			podName = node.Name
		}
		entry := PodResourceUsage{
			PodName:              podName,
			NodeID:               node.ID,
			NodeName:             node.HostNodeName,
			TemplateName:         node.TemplateName,
			ObservedAt:           observedAt,
			LiveMetricsAvailable: false,
		}
		durationCPU, durationMemory := formatResourcesDuration(node.ResourcesDuration)
		entry.CPUUsage = durationCPU
		entry.MemoryUsage = durationMemory
		entry.CPUResourceDuration = durationCPU
		entry.MemoryResourceDuration = durationMemory
		entry.ResourceDuration = ResourceValues{CPU: durationCPU, Memory: durationMemory}
		if spec, ok := reqs[node.TemplateName]; ok {
			entry.CPURequest = spec.CPURequest
			entry.MemoryRequest = spec.MemoryRequest
			entry.CPULimit = spec.CPULimit
			entry.MemoryLimit = spec.MemoryLimit
			entry.Requests = ResourceValues{CPU: spec.CPURequest, Memory: spec.MemoryRequest}
			entry.Limits = ResourceValues{CPU: spec.CPULimit, Memory: spec.MemoryLimit}
		}
		pods = append(pods, entry)
	}
	return pods
}

func specSource(manifest *string) string {
	if manifest == nil || strings.TrimSpace(*manifest) == "" {
		return "unavailable"
	}
	return "stored-manifest"
}

func formatResourcesDuration(d wfv1.ResourcesDuration) (cpu, mem string) {
	if len(d) == 0 {
		return "", ""
	}
	if v, ok := d[corev1.ResourceCPU]; ok && v > 0 {
		cpu = v.String()
	}
	if v, ok := d[corev1.ResourceMemory]; ok && v > 0 {
		mem = v.String()
	}
	return cpu, mem
}

func templateResourcesFromManifest(manifest *string) map[string]templateResourceSpec {
	out := make(map[string]templateResourceSpec)
	if manifest == nil || strings.TrimSpace(*manifest) == "" {
		return out
	}
	var wf wfv1.Workflow
	if err := sigsyaml.Unmarshal([]byte(*manifest), &wf); err != nil {
		return out
	}
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == "" {
			continue
		}
		var res corev1.ResourceRequirements
		switch {
		case tmpl.Container != nil:
			res = tmpl.Container.Resources
		case tmpl.Script != nil:
			res = tmpl.Script.Resources
		default:
			continue
		}
		out[tmpl.Name] = templateResourceSpec{
			CPURequest:    quantityString(res.Requests[corev1.ResourceCPU]),
			MemoryRequest: quantityString(res.Requests[corev1.ResourceMemory]),
			CPULimit:      quantityString(res.Limits[corev1.ResourceCPU]),
			MemoryLimit:   quantityString(res.Limits[corev1.ResourceMemory]),
		}
	}
	return out
}

func quantityString(q resource.Quantity) string {
	if q.IsZero() {
		return ""
	}
	return q.String()
}
