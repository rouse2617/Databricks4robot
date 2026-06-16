package workflow

import (
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func workflowCanRetry(wf *wfv1.Workflow) bool {
	if wf == nil {
		return false
	}
	phase := wf.Status.Phase
	if phase != wfv1.WorkflowFailed && phase != wfv1.WorkflowError {
		return false
	}
	if workflowWasStopped(wf) {
		return false
	}
	return workflowHasRetryableFailedNodes(wf)
}

func workflowWasStopped(wf *wfv1.Workflow) bool {
	message := strings.ToLower(strings.TrimSpace(wf.Status.Message))
	return strings.Contains(message, "stopped")
}

func workflowHasRetryableFailedNodes(wf *wfv1.Workflow) bool {
	if wf == nil || len(wf.Status.Nodes) == 0 {
		return false
	}
	for _, node := range wf.Status.Nodes {
		if !isRetryableWorkflowNode(node) {
			continue
		}
		if node.Phase == wfv1.NodeFailed || node.Phase == wfv1.NodeError {
			return true
		}
	}
	return false
}

func isRetryableWorkflowNode(node wfv1.NodeStatus) bool {
	switch node.Type {
	case wfv1.NodeTypeDAG, wfv1.NodeTypeSteps, wfv1.NodeTypeStepGroup, wfv1.NodeTypeRetry, wfv1.NodeTypeSkipped:
		return false
	default:
		return true
	}
}

func workflowRetryBlockedMessage(wf *wfv1.Workflow) string {
	if workflowWasStopped(wf) {
		return "workflow was manually stopped; retry only reruns failed nodes — use resubmit to run again from the start"
	}
	if wf != nil && (wf.Status.Phase == wfv1.WorkflowFailed || wf.Status.Phase == wfv1.WorkflowError) {
		return "workflow has no failed or errored task nodes to retry — use resubmit to run again from the start"
	}
	return "workflow cannot be retried in its current state"
}
