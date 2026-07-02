package argo

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	workflowapi "github.com/CyberOrigin2077/cyber-databrew/internal/argo"
	"github.com/CyberOrigin2077/cyber-databrew/internal/runtimeos/adapter"
)

const RuntimeType = "argo"

type workflowClient interface {
	CreateWorkflow(ctx context.Context, wf *wfv1.Workflow, namespace string) error
	GetWorkflow(ctx context.Context, name, namespace string) (*wfv1.Workflow, error)
	StopWorkflow(ctx context.Context, name, namespace string) error
	RetryWorkflow(ctx context.Context, name, namespace string) error
	ResubmitWorkflow(ctx context.Context, name, namespace string) error
	SuspendWorkflow(ctx context.Context, name, namespace string) error
	ResumeWorkflow(ctx context.Context, name, namespace string) error
	TerminateWorkflow(ctx context.Context, name, namespace string) error
	GetWorkflowLogs(ctx context.Context, workflowName, podName, namespace string, opts workflowapi.WorkflowLogOptions) (workflowapi.WorkflowLogResult, error)
	GetWorkflowLogStream(ctx context.Context, workflowName, podName, namespace string, opts workflowapi.WorkflowLogOptions) (io.ReadCloser, error)
}

type resubmitResultClient interface {
	ResubmitWorkflowWithResult(ctx context.Context, name, namespace string) (*wfv1.Workflow, error)
}

// Adapter implements RuntimeAdapter for Argo Workflows.
type Adapter struct {
	client           workflowClient
	defaultNamespace string
}

func New(client workflowClient, defaultNamespace string) *Adapter {
	return &Adapter{
		client:           client,
		defaultNamespace: strings.TrimSpace(defaultNamespace),
	}
}

var _ adapter.RuntimeAdapter = (*Adapter)(nil)

func (a *Adapter) Submit(ctx context.Context, _ adapter.RunRef, spec adapter.RuntimeSpec) (*adapter.RuntimeJob, error) {
	wf, err := workflowFromSpec(spec)
	if err != nil {
		return nil, err
	}
	namespace := a.namespace(firstNonEmpty(spec.Namespace, wf.Namespace))
	if err := a.client.CreateWorkflow(ctx, wf, namespace); err != nil {
		return nil, err
	}
	return runtimeJobFromWorkflow(wf, namespace), nil
}

func (a *Adapter) Get(ctx context.Context, ref adapter.RuntimeRef) (*adapter.RuntimeJobStatus, error) {
	name, namespace, err := a.nameNamespace(ref)
	if err != nil {
		return nil, err
	}
	wf, err := a.client.GetWorkflow(ctx, name, namespace)
	if err != nil {
		return nil, err
	}
	status := &adapter.RuntimeJobStatus{
		Ref:        runtimeRefFromWorkflow(wf, namespace),
		Status:     string(wf.Status.Phase),
		Message:    strings.TrimSpace(wf.Status.Message),
		StartedAt:  timePtr(wf.Status.StartedAt),
		FinishedAt: timePtr(wf.Status.FinishedAt),
		Raw:        wf,
	}
	return status, nil
}

func (a *Adapter) Stop(ctx context.Context, ref adapter.RuntimeRef) error {
	return a.workflowOperation(ctx, ref, a.client.StopWorkflow)
}

func (a *Adapter) Suspend(ctx context.Context, ref adapter.RuntimeRef) error {
	return a.workflowOperation(ctx, ref, a.client.SuspendWorkflow)
}

func (a *Adapter) Resume(ctx context.Context, ref adapter.RuntimeRef) error {
	return a.workflowOperation(ctx, ref, a.client.ResumeWorkflow)
}

func (a *Adapter) Terminate(ctx context.Context, ref adapter.RuntimeRef) error {
	return a.workflowOperation(ctx, ref, a.client.TerminateWorkflow)
}

func (a *Adapter) Retry(ctx context.Context, ref adapter.RuntimeRef, _ adapter.RetryOptions) (*adapter.RuntimeJob, error) {
	name, namespace, err := a.nameNamespace(ref)
	if err != nil {
		return nil, err
	}
	if err := a.client.RetryWorkflow(ctx, name, namespace); err != nil {
		return nil, err
	}
	return &adapter.RuntimeJob{Ref: refWithDefaults(ref, name, namespace)}, nil
}

func (a *Adapter) Resubmit(ctx context.Context, ref adapter.RuntimeRef, _ adapter.ResubmitOptions) (*adapter.RuntimeJob, error) {
	name, namespace, err := a.nameNamespace(ref)
	if err != nil {
		return nil, err
	}
	if client, ok := a.client.(resubmitResultClient); ok {
		wf, err := client.ResubmitWorkflowWithResult(ctx, name, namespace)
		if err != nil {
			return nil, err
		}
		return runtimeJobFromWorkflow(wf, namespace), nil
	}
	if err := a.client.ResubmitWorkflow(ctx, name, namespace); err != nil {
		return nil, err
	}
	return &adapter.RuntimeJob{Ref: refWithDefaults(ref, name, namespace)}, nil
}

func (a *Adapter) Logs(ctx context.Context, ref adapter.RuntimeRef, nodeID string, opts adapter.LogOptions) (*adapter.LogResult, error) {
	name, namespace, err := a.nameNamespace(ref)
	if err != nil {
		return nil, err
	}
	result, err := a.client.GetWorkflowLogs(ctx, name, nodeID, namespace, toWorkflowLogOptions(opts))
	if err != nil {
		return nil, err
	}
	return &adapter.LogResult{
		Logs:       result.Logs,
		LineCount:  result.LineCount,
		Truncated:  result.Truncated,
		LimitBytes: result.LimitBytes,
	}, nil
}

func (a *Adapter) LogStream(ctx context.Context, ref adapter.RuntimeRef, nodeID string, opts adapter.LogOptions) (io.ReadCloser, error) {
	name, namespace, err := a.nameNamespace(ref)
	if err != nil {
		return nil, err
	}
	opts.Follow = true
	return a.client.GetWorkflowLogStream(ctx, name, nodeID, namespace, toWorkflowLogOptions(opts))
}

func (a *Adapter) workflowOperation(ctx context.Context, ref adapter.RuntimeRef, op func(context.Context, string, string) error) error {
	name, namespace, err := a.nameNamespace(ref)
	if err != nil {
		return err
	}
	return op(ctx, name, namespace)
}

func (a *Adapter) nameNamespace(ref adapter.RuntimeRef) (string, string, error) {
	name := strings.TrimSpace(ref.Name)
	if name == "" {
		return "", "", fmt.Errorf("runtime ref name is required")
	}
	return name, a.namespace(ref.Namespace), nil
}

func (a *Adapter) namespace(namespace string) string {
	namespace = strings.TrimSpace(namespace)
	if namespace != "" {
		return namespace
	}
	return a.defaultNamespace
}

func workflowFromSpec(spec adapter.RuntimeSpec) (*wfv1.Workflow, error) {
	switch wf := spec.Manifest.(type) {
	case *wfv1.Workflow:
		if wf == nil {
			return nil, fmt.Errorf("argo runtime spec workflow is nil")
		}
		return wf, nil
	case wfv1.Workflow:
		return &wf, nil
	default:
		return nil, fmt.Errorf("argo runtime spec requires workflow manifest")
	}
}

func runtimeJobFromWorkflow(wf *wfv1.Workflow, namespace string) *adapter.RuntimeJob {
	return &adapter.RuntimeJob{
		Ref: runtimeRefFromWorkflow(wf, namespace),
		Raw: wf,
	}
}

func runtimeRefFromWorkflow(wf *wfv1.Workflow, namespace string) adapter.RuntimeRef {
	if wf == nil {
		return adapter.RuntimeRef{RuntimeType: RuntimeType, Namespace: namespace}
	}
	return adapter.RuntimeRef{
		RuntimeType: RuntimeType,
		Name:        wf.Name,
		Namespace:   firstNonEmpty(wf.Namespace, namespace),
		UID:         string(wf.UID),
	}
}

func refWithDefaults(ref adapter.RuntimeRef, name, namespace string) adapter.RuntimeRef {
	ref.RuntimeType = firstNonEmpty(ref.RuntimeType, RuntimeType)
	ref.Name = firstNonEmpty(ref.Name, name)
	ref.Namespace = firstNonEmpty(ref.Namespace, namespace)
	return ref
}

func toWorkflowLogOptions(opts adapter.LogOptions) workflowapi.WorkflowLogOptions {
	return workflowapi.WorkflowLogOptions{
		Container:    opts.Container,
		TailLines:    opts.TailLines,
		LimitBytes:   opts.LimitBytes,
		SinceSeconds: opts.SinceSeconds,
		SinceTime:    opts.SinceTime,
		Previous:     opts.Previous,
		Timestamps:   opts.Timestamps,
		Follow:       opts.Follow,
	}
}

func timePtr(value metav1.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	t := value.Time
	return &t
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
