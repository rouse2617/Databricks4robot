package argo

import (
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

// PodNameForNode derives the K8s pod name for a Pod-type workflow node using
// argo's POD_NAMES=v2 scheme (argo v3.4+ default): the pod is named
// "<workflow>-<sanitized template>-<suffix>", where suffix is the node ID with
// the "<workflow>-" prefix stripped. NodeStatus carries no pod-name field, so
// the name must be reconstructed rather than read — and it is NOT equal to
// node.ID (which lacks the template segment). Returns (name, true) for a
// resolvable Pod node; ("", false) when wf/node is missing or the node is not
// a Pod.
//
// This is the single source of truth for pod naming, shared by the argo CRD
// client (RetryWorkflow pod deletion, multi-pod log enumeration) and the
// workflow handler (log/terminal/diagnostics). See CYB-3486d: the CRD client
// previously used node.ID directly, so on POD_NAMES=v2 clusters it never
// matched the real pod — retry silently skipped stale pods and aggregate log
// fetch came back empty.
func PodNameForNode(wf *wfv1.Workflow, nodeID string) (string, bool) {
	if wf == nil || strings.TrimSpace(nodeID) == "" {
		return "", false
	}
	node, ok := wf.Status.Nodes[nodeID]
	if !ok {
		return "", false
	}
	if node.Type != wfv1.NodeTypePod {
		return "", false
	}

	workflowName := strings.TrimSpace(wf.Name)
	stepName := strings.TrimSpace(node.TemplateName)
	if stepName == "" {
		stepName = strings.TrimSpace(node.DisplayName)
	}
	// Without a workflow name or template segment the v2 scheme has nothing to
	// build from — fall back to the node ID (also the argo behavior for the
	// single-node "<workflow>" root pod).
	if workflowName == "" || stepName == "" {
		return node.ID, true
	}

	suffix := strings.TrimPrefix(node.ID, workflowName+"-")
	if suffix == "" || suffix == node.ID {
		return node.ID, true
	}
	return workflowName + "-" + sanitizePodNamePart(stepName) + "-" + suffix, true
}

// sanitizePodNamePart lower-cases and strips a template name down to the
// [a-z0-9-] alphabet argo uses for the pod-name template segment, collapsing
// runs of invalid characters into a single dash and trimming leading/trailing
// dashes.
func sanitizePodNamePart(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var out strings.Builder
	lastDash := false
	for _, r := range value {
		valid := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if valid {
			out.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			out.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(out.String(), "-")
}
