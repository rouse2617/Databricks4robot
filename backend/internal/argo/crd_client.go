package argo

import (
	"context"
	"errors"
	"fmt"
	"io"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// workflowGVR is the argoproj.io Workflow CRD GroupVersionResource. Argo v3.x
// stays on v1alpha1; a future v4 bump would need re-mapping (Non-goal per
// design doc D3).
var workflowGVR = schema.GroupVersionResource{
	Group:    "argoproj.io",
	Version:  "v1alpha1",
	Resource: "workflows",
}

// errCRDMethodNotImplemented is returned by crdWorkflowClient methods that
// belong to later PRs in the 4d series (lifecycle ops → 4d.2, retry/resubmit
// → 4d.3, log streaming → 4d.4). Kept as a sentinel so tests can assert the
// scope boundary explicitly.
var errCRDMethodNotImplemented = errors.New("argo crd client: method not implemented in PR 4d.1 skeleton")

// crdWorkflowClient talks to argo-workflows directly via the K8s Workflow CRD
// (argoproj.io/v1alpha1) using dynamic + typed clients handed in by
// k8s.ClientFactory. It exists so DataBrew can reach clusters that do not
// expose argo-server on a network the backend can dial — see design doc D1/D2
// in openspec/changes/CYB-3486d-argo-crd-mode/design.md.
type crdWorkflowClient struct {
	dyn       dynamic.Interface
	pods      kubernetes.Interface // reserved for 4d.4 log streaming; unused today
	defaultNS string               // fallback for List calls without explicit ns
}

// newCRDWorkflowClient constructs the CRD-mode client. Callers get it from
// argo.ClientFactory when the cluster row lacks a usable argo-server URL.
func newCRDWorkflowClient(dyn dynamic.Interface, pods kubernetes.Interface, defaultNS string) *crdWorkflowClient {
	return &crdWorkflowClient{dyn: dyn, pods: pods, defaultNS: defaultNS}
}

var _ WorkflowClient = (*crdWorkflowClient)(nil)

// CreateWorkflow posts a new Workflow CRD to the cluster. The workflow's
// TypeMeta is normalized to argoproj.io/v1alpha1 Workflow so callers can hand
// in transpiler output without setting those fields themselves.
func (c *crdWorkflowClient) CreateWorkflow(ctx context.Context, wf *wfv1.Workflow, namespace string) error {
	if wf == nil {
		return fmt.Errorf("workflow is nil")
	}
	ns := c.resolveNS(namespace)
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(wf)
	if err != nil {
		return fmt.Errorf("argo crd: encode workflow: %w", err)
	}
	u := &unstructured.Unstructured{Object: obj}
	u.SetGroupVersionKind(schema.GroupVersionKind{Group: "argoproj.io", Version: "v1alpha1", Kind: "Workflow"})
	if _, err := c.dyn.Resource(workflowGVR).Namespace(ns).Create(ctx, u, metav1.CreateOptions{}); err != nil {
		return translateK8sErr(err)
	}
	return nil
}

// GetWorkflow reads a Workflow CRD by name and decodes it into wfv1.Workflow
// so the rest of the backend sees the same type argo-server would return.
func (c *crdWorkflowClient) GetWorkflow(ctx context.Context, name, namespace string) (*wfv1.Workflow, error) {
	ns := c.resolveNS(namespace)
	u, err := c.dyn.Resource(workflowGVR).Namespace(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, translateK8sErr(err)
	}
	return unstructuredToWorkflow(u)
}

// GetWorkflowStatus is a convenience over GetWorkflow that extracts the phase.
func (c *crdWorkflowClient) GetWorkflowStatus(ctx context.Context, name, namespace string) (wfv1.WorkflowPhase, error) {
	wf, err := c.GetWorkflow(ctx, name, namespace)
	if err != nil {
		return wfv1.WorkflowUnknown, err
	}
	return wf.Status.Phase, nil
}

// DeleteWorkflow removes a Workflow CRD by name.
func (c *crdWorkflowClient) DeleteWorkflow(ctx context.Context, name, namespace string) error {
	ns := c.resolveNS(namespace)
	if err := c.dyn.Resource(workflowGVR).Namespace(ns).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return translateK8sErr(err)
	}
	return nil
}

// ListWorkflows lists Workflow CRDs in the namespace, filtered by label
// selector. Empty selector returns everything the caller can see.
func (c *crdWorkflowClient) ListWorkflows(ctx context.Context, namespace, labelSelector string) ([]wfv1.Workflow, error) {
	ns := c.resolveNS(namespace)
	list, err := c.dyn.Resource(workflowGVR).Namespace(ns).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, translateK8sErr(err)
	}
	out := make([]wfv1.Workflow, 0, len(list.Items))
	for i := range list.Items {
		wf, err := unstructuredToWorkflow(&list.Items[i])
		if err != nil {
			return nil, err
		}
		out = append(out, *wf)
	}
	return out, nil
}

// StopWorkflow is implemented in PR 4d.2 (spec.shutdown=Stop patch).
func (c *crdWorkflowClient) StopWorkflow(context.Context, string, string) error {
	return errCRDMethodNotImplemented
}

// TerminateWorkflow is implemented in PR 4d.2 (spec.shutdown=Terminate patch).
func (c *crdWorkflowClient) TerminateWorkflow(context.Context, string, string) error {
	return errCRDMethodNotImplemented
}

// SuspendWorkflow is implemented in PR 4d.2 (spec.suspend=true patch).
func (c *crdWorkflowClient) SuspendWorkflow(context.Context, string, string) error {
	return errCRDMethodNotImplemented
}

// ResumeWorkflow is implemented in PR 4d.2 (spec.suspend=false patch).
func (c *crdWorkflowClient) ResumeWorkflow(context.Context, string, string) error {
	return errCRDMethodNotImplemented
}

// RetryWorkflow is implemented in PR 4d.3 (argo v3.5 retry semantics).
func (c *crdWorkflowClient) RetryWorkflow(context.Context, string, string) error {
	return errCRDMethodNotImplemented
}

// ResubmitWorkflow is implemented in PR 4d.3 (deep-copy spec, clear metadata).
func (c *crdWorkflowClient) ResubmitWorkflow(context.Context, string, string) error {
	return errCRDMethodNotImplemented
}

// GetWorkflowLogs is implemented in PR 4d.4 (typed pod log API).
func (c *crdWorkflowClient) GetWorkflowLogs(context.Context, string, string, string, WorkflowLogOptions) (WorkflowLogResult, error) {
	return WorkflowLogResult{}, errCRDMethodNotImplemented
}

// GetWorkflowLogStream is implemented in PR 4d.4 (typed pod log API stream).
func (c *crdWorkflowClient) GetWorkflowLogStream(context.Context, string, string, string, WorkflowLogOptions) (io.ReadCloser, error) {
	return nil, errCRDMethodNotImplemented
}

// resolveNS falls back to the cluster's default namespace when the caller
// passes an empty string. Consumers today always pass an explicit ns; keeping
// the fallback avoids nil-ns 400s if that convention slips.
func (c *crdWorkflowClient) resolveNS(ns string) string {
	if ns != "" {
		return ns
	}
	return c.defaultNS
}

// unstructuredToWorkflow decodes an unstructured Workflow CRD into wfv1.
// Failure to decode is a hard error — the alternative (returning a partial
// object) would silently drop fields callers depend on.
func unstructuredToWorkflow(u *unstructured.Unstructured) (*wfv1.Workflow, error) {
	if u == nil {
		return nil, fmt.Errorf("argo crd: nil workflow unstructured")
	}
	wf := &wfv1.Workflow{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, wf); err != nil {
		return nil, fmt.Errorf("argo crd: decode workflow %q: %w", u.GetName(), err)
	}
	return wf, nil
}

// translateK8sErr maps K8s API errors into argo package sentinels the rest of
// the backend already knows how to handle. IsNotFound → ErrNotFound;
// IsAlreadyExists → ErrAlreadyExists; everything else passes through.
func translateK8sErr(err error) error {
	if err == nil {
		return nil
	}
	if apierrors.IsNotFound(err) {
		return fmt.Errorf("%w: %v", ErrNotFound, err)
	}
	if apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("%w: %v", ErrAlreadyExists, err)
	}
	return err
}
