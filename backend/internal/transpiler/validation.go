package transpiler

import (
	"fmt"
	"strings"
)

// ValidationError describes a user-correctable pipeline definition problem.
type ValidationError struct {
	Problems []string
}

func (e *ValidationError) Error() string {
	if e == nil || len(e.Problems) == 0 {
		return "invalid pipeline"
	}
	return "invalid pipeline: " + strings.Join(e.Problems, "; ")
}

func (e *ValidationError) Is(target error) bool {
	_, ok := target.(*ValidationError)
	return ok
}

// ValidatePipeline checks Argo-facing constraints that are easy to diagnose
// before a workflow is submitted.
func ValidatePipeline(p *Pipeline) error {
	if p == nil {
		return &ValidationError{Problems: []string{"pipeline is required"}}
	}
	var problems []string
	problems = append(problems, validateDuplicateTargetInputs(p)...)
	if len(problems) > 0 {
		return &ValidationError{Problems: problems}
	}
	return nil
}

func validatePortTypes(p *Pipeline) []string {
	nodes := map[string]Node{}
	for _, node := range p.Nodes {
		nodes[node.ID] = node
	}
	var problems []string
	for _, edge := range p.Edges {
		sourceNode, sourcePort := edge.ResolveSource()
		targetNode, targetPort := edge.ResolveTarget()
		if sourceNode == "" || sourcePort == "" || targetNode == "" || targetPort == "" {
			continue
		}
		srcNode, ok := nodes[sourceNode]
		if !ok {
			continue
		}
		tgtNode, ok := nodes[targetNode]
		if !ok {
			continue
		}
		var srcType, tgtType string
		for _, port := range srcNode.Outputs {
			if port.Name == sourcePort {
				srcType = port.Type
				break
			}
		}
		for _, port := range tgtNode.Inputs {
			if port.Name == targetPort {
				tgtType = port.Type
				break
			}
		}
		if srcType != "" && tgtType != "" && srcType != tgtType {
			problems = append(problems, fmt.Sprintf(
				"port type mismatch: %s.%s (%s) → %s.%s (%s)",
				sourceNode, sourcePort, srcType, targetNode, targetPort, tgtType,
			))
		}
	}
	for _, node := range p.Nodes {
		if len(node.SubNodes) == 0 {
			continue
		}
		sub := &Pipeline{Nodes: node.SubNodes, Edges: node.SubEdges}
		problems = append(problems, validatePortTypes(sub)...)
	}
	return problems
}

// NormalizePipeline mutates a pipeline into the canonical shape accepted by
// the transpiler. It is intentionally conservative: only shell script arg
// duplication created by component forms is normalized.
func NormalizePipeline(p *Pipeline) {
	if p == nil {
		return
	}
	for i := range p.Nodes {
		normalizeNode(&p.Nodes[i])
	}
}

func normalizeNode(node *Node) {
	node.Component.Args = NormalizeShellArgs(node.Component.Command, node.Component.Args)
	for i := range node.SubNodes {
		normalizeNode(&node.SubNodes[i])
	}
}

// NormalizeShellArgs turns command=["sh","-c"] plus args=["sh","-c",script]
// or args=["-c",script] into args=[script].
func NormalizeShellArgs(command []string, args []Argument) []Argument {
	if len(command) != 2 || !isShellBinary(command[0]) || command[1] != "-c" || len(args) == 0 {
		return args
	}
	values := argumentValues(args)
	switch {
	case len(values) >= 3 && isShellBinary(values[0]) && values[1] == "-c":
		return []Argument{{Name: scriptArgName(args[len(args)-1]), Value: values[len(values)-1]}}
	case len(values) >= 2 && values[0] == "-c":
		return []Argument{{Name: scriptArgName(args[len(args)-1]), Value: values[len(values)-1]}}
	default:
		return args
	}
}

func argumentValues(args []Argument) []string {
	values := make([]string, 0, len(args))
	for _, arg := range args {
		if arg.From != "" {
			values = append(values, "")
			continue
		}
		values = append(values, strings.TrimSpace(arg.Value))
	}
	return values
}

func scriptArgName(arg Argument) string {
	if strings.TrimSpace(arg.Name) != "" {
		return arg.Name
	}
	return "script"
}

func validateDuplicateTargetInputs(p *Pipeline) []string {
	seen := map[string]Edge{}
	var problems []string
	for _, edge := range p.Edges {
		targetNode, targetPort := edge.ResolveTarget()
		if targetNode == "" || targetPort == "" {
			continue
		}
		key := targetNode + "." + safeParamName(targetPort)
		if prior, ok := seen[key]; ok {
			problems = append(problems, fmt.Sprintf("target input %q has multiple upstream bindings (%s and %s)", key, prior.Source, edge.Source))
			continue
		}
		seen[key] = edge
	}
	for _, node := range p.Nodes {
		if len(node.SubNodes) == 0 {
			continue
		}
		sub := &Pipeline{Nodes: node.SubNodes, Edges: node.SubEdges}
		for _, problem := range validateDuplicateTargetInputs(sub) {
			problems = append(problems, fmt.Sprintf("%s: %s", node.ID, problem))
		}
	}
	return problems
}

func validateConsumedOutputFiles(p *Pipeline) []string {
	nodes := map[string]Node{}
	for _, node := range p.Nodes {
		nodes[node.ID] = node
	}
	var problems []string
	for _, edge := range p.Edges {
		sourceNode, sourcePort := edge.ResolveSource()
		if sourceNode == "" || sourcePort == "" {
			continue
		}
		node, ok := nodes[sourceNode]
		if !ok {
			continue
		}
		if !declaresOutput(node, sourcePort) {
			continue
		}
		if componentWritesOutputPath(node.Component, sourcePort) {
			continue
		}
		problems = append(problems, fmt.Sprintf("consumed output %q must write /tmp/outputs/%s before it can feed %s", edge.Source, sourcePort, edge.Target))
	}
	for _, node := range p.Nodes {
		if len(node.SubNodes) == 0 {
			continue
		}
		sub := &Pipeline{Nodes: node.SubNodes, Edges: node.SubEdges}
		for _, problem := range validateConsumedOutputFiles(sub) {
			problems = append(problems, fmt.Sprintf("%s: %s", node.ID, problem))
		}
	}
	return problems
}

func declaresOutput(node Node, portName string) bool {
	safe := safeParamName(portName)
	for _, port := range node.Outputs {
		if port.Name == portName || safeParamName(port.Name) == safe {
			return true
		}
	}
	return false
}
