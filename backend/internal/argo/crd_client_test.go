package argo

import (
	"context"
	"errors"
	"testing"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

// newFakeCRDClient builds a crdWorkflowClient backed by a fake dynamic client
// that knows the argoproj.io Workflow GVR → WorkflowList mapping. Seed objects
// are handed in as unstructured Workflows so tests can populate the tracker
// without paying scheme-registration ceremony.
func newFakeCRDClient(t *testing.T, ns string, seed ...*unstructured.Unstructured) (*crdWorkflowClient, *dynamicfake.FakeDynamicClient) {
	t.Helper()
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: "argoproj.io", Version: "v1alpha1", Kind: "Workflow"}, &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: "argoproj.io", Version: "v1alpha1", Kind: "WorkflowList"}, &unstructured.UnstructuredList{})
	gvrToListKind := map[schema.GroupVersionResource]string{
		workflowGVR: "WorkflowList",
	}
	objs := make([]runtime.Object, 0, len(seed))
	for _, s := range seed {
		objs = append(objs, s)
	}
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind, objs...)
	pods := k8sfake.NewSimpleClientset()
	return newCRDWorkflowClient(dyn, pods, ns), dyn
}

func makeUnstructuredWorkflow(name, namespace, phase string, labels map[string]string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Workflow",
		"metadata": map[string]any{
			"name":      name,
			"namespace": namespace,
		},
	}}
	if len(labels) > 0 {
		labelMap := make(map[string]any, len(labels))
		for k, v := range labels {
			labelMap[k] = v
		}
		metadata := u.Object["metadata"].(map[string]any)
		metadata["labels"] = labelMap
	}
	if phase != "" {
		u.Object["status"] = map[string]any{"phase": phase}
	}
	return u
}

func TestCRDClient_CreateWorkflow_Success(t *testing.T) {
	client, dyn := newFakeCRDClient(t, "argo")
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "wf-1", Namespace: "argo"},
	}
	if err := client.CreateWorkflow(context.Background(), wf, "argo"); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := dyn.Resource(workflowGVR).Namespace("argo").Get(context.Background(), "wf-1", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("post-create get: %v", err)
	}
	if got.GetAPIVersion() != "argoproj.io/v1alpha1" || got.GetKind() != "Workflow" {
		t.Errorf("gvk not normalized: apiVersion=%s kind=%s", got.GetAPIVersion(), got.GetKind())
	}
}

func TestCRDClient_CreateWorkflow_NilWorkflow(t *testing.T) {
	client, _ := newFakeCRDClient(t, "argo")
	if err := client.CreateWorkflow(context.Background(), nil, "argo"); err == nil {
		t.Fatal("nil workflow should error")
	}
}

func TestCRDClient_CreateWorkflow_AlreadyExistsTranslated(t *testing.T) {
	seed := makeUnstructuredWorkflow("wf-dup", "argo", "", nil)
	client, _ := newFakeCRDClient(t, "argo", seed)
	err := client.CreateWorkflow(context.Background(), &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "wf-dup", Namespace: "argo"},
	}, "argo")
	if err == nil {
		t.Fatal("expected AlreadyExists error")
	}
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("want ErrAlreadyExists, got %v", err)
	}
}

func TestCRDClient_GetWorkflow_Success(t *testing.T) {
	seed := makeUnstructuredWorkflow("wf-1", "argo", "Running", nil)
	client, _ := newFakeCRDClient(t, "argo", seed)

	wf, err := client.GetWorkflow(context.Background(), "wf-1", "argo")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if wf.Name != "wf-1" {
		t.Errorf("name mismatch: %s", wf.Name)
	}
	if wf.Status.Phase != wfv1.WorkflowRunning {
		t.Errorf("phase mismatch: %s", wf.Status.Phase)
	}
}

func TestCRDClient_GetWorkflow_NotFoundTranslated(t *testing.T) {
	client, _ := newFakeCRDClient(t, "argo")
	_, err := client.GetWorkflow(context.Background(), "missing", "argo")
	if err == nil {
		t.Fatal("expected error for missing workflow")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestCRDClient_GetWorkflowStatus_ReturnsPhase(t *testing.T) {
	seed := makeUnstructuredWorkflow("wf-1", "argo", "Succeeded", nil)
	client, _ := newFakeCRDClient(t, "argo", seed)
	phase, err := client.GetWorkflowStatus(context.Background(), "wf-1", "argo")
	if err != nil {
		t.Fatalf("get status: %v", err)
	}
	if phase != wfv1.WorkflowSucceeded {
		t.Errorf("phase mismatch: %s", phase)
	}
}

func TestCRDClient_GetWorkflowStatus_NotFoundReturnsUnknown(t *testing.T) {
	client, _ := newFakeCRDClient(t, "argo")
	phase, err := client.GetWorkflowStatus(context.Background(), "missing", "argo")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	if phase != wfv1.WorkflowUnknown {
		t.Errorf("phase must be Unknown on not-found, got %s", phase)
	}
}

func TestCRDClient_DeleteWorkflow_Success(t *testing.T) {
	seed := makeUnstructuredWorkflow("wf-1", "argo", "Succeeded", nil)
	client, dyn := newFakeCRDClient(t, "argo", seed)
	if err := client.DeleteWorkflow(context.Background(), "wf-1", "argo"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := dyn.Resource(workflowGVR).Namespace("argo").Get(context.Background(), "wf-1", metav1.GetOptions{}); err == nil {
		t.Fatal("workflow should be gone after delete")
	}
}

func TestCRDClient_DeleteWorkflow_NotFoundTranslated(t *testing.T) {
	client, _ := newFakeCRDClient(t, "argo")
	err := client.DeleteWorkflow(context.Background(), "missing", "argo")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestCRDClient_ListWorkflows_HappyPath(t *testing.T) {
	seed1 := makeUnstructuredWorkflow("wf-1", "argo", "Running", map[string]string{"pipeline": "p1"})
	seed2 := makeUnstructuredWorkflow("wf-2", "argo", "Succeeded", map[string]string{"pipeline": "p2"})
	client, _ := newFakeCRDClient(t, "argo", seed1, seed2)

	list, err := client.ListWorkflows(context.Background(), "argo", "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("want 2 workflows, got %d", len(list))
	}
}

func TestCRDClient_ListWorkflows_EmptyNamespaceUsesDefault(t *testing.T) {
	seed := makeUnstructuredWorkflow("wf-1", "argo-default", "Running", nil)
	client, _ := newFakeCRDClient(t, "argo-default", seed)
	list, err := client.ListWorkflows(context.Background(), "", "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("empty ns should fall back to defaultNS 'argo-default'; got %d items", len(list))
	}
}

func TestCRDClient_StubbedMethodsReturnNotImplemented(t *testing.T) {
	client, _ := newFakeCRDClient(t, "argo")
	ctx := context.Background()
	cases := []struct {
		name string
		call func() error
	}{
		{"Stop", func() error { return client.StopWorkflow(ctx, "wf", "argo") }},
		{"Terminate", func() error { return client.TerminateWorkflow(ctx, "wf", "argo") }},
		{"Suspend", func() error { return client.SuspendWorkflow(ctx, "wf", "argo") }},
		{"Resume", func() error { return client.ResumeWorkflow(ctx, "wf", "argo") }},
		{"Retry", func() error { return client.RetryWorkflow(ctx, "wf", "argo") }},
		{"Resubmit", func() error { return client.ResubmitWorkflow(ctx, "wf", "argo") }},
		{"GetWorkflowLogs", func() error {
			_, err := client.GetWorkflowLogs(ctx, "wf", "pod", "argo", WorkflowLogOptions{})
			return err
		}},
		{"GetWorkflowLogStream", func() error {
			_, err := client.GetWorkflowLogStream(ctx, "wf", "pod", "argo", WorkflowLogOptions{})
			return err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.call(); !errors.Is(err, errCRDMethodNotImplemented) {
				t.Errorf("want errCRDMethodNotImplemented, got %v", err)
			}
		})
	}
}
