package workflow

import (
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func resolveWorkflowPodName(wf *wfv1.Workflow, nodeID string) (string, bool) {
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
	if workflowName == "" || stepName == "" {
		return node.ID, true
	}

	suffix := strings.TrimPrefix(node.ID, workflowName+"-")
	if suffix == "" || suffix == node.ID {
		return node.ID, true
	}
	return workflowName + "-" + sanitizePodNamePart(stepName) + "-" + suffix, true
}

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
