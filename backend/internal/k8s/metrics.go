package k8s

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
)

// PodResourceUsage represents CPU/memory usage for a single pod.
type PodResourceUsage struct {
	PodName       string            `json:"pod_name"`
	NodeName      string            `json:"node_name,omitempty"`
	CPUUsage      string            `json:"cpu_usage"`      // e.g. "125m"
	MemoryUsage   string            `json:"memory_usage"`   // e.g. "64Mi"
	CPURequest    string            `json:"cpu_request"`    // from pod spec
	MemoryRequest string            `json:"memory_request"` // from pod spec
	CPULimit      string            `json:"cpu_limit"`      // from pod spec
	MemoryLimit   string            `json:"memory_limit"`   // from pod spec
}

// MetricsClient defines the interface for querying pod resource usage.
type MetricsClient interface {
	// GetWorkflowResourceUsage returns per-pod resource usage for pods
	// belonging to the given workflow name in the given namespace.
	GetWorkflowResourceUsage(ctx context.Context, workflowName, namespace string) ([]PodResourceUsage, error)
}

// K8sMetricsClient implements MetricsClient using the K8s Metrics API.
type K8sMetricsClient struct {
	kubeClientset   kubernetes.Interface
	metricsClientset metricsclientset.Interface
}

// NewMetricsClient creates a K8sMetricsClient.
func NewMetricsClient(kubeClientset kubernetes.Interface, metricsClientset metricsclientset.Interface) *K8sMetricsClient {
	return &K8sMetricsClient{kubeClientset: kubeClientset, metricsClientset: metricsClientset}
}

// GetWorkflowResourceUsage fetches pod metrics and pod specs for a workflow.
func (m *K8sMetricsClient) GetWorkflowResourceUsage(ctx context.Context, workflowName, namespace string) ([]PodResourceUsage, error) {
	selector := fmt.Sprintf("workflows.argoproj.io/workflow=%s", workflowName)

	pods, err := m.kubeClientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, fmt.Errorf("list pods: %w", err)
	}
	if len(pods.Items) == 0 {
		return nil, nil
	}

	// Get pod metrics.
	podMetrics := make(map[string]*metricsv1beta1.PodMetrics)
	if m.metricsClientset != nil {
		metricsList, err := m.metricsClientset.MetricsV1beta1().PodMetricses(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
		if err == nil {
			for i := range metricsList.Items {
				podMetrics[metricsList.Items[i].Name] = &metricsList.Items[i]
			}
		}
	}

	var result []PodResourceUsage
	for _, pod := range pods.Items {
		usage := PodResourceUsage{
			PodName:  pod.Name,
			NodeName: pod.Spec.NodeName,
		}

		// Get resource requests/limits from pod spec (first container or aggregate).
		usage.CPURequest, usage.MemoryRequest, usage.CPULimit, usage.MemoryLimit = aggregatePodResources(pod)

		// Get actual usage from metrics.
		if pm, ok := podMetrics[pod.Name]; ok {
			usage.CPUUsage, usage.MemoryUsage = aggregatePodMetrics(*pm)
		}

		result = append(result, usage)
	}

	return result, nil
}

// aggregatePodResources sums CPU/memory requests and limits across all containers.
func aggregatePodResources(pod corev1.Pod) (cpuReq, memReq, cpuLim, memLim string) {
	var totalCPUReq, totalMemReq, totalCPULim, totalMemLim resource.Quantity
	for _, c := range pod.Spec.Containers {
		if r, ok := c.Resources.Requests[corev1.ResourceCPU]; ok {
			totalCPUReq.Add(r)
		}
		if r, ok := c.Resources.Requests[corev1.ResourceMemory]; ok {
			totalMemReq.Add(r)
		}
		if r, ok := c.Resources.Limits[corev1.ResourceCPU]; ok {
			totalCPULim.Add(r)
		}
		if r, ok := c.Resources.Limits[corev1.ResourceMemory]; ok {
			totalMemLim.Add(r)
		}
	}
	if !totalCPUReq.IsZero() {
		cpuReq = totalCPUReq.String()
	}
	if !totalMemReq.IsZero() {
		memReq = totalMemReq.String()
	}
	if !totalCPULim.IsZero() {
		cpuLim = totalCPULim.String()
	}
	if !totalMemLim.IsZero() {
		memLim = totalMemLim.String()
	}
	return
}

// aggregatePodMetrics sums CPU/memory usage across all containers.
func aggregatePodMetrics(pm metricsv1beta1.PodMetrics) (cpu, mem string) {
	var totalCPU, totalMem resource.Quantity
	for _, c := range pm.Containers {
		totalCPU.Add(c.Usage[corev1.ResourceCPU])
		totalMem.Add(c.Usage[corev1.ResourceMemory])
	}
	if !totalCPU.IsZero() {
		cpu = totalCPU.String()
	}
	if !totalMem.IsZero() {
		mem = totalMem.String()
	}
	return
}
