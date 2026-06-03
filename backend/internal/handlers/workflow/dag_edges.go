package workflow

import (
	"sort"
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type workflowDagEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Kind   string `json:"kind"`
}

func buildWorkflowDagEdges(wf *wfv1.Workflow) []workflowDagEdge {
	if wf == nil {
		return nil
	}

	nodes := wf.Status.Nodes
	edges := make([]workflowDagEdge, 0)
	seen := make(map[string]struct{})

	addEdge := func(source, target, kind string) {
		if source == "" || target == "" || source == target {
			return
		}
		key := source + "\x00" + target
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		edges = append(edges, workflowDagEdge{
			ID:     "e-" + source + "-" + target,
			Source: source,
			Target: target,
			Kind:   kind,
		})
	}

	for _, id := range sortedNodeIDs(nodes) {
		sourceNode, ok := nodes[id]
		if !ok {
			continue
		}
		sourceID := nearestVisibleAncestorID(sourceNode, nodes)
		for _, childID := range sourceNode.Children {
			childNode, ok := nodes[childID]
			if !ok {
				continue
			}
			targetID := nearestVisibleDescendantID(childNode, nodes)
			addEdge(sourceID, targetID, "runtime")
		}
	}

	taskNodes := workflowTaskVisibleNodeIDs(nodes)
	for _, task := range workflowStaticDAGTasks(wf) {
		taskName := strings.TrimSpace(task.Name)
		if taskName == "" {
			continue
		}
		if _, exists := taskNodes[taskName]; !exists {
			taskNodes[taskName] = taskName
		}
	}
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.DAG == nil {
			continue
		}
		for _, task := range tmpl.DAG.Tasks {
			targetID := taskNodes[task.Name]
			for _, depName := range task.Dependencies {
				addEdge(taskNodes[depName], targetID, "dag")
			}
		}
	}

	nodesByName := make(map[string]wfv1.NodeStatus, len(nodes))
	for _, id := range sortedNodeIDs(nodes) {
		node := nodes[id]
		nodesByName[node.Name] = node
	}
	for _, id := range sortedNodeIDs(nodes) {
		node := nodes[id]
		if !isWorkflowDagDisplayableNode(node) {
			continue
		}
		if sourceID := nearestVisibleNameParentID(node, nodesByName); sourceID != "" {
			addEdge(sourceID, node.ID, "fallback")
		}
	}

	return edges
}

func sortedNodeIDs(nodes wfv1.Nodes) []string {
	ids := make([]string, 0, len(nodes))
	for id := range nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func isWorkflowDagDisplayableNode(node wfv1.NodeStatus) bool {
	nodeType := strings.ToLower(string(node.Type))
	phase := string(node.Phase)
	parentPath := ""
	if idx := strings.LastIndex(node.ID, "."); idx > 0 {
		parentPath = node.ID[:idx]
	}
	isRootDAGNode := string(node.Type) == "" &&
		node.Name != "" &&
		(node.Name == node.ID || (parentPath != "" && node.Name == parentPath))
	hasMeaningfulPhase := phase == "Skipped" ||
		phase == "Omitted" ||
		nodeType == "pod" ||
		nodeType == "template"

	return hasMeaningfulPhase && !isRootDAGNode
}

func nearestVisibleAncestorID(node wfv1.NodeStatus, nodes wfv1.Nodes) string {
	current := node
	for {
		if isWorkflowDagDisplayableNode(current) {
			return current.ID
		}
		if current.BoundaryID != "" {
			if parent, ok := nodes[current.BoundaryID]; ok && parent.ID != current.ID {
				current = parent
				continue
			}
		}
		idx := strings.LastIndex(current.Name, ".")
		if idx <= 0 {
			return ""
		}
		parentName := current.Name[:idx]
		found := false
		for _, id := range sortedNodeIDs(nodes) {
			parent := nodes[id]
			if parent.Name == parentName {
				current = parent
				found = true
				break
			}
		}
		if !found {
			return ""
		}
	}
}

func nearestVisibleDescendantID(node wfv1.NodeStatus, nodes wfv1.Nodes) string {
	if isWorkflowDagDisplayableNode(node) {
		return node.ID
	}
	queue := append([]string(nil), node.Children...)
	seen := make(map[string]struct{})
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		child, ok := nodes[id]
		if !ok {
			continue
		}
		if isWorkflowDagDisplayableNode(child) {
			return child.ID
		}
		queue = append(queue, child.Children...)
	}
	return ""
}

func workflowTaskVisibleNodeIDs(nodes wfv1.Nodes) map[string]string {
	taskNodes := make(map[string]string)
	for _, id := range sortedNodeIDs(nodes) {
		node := nodes[id]
		if !isWorkflowDagDisplayableNode(node) {
			continue
		}
		for _, key := range workflowNodeTaskNameCandidates(node) {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			if _, exists := taskNodes[key]; !exists {
				taskNodes[key] = node.ID
			}
		}
	}
	return taskNodes
}

func workflowNodeTaskNameCandidates(node wfv1.NodeStatus) []string {
	candidates := make([]string, 0, 3)
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		for _, existing := range candidates {
			if existing == value {
				return
			}
		}
		candidates = append(candidates, value)
	}

	add(node.DisplayName)
	if idx := strings.LastIndex(node.Name, "."); idx >= 0 && idx < len(node.Name)-1 {
		add(node.Name[idx+1:])
	}
	add(node.TemplateName)
	return candidates
}

func workflowStaticDAGTasks(wf *wfv1.Workflow) []wfv1.DAGTask {
	if wf == nil {
		return nil
	}
	tasks := make([]wfv1.DAGTask, 0)
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.DAG == nil {
			continue
		}
		tasks = append(tasks, tmpl.DAG.Tasks...)
	}
	return tasks
}

func workflowTemplateByName(wf *wfv1.Workflow, name string) *wfv1.Template {
	if wf == nil {
		return nil
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	for i := range wf.Spec.Templates {
		if wf.Spec.Templates[i].Name == name {
			return &wf.Spec.Templates[i]
		}
	}
	return nil
}

func nearestVisibleNameParentID(
	node wfv1.NodeStatus,
	nodesByName map[string]wfv1.NodeStatus,
) string {
	parts := strings.Split(node.Name, ".")
	for len(parts) > 1 {
		parts = parts[:len(parts)-1]
		parent, ok := nodesByName[strings.Join(parts, ".")]
		if !ok {
			continue
		}
		if isWorkflowDagDisplayableNode(parent) {
			return parent.ID
		}
	}
	return ""
}
