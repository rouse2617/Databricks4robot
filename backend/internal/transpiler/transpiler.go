package transpiler

import (
	"fmt"
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// inputSpec describes one input parameter that comes from an upstream node's output.
type inputSpec struct {
	paramName string
	srcNode   string
	srcPort   string
}

// Options controls how the pipeline is transpiled.
type Options struct {
	Name                 string
	Namespace            string
	ServiceAccount       string
	ImagePullSecrets     []string
	TTLSecondsAfter      int32
	RetryStrategy        *RetryStrategy
	ActiveDeadlineSeconds int64
	WorkflowParams       []Param // workflow-level parameters (e.g. asset_ids)
}

// RetryStrategy defines automatic retry policy for each step.
type RetryStrategy struct {
	Limit     int32 `json:"limit,omitempty"`
}

// Transpile converts a Pipeline definition into an Argo Workflow CRD.
func Transpile(p *Pipeline, opts *Options) (*wfv1.Workflow, error) {
	if opts == nil {
		opts = &Options{}
	}
	if opts.TTLSecondsAfter == 0 {
		opts.TTLSecondsAfter = 3600
	}
	name := opts.Name
	if name == "" {
		name = p.Name
	}
	ns := opts.Namespace
	if ns == "" {
		ns = "default"
	}

	wf := &wfv1.Workflow{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "argoproj.io/v1alpha1",
			Kind:       "Workflow",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: wfv1.WorkflowSpec{
			Entrypoint: "dag",
			TTLStrategy: &wfv1.TTLStrategy{
				SecondsAfterCompletion: &opts.TTLSecondsAfter,
			},
		},
	}

	if opts.ServiceAccount != "" {
		wf.Spec.ServiceAccountName = opts.ServiceAccount
	}
	for _, s := range opts.ImagePullSecrets {
		wf.Spec.ImagePullSecrets = append(wf.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: s})
	}

	// Workflow-level parameters (e.g. asset_ids passed at deploy time)
	if len(opts.WorkflowParams) > 0 {
		var params []wfv1.Parameter
		for _, p := range opts.WorkflowParams {
			params = append(params, wfv1.Parameter{
				Name:  p.Name,
				Value: wfv1.AnyStringPtr(p.Value),
			})
		}
		wf.Spec.Arguments = wfv1.Arguments{Parameters: params}
	}

	// Build node input specs: for each node, which input params come from where
	nodeInputs := buildInputSpecs(p)
	nodeTemplates := make(map[string]string)

	for _, node := range p.Nodes {
		tmpl := buildNodeTemplate(node, nodeInputs[node.ID], opts)
		nodeTemplates[node.ID] = tmpl.Name
		wf.Spec.Templates = append(wf.Spec.Templates, *tmpl)
	}

	dagTmpl := buildDAGTemplate(p.Nodes, p.Edges, nodeTemplates, nodeInputs)
	wf.Spec.Templates = append(wf.Spec.Templates, *dagTmpl)

	wf.Spec.Volumes = []corev1.Volume{
		{Name: "temp", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
	}

	return wf, nil
}

// buildInputSpecs collects all input parameter specs from edges and arg.From references.
func buildInputSpecs(p *Pipeline) map[string][]inputSpec {
	m := make(map[string][]inputSpec)
	for _, edge := range p.Edges {
		targetNode, targetPort := edge.ResolveTarget()
		srcNode, srcPort := edge.ResolveSource()
		m[targetNode] = append(m[targetNode], inputSpec{
			paramName: safeParamName(targetPort),
			srcNode:   srcNode,
			srcPort:   safeParamName(srcPort),
		})
	}
	for _, node := range p.Nodes {
		for _, arg := range node.Component.Args {
			if arg.From == "" {
				continue
			}
			refNode, refPort := splitRef(arg.From)
			if refNode == "" || refPort == "" {
				continue
			}
			pn := safeParamName(arg.Name)
			already := false
			for _, is := range m[node.ID] {
				if is.paramName == pn {
					already = true
					break
				}
			}
			if !already {
				m[node.ID] = append(m[node.ID], inputSpec{
					paramName: pn,
					srcNode:   refNode,
					srcPort:   safeParamName(refPort),
				})
			}
		}
		for _, env := range node.Component.Env {
			if env.From == "" {
				continue
			}
			refNode, refPort := splitRef(env.From)
			if refNode == "" || refPort == "" {
				continue
			}
			pn := safeParamName(env.Name)
			already := false
			for _, is := range m[node.ID] {
				if is.paramName == pn {
					already = true
					break
				}
			}
			if !already {
				m[node.ID] = append(m[node.ID], inputSpec{
					paramName: pn,
					srcNode:   refNode,
					srcPort:   safeParamName(refPort),
				})
			}
		}
	}
	return m
}

// buildNodeTemplate creates a Container template. Input params are name-only —
// actual values come from DAG task arguments.
func buildNodeTemplate(node Node, inputs []inputSpec, opts *Options) *wfv1.Template {
	pullPolicy := corev1.PullIfNotPresent
	switch node.Component.ImagePullPolicy {
	case "Always":
		pullPolicy = corev1.PullAlways
	case "Never":
		pullPolicy = corev1.PullNever
	case "IfNotPresent":
		pullPolicy = corev1.PullIfNotPresent
	}
	tmpl := wfv1.Template{
		Name: templateName(node.ID),
		Container: &corev1.Container{
			Image:           node.Component.Image,
			Command:         node.Component.Command,
			ImagePullPolicy: pullPolicy,
		},
	}

	// Resources
	if node.Component.Resources != nil {
		res := node.Component.Resources
		limits := corev1.ResourceList{}
		requests := corev1.ResourceList{}

		if res.CPU != "" {
			if q, err := resource.ParseQuantity(res.CPU); err == nil {
				limits[corev1.ResourceCPU] = q
				requests[corev1.ResourceCPU] = q
			}
		}
		if res.Memory != "" {
			if q, err := resource.ParseQuantity(res.Memory); err == nil {
				limits[corev1.ResourceMemory] = q
				requests[corev1.ResourceMemory] = q
			}
		}
		if res.Disk != "" {
			if q, err := resource.ParseQuantity(res.Disk); err == nil {
				limits[corev1.ResourceEphemeralStorage] = q
				requests[corev1.ResourceEphemeralStorage] = q
			}
		}
		if len(limits) > 0 || len(requests) > 0 {
			tmpl.Container.Resources = corev1.ResourceRequirements{Limits: limits, Requests: requests}
		}
	}

	// Input param declarations (names only — values come from DAG task arguments)
	var inputParams []wfv1.Parameter
	for _, is := range inputs {
		inputParams = append(inputParams, wfv1.Parameter{Name: is.paramName})
	}
	if len(inputParams) > 0 {
		tmpl.Inputs = wfv1.Inputs{Parameters: inputParams}
	}

	// Output parameters (captured from file paths)
	var outputParams []wfv1.Parameter
	for _, out := range node.Outputs {
		outputParams = append(outputParams, wfv1.Parameter{
			Name:      safeParamName(out.Name),
			ValueFrom: &wfv1.ValueFrom{Path: fmt.Sprintf("/tmp/outputs/%s", out.Name)},
		})
	}
	if len(outputParams) > 0 {
		tmpl.Outputs = wfv1.Outputs{Parameters: outputParams}
	}

	// Container args
	var containerArgs []string
	for _, arg := range node.Component.Args {
		if arg.From != "" {
			containerArgs = append(containerArgs, fmt.Sprintf("{{inputs.parameters.%s}}", safeParamName(arg.Name)))
		} else if arg.Value != "" {
			containerArgs = append(containerArgs, arg.Value)
		} else {
			containerArgs = append(containerArgs, fmt.Sprintf("{{inputs.parameters.%s}}", safeParamName(arg.Name)))
		}
	}
	tmpl.Container.Args = containerArgs

	// Environment variables
	var envVars []corev1.EnvVar
	for _, env := range node.Component.Env {
		if env.From != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  env.Name,
				Value: fmt.Sprintf("{{inputs.parameters.%s}}", safeParamName(env.Name)),
			})
		} else {
			envVars = append(envVars, corev1.EnvVar{Name: env.Name, Value: env.Value})
		}
	}
	if len(envVars) > 0 {
		tmpl.Container.Env = envVars
	}

	// Retry strategy
	if opts.RetryStrategy != nil && opts.RetryStrategy.Limit > 0 {
		limit := intstr.FromInt(int(opts.RetryStrategy.Limit))
		tmpl.RetryStrategy = &wfv1.RetryStrategy{
			Limit: &limit,
		}
	}

	// Timeout per step
	if opts.ActiveDeadlineSeconds > 0 {
		d := intstr.FromInt(int(opts.ActiveDeadlineSeconds))
		tmpl.ActiveDeadlineSeconds = &d
	}

	return &tmpl
}

// buildDAGTemplate creates the entrypoint DAG. Parameter values are set as task
// arguments so Argo resolves cross-task references correctly.
func buildDAGTemplate(nodes []Node, edges []Edge, nodeTemplates map[string]string, nodeInputs map[string][]inputSpec) *wfv1.Template {
	var tasks []wfv1.DAGTask

	for _, node := range nodes {
		tmplName := nodeTemplates[node.ID]
		task := wfv1.DAGTask{
			Name:     tmplName,
			Template: tmplName,
		}

		// Dependencies
		var deps []string
		for _, edge := range edges {
			targetNode, _ := edge.ResolveTarget()
			if targetNode != node.ID {
				continue
			}
			srcNode, _ := edge.ResolveSource()
			deps = append(deps, nodeTemplates[srcNode])
		}
		deps = unique(deps)
		if len(deps) > 0 {
			task.Dependencies = deps
		}

		// Parameter values as task arguments
		if inputs, ok := nodeInputs[node.ID]; ok && len(inputs) > 0 {
			var params []wfv1.Parameter
			for _, is := range inputs {
				params = append(params, wfv1.Parameter{
					Name:  is.paramName,
					Value: wfv1.AnyStringPtr(fmt.Sprintf("{{tasks.%s.outputs.parameters.%s}}", templateName(is.srcNode), is.srcPort)),
				})
			}
			task.Arguments = wfv1.Arguments{Parameters: params}
		}

		tasks = append(tasks, task)
	}

	return &wfv1.Template{
		Name: "dag",
		DAG:  &wfv1.DAGTemplate{Tasks: tasks},
	}
}

// --- helpers ---

func templateName(nodeID string) string {
	return "step-" + strings.ReplaceAll(nodeID, "_", "-")
}

func safeParamName(name string) string {
	return strings.ReplaceAll(name, ".", "-")
}

func unique(s []string) []string {
	seen := make(map[string]bool, len(s))
	r := make([]string, 0, len(s))
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			r = append(r, v)
		}
	}
	return r
}
