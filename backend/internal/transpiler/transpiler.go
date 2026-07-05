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

// DefaultTTLSecondsAfterCompletion keeps Argo workflow objects available long
// enough for operator diagnostics while DataBrew keeps the durable run ledger.
const DefaultTTLSecondsAfterCompletion int32 = 30 * 24 * 60 * 60

// inputSpec describes one input parameter that comes from an upstream node's output.
type inputSpec struct {
	paramName string
	srcNode   string
	srcPort   string
}

// Volume represents a named volume that can be mounted.
type Volume struct {
	Name                   string
	IsEmptyDir             bool
	PVCName                string
	ConfigMapName          string
	ConfigMapKey           string
	CSIDriver              string
	CSISecretProviderClass string
	ReadOnly               bool
}

// Options controls how the pipeline is transpiled.
type Options struct {
	Name                  string
	Namespace             string
	ServiceAccount        string
	ImagePullSecrets      []string
	TemplateNodeSelector  map[string]string
	TemplateTolerations   []corev1.Toleration
	TTLSecondsAfter       int32
	RetryStrategy         *RetryStrategy
	ActiveDeadlineSeconds int64
	WorkflowParams        []Param // workflow-level parameters (e.g. asset_ids)
	// GlobalEnv are environment variables injected into every node container (e.g. asset paths).
	GlobalEnv    []EnvVar
	ExtraVolumes []Volume // additional workflow-level volumes

	// ExitHookURL, when non-empty, injects a workflow-level exit lifecycle hook
	// that POSTs a lightweight "run finished" poke to DataBrew when the workflow
	// reaches a terminal phase. Empty disables the hook entirely (kill switch).
	ExitHookURL string
	// ExitHookTokenSecretName / ExitHookTokenSecretKey reference a Kubernetes
	// Secret (in the workflow namespace) holding the webhook auth token, injected
	// as an env var via valueFrom.secretKeyRef so the token is never embedded in
	// the manifest.
	ExitHookTokenSecretName string
	ExitHookTokenSecretKey  string
	// ExitHookImage is the container image (must contain curl) used by the exit
	// notify handler. Empty falls back to defaultExitNotifyImage.
	ExitHookImage string
}

// ExitNotifyTemplateName is the template invoked by the workflow-level exit hook.
// It is distinct from node templates (which are prefixed "step-") so DataBrew can
// exclude this node from run status derivation and the step list.
const ExitNotifyTemplateName = "databrew-exit-notify"

// exitNotifyHeaderName is the HTTP header carrying the webhook auth token.
const exitNotifyHeaderName = "X-Databrew-Webhook-Token"

// defaultExitNotifyImage is the fallback image for the exit notify handler.
// It must contain curl. Overridable via Options.ExitHookImage.
const defaultExitNotifyImage = "curlimages/curl:8.11.1"

// buildExitNotifyTemplate builds a plain container template that pokes DataBrew
// on workflow completion via curl. A container (not an Argo `http` template) is
// used deliberately: the http template runs on the Argo agent pod, which on this
// cluster cannot mount its service-account token (K8s 1.24+ dropped the legacy
// secret) and hangs, stalling the workflow. A normal pod schedules fine and its
// exit code is controlled here (always 0) so a webhook error never fails or
// hangs the workflow. The token is injected via env valueFrom.secretKeyRef so it
// never lands in the manifest; DataBrew treats the call as a trigger and re-reads
// authoritative state, so the body only needs to identify the workflow.
func buildExitNotifyTemplate(opts *Options) wfv1.Template {
	image := opts.ExitHookImage
	if image == "" {
		image = defaultExitNotifyImage
	}
	env := []corev1.EnvVar{{Name: "DATABREW_WEBHOOK_URL", Value: opts.ExitHookURL}}
	if opts.ExitHookTokenSecretName != "" && opts.ExitHookTokenSecretKey != "" {
		env = append(env, corev1.EnvVar{
			Name: "DATABREW_WEBHOOK_TOKEN",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: opts.ExitHookTokenSecretName},
					Key:                  opts.ExitHookTokenSecretKey,
				},
			},
		})
	}
	// Best-effort: never fail the workflow on a webhook error (|| true, exit 0).
	body := `{"workflowName":"{{workflow.name}}","namespace":"{{workflow.namespace}}","uid":"{{workflow.uid}}","phase":"{{workflow.status}}"}`
	script := `curl -sS --max-time 10 -o /dev/null -w 'databrew webhook: HTTP %{http_code}\n' ` +
		`-X POST "$DATABREW_WEBHOOK_URL" ` +
		`-H 'Content-Type: application/json' ` +
		`-H "` + exitNotifyHeaderName + `: $DATABREW_WEBHOOK_TOKEN" ` +
		`-d '` + body + `' || true; exit 0`
	return wfv1.Template{
		Name: ExitNotifyTemplateName,
		Container: &corev1.Container{
			Image:   image,
			Command: []string{"sh", "-c"},
			Args:    []string{script},
			Env:     env,
		},
	}
}

// RetryStrategy defines automatic retry policy for each step.
type RetryStrategy struct {
	Limit int32 `json:"limit,omitempty"`
}

// Transpile converts a Pipeline definition into an Argo Workflow CRD.
func Transpile(p *Pipeline, opts *Options) (*wfv1.Workflow, error) {
	NormalizePipeline(p)
	if err := ValidatePipeline(p); err != nil {
		return nil, err
	}
	if opts == nil {
		opts = &Options{}
	}
	if opts.TTLSecondsAfter == 0 {
		opts.TTLSecondsAfter = DefaultTTLSecondsAfterCompletion
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

	// Workflow-level exit hook: poke DataBrew on terminal phase (push status).
	// Kept out of the entrypoint DAG so the notify node is never treated as a
	// business step; DataBrew excludes it from run status derivation.
	if opts.ExitHookURL != "" {
		wf.Spec.Templates = append(wf.Spec.Templates, buildExitNotifyTemplate(opts))
		wf.Spec.Hooks = wfv1.LifecycleHooks{
			wfv1.ExitLifecycleEvent: wfv1.LifecycleHook{Template: ExitNotifyTemplateName},
		}
	}

	if p.Parallelism > 0 {
		v := int64(p.Parallelism)
		wf.Spec.Parallelism = &v
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
	outputConsumers := buildOutputConsumers(p)
	nodeTemplates := make(map[string]string)
	allTmpls, err := buildAllNodeTemplates(p.Nodes, nodeInputs, outputConsumers, opts)
	if err != nil {
		return nil, fmt.Errorf("build node templates: %w", err)
	}
	for _, tmpl := range allTmpls {
		wf.Spec.Templates = append(wf.Spec.Templates, tmpl)
	}
	for _, node := range p.Nodes {
		nodeTemplates[node.ID] = templateName(node.ID)
	}

	dagTmpl := buildDAGTemplate(p.Nodes, p.Edges, nodeTemplates, nodeInputs)
	wf.Spec.Templates = append(wf.Spec.Templates, *dagTmpl)

	wf.Spec.Volumes = buildWorkflowVolumes(p.Nodes, opts.ExtraVolumes)

	return wf, nil
}

// buildWorkflowVolumes collects volume declarations from node volume mounts and extra volumes.
func buildWorkflowVolumes(nodes []Node, extra []Volume) []corev1.Volume {
	seen := make(map[string]bool)
	var vols []corev1.Volume
	for _, v := range extra {
		if seen[v.Name] {
			continue
		}
		vol := corev1.Volume{Name: v.Name}
		if v.IsEmptyDir {
			vol.VolumeSource = corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			}
		} else if v.CSISecretProviderClass != "" {
			vol.VolumeSource = corev1.VolumeSource{
				CSI: &corev1.CSIVolumeSource{
					Driver:           csiDriver(v.CSIDriver),
					ReadOnly:         boolPtr(true),
					VolumeAttributes: map[string]string{"secretProviderClass": v.CSISecretProviderClass},
				},
			}
		} else if v.ConfigMapName != "" {
			vol.VolumeSource = corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: v.ConfigMapName},
				},
			}
		} else if v.PVCName != "" {
			vol.VolumeSource = corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: v.PVCName,
					ReadOnly:  v.ReadOnly,
				},
			}
		}
		vols = append(vols, vol)
		seen[v.Name] = true
	}
	// Collect volumes from all nodes (including sub-graph nodes).
	var collect func(nodes []Node)
	collect = func(ns []Node) {
		for _, n := range ns {
			for _, vm := range n.VolumeMounts {
				if seen[vm.Name] {
					continue
				}
				if vm.EmptyDir {
					vols = append(vols, corev1.Volume{
						Name: vm.Name,
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					})
					seen[vm.Name] = true
				} else if vm.CSISecretProviderClass != "" {
					vols = append(vols, corev1.Volume{
						Name: vm.Name,
						VolumeSource: corev1.VolumeSource{
							CSI: &corev1.CSIVolumeSource{
								Driver:           csiDriver(vm.CSIDriver),
								ReadOnly:         boolPtr(true),
								VolumeAttributes: map[string]string{"secretProviderClass": vm.CSISecretProviderClass},
							},
						},
					})
					seen[vm.Name] = true
				} else if vm.PVCName != "" {
					vols = append(vols, corev1.Volume{
						Name: vm.Name,
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: vm.PVCName,
								ReadOnly:  vm.ReadOnly,
							},
						},
					})
					seen[vm.Name] = true
				}
			}
			if len(n.SubNodes) > 0 {
				collect(n.SubNodes)
			}
		}
	}
	collect(nodes)
	// Always include a temp emptyDir for scratch space.
	if !seen["temp"] {
		vols = append(vols, corev1.Volume{Name: "temp", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}})
	}
	return vols
}

func csiDriver(driver string) string {
	driver = strings.TrimSpace(driver)
	if driver == "" {
		return "secrets-store-gke.csi.k8s.io"
	}
	return driver
}

func boolPtr(value bool) *bool {
	return &value
}

// buildInputSpecs collects all input parameter specs from edges and arg.From references.
func buildInputSpecs(p *Pipeline) map[string][]inputSpec {
	m := make(map[string][]inputSpec)
	for _, edge := range p.Edges {
		targetNode, targetPort := edge.ResolveTarget()
		srcNode, srcPort := edge.ResolveSource()
		if targetPort == "" {
			continue
		}
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

func buildOutputConsumers(p *Pipeline) map[string]map[string]bool {
	m := make(map[string]map[string]bool)
	mark := func(nodeID, port string) {
		if nodeID == "" || port == "" {
			return
		}
		if m[nodeID] == nil {
			m[nodeID] = make(map[string]bool)
		}
		m[nodeID][port] = true
		m[nodeID][safeParamName(port)] = true
	}
	for _, edge := range p.Edges {
		nodeID, port := edge.ResolveSource()
		mark(nodeID, port)
	}
	for _, node := range p.Nodes {
		for _, arg := range node.Component.Args {
			refNode, refPort := splitRef(arg.From)
			mark(refNode, refPort)
		}
	}
	return m
}

func outputParamDecls(node Node, consumed map[string]bool) []wfv1.Parameter {
	var outputParams []wfv1.Parameter
	for _, out := range node.Outputs {
		if !consumed[out.Name] && !consumed[safeParamName(out.Name)] && !componentWritesOutputPath(node.Component, out.Name) {
			continue
		}
		outputParams = append(outputParams, wfv1.Parameter{
			Name:      safeParamName(out.Name),
			ValueFrom: &wfv1.ValueFrom{Path: fmt.Sprintf("/tmp/outputs/%s", out.Name)},
		})
	}
	return outputParams
}

func componentWritesOutputPath(c Component, outputName string) bool {
	path := fmt.Sprintf("/tmp/outputs/%s", outputName)
	if strings.Contains(c.Source, path) {
		return true
	}
	for _, part := range c.Command {
		if strings.Contains(part, path) {
			return true
		}
	}
	for _, arg := range c.Args {
		if strings.Contains(arg.Value, path) {
			return true
		}
	}
	return false
}

// buildContainerTemplate creates a Container template. Input params are name-only —
// actual values come from DAG task arguments.
func buildContainerTemplate(node Node, inputs []inputSpec, consumedOutputs map[string]bool, opts *Options) *wfv1.Template {
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

	tmpl.Container.Resources = buildK8sResources(node.Component.Resources)
	applyTemplateSchedulingDefaults(&tmpl, opts)
	applySchedulingHints(&tmpl, node.Component.Resources)

	// Input param declarations (names only — values come from DAG task arguments)
	var inputParams []wfv1.Parameter
	for _, is := range inputs {
		inputParams = append(inputParams, wfv1.Parameter{Name: is.paramName})
	}
	if len(inputParams) > 0 {
		tmpl.Inputs = wfv1.Inputs{Parameters: inputParams}
	}

	// Output parameters (captured from file paths)
	outputParams := outputParamDecls(node, consumedOutputs)
	if len(outputParams) > 0 {
		tmpl.Outputs = wfv1.Outputs{Parameters: outputParams}
	}

	// Container args
	var containerArgs []string
	for _, arg := range node.Component.Args {
		if arg.From != "" {
			containerArgs = append(containerArgs, fmt.Sprintf("{{inputs.parameters.%s}}", safeParamName(arg.Name)))
		} else if arg.Value != "" {
			containerArgs = append(containerArgs, splitArgValue(arg.Value)...)
		} else {
			containerArgs = append(containerArgs, fmt.Sprintf("{{inputs.parameters.%s}}", safeParamName(arg.Name)))
		}
	}
	// Auto-create /tmp/outputs/ when the component declares output parameters
	// and the container is running a shell command (sh -c).
	// Typical case: Command=["sh"], Args=[{Value:"-c"}, {Value:"echo ... > /tmp/outputs/output"}]
	// → containerArgs = ["-c", "echo ..."]
	if len(outputParams) > 0 {
		if scriptArgIndex := shellScriptArgIndex(node.Component.Command, containerArgs); scriptArgIndex >= 0 {
			containerArgs[scriptArgIndex] = "mkdir -p /tmp/outputs && " + containerArgs[scriptArgIndex]
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
	// Volume mounts
	if len(node.VolumeMounts) > 0 {
		var volMounts []corev1.VolumeMount
		for _, vm := range node.VolumeMounts {
			volMounts = append(volMounts, corev1.VolumeMount{
				Name:      vm.Name,
				MountPath: vm.MountPath,
				SubPath:   vm.SubPath,
				ReadOnly:  vm.ReadOnly,
			})
		}
		tmpl.Container.VolumeMounts = volMounts
	}
	// GlobalEnv (asset IDs, deployment ID, etc.) injected into every container.
	// Global values override a component env of the same name instead of
	// emitting duplicate entries, which Kubernetes resolves ambiguously.
	tmpl.Container.Env = mergeGlobalEnv(tmpl.Container.Env, opts.GlobalEnv)

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

// buildAllNodeTemplates recursively builds templates for a list of nodes.
// For container nodes returns 1 template; for sub-graph nodes returns N+1
// templates (1 DAG template + N leaf templates for sub-nodes).
func buildAllNodeTemplates(nodes []Node, inputs map[string][]inputSpec, outputConsumers map[string]map[string]bool, opts *Options) ([]wfv1.Template, error) {
	var all []wfv1.Template
	for _, node := range nodes {
		tms, err := buildNodeTemplates(node, inputs[node.ID], outputConsumers[node.ID], opts)
		if err != nil {
			return nil, err
		}
		all = append(all, tms...)
	}
	return all, nil
}

// buildNodeTemplates returns all templates for a single node.
// For sub-graph nodes this recursively includes sub-node templates.
func buildNodeTemplates(node Node, inputs []inputSpec, consumedOutputs map[string]bool, opts *Options) ([]wfv1.Template, error) {
	if len(node.SubNodes) > 0 {
		return buildSubGraphTemplates(node, inputs, opts)
	}
	if node.Component.Mode == "script" {
		return []wfv1.Template{*buildScriptTemplate(node, inputs, consumedOutputs, opts)}, nil
	}
	return []wfv1.Template{*buildContainerTemplate(node, inputs, consumedOutputs, opts)}, nil
}

// buildSubGraphTemplates builds templates for a sub-graph node.
// Returns container templates for sub-nodes plus the DAG template.
func buildSubGraphTemplates(node Node, inputs []inputSpec, opts *Options) ([]wfv1.Template, error) {
	subPipe := &Pipeline{Nodes: node.SubNodes, Edges: node.SubEdges}
	subInputs := buildInputSpecs(subPipe)
	subOutputConsumers := buildOutputConsumers(subPipe)

	var templates []wfv1.Template
	for _, subNode := range node.SubNodes {
		tms, err := buildNodeTemplates(subNode, subInputs[subNode.ID], subOutputConsumers[subNode.ID], opts)
		if err != nil {
			return nil, err
		}
		templates = append(templates, tms...)
	}

	subTemplateNames := make(map[string]string)
	for _, subNode := range node.SubNodes {
		subTemplateNames[subNode.ID] = templateName(subNode.ID)
	}

	dagTmpl := buildDAGTemplate(node.SubNodes, node.SubEdges, subTemplateNames, subInputs)
	dagTmpl.Name = templateName(node.ID)
	templates = append(templates, *dagTmpl)
	return templates, nil
}

// buildScriptTemplate creates a Script template (Argo script template).
// In script mode:
//   - Command is the interpreter (default: ["sh"])
//   - Source is the inline script body
//   - Argo writes source to a temp file and runs `command < tmpfile`
//   - Stdout is automatically captured as outputs.result
//   - File-based output params use valueFrom.path (same as container mode)
func buildScriptTemplate(node Node, inputs []inputSpec, consumedOutputs map[string]bool, opts *Options) *wfv1.Template {
	pullPolicy := corev1.PullIfNotPresent
	switch node.Component.ImagePullPolicy {
	case "Always":
		pullPolicy = corev1.PullAlways
	case "Never":
		pullPolicy = corev1.PullNever
	case "IfNotPresent":
		pullPolicy = corev1.PullIfNotPresent
	}

	// Determine interpreter command
	command := node.Component.Command
	if len(command) == 0 {
		command = []string{"sh"} // default interpreter
	}

	// Determine source
	source := node.Component.Source
	if source == "" && len(node.Component.Args) > 0 {
		// Fallback: derive source from positional args
		for _, a := range node.Component.Args {
			if a.Value != "" {
				source += a.Value + "\n"
			} else if a.From != "" {
				source += fmt.Sprintf("# value from {{inputs.parameters.%s}}\n", safeParamName(a.Name))
			} else {
				source += fmt.Sprintf("# value from {{inputs.parameters.%s}}\n", safeParamName(a.Name))
			}
		}
	}

	tmpl := wfv1.Template{
		Name: templateName(node.ID),
		Script: &wfv1.ScriptTemplate{
			Container: corev1.Container{
				Image:           node.Component.Image,
				Command:         command,
				ImagePullPolicy: pullPolicy,
			},
			Source: source,
		},
	}

	tmpl.Script.Resources = buildK8sResources(node.Component.Resources)
	applyTemplateSchedulingDefaults(&tmpl, opts)
	applySchedulingHints(&tmpl, node.Component.Resources)

	// Input param declarations (names only — values come from DAG task arguments)
	var inputParams []wfv1.Parameter
	for _, is := range inputs {
		inputParams = append(inputParams, wfv1.Parameter{Name: is.paramName})
	}
	if len(inputParams) > 0 {
		tmpl.Inputs = wfv1.Inputs{Parameters: inputParams}
	}

	// Output parameters (captured from file paths)
	outputParams := outputParamDecls(node, consumedOutputs)
	if len(outputParams) > 0 {
		tmpl.Outputs = wfv1.Outputs{Parameters: outputParams}
		// Auto-create /tmp/outputs/ directory in the script source
		tmpl.Script.Source = "mkdir -p /tmp/outputs\n" + tmpl.Script.Source
	}

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
		tmpl.Script.Env = envVars
	}
	// Volume mounts
	if len(node.VolumeMounts) > 0 {
		var volMounts []corev1.VolumeMount
		for _, vm := range node.VolumeMounts {
			volMounts = append(volMounts, corev1.VolumeMount{
				Name:      vm.Name,
				MountPath: vm.MountPath,
				SubPath:   vm.SubPath,
				ReadOnly:  vm.ReadOnly,
			})
		}
		tmpl.Script.VolumeMounts = volMounts
	}
	// GlobalEnv injected into every node (override duplicates, see container path).
	tmpl.Script.Env = mergeGlobalEnv(tmpl.Script.Env, opts.GlobalEnv)

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

// --- helpers ---

// mergeGlobalEnv appends platform-injected global env vars to a container's env
// list. When a global var shares a name with an existing component env var the
// global value overrides it in place rather than producing a duplicate entry
// (Kubernetes resolves duplicate env names ambiguously and warns).
func mergeGlobalEnv(base []corev1.EnvVar, global []EnvVar) []corev1.EnvVar {
	if len(global) == 0 {
		return base
	}
	index := make(map[string]int, len(base))
	for i, e := range base {
		index[e.Name] = i
	}
	for _, env := range global {
		if i, ok := index[env.Name]; ok {
			base[i].Value = env.Value
			continue
		}
		base = append(base, corev1.EnvVar{Name: env.Name, Value: env.Value})
		index[env.Name] = len(base) - 1
	}
	return base
}

func buildK8sResources(res *ResourceRequirements) corev1.ResourceRequirements {
	if res == nil {
		return corev1.ResourceRequirements{}
	}
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
	if res.GPU != "" {
		if q, err := resource.ParseQuantity(res.GPU); err == nil {
			limits[corev1.ResourceName("nvidia.com/gpu")] = q
		}
	}
	if len(limits) == 0 && len(requests) == 0 {
		return corev1.ResourceRequirements{}
	}
	return corev1.ResourceRequirements{Limits: limits, Requests: requests}
}

func applySchedulingHints(tmpl *wfv1.Template, res *ResourceRequirements) {
	if tmpl == nil || !requiresGPU(res) {
		return
	}
	if tmpl.NodeSelector == nil {
		tmpl.NodeSelector = map[string]string{}
	}
	if isL4ComputeTier(res.ComputeTier) {
		tmpl.NodeSelector["cloud.google.com/gke-accelerator"] = "nvidia-l4"
	}
	appendTemplateToleration(tmpl, corev1.Toleration{
		Key:      "nvidia.com/gpu",
		Operator: corev1.TolerationOpEqual,
		Value:    "present",
		Effect:   corev1.TaintEffectNoSchedule,
	})
}

func applyTemplateSchedulingDefaults(tmpl *wfv1.Template, opts *Options) {
	if tmpl == nil || opts == nil {
		return
	}
	if len(opts.TemplateNodeSelector) > 0 {
		if tmpl.NodeSelector == nil {
			tmpl.NodeSelector = map[string]string{}
		}
		for key, value := range opts.TemplateNodeSelector {
			if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
				continue
			}
			tmpl.NodeSelector[key] = value
		}
	}
	for _, toleration := range opts.TemplateTolerations {
		if strings.TrimSpace(toleration.Key) == "" && toleration.Operator != corev1.TolerationOpExists {
			continue
		}
		appendTemplateToleration(tmpl, toleration)
	}
}

func requiresGPU(res *ResourceRequirements) bool {
	if res == nil || strings.TrimSpace(res.GPU) == "" {
		return false
	}
	q, err := resource.ParseQuantity(strings.TrimSpace(res.GPU))
	if err != nil {
		return false
	}
	return q.Sign() > 0
}

func isL4ComputeTier(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(normalized, "l4")
}

func appendTemplateToleration(tmpl *wfv1.Template, toleration corev1.Toleration) {
	for _, existing := range tmpl.Tolerations {
		if existing.Key == toleration.Key &&
			existing.Operator == toleration.Operator &&
			existing.Value == toleration.Value &&
			existing.Effect == toleration.Effect {
			return
		}
	}
	tmpl.Tolerations = append(tmpl.Tolerations, toleration)
}

// isShellName reports whether cmd is a single shell name (e.g. ["sh"], ["/bin/sh"]).
func isShellBinary(name string) bool {
	switch name {
	case "sh", "bash", "dash", "zsh", "/bin/sh", "/bin/bash", "/usr/bin/sh", "/usr/bin/bash":
		return true
	}
	return false
}

// shellScriptArgIndex finds the args entry that contains the shell script body.
// Argo container templates commonly use either Command=["sh"], Args=["-c", "..."]
// splitArgValue splits a combined arg value like "--flag=value" or
// " --flag value" into separate elements so the container runtime passes
// them as individual argv entries.
func splitArgValue(raw string) []string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return nil
	}
	// "--flag=value" → ["--flag", "value"]
	if idx := strings.IndexByte(v, '='); idx > 2 && strings.HasPrefix(v, "--") {
		return []string{v[:idx], v[idx+1:]}
	}
	// "--flag value" → ["--flag", "value"]
	if idx := strings.IndexByte(v, ' '); idx > 2 && strings.HasPrefix(v, "--") {
		return []string{v[:idx], strings.TrimSpace(v[idx+1:])}
	}
	return []string{v}
}

// or Command=["sh", "-c"], Args=["..."].
func shellScriptArgIndex(cmd []string, args []string) int {
	if len(cmd) != 1 {
		if len(cmd) == 2 && isShellBinary(cmd[0]) && cmd[1] == "-c" && len(args) == 1 {
			return 0
		}
		return -1
	}
	if !isShellBinary(cmd[0]) {
		return -1
	}
	if len(args) >= 2 && args[0] == "-c" {
		return 1
	}
	if len(args) == 1 {
		return 0
	}
	return -1
}

func templateName(nodeID string) string {
	normalized := strings.ReplaceAll(nodeID, "_", "-")
	if strings.HasPrefix(normalized, "step-") {
		return normalized
	}
	return "step-" + normalized
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
