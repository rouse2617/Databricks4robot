package k8s

import (
	"context"
	"fmt"
	"io"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	argowfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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

	// StopWorkflow sets the Shutdown strategy on a running workflow to stop it.
	StopWorkflow(ctx context.Context, name, namespace string) error

	// GetWorkflowLogs returns logs for a specific workflow node (pod) identified by nodeId.
	GetWorkflowLogs(ctx context.Context, workflowName, nodeId, namespace string) (string, error)
}

// ArgoClient implements WorkflowClient backed by an Argo Workflow clientset.
type ArgoClient struct {
	client        argowfclientset.Interface
	kubeClientset kubernetes.Interface
}

// NewArgoClient creates a new ArgoClient from an Argo Workflow clientset and a
// Kubernetes clientset (needed for pod log access).
func NewArgoClient(client argowfclientset.Interface, kubeClientset kubernetes.Interface) *ArgoClient {
	return &ArgoClient{client: client, kubeClientset: kubeClientset}
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

// StopWorkflow sets the Shutdown strategy to Stop on a running workflow.
func (a *ArgoClient) StopWorkflow(ctx context.Context, name, namespace string) error {
	wf, err := a.client.ArgoprojV1alpha1().Workflows(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get workflow: %w", err)
	}
	wf.Spec.Shutdown = wfv1.ShutdownStrategyStop
	_, err = a.client.ArgoprojV1alpha1().Workflows(namespace).Update(ctx, wf, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("stop workflow: %w", err)
	}
	return nil
}

// GetWorkflowLogs returns logs for a specific workflow node (pod) identified by nodeId.
// It finds the pod using the Argo label workflows.argoproj.io/node-id=<nodeId>.
func (a *ArgoClient) GetWorkflowLogs(ctx context.Context, workflowName, nodeId, namespace string) (string, error) {
	pods, err := a.kubeClientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf(
			"workflows.argoproj.io/workflow=%s,workflows.argoproj.io/node-id=%s",
			workflowName, nodeId,
		),
	})
	if err != nil {
		return "", fmt.Errorf("list pods for node %s: %w", nodeId, err)
	}
	if len(pods.Items) == 0 {
		return "", fmt.Errorf("no pod found for node %s in workflow %s", nodeId, workflowName)
	}

	// Get logs from the main container of the first matching pod.
	podName := pods.Items[0].Name
	req := a.kubeClientset.CoreV1().Pods(namespace).GetLogs(podName, &v1.PodLogOptions{Container: "main"})
	stream, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("get log stream for pod %s: %w", podName, err)
	}
	defer stream.Close()

	raw, err := io.ReadAll(stream)
	if err != nil {
		return "", fmt.Errorf("read logs for pod %s: %w", podName, err)
	}
	return string(raw), nil
}
