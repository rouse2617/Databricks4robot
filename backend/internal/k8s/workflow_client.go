package k8s

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	argowfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkflowClient defines the interface for managing Argo Workflows.
type WorkflowClient interface {
	// CreateWorkflow creates a new workflow in the given namespace.
	CreateWorkflow(ctx context.Context, wf *wfv1.Workflow, namespace string) error

	// GetWorkflowStatus returns the current phase of a workflow.
	GetWorkflowStatus(ctx context.Context, name, namespace string) (wfv1.WorkflowPhase, error)

	// DeleteWorkflow removes a workflow by name.
	DeleteWorkflow(ctx context.Context, name, namespace string) error

	// ListWorkflows returns all workflows matching the label selector.
	ListWorkflows(ctx context.Context, namespace string, labelSelector string) ([]wfv1.Workflow, error)

	// GetWorkflow returns the full workflow object by name.
	GetWorkflow(ctx context.Context, name, namespace string) (*wfv1.Workflow, error)
}

// ArgoClient implements WorkflowClient backed by an Argo Workflow clientset.
type ArgoClient struct {
	client argowfclientset.Interface
}

// NewArgoClient creates a new ArgoClient from an Argo Workflow clientset.
func NewArgoClient(client argowfclientset.Interface) *ArgoClient {
	return &ArgoClient{client: client}
}

// CreateWorkflow creates a workflow in the specified namespace.
func (a *ArgoClient) CreateWorkflow(ctx context.Context, wf *wfv1.Workflow, namespace string) error {
	_, err := a.client.ArgoprojV1alpha1().Workflows(namespace).Create(ctx, wf, metav1.CreateOptions{})
	return err
}

// GetWorkflowStatus returns the current phase of the named workflow.
func (a *ArgoClient) GetWorkflowStatus(ctx context.Context, name, namespace string) (wfv1.WorkflowPhase, error) {
	wf, err := a.client.ArgoprojV1alpha1().Workflows(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return wfv1.WorkflowUnknown, err
	}
	return wf.Status.Phase, nil
}

// DeleteWorkflow removes a workflow by name.
func (a *ArgoClient) DeleteWorkflow(ctx context.Context, name, namespace string) error {
	return a.client.ArgoprojV1alpha1().Workflows(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// ListWorkflows returns workflows in the namespace filtered by label selector.
func (a *ArgoClient) ListWorkflows(ctx context.Context, namespace string, labelSelector string) ([]wfv1.Workflow, error) {
	list, err := a.client.ArgoprojV1alpha1().Workflows(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// GetWorkflow returns the full workflow object by name.
func (a *ArgoClient) GetWorkflow(ctx context.Context, name, namespace string) (*wfv1.Workflow, error) {
	return a.client.ArgoprojV1alpha1().Workflows(namespace).Get(ctx, name, metav1.GetOptions{})
}
