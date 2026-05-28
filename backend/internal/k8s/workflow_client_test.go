package k8s

import (
	"context"
	"testing"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	argowffake "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func TestArgoClient_CreateGetDeleteWorkflow(t *testing.T) {
	fakeClientset := argowffake.NewSimpleClientset()
	client := NewArgoClient(fakeClientset, k8sfake.NewSimpleClientset())
	ctx := context.Background()
	ns := "default"

	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-wf",
			Namespace: ns,
		},
		Spec: wfv1.WorkflowSpec{
			Entrypoint: "main",
			Templates: []wfv1.Template{
				{
					Name: "main",
					Container: &corev1.Container{
						Image: "alpine:latest",
					},
				},
			},
		},
	}

	// Create the workflow.
	if err := client.CreateWorkflow(ctx, wf, ns); err != nil {
		t.Fatalf("CreateWorkflow failed: %v", err)
	}
	t.Log("Workflow created successfully")

	// Verify initial status (fake returns empty phase).
	phase, err := client.GetWorkflowStatus(ctx, "test-wf", ns)
	if err != nil {
		t.Fatalf("GetWorkflowStatus failed: %v", err)
	}
	t.Logf("Workflow phase after creation: %q", phase)

	// Delete the workflow.
	if err := client.DeleteWorkflow(ctx, "test-wf", ns); err != nil {
		t.Fatalf("DeleteWorkflow failed: %v", err)
	}
	t.Log("Workflow deleted successfully")

	// Verify the workflow is gone.
	_, err = client.GetWorkflowStatus(ctx, "test-wf", ns)
	if err == nil {
		t.Fatal("expected error when getting deleted workflow, got nil")
	}
	t.Logf("Expected error after deletion: %v", err)
}

func TestArgoClient_ListWorkflows(t *testing.T) {
	fakeClientset := argowffake.NewSimpleClientset()
	client := NewArgoClient(fakeClientset, k8sfake.NewSimpleClientset())
	ctx := context.Background()
	ns := "default"

	// Create two workflows with different labels.
	wf1 := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "wf-a",
			Namespace: ns,
			Labels:    map[string]string{"app": "test", "env": "a"},
		},
		Spec: wfv1.WorkflowSpec{
			Entrypoint: "main",
			Templates: []wfv1.Template{
				{Name: "main", Container: &corev1.Container{Image: "alpine:latest"}},
			},
		},
	}
	wf2 := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "wf-b",
			Namespace: ns,
			Labels:    map[string]string{"app": "test", "env": "b"},
		},
		Spec: wfv1.WorkflowSpec{
			Entrypoint: "main",
			Templates: []wfv1.Template{
				{Name: "main", Container: &corev1.Container{Image: "alpine:latest"}},
			},
		},
	}

	for _, wf := range []*wfv1.Workflow{wf1, wf2} {
		if err := client.CreateWorkflow(ctx, wf, ns); err != nil {
			t.Fatalf("CreateWorkflow(%s) failed: %v", wf.Name, err)
		}
	}

	t.Run("list all workflows in namespace", func(t *testing.T) {
		list, err := client.ListWorkflows(ctx, ns, "")
		if err != nil {
			t.Fatalf("ListWorkflows failed: %v", err)
		}
		if len(list) != 2 {
			t.Fatalf("expected 2 workflows, got %d", len(list))
		}
	})

	t.Run("list workflows with label selector", func(t *testing.T) {
		list, err := client.ListWorkflows(ctx, ns, "env=a")
		if err != nil {
			t.Fatalf("ListWorkflows with selector failed: %v", err)
		}
		if len(list) != 1 {
			t.Fatalf("expected 1 workflow matching 'env=a', got %d", len(list))
		}
		if list[0].Name != "wf-a" {
			t.Fatalf("expected 'wf-a', got %q", list[0].Name)
		}
	})
}
