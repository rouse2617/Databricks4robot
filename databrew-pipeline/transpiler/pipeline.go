package transpiler

// Pipeline is the top-level DAG definition.
// It can be created by the UI (drag-and-drop) or hand-written (or AI-generated).
type Pipeline struct {
	Name    string `json:"name" yaml:"name"`
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
	Nodes   []Node `json:"nodes" yaml:"nodes"`
	Edges   []Edge `json:"edges" yaml:"edges"`
}

// Node is a single step in the pipeline.
type Node struct {
	ID        string    `json:"id" yaml:"id"`
	Component Component `json:"component" yaml:"component"`
	Inputs    []Port    `json:"inputs,omitempty" yaml:"inputs,omitempty"`
	Outputs   []Port    `json:"outputs,omitempty" yaml:"outputs,omitempty"`
}

// Component is a pipeline step backed by a container image.
type Component struct {
	Name      string               `json:"name" yaml:"name"`
	Image     string               `json:"image" yaml:"image"`
	Command   []string             `json:"command,omitempty" yaml:"command,omitempty"`
	Args      []Argument           `json:"args,omitempty" yaml:"args,omitempty"`
	Resources *ResourceRequirements `json:"resources,omitempty" yaml:"resources,omitempty"`
}

// ResourceRequirements defines compute resources for a component.
type ResourceRequirements struct {
	CPU    string `json:"cpu,omitempty" yaml:"cpu,omitempty"`       // e.g. "500m", "2"
	Memory string `json:"memory,omitempty" yaml:"memory,omitempty"` // e.g. "256Mi", "1Gi"
	Disk   string `json:"disk,omitempty" yaml:"disk,omitempty"`     // ephemeral storage, e.g. "1Gi"
}

// Argument defines a parameter passed to a component.
// Value is used for static values; From references another node's output.
type Argument struct {
	Name  string `json:"name" yaml:"name"`
	Value string `json:"value,omitempty" yaml:"value,omitempty"`
	From  string `json:"from,omitempty" yaml:"from,omitempty"`
}

// Edge connects an output port of one node to an input port of another.
// Format: "node-id.port-name"
type Edge struct {
	Source string `json:"source" yaml:"source"`
	Target string `json:"target" yaml:"target"`
}

// Port describes an input or output port on a node.
type Port struct {
	Name string `json:"name" yaml:"name"`
	Type string `json:"type" yaml:"type"`
}

// ResolveSource splits a "node-id.port-name" reference.
func (e Edge) ResolveSource() (nodeID, portName string) { return splitRef(e.Source) }

// ResolveTarget splits a "node-id.port-name" reference.
func (e Edge) ResolveTarget() (nodeID, portName string) { return splitRef(e.Target) }

func splitRef(ref string) (string, string) {
	dot := -1
	for i := len(ref) - 1; i >= 0; i-- {
		if ref[i] == '.' {
			dot = i
			break
		}
	}
	if dot < 0 {
		return ref, ""
	}
	return ref[:dot], ref[dot+1:]
}
