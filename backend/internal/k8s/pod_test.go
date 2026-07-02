package k8s

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
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

func TestGetPodDiagnostics_AllowsEventListFailure(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wf-step-a",
				Namespace: "cyber-databrew-dev",
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{Name: "main", Image: "alpine:3.20", Ready: true},
				},
			},
		},
	)
	clientset.Fake.PrependReactor("list", "events", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "events"}, "", nil)
	})

	client := NewPodClientWithClientset(clientset, "dev-gke")
	diag, err := client.GetPodDiagnostics(context.Background(), "cyber-databrew-dev", "wf-step-a")
	if err != nil {
		t.Fatalf("GetPodDiagnostics should not fail when events cannot be listed: %v", err)
	}

	if diag.PodName != "wf-step-a" || len(diag.Containers) != 1 {
		t.Fatalf("expected pod diagnostics without events, got %#v", diag)
	}
	if len(diag.Events) != 0 {
		t.Fatalf("expected empty events after list failure, got %#v", diag.Events)
	}
}
