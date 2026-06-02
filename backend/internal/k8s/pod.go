package k8s

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type PodClient interface {
	GetPodDiagnostics(ctx context.Context, namespace, podName string) (*PodDiagnostics, error)
}

type PodDiagnostics struct {
	Cluster            string             `json:"cluster,omitempty"`
	Namespace          string             `json:"namespace"`
	PodName            string             `json:"podName"`
	PodIP              string             `json:"podIp,omitempty"`
	ServiceAccountName string             `json:"serviceAccountName,omitempty"`
	RestartCount       int32              `json:"restartCount"`
	Containers         []ContainerInfo    `json:"containers"`
	Conditions         []PodConditionInfo `json:"podConditions"`
	Events             []EventInfo        `json:"podEvents"`
}

type ContainerInfo struct {
	Name         string `json:"name"`
	Image        string `json:"image,omitempty"`
	Ready        bool   `json:"ready"`
	RestartCount int32  `json:"restartCount"`
	State        string `json:"state,omitempty"`
}

type PodConditionInfo struct {
	Type               string  `json:"type"`
	Status             string  `json:"status"`
	Reason             string  `json:"reason,omitempty"`
	Message            string  `json:"message,omitempty"`
	LastTransitionTime *string `json:"lastTransitionTime,omitempty"`
}

type EventInfo struct {
	Type           string  `json:"type"`
	Reason         string  `json:"reason"`
	Message        string  `json:"message"`
	Count          int32   `json:"count,omitempty"`
	FirstTimestamp *string `json:"firstTimestamp,omitempty"`
	LastTimestamp  *string `json:"lastTimestamp,omitempty"`
}

type podClient struct {
	clientset kubernetes.Interface
	cluster   string
}

func NewPodClient(kubeconfigPath string) (PodClient, error) {
	cs, err := NewClientset(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return NewPodClientWithClientset(cs, os.Getenv("K8S_CLUSTER_NAME")), nil
}

func NewPodClientWithClientset(clientset kubernetes.Interface, cluster string) PodClient {
	return &podClient{clientset: clientset, cluster: cluster}
}

func (c *podClient) GetPodDiagnostics(ctx context.Context, namespace, podName string) (*PodDiagnostics, error) {
	pod, err := c.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get pod %s/%s: %w", namespace, podName, err)
	}

	diag := &PodDiagnostics{
		Cluster:            c.cluster,
		Namespace:          pod.Namespace,
		PodName:            pod.Name,
		PodIP:              pod.Status.PodIP,
		ServiceAccountName: pod.Spec.ServiceAccountName,
		Containers:         []ContainerInfo{},
		Conditions:         []PodConditionInfo{},
		Events:             []EventInfo{},
	}

	var totalRestarts int32
	for _, cs := range pod.Status.ContainerStatuses {
		totalRestarts += cs.RestartCount
		diag.Containers = append(diag.Containers, ContainerInfo{
			Name:         cs.Name,
			Image:        cs.Image,
			Ready:        cs.Ready,
			RestartCount: cs.RestartCount,
			State:        containerStateString(cs.State),
		})
	}
	diag.RestartCount = totalRestarts

	for _, cond := range pod.Status.Conditions {
		item := PodConditionInfo{
			Type:    string(cond.Type),
			Status:  string(cond.Status),
			Reason:  cond.Reason,
			Message: cond.Message,
		}
		if !cond.LastTransitionTime.IsZero() {
			t := cond.LastTransitionTime.Time.UTC().Format("2006-01-02T15:04:05Z")
			item.LastTransitionTime = &t
		}
		diag.Conditions = append(diag.Conditions, item)
	}

	events, err := c.clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.kind=Pod,involvedObject.name=%s", podName),
	})
	if err != nil {
		slog.Warn("failed to list events for pod diagnostics", "namespace", namespace, "podName", podName, "err", err)
	} else {
		for _, ev := range events.Items {
			item := EventInfo{
				Type:    ev.Type,
				Reason:  ev.Reason,
				Message: ev.Message,
				Count:   ev.Count,
			}
			if !ev.FirstTimestamp.IsZero() {
				t := ev.FirstTimestamp.Time.UTC().Format("2006-01-02T15:04:05Z")
				item.FirstTimestamp = &t
			}
			if !ev.LastTimestamp.IsZero() {
				t := ev.LastTimestamp.Time.UTC().Format("2006-01-02T15:04:05Z")
				item.LastTimestamp = &t
			}
			diag.Events = append(diag.Events, item)
		}
	}

	return diag, nil
}

func containerStateString(state corev1.ContainerState) string {
	switch {
	case state.Running != nil:
		return "Running"
	case state.Waiting != nil:
		return "Waiting"
	case state.Terminated != nil:
		return "Terminated"
	default:
		return ""
	}
}
