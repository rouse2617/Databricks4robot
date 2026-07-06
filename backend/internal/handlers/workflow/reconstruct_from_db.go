package workflow

import (
	"context"
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// reconstructWorkflowFromDB builds a *wfv1.Workflow good enough for
// buildWorkflowDetailNodes/buildWorkflowDagEdges to render a terminal run's
// detail view without querying the live Argo API. It returns ok=false when
// the data needed to do so is missing or invalid, so the caller falls back
// to the direct Argo path rather than serving an incomplete response. The
// returned []models.PipelineRunNode is the same rows used to build wf.Status
// — the caller passes it to backfillNodeItemDataFields to restore
// Inputs/Outputs/ResourcesDuration, which are intentionally NOT carried
// through wfv1.NodeStatus (see CYB-3063 decisions.md).
func (h *Handler) reconstructWorkflowFromDB(ctx context.Context, run *models.PipelineRun) (*wfv1.Workflow, []models.PipelineRunNode, bool) {
	if run == nil || run.Manifest == nil || strings.TrimSpace(*run.Manifest) == "" {
		return nil, nil, false
	}
	var wf wfv1.Workflow
	if err := sigsyaml.Unmarshal([]byte(*run.Manifest), &wf); err != nil {
		return nil, nil, false
	}
	if wf.Name == "" {
		wf.Name = run.WorkflowName
	}
	if wf.Namespace == "" {
		wf.Namespace = run.ArgoNamespace
	}
	// The manifest is a submission-time snapshot, so its CreationTimestamp is
	// zero-value (Argo had not created the object yet). Use the run's own
	// CreatedAt instead so the response's createdAt reflects reality.
	if !run.CreatedAt.IsZero() {
		wf.CreationTimestamp = metav1.NewTime(run.CreatedAt)
	}

	var runNodes []models.PipelineRunNode
	nodes := wfv1.Nodes{}
	if h.runNodeRepo != nil {
		var err error
		runNodes, err = h.runNodeRepo.FindByRunID(ctx, run.ID)
		if err != nil {
			return nil, nil, false
		}
		for _, n := range runNodes {
			status := pipelineRunNodeToNodeStatus(n)
			nodes[status.ID] = status
		}
	}
	wf.Status.Nodes = nodes
	wf.Status.Phase = wfv1.WorkflowPhase(run.Status)
	wf.Status.Message = run.Message
	if run.StartedAt != nil {
		wf.Status.StartedAt = metav1.NewTime(*run.StartedAt)
	}
	if run.FinishedAt != nil {
		wf.Status.FinishedAt = metav1.NewTime(*run.FinishedAt)
	}
	return &wf, runNodes, true
}

// pipelineRunNodeToNodeStatus maps the persisted node row to the topology
// fields buildWorkflowDetailNodes/buildWorkflowDagEdges actually read
// (ID/Name/DisplayName/Type/TemplateName/Phase/Message/BoundaryID-adjacent
// Children/StartedAt/FinishedAt/HostNodeName). Inputs/Outputs/ResourcesDuration
// are intentionally left zero-value here: the DB stores them as already-flat
// maps, not Argo's native struct types, and are backfilled onto the resulting
// workflowNodeItem afterward instead (see backfillNodeItemDataFields).
func pipelineRunNodeToNodeStatus(n models.PipelineRunNode) wfv1.NodeStatus {
	status := wfv1.NodeStatus{
		ID:           n.ArgoNodeID,
		Name:         n.ArgoNodeName,
		DisplayName:  n.DisplayName,
		Type:         wfv1.NodeType(n.Type),
		TemplateName: n.TemplateName,
		Phase:        wfv1.NodePhase(n.Phase),
		Message:      n.Message,
		HostNodeName: n.HostNodeName,
		Children:     n.Children,
	}
	if n.StartedAt != nil {
		status.StartedAt = metav1.NewTime(*n.StartedAt)
	}
	if n.FinishedAt != nil {
		status.FinishedAt = metav1.NewTime(*n.FinishedAt)
	}
	return status
}

// backfillNodeItemDataFields fills Inputs/Outputs/ResourcesDuration back onto
// items built from a DB-reconstructed workflow, matching by Argo node ID —
// see pipelineRunNodeToNodeStatus for why these fields are not carried through
// wfv1.NodeStatus.
func backfillNodeItemDataFields(items []workflowNodeItem, runNodes []models.PipelineRunNode) []workflowNodeItem {
	byID := make(map[string]models.PipelineRunNode, len(runNodes))
	for _, n := range runNodes {
		if n.ArgoNodeID != "" {
			byID[n.ArgoNodeID] = n
		}
	}
	for i := range items {
		n, ok := byID[items[i].ID]
		if !ok {
			continue
		}
		if len(n.Inputs) > 0 {
			items[i].Inputs = n.Inputs
		}
		if len(n.Outputs) > 0 {
			items[i].Outputs = n.Outputs
		}
		if len(n.ResourcesDuration) > 0 {
			items[i].ResourcesDuration = n.ResourcesDuration
		}
		if items[i].PodName == "" {
			items[i].PodName = n.PodName
		}
	}
	return items
}
