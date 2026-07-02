package argo

import (
	"context"
	"io"
	"strings"
	"testing"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	workflowapi "github.com/CyberOrigin2077/cyber-databrew/internal/argo"
	"github.com/CyberOrigin2077/cyber-databrew/internal/runtimeos/adapter"
)

type fakeWorkflowClient struct {
	createdWorkflow *wfv1.Workflow
	createdNS       string
	gotName         string
	gotNS           string
	operations      []string
	logName         string
	logPod          string
	logNS           string
	logOpts         workflowapi.WorkflowLogOptions
	streamOpts      workflowapi.WorkflowLogOptions
	resubmitName    string
	resubmitNS      string
}

func (f *fakeWorkflowClient) CreateWorkflow(_ context.Context, wf *wfv1.Workflow, namespace string) error {
	f.createdWorkflow = wf
	f.createdNS = namespace
	return nil
}

func (f *fakeWorkflowClient) GetWorkflow(_ context.Context, name, namespace string) (*wfv1.Workflow, error) {
	f.gotName = name
	f.gotNS = namespace
	return &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, UID: types.UID("uid-1")},
		Status: wfv1.WorkflowStatus{
			Phase:   wfv1.WorkflowRunning,
			Message: "running pods",
		},
	}, nil
}

func (f *fakeWorkflowClient) StopWorkflow(_ context.Context, name, namespace string) error {
	f.operations = append(f.operations, "stop:"+namespace+"/"+name)
	return nil
}

func (f *fakeWorkflowClient) RetryWorkflow(_ context.Context, name, namespace string) error {
	f.operations = append(f.operations, "retry:"+namespace+"/"+name)
	return nil
}

func (f *fakeWorkflowClient) ResubmitWorkflow(_ context.Context, name, namespace string) error {
	f.operations = append(f.operations, "resubmit:"+namespace+"/"+name)
	return nil
}

func (f *fakeWorkflowClient) ResubmitWorkflowWithResult(_ context.Context, name, namespace string) (*wfv1.Workflow, error) {
	f.resubmitName = name
	f.resubmitNS = namespace
	return &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: name + "-resubmitted", Namespace: namespace, UID: types.UID("uid-2")},
	}, nil
}

func (f *fakeWorkflowClient) SuspendWorkflow(_ context.Context, name, namespace string) error {
	f.operations = append(f.operations, "suspend:"+namespace+"/"+name)
	return nil
}

func (f *fakeWorkflowClient) ResumeWorkflow(_ context.Context, name, namespace string) error {
	f.operations = append(f.operations, "resume:"+namespace+"/"+name)
	return nil
}

func (f *fakeWorkflowClient) TerminateWorkflow(_ context.Context, name, namespace string) error {
	f.operations = append(f.operations, "terminate:"+namespace+"/"+name)
	return nil
}

func (f *fakeWorkflowClient) GetWorkflowLogs(_ context.Context, workflowName, podName, namespace string, opts workflowapi.WorkflowLogOptions) (workflowapi.WorkflowLogResult, error) {
	f.logName = workflowName
	f.logPod = podName
	f.logNS = namespace
	f.logOpts = opts
	return workflowapi.WorkflowLogResult{Logs: "hello\n", LineCount: 1, LimitBytes: 1024}, nil
}

func (f *fakeWorkflowClient) GetWorkflowLogStream(_ context.Context, workflowName, podName, namespace string, opts workflowapi.WorkflowLogOptions) (io.ReadCloser, error) {
	f.logName = workflowName
	f.logPod = podName
	f.logNS = namespace
	f.streamOpts = opts
	return io.NopCloser(strings.NewReader("live\n")), nil
}

func TestAdapterSubmitAndGet(t *testing.T) {
	fake := &fakeWorkflowClient{}
	a := New(fake, "video-proc-dev")

	job, err := a.Submit(context.Background(), adapter.RunRef{ID: "run-1"}, adapter.RuntimeSpec{
		Manifest: &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "wf-1"}},
	})
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}
	if fake.createdWorkflow == nil || fake.createdWorkflow.Name != "wf-1" {
		t.Fatalf("workflow was not submitted: %#v", fake.createdWorkflow)
	}
	if fake.createdNS != "video-proc-dev" {
		t.Fatalf("unexpected submit namespace: %q", fake.createdNS)
	}
	if job.Ref.Name != "wf-1" || job.Ref.Namespace != "video-proc-dev" || job.Ref.RuntimeType != RuntimeType {
		t.Fatalf("unexpected runtime ref: %#v", job.Ref)
	}

	status, err := a.Get(context.Background(), adapter.RuntimeRef{Name: "wf-1"})
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if fake.gotName != "wf-1" || fake.gotNS != "video-proc-dev" {
		t.Fatalf("Get used wrong runtime identity: %s/%s", fake.gotNS, fake.gotName)
	}
	if status.Status != string(wfv1.WorkflowRunning) || status.Message != "running pods" || status.Ref.UID != "uid-1" {
		t.Fatalf("unexpected status: %#v", status)
	}
}

func TestAdapterRuntimeOperations(t *testing.T) {
	fake := &fakeWorkflowClient{}
	a := New(fake, "default-ns")
	ref := adapter.RuntimeRef{Name: "wf-ops", Namespace: "run-ns"}

	if err := a.Stop(context.Background(), ref); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if _, err := a.Retry(context.Background(), ref, adapter.RetryOptions{}); err != nil {
		t.Fatalf("Retry: %v", err)
	}
	if err := a.Suspend(context.Background(), ref); err != nil {
		t.Fatalf("Suspend: %v", err)
	}
	if err := a.Resume(context.Background(), ref); err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if err := a.Terminate(context.Background(), ref); err != nil {
		t.Fatalf("Terminate: %v", err)
	}
	want := []string{
		"stop:run-ns/wf-ops",
		"retry:run-ns/wf-ops",
		"suspend:run-ns/wf-ops",
		"resume:run-ns/wf-ops",
		"terminate:run-ns/wf-ops",
	}
	if strings.Join(fake.operations, ",") != strings.Join(want, ",") {
		t.Fatalf("unexpected operations: %#v", fake.operations)
	}
}

func TestAdapterResubmitReturnsNewWorkflowRef(t *testing.T) {
	fake := &fakeWorkflowClient{}
	a := New(fake, "default-ns")

	job, err := a.Resubmit(context.Background(), adapter.RuntimeRef{Name: "wf-source"}, adapter.ResubmitOptions{})
	if err != nil {
		t.Fatalf("Resubmit returned error: %v", err)
	}
	if fake.resubmitName != "wf-source" || fake.resubmitNS != "default-ns" {
		t.Fatalf("resubmit used wrong identity: %s/%s", fake.resubmitNS, fake.resubmitName)
	}
	if job.Ref.Name != "wf-source-resubmitted" || job.Ref.UID != "uid-2" {
		t.Fatalf("unexpected resubmitted job ref: %#v", job.Ref)
	}
}

func TestAdapterLogsMapOptions(t *testing.T) {
	fake := &fakeWorkflowClient{}
	a := New(fake, "default-ns")
	tail := int64(25)
	limit := int64(1024)

	result, err := a.Logs(context.Background(), adapter.RuntimeRef{Name: "wf-log"}, "pod-1", adapter.LogOptions{
		Container:  "main",
		TailLines:  &tail,
		LimitBytes: &limit,
		Timestamps: true,
	})
	if err != nil {
		t.Fatalf("Logs returned error: %v", err)
	}
	if result.LineCount != 1 || result.LimitBytes != 1024 {
		t.Fatalf("unexpected log result: %#v", result)
	}
	if fake.logName != "wf-log" || fake.logPod != "pod-1" || fake.logNS != "default-ns" {
		t.Fatalf("logs used wrong identity: %s/%s pod=%s", fake.logNS, fake.logName, fake.logPod)
	}
	if fake.logOpts.Container != "main" || fake.logOpts.TailLines == nil || *fake.logOpts.TailLines != 25 || !fake.logOpts.Timestamps {
		t.Fatalf("log options were not mapped: %#v", fake.logOpts)
	}

	stream, err := a.LogStream(context.Background(), adapter.RuntimeRef{Name: "wf-log"}, "pod-1", adapter.LogOptions{})
	if err != nil {
		t.Fatalf("LogStream returned error: %v", err)
	}
	defer stream.Close()
	if !fake.streamOpts.Follow {
		t.Fatalf("LogStream did not force follow")
	}
}

func TestAdapterRejectsInvalidRuntimeIdentity(t *testing.T) {
	a := New(&fakeWorkflowClient{}, "default-ns")

	if _, err := a.Submit(context.Background(), adapter.RunRef{}, adapter.RuntimeSpec{}); err == nil {
		t.Fatalf("Submit accepted missing workflow manifest")
	}
	if err := a.Stop(context.Background(), adapter.RuntimeRef{}); err == nil {
		t.Fatalf("Stop accepted empty runtime ref")
	}
}
