package transpiler

// Pipeline is the top-level DAG definition.
// It can be created by the UI (drag-and-drop) or hand-written (or AI-generated).
type Pipeline struct {
	Name        string `json:"name" yaml:"name"`
	Version     string `json:"version,omitempty" yaml:"version,omitempty"`
	Parallelism int32  `json:"parallelism,omitempty" yaml:"parallelism,omitempty"` // max concurrent pods (0 = unlimited)
	Nodes       []Node `json:"nodes" yaml:"nodes"`
	Edges       []Edge `json:"edges" yaml:"edges"`
}

// Node is a single step in the pipeline.
// When SubNodes is non-empty this node is a sub-graph (nested DAG).
type Node struct {
	ID             string                       `json:"id" yaml:"id"`
	Component      Component                    `json:"component" yaml:"component"`
	Inputs         []Port                       `json:"inputs,omitempty" yaml:"inputs,omitempty"`
	Outputs        []Port                       `json:"outputs,omitempty" yaml:"outputs,omitempty"`
	RuntimeConfig  *RuntimeConfigBinding        `json:"runtimeConfig,omitempty" yaml:"runtimeConfig,omitempty"`
	RuntimeSecrets []RuntimeSecretMountBinding  `json:"runtimeSecrets,omitempty" yaml:"runtimeSecrets,omitempty"`
	StorageMounts  []RuntimeStorageMountBinding `json:"storageMounts,omitempty" yaml:"storageMounts,omitempty"`
	SubNodes       []Node                       `json:"sub_nodes,omitempty" yaml:"sub_nodes,omitempty"`
	SubEdges       []Edge                       `json:"sub_edges,omitempty" yaml:"sub_edges,omitempty"`
	VolumeMounts   []VolumeMount                `json:"volume_mounts,omitempty" yaml:"volume_mounts,omitempty"`
}

// RuntimeConfigBinding references a saved config-library version for one node.
type RuntimeConfigBinding struct {
	Mode           string `json:"mode,omitempty" yaml:"mode,omitempty"`
	ConfigID       string `json:"configId,omitempty" yaml:"configId,omitempty"`
	Version        int    `json:"version,omitempty" yaml:"version,omitempty"`
	FileName       string `json:"fileName,omitempty" yaml:"fileName,omitempty"`
	MountPath      string `json:"mountPath,omitempty" yaml:"mountPath,omitempty"`
	TargetFilename string `json:"targetFilename,omitempty" yaml:"targetFilename,omitempty"`
}

// RuntimeSecretMountBinding references a platform-defined SecretProviderClass
// resource that should be mounted into only this node at runtime.
type RuntimeSecretMountBinding struct {
	ResourceID  string `json:"resourceId" yaml:"resourceId"`
	MountPath   string `json:"mountPath,omitempty" yaml:"mountPath,omitempty"`
	DisplayName string `json:"displayName,omitempty" yaml:"displayName,omitempty"`
}

// RuntimeStorageMountBinding references a platform-defined storage resource
// such as an existing PVC or a per-pod emptyDir.
type RuntimeStorageMountBinding struct {
	ResourceID  string `json:"resourceId" yaml:"resourceId"`
	MountPath   string `json:"mountPath,omitempty" yaml:"mountPath,omitempty"`
	ReadOnly    *bool  `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`
	DisplayName string `json:"displayName,omitempty" yaml:"displayName,omitempty"`
}

// Component is a pipeline step backed by a container image.
type Component struct {
	Name            string     `json:"name" yaml:"name"`
	Image           string     `json:"image" yaml:"image"`
	ImagePullPolicy string     `json:"imagePullPolicy,omitempty" yaml:"imagePullPolicy,omitempty"`
	Command         []string   `json:"command,omitempty" yaml:"command,omitempty"`
	Args            []Argument `json:"args,omitempty" yaml:"args,omitempty"`
	// Mode controls the template type: "container" (default) or "script".
	// "container" emits an Argo container template (suitable for any image).
	// "script" emits an Argo script template: Source is injected as inline script,
	// and Command is treated as the interpreter (default: ["sh"]).
	Mode      string                `json:"mode,omitempty" yaml:"mode,omitempty"`
	Source    string                `json:"source,omitempty" yaml:"source,omitempty"` // inline script body (script mode only)
	Env       []EnvVar              `json:"env,omitempty" yaml:"env,omitempty"`
	Resources *ResourceRequirements `json:"resources,omitempty" yaml:"resources,omitempty"`
}

// ResourceRequirements defines compute resources for a component.
type ResourceRequirements struct {
	CPU         string `json:"cpu,omitempty" yaml:"cpu,omitempty"`                 // e.g. "500m", "2"
	Memory      string `json:"memory,omitempty" yaml:"memory,omitempty"`           // e.g. "256Mi", "1Gi"
	Disk        string `json:"disk,omitempty" yaml:"disk,omitempty"`               // ephemeral storage, e.g. "1Gi"
	GPU         string `json:"gpu,omitempty" yaml:"gpu,omitempty"`                 // Kubernetes nvidia.com/gpu quantity, e.g. "1"
	ComputeTier string `json:"computeTier,omitempty" yaml:"computeTier,omitempty"` // DataBrew scheduling/cost metadata
}

// Param is a key-value pair for workflow-level parameters.
type Param struct {
	Name  string `json:"name" yaml:"name"`
	Value string `json:"value" yaml:"value"`
}

// Argument defines a parameter passed to a component.
// Value is used for static values; From references another node's output.
type Argument struct {
	Name  string `json:"name" yaml:"name"`
	Value string `json:"value,omitempty" yaml:"value,omitempty"`
	From  string `json:"from,omitempty" yaml:"from,omitempty"`
}

// EnvVar defines an environment variable for a component.
type EnvVar struct {
	Name  string `json:"name" yaml:"name"`
	Value string `json:"value,omitempty" yaml:"value,omitempty"`
	From  string `json:"from,omitempty" yaml:"from,omitempty"`
}

// VolumeMount describes a volume mount on a node's container.
type VolumeMount struct {
	Name                   string `json:"name" yaml:"name"`
	MountPath              string `json:"mountPath" yaml:"mountPath"`
	SubPath                string `json:"subPath,omitempty" yaml:"subPath,omitempty"`
	ReadOnly               bool   `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`
	PVCName                string `json:"pvcName,omitempty" yaml:"pvcName,omitempty"`                               // existing PVC
	EmptyDir               bool   `json:"emptyDir,omitempty" yaml:"emptyDir,omitempty"`                             // ephemeral volume
	CSIDriver              string `json:"csiDriver,omitempty" yaml:"csiDriver,omitempty"`                           // CSI driver name
	CSISecretProviderClass string `json:"csiSecretProviderClass,omitempty" yaml:"csiSecretProviderClass,omitempty"` // SecretProviderClass name
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
