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
	problems = append(problems, validateDanglingEdges(p)...)
	problems = append(problems, validateNormalizedNameCollisions(p)...)
	problems = append(problems, validateDuplicateTargetInputs(p)...)
	problems = append(problems, validateConsumedOutputFiles(p)...)
	problems = append(problems, validateNoCycles(p)...)
	if len(problems) > 0 {
		return &ValidationError{Problems: problems}
	}
	return nil
}

// validateDanglingEdges reports edges whose source or target node id does not
// exist in the pipeline, which would otherwise produce broken Argo dependencies.
func validateDanglingEdges(p *Pipeline) []string {
	nodes := make(map[string]bool, len(p.Nodes))
	for _, node := range p.Nodes {
		nodes[node.ID] = true
	}
	var problems []string
	for _, edge := range p.Edges {
		src, _ := edge.ResolveSource()
		tgt, _ := edge.ResolveTarget()
		if src != "" && !nodes[src] {
			problems = append(problems, fmt.Sprintf("edge %q -> %q references unknown source node %q", edge.Source, edge.Target, src))
		}
		if tgt != "" && !nodes[tgt] {
			problems = append(problems, fmt.Sprintf("edge %q -> %q references unknown target node %q", edge.Source, edge.Target, tgt))
		}
	}
	for _, node := range p.Nodes {
		if len(node.SubNodes) == 0 {
			continue
		}
		sub := &Pipeline{Nodes: node.SubNodes, Edges: node.SubEdges}
		for _, problem := range validateDanglingEdges(sub) {
			problems = append(problems, fmt.Sprintf("%s: %s", node.ID, problem))
		}
	}
	return problems
}

// validateNoCycles reports a cycle in the node dependency graph. Argo rejects
// cyclic DAGs at submission time; catching it earlier yields a clearer error.
func validateNoCycles(p *Pipeline) []string {
	exists := make(map[string]bool, len(p.Nodes))
	for _, node := range p.Nodes {
		exists[node.ID] = true
	}
	adj := make(map[string][]string)
	for _, edge := range p.Edges {
		src, _ := edge.ResolveSource()
		tgt, _ := edge.ResolveTarget()
		if !exists[src] || !exists[tgt] {
			continue // dangling edges are reported separately
		}
		adj[src] = append(adj[src], tgt)
	}

	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make(map[string]int, len(p.Nodes))
	var problems []string
	cycleFound := false
	var dfs func(node string)
	dfs = func(node string) {
		if cycleFound {
			return
		}
		color[node] = gray
		for _, next := range adj[node] {
			if cycleFound {
				return
			}
			switch color[next] {
			case gray:
				cycleFound = true
				problems = append(problems, fmt.Sprintf("pipeline graph has a cycle involving node %q", next))
				return
			case white:
				dfs(next)
			}
		}
		color[node] = black
	}
	for _, node := range p.Nodes {
		if color[node.ID] == white {
			dfs(node.ID)
		}
	}

	for _, node := range p.Nodes {
		if len(node.SubNodes) == 0 {
			continue
		}
		sub := &Pipeline{Nodes: node.SubNodes, Edges: node.SubEdges}
		for _, problem := range validateNoCycles(sub) {
			problems = append(problems, fmt.Sprintf("%s: %s", node.ID, problem))
		}
	}
	return problems
}

// validateNormalizedNameCollisions reports node ids or port names that are
// distinct in the DSL but collapse to the same Argo template/parameter name
// after normalization (templateName / safeParamName), which would otherwise
// silently overwrite templates or parameters.
func validateNormalizedNameCollisions(p *Pipeline) []string {
	var problems []string

	// Template-name collisions across the full flat node set (top-level +
	// sub-graphs). buildStepTemplateNames is dup-aware, so two nodes sharing a
	// component name are NOT a collision (they get a uuid suffix); a residual
	// clash means duplicate node ids.
	names := buildStepTemplateNames(p.Nodes)
	seenTemplate := make(map[string]string, len(names))
	for _, node := range flattenNodes(p.Nodes) {
		norm := names[node.ID]
		if prior, ok := seenTemplate[norm]; ok && prior != node.ID {
			problems = append(problems, fmt.Sprintf("node ids %q and %q map to the same workflow template name %q", prior, node.ID, norm))
			continue
		}
		seenTemplate[norm] = node.ID
	}

	// Port-name collisions (per node, recursing into sub-graphs).
	problems = append(problems, validatePortNameCollisionsRecursive(p.Nodes)...)
	return problems
}

func validatePortNameCollisionsRecursive(nodes []Node) []string {
	var problems []string
	for _, node := range nodes {
		problems = append(problems, validatePortNameCollisions(node)...)
		if len(node.SubNodes) > 0 {
			for _, problem := range validatePortNameCollisionsRecursive(node.SubNodes) {
				problems = append(problems, fmt.Sprintf("%s: %s", node.ID, problem))
			}
		}
	}
	return problems
}

func validatePortNameCollisions(node Node) []string {
	var problems []string
	check := func(kind string, ports []Port) {
		seen := make(map[string]string, len(ports))
		for _, port := range ports {
			norm := safeParamName(port.Name)
			if prior, ok := seen[norm]; ok {
				problems = append(problems, fmt.Sprintf("node %q has %s ports %q and %q that map to the same parameter name %q", node.ID, kind, prior, port.Name, norm))
				continue
			}
			seen[norm] = port.Name
		}
	}
	check("input", node.Inputs)
	check("output", node.Outputs)
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
