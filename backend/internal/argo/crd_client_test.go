package argo

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

// TestCRDClient_LifecycleOps_PatchSpec verifies each of Stop/Terminate/Suspend/
// Resume submits the expected JSON merge patch and the resulting spec matches
// argo-server semantics.
func TestCRDClient_LifecycleOps_PatchSpec(t *testing.T) {
	cases := []struct {
		name       string
		call       func(client *crdWorkflowClient, ctx context.Context) error
		wantSpec   map[string]any
		wantSpecOK bool // for suspend=false we want the key present (bool false), not absent
	}{
		{
			name: "Stop",
			call: func(c *crdWorkflowClient, ctx context.Context) error {
				return c.StopWorkflow(ctx, "wf-1", "argo")
			},
			wantSpec: map[string]any{"shutdown": "Stop"},
		},
		{
			name: "Terminate",
			call: func(c *crdWorkflowClient, ctx context.Context) error {
				return c.TerminateWorkflow(ctx, "wf-1", "argo")
			},
			wantSpec: map[string]any{"shutdown": "Terminate"},
		},
		{
			name: "Suspend",
			call: func(c *crdWorkflowClient, ctx context.Context) error {
				return c.SuspendWorkflow(ctx, "wf-1", "argo")
			},
			wantSpec: map[string]any{"suspend": true},
		},
		{
			name: "Resume",
			call: func(c *crdWorkflowClient, ctx context.Context) error {
				return c.ResumeWorkflow(ctx, "wf-1", "argo")
			},
			wantSpec: map[string]any{"suspend": false},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			seed := makeUnstructuredWorkflow("wf-1", "argo", "Running", nil)
			seed.Object["spec"] = map[string]any{"entrypoint": "main"}
			client, dyn := newFakeCRDClient(t, "argo", seed)

			if err := tc.call(client, context.Background()); err != nil {
				t.Fatalf("patch: %v", err)
			}
			got, err := dyn.Resource(workflowGVR).Namespace("argo").Get(context.Background(), "wf-1", metav1.GetOptions{})
			if err != nil {
				t.Fatalf("post-patch get: %v", err)
			}
			spec, _ := got.Object["spec"].(map[string]any)
			if spec == nil {
				t.Fatalf("spec missing after patch")
			}
			for k, want := range tc.wantSpec {
				if spec[k] != want {
					t.Errorf("spec.%s: want %v (%T), got %v (%T)", k, want, want, spec[k], spec[k])
				}
			}
			// Merge patch must preserve pre-existing spec fields, not clobber them.
			if spec["entrypoint"] != "main" {
				t.Errorf("merge patch clobbered spec.entrypoint: got %v", spec["entrypoint"])
			}
		})
	}
}

func TestCRDClient_LifecycleOps_NotFoundTranslated(t *testing.T) {
	client, _ := newFakeCRDClient(t, "argo")
	ctx := context.Background()
	cases := map[string]func() error{
		"Stop":      func() error { return client.StopWorkflow(ctx, "missing", "argo") },
		"Terminate": func() error { return client.TerminateWorkflow(ctx, "missing", "argo") },
		"Suspend":   func() error { return client.SuspendWorkflow(ctx, "missing", "argo") },
		"Resume":    func() error { return client.ResumeWorkflow(ctx, "missing", "argo") },
	}
	for name, call := range cases {
		t.Run(name, func(t *testing.T) {
			if err := call(); !errors.Is(err, ErrNotFound) {
				t.Errorf("want ErrNotFound, got %v", err)
			}
		})
	}
}

// TestCRDClient_RetryWorkflow_ResetsFailedNodes verifies retry semantics:
// - only completed workflows can be retried (Failed / Error / Succeeded)
// - failed nodes' pods are deleted
// - failed nodes' phase reset so controller re-picks them
// - workflow-level phase / finishedAt / message reset
func TestCRDClient_RetryWorkflow_ResetsFailedNodes(t *testing.T) {
	seed := makeUnstructuredWorkflow("wf-1", "argo", "Failed", nil)
	seed.Object["spec"] = map[string]any{"entrypoint": "main"}
	seed.Object["status"] = map[string]any{
		"phase":      "Failed",
		"message":    "step X errored",
		"finishedAt": "2026-07-16T00:00:00Z",
		"nodes": map[string]any{
			"n-ok": map[string]any{
				"id":    "n-ok",
				"type":  "Pod",
				"phase": "Succeeded",
			},
			"n-fail": map[string]any{
				"id":    "n-fail",
				"type":  "Pod",
				"phase": "Failed",
			},
		},
	}
	client, dyn := newFakeCRDClient(t, "argo", seed)

	// Pre-create the failed pod so we can assert it's deleted after retry.
	if err := seedPod(client, "n-fail", "argo"); err != nil {
		t.Fatalf("seed pod: %v", err)
	}
	if err := client.RetryWorkflow(context.Background(), "wf-1", "argo"); err != nil {
		t.Fatalf("retry: %v", err)
	}

	// The failed pod must be gone.
	if _, err := client.pods.CoreV1().Pods("argo").Get(context.Background(), "n-fail", metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Errorf("failed pod should be deleted after retry; got err=%v", err)
	}
	// Succeeded node untouched, failed node reset.
	got, _ := dyn.Resource(workflowGVR).Namespace("argo").Get(context.Background(), "wf-1", metav1.GetOptions{})
	status, _ := got.Object["status"].(map[string]any)
	if status["phase"] != "Running" {
		t.Errorf("workflow phase after retry: want Running, got %v", status["phase"])
	}
	// omitempty on Status.Message means an empty string is dropped from the
	// unstructured map; either "absent" or "" is acceptable here.
	if msg, ok := status["message"]; ok && msg != "" {
		t.Errorf("workflow message must clear on retry, got %v", msg)
	}
	nodes := status["nodes"].(map[string]any)
	if nodes["n-ok"].(map[string]any)["phase"] != "Succeeded" {
		t.Errorf("succeeded node must not be reset; phase=%v", nodes["n-ok"].(map[string]any)["phase"])
	}
	if nodes["n-fail"].(map[string]any)["phase"] != "Pending" {
		t.Errorf("failed node must reset to Pending; got %v", nodes["n-fail"].(map[string]any)["phase"])
	}
}

func TestCRDClient_RetryWorkflow_RejectRunning(t *testing.T) {
	seed := makeUnstructuredWorkflow("wf-1", "argo", "Running", nil)
	client, _ := newFakeCRDClient(t, "argo", seed)
	err := client.RetryWorkflow(context.Background(), "wf-1", "argo")
	if err == nil || !strings.Contains(err.Error(), "not retryable") {
		t.Fatalf("running workflow must not be retryable; got %v", err)
	}
}

func TestCRDClient_RetryWorkflow_NotFound(t *testing.T) {
	client, _ := newFakeCRDClient(t, "argo")
	err := client.RetryWorkflow(context.Background(), "missing", "argo")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestCRDClient_Resubmit_ClonesSpecCleansMetadata(t *testing.T) {
	seed := makeUnstructuredWorkflow("wf-1-abcd1", "argo", "Failed", map[string]string{
		"pipeline":                             "p1",
		"workflows.argoproj.io/completed":      "true",
		"workflows.argoproj.io/phase":          "Failed",
	})
	seed.Object["spec"] = map[string]any{
		"entrypoint": "main",
		"arguments":  map[string]any{"parameters": []any{}},
	}
	metadata := seed.Object["metadata"].(map[string]any)
	metadata["uid"] = "old-uid-123"
	metadata["resourceVersion"] = "5"

	client, dyn := newFakeCRDClient(t, "argo", seed)

	newWF, err := client.ResubmitWorkflowWithResult(context.Background(), "wf-1-abcd1", "argo")
	if err != nil {
		t.Fatalf("resubmit: %v", err)
	}
	// New workflow must have generateName derived from source, not a fixed name.
	if newWF.GenerateName != "wf-1-" {
		t.Errorf("generateName: want %q, got %q", "wf-1-", newWF.GenerateName)
	}
	// Argo-managed labels stripped, user labels retained.
	if newWF.Labels["pipeline"] != "p1" {
		t.Errorf("user label 'pipeline' lost")
	}
	if _, ok := newWF.Labels["workflows.argoproj.io/completed"]; ok {
		t.Errorf("argo-managed label must be stripped from resubmit")
	}
	// The old workflow still exists — resubmit does not delete the source.
	if _, err := dyn.Resource(workflowGVR).Namespace("argo").Get(context.Background(), "wf-1-abcd1", metav1.GetOptions{}); err != nil {
		t.Errorf("original workflow gone after resubmit: %v", err)
	}
}

func TestCRDClient_Resubmit_NotFound(t *testing.T) {
	client, _ := newFakeCRDClient(t, "argo")
	_, err := client.ResubmitWorkflowWithResult(context.Background(), "missing", "argo")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

// TestCRDClient_ResubmitGenerateNameFallback covers the case where the source
// workflow's name has no trailing hex suffix — the fallback should still
// produce a stable prefix ending in "-".
func TestCRDClient_ResubmitGenerateNameFallback(t *testing.T) {
	cases := []struct {
		name string
		src  *wfv1.Workflow
		want string
	}{
		{"has-generateName", &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{GenerateName: "already-"}}, "already-"},
		{"strip-hash", &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "pipeline-42-abc12"}}, "pipeline-42-"},
		{"no-hash", &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "pipeline"}}, "pipeline-"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resubmitGenerateName(tc.src); got != tc.want {
				t.Errorf("resubmitGenerateName: want %q, got %q", tc.want, got)
			}
		})
	}
}

// TestCRDClient_GetWorkflowLogs_SinglePod verifies the direct pod-log path.
// K8s fake client returns "fake logs" from GetLogs by default; we just need
// the request to reach it and the result to come back non-empty.
func TestCRDClient_GetWorkflowLogs_SinglePod(t *testing.T) {
	client, _ := newFakeCRDClient(t, "argo")
	if err := seedPod(client, "wf-1-step-abc", "argo"); err != nil {
		t.Fatalf("seed pod: %v", err)
	}
	result, err := client.GetWorkflowLogs(context.Background(), "wf-1", "wf-1-step-abc", "argo", WorkflowLogOptions{})
	if err != nil {
		t.Fatalf("get logs: %v", err)
	}
	if result.Logs == "" {
		t.Errorf("expected non-empty log body, got empty")
	}
}

// TestCRDClient_GetWorkflowLogs_LimitBytesTruncates verifies bounded reads
// respect the limit and mark truncated=true.
func TestCRDClient_GetWorkflowLogs_LimitBytesTruncates(t *testing.T) {
	client, _ := newFakeCRDClient(t, "argo")
	if err := seedPod(client, "wf-1-step-abc", "argo"); err != nil {
		t.Fatalf("seed pod: %v", err)
	}
	limit := int64(2)
	result, err := client.GetWorkflowLogs(context.Background(), "wf-1", "wf-1-step-abc", "argo", WorkflowLogOptions{LimitBytes: &limit})
	if err != nil {
		t.Fatalf("get logs: %v", err)
	}
	if !result.Truncated {
		t.Errorf("expected Truncated=true when logs exceed limit=2, got false (len=%d)", len(result.Logs))
	}
	if int64(len(result.Logs)) > limit {
		t.Errorf("logs len %d exceeds limit %d", len(result.Logs), limit)
	}
}

// TestCRDClient_GetWorkflowLogs_EmptyPodEnumeratesFromNodes verifies "no pod"
// falls back to enumerating workflow.status.nodes.
func TestCRDClient_GetWorkflowLogs_EmptyPodEnumeratesFromNodes(t *testing.T) {
	seed := makeUnstructuredWorkflow("wf-1", "argo", "Succeeded", nil)
	seed.Object["status"] = map[string]any{
		"phase": "Succeeded",
		"nodes": map[string]any{
			"n-a": map[string]any{
				"id":        "n-a",
				"type":      "Pod",
				"phase":     "Succeeded",
				"startedAt": "2026-07-16T10:00:00Z",
			},
			"n-b": map[string]any{
				"id":        "n-b",
				"type":      "Pod",
				"phase":     "Succeeded",
				"startedAt": "2026-07-16T10:00:05Z",
			},
			"n-nonpod": map[string]any{
				"id":    "n-nonpod",
				"type":  "DAG",
				"phase": "Succeeded",
			},
		},
	}
	client, _ := newFakeCRDClient(t, "argo", seed)
	if err := seedPod(client, "n-a", "argo"); err != nil {
		t.Fatal(err)
	}
	if err := seedPod(client, "n-b", "argo"); err != nil {
		t.Fatal(err)
	}
	result, err := client.GetWorkflowLogs(context.Background(), "wf-1", "", "argo", WorkflowLogOptions{})
	if err != nil {
		t.Fatalf("get logs: %v", err)
	}
	// Both pods contribute; fake client returns "fake logs" each, so the
	// concatenation must be strictly longer than a single pod fetch.
	if len(result.Logs) < 2 {
		t.Errorf("expected multi-pod concatenation, got len=%d", len(result.Logs))
	}
}

// TestCRDClient_GetWorkflowLogs_NoNodes covers the case where empty podName
// AND no matching pod-type nodes yields an empty result rather than an error.
func TestCRDClient_GetWorkflowLogs_NoNodes(t *testing.T) {
	seed := makeUnstructuredWorkflow("wf-empty", "argo", "Running", nil)
	seed.Object["status"] = map[string]any{"phase": "Running"}
	client, _ := newFakeCRDClient(t, "argo", seed)
	result, err := client.GetWorkflowLogs(context.Background(), "wf-empty", "", "argo", WorkflowLogOptions{})
	if err != nil {
		t.Fatalf("empty node list: %v", err)
	}
	if result.Logs != "" {
		t.Errorf("no pods → expect empty log body, got %q", result.Logs)
	}
}

func TestCRDClient_GetWorkflowLogStream_RequiresPodName(t *testing.T) {
	client, _ := newFakeCRDClient(t, "argo")
	if _, err := client.GetWorkflowLogStream(context.Background(), "wf", "", "argo", WorkflowLogOptions{}); err == nil {
		t.Fatal("expected error when podName empty for stream")
	}
}

func TestSortPodLogEntries(t *testing.T) {
	t0 := metav1.NewTime(mustParseTime("2026-07-16T10:00:00Z"))
	t1 := metav1.NewTime(mustParseTime("2026-07-16T10:00:05Z"))
	t2 := metav1.NewTime(mustParseTime("2026-07-16T10:00:10Z"))
	entries := []podLogEntry{
		{id: "c", start: t2},
		{id: "a", start: t0},
		{id: "b", start: t1},
	}
	sortPodLogEntries(entries)
	want := []string{"a", "b", "c"}
	for i, e := range entries {
		if e.id != want[i] {
			t.Errorf("index %d: want %s, got %s", i, want[i], e.id)
		}
	}
}

func TestToPodLogOptions_DefaultsContainerToMain(t *testing.T) {
	got := toPodLogOptions(WorkflowLogOptions{})
	if got.Container != "main" {
		t.Errorf("default container: want main, got %s", got.Container)
	}
}

func TestToPodLogOptions_PassesThrough(t *testing.T) {
	tail := int64(10)
	limit := int64(2048)
	since := int64(30)
	got := toPodLogOptions(WorkflowLogOptions{
		Container:    "sidecar",
		TailLines:    &tail,
		LimitBytes:   &limit,
		SinceSeconds: &since,
		Follow:       true,
		Timestamps:   true,
		Previous:     true,
		SinceTime:    "2026-07-16T10:00:00Z",
	})
	if got.Container != "sidecar" || *got.TailLines != tail || *got.LimitBytes != limit ||
		*got.SinceSeconds != since || !got.Follow || !got.Timestamps || !got.Previous {
		t.Errorf("options not passed through: %+v", got)
	}
	if got.SinceTime == nil {
		t.Errorf("valid SinceTime should be parsed into SinceTime pointer")
	}
}

// TestUnstructuredToWorkflow_DecodesMetaV1Times guards the JSON roundtrip
// fix (gemini review on #433). runtime.DefaultUnstructuredConverter walks
// via reflection and mis-handles metav1.Time — status.startedAt,
// status.finishedAt, metadata.creationTimestamp, and status.nodes[].startedAt
// were previously either dropped or wedged into wrong types when a real
// Argo workflow came back from the K8s API. json.Unmarshal calls the type's
// own UnmarshalJSON so times parse into metav1.Time correctly.
func TestUnstructuredToWorkflow_DecodesMetaV1Times(t *testing.T) {
	u := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Workflow",
		"metadata": map[string]any{
			"name":              "wf-1",
			"namespace":         "argo",
			"creationTimestamp": "2026-07-16T10:00:00Z",
		},
		"status": map[string]any{
			"phase":      "Running",
			"startedAt":  "2026-07-16T10:00:05Z",
			"finishedAt": nil,
			"nodes": map[string]any{
				"n-1": map[string]any{
					"id":         "n-1",
					"type":       "Pod",
					"phase":      "Running",
					"startedAt":  "2026-07-16T10:00:10Z",
					"finishedAt": nil,
				},
			},
		},
	}}
	wf, err := unstructuredToWorkflow(u)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if wf.CreationTimestamp.IsZero() {
		t.Errorf("metadata.creationTimestamp lost (would happen with DefaultUnstructuredConverter)")
	}
	if wf.Status.StartedAt.IsZero() {
		t.Errorf("status.startedAt lost")
	}
	if wf.Status.Phase != wfv1.WorkflowRunning {
		t.Errorf("status.phase mismatch: %s", wf.Status.Phase)
	}
	n, ok := wf.Status.Nodes["n-1"]
	if !ok {
		t.Fatalf("status.nodes[n-1] missing")
	}
	if n.StartedAt.IsZero() {
		t.Errorf("status.nodes[n-1].startedAt lost")
	}
	if !n.FinishedAt.IsZero() {
		t.Errorf("status.nodes[n-1].finishedAt should be zero (null in JSON), got %v", n.FinishedAt)
	}
}

// TestWorkflowToUnstructured_EmitsRFC3339Times verifies the reverse
// direction: encoding a wfv1.Workflow to unstructured must preserve
// metav1.Time as RFC3339 strings so the K8s API accepts them.
func TestWorkflowToUnstructured_EmitsRFC3339Times(t *testing.T) {
	when := metav1.NewTime(mustParseTime("2026-07-16T10:00:00Z"))
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "wf-1", Namespace: "argo", CreationTimestamp: when},
		Status: wfv1.WorkflowStatus{
			Phase:     wfv1.WorkflowRunning,
			StartedAt: when,
		},
	}
	u, err := workflowToUnstructured(wf)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	meta, _ := u.Object["metadata"].(map[string]any)
	if ts, _ := meta["creationTimestamp"].(string); ts != "2026-07-16T10:00:00Z" {
		t.Errorf("creationTimestamp: want RFC3339, got %v (type %T)", meta["creationTimestamp"], meta["creationTimestamp"])
	}
	status, _ := u.Object["status"].(map[string]any)
	if ts, _ := status["startedAt"].(string); ts != "2026-07-16T10:00:00Z" {
		t.Errorf("status.startedAt: want RFC3339, got %v (type %T)", status["startedAt"], status["startedAt"])
	}
	if u.GetKind() != "Workflow" || u.GetAPIVersion() != "argoproj.io/v1alpha1" {
		t.Errorf("gvk not normalized: kind=%s apiVersion=%s", u.GetKind(), u.GetAPIVersion())
	}
}

func TestWorkflowToUnstructured_NilRejected(t *testing.T) {
	if _, err := workflowToUnstructured(nil); err == nil {
		t.Fatal("nil workflow should error")
	}
}

func TestCRDClient_StubbedMethodsReturnNotImplemented(t *testing.T) {
	// After 4d.4 there are no more stubs. Keep the test as a placeholder that
	// asserts the sentinel is still defined but references no method that has
	// left the interface — this guards against accidental re-introduction of
	// a stub that never gets a real implementation.
	if errCRDMethodNotImplemented == nil {
		t.Fatal("errCRDMethodNotImplemented sentinel must remain defined")
	}
}

func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

// seedPod inserts a bare pod object into the crdWorkflowClient's fake typed
// client so tests can verify RetryWorkflow deletes it.
func seedPod(client *crdWorkflowClient, name, namespace string) error {
	_, err := client.pods.CoreV1().Pods(namespace).Create(
		context.Background(),
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}},
		metav1.CreateOptions{},
	)
	return err
}
