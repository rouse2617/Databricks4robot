package k8s

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetPodDiagnostics(t *testing.T) {
	now := metav1.Now()
	clientset := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wf-step-a",
				Namespace: "cyber-databrew-dev",
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: "workflow-sa",
			},
			Status: corev1.PodStatus{
				PodIP: "10.2.3.4",
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name:         "main",
						Image:        "alpine:3.20",
						Ready:        true,
						RestartCount: 2,
						State: corev1.ContainerState{
							Running: &corev1.ContainerStateRunning{StartedAt: now},
						},
					},
				},
				Conditions: []corev1.PodCondition{
					{
						Type:               corev1.PodReady,
						Status:             corev1.ConditionTrue,
						Reason:             "ContainersReady",
						Message:            "ready",
						LastTransitionTime: now,
					},
				},
			},
		},
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wf-step-a.123",
				Namespace: "cyber-databrew-dev",
			},
			InvolvedObject: corev1.ObjectReference{
				Kind:      "Pod",
				Name:      "wf-step-a",
				Namespace: "cyber-databrew-dev",
			},
			Type:           "Normal",
			Reason:         "Pulled",
			Message:        "container image pulled",
			Count:          1,
			FirstTimestamp: now,
			LastTimestamp:  now,
		},
	)

	client := NewPodClientWithClientset(clientset, "dev-gke")
	diag, err := client.GetPodDiagnostics(context.Background(), "cyber-databrew-dev", "wf-step-a")
	if err != nil {
		t.Fatalf("GetPodDiagnostics: %v", err)
	}

	if diag.Cluster != "dev-gke" {
		t.Fatalf("expected cluster dev-gke, got %q", diag.Cluster)
	}
	if diag.Namespace != "cyber-databrew-dev" || diag.PodName != "wf-step-a" {
		t.Fatalf("unexpected pod identity %#v", diag)
	}
	if diag.RestartCount != 2 {
		t.Fatalf("expected restart count 2, got %d", diag.RestartCount)
	}
	if len(diag.Containers) != 1 || diag.Containers[0].State != "Running" {
		t.Fatalf("unexpected containers %#v", diag.Containers)
	}
	if len(diag.Conditions) != 1 || diag.Conditions[0].Type != "Ready" {
		t.Fatalf("unexpected conditions %#v", diag.Conditions)
	}
	if len(diag.Events) != 1 || diag.Events[0].Reason != "Pulled" {
		t.Fatalf("unexpected events %#v", diag.Events)
	}
}
