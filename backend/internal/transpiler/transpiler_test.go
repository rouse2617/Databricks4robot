package transpiler

import (
	"strings"
	"testing"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func TestTranspileEdgesCreateDependenciesOnly(t *testing.T) {
	p := &Pipeline{
		Name: "test",
		Nodes: []Node{
			{
				ID: "a",
				Component: Component{
					Name:    "a",
					Image:   "busybox:latest",
					Command: []string{"sh", "-c"},
					Args:    []Argument{{Name: "script", Value: "echo a"}},
				},
				Outputs: []Port{{Name: "out", Type: "string"}},
			},
			{
				ID: "b",
				Component: Component{
					Name:    "b",
					Image:   "busybox:latest",
					Command: []string{"sh", "-c"},
					Args:    []Argument{{Name: "script", Value: "echo b"}},
				},
				Inputs: []Port{{Name: "in", Type: "string"}},
			},
		},
		Edges: []Edge{{Source: "a.out", Target: "b.in"}},
	}

	wf, err := Transpile(p, &Options{Name: "wf-test", Namespace: "default"})
	if err != nil {
		t.Fatal(err)
	}
	if wf.Spec.Entrypoint != "dag" {
		t.Fatalf("entrypoint = %q", wf.Spec.Entrypoint)
	}
	task := findDAGTask(t, wf, "step-b")
	if got := task.Dependencies; len(got) != 1 || got[0] != "step-a" {
		t.Fatalf("dependencies = %#v, want [step-a]", got)
	}
	if len(task.Arguments.Parameters) != 0 {
		t.Fatalf("task arguments = %+v, want none for dependency-only edge", task.Arguments.Parameters)
	}
	tmpl := findTemplate(t, wf, "step-a")
	if len(tmpl.Outputs.Parameters) != 0 {
		t.Fatalf("upstream outputs = %+v, want none for dependency-only edge", tmpl.Outputs.Parameters)
	}
}

func TestTranspileMismatchedPortEdgeIsDependencyOnly(t *testing.T) {
	p := &Pipeline{
		Name: "mismatched-edge",
		Nodes: []Node{
			{
				ID: "step-1",
				Component: Component{
					Name:    "cloudrun-e2e-test",
					Image:   "alpine:latest",
					Command: []string{"sh", "-c"},
					Args:    []Argument{{Name: "script", Value: "echo '[E2E] OK'"}},
				},
				Inputs:  []Port{{Name: "input", Type: "asset"}},
				Outputs: []Port{{Name: "output", Type: "asset"}},
			},
			{
				ID: "step-2",
				Component: Component{
					Name:    "codex-smoke-output",
					Image:   "alpine:3.20",
					Command: []string{"sh", "-c"},
					Args:    []Argument{{Name: "script", Value: "mkdir -p /tmp/outputs && echo ok | tee /tmp/outputs/output"}},
				},
				Inputs:  []Port{{Name: "input", Type: "string"}},
				Outputs: []Port{{Name: "output", Type: "string"}},
			},
		},
		Edges: []Edge{{Source: "step-1.output", Target: "step-2.input"}},
	}

	wf, err := Transpile(p, &Options{Name: "mismatched-edge"})
	if err != nil {
		t.Fatal(err)
	}
	task := findDAGTask(t, wf, "step-step-2")
	if got := task.Dependencies; len(got) != 1 || got[0] != "step-step-1" {
		t.Fatalf("dependencies = %#v, want [step-step-1]", got)
	}
	if len(task.Arguments.Parameters) != 0 {
		t.Fatalf("task arguments = %+v, want none for dependency-only edge", task.Arguments.Parameters)
	}
	upstream := findTemplate(t, wf, "step-step-1")
	if len(upstream.Outputs.Parameters) != 0 {
		t.Fatalf("upstream outputs = %+v, want no implicit output parameter", upstream.Outputs.Parameters)
	}
}

func TestTranspileAllowsDuplicateTargetInputsForDependencyEdges(t *testing.T) {
	p := &Pipeline{
		Name: "dup-target",
		Nodes: []Node{
			{
				ID: "a",
				Component: Component{
					Name:    "a",
					Image:   "busybox:latest",
					Command: []string{"sh", "-c"},
					Args:    []Argument{{Name: "script", Value: "echo a > /tmp/outputs/out"}},
				},
				Outputs: []Port{{Name: "out", Type: "string"}},
			},
			{
				ID: "b",
				Component: Component{
					Name:    "b",
					Image:   "busybox:latest",
					Command: []string{"sh", "-c"},
					Args:    []Argument{{Name: "script", Value: "echo b > /tmp/outputs/out"}},
				},
				Outputs: []Port{{Name: "out", Type: "string"}},
			},
			{
				ID: "join",
				Component: Component{
					Name:    "join",
					Image:   "busybox:latest",
					Command: []string{"sh", "-c"},
					Args:    []Argument{{Name: "script", Value: "echo join"}},
				},
				Inputs: []Port{{Name: "input", Type: "string"}},
			},
		},
		Edges: []Edge{
			{Source: "a.out", Target: "join.input"},
			{Source: "b.out", Target: "join.input"},
		},
	}

	wf, err := Transpile(p, &Options{Name: "dup-target"})
	if err != nil {
		t.Fatalf("expected dependency-only edges to allow duplicate target handles, got %v", err)
	}
	task := findDAGTask(t, wf, "step-join")
	if got := task.Dependencies; len(got) != 2 || got[0] != "step-a" || got[1] != "step-b" {
		t.Fatalf("dependencies = %#v, want [step-a step-b]", got)
	}
	if len(task.Arguments.Parameters) != 0 {
		t.Fatalf("task arguments = %+v, want none for dependency-only edges", task.Arguments.Parameters)
	}
}

func TestTranspileAllowsDistinctFanInInputs(t *testing.T) {
	p := &Pipeline{
		Name: "distinct-target",
		Nodes: []Node{
			{
				ID: "left",
				Component: Component{
					Name:    "left",
					Image:   "busybox:latest",
					Command: []string{"sh", "-c"},
					Args:    []Argument{{Name: "script", Value: "echo left > /tmp/outputs/out"}},
				},
				Outputs: []Port{{Name: "out", Type: "string"}},
			},
			{
				ID: "right",
				Component: Component{
					Name:    "right",
					Image:   "busybox:latest",
					Command: []string{"sh", "-c"},
					Args:    []Argument{{Name: "script", Value: "echo right > /tmp/outputs/out"}},
				},
				Outputs: []Port{{Name: "out", Type: "string"}},
			},
			{
				ID: "join",
				Component: Component{
					Name:    "join",
					Image:   "busybox:latest",
					Command: []string{"sh", "-c"},
					Args:    []Argument{{Name: "script", Value: "echo join"}},
				},
				Inputs: []Port{{Name: "left", Type: "string"}, {Name: "right", Type: "string"}},
			},
		},
		Edges: []Edge{
			{Source: "left.out", Target: "join.left"},
			{Source: "right.out", Target: "join.right"},
		},
	}

	if _, err := Transpile(p, &Options{Name: "distinct-target"}); err != nil {
		t.Fatal(err)
	}
}

func TestTranspileEmitsGPUResourceLimit(t *testing.T) {
	p := &Pipeline{
		Name: "gpu-pipeline",
		Nodes: []Node{{
			ID: "gpu-step",
			Component: Component{
				Name:    "gpu",
				Image:   "nvidia/cuda:12.4.1-base-ubuntu22.04",
				Command: []string{"sh", "-c"},
				Args:    []Argument{{Name: "script", Value: "nvidia-smi"}},
				Resources: &ResourceRequirements{
					CPU:         "4000m",
					Memory:      "16Gi",
					Disk:        "50Gi",
					GPU:         "1",
					ComputeTier: "gpu-l4",
				},
			},
		}},
	}

	wf, err := Transpile(p, &Options{Name: "gpu-pipeline"})
	if err != nil {
		t.Fatal(err)
	}
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name != "step-gpu-step" {
			continue
		}
		if tmpl.Container == nil {
			t.Fatal("expected container template")
		}
		gpu := tmpl.Container.Resources.Limits[corev1.ResourceName("nvidia.com/gpu")]
		if gpu.String() != "1" {
			t.Fatalf("gpu limit = %q, want 1", gpu.String())
		}
		if _, ok := tmpl.Container.Resources.Requests[corev1.ResourceName("nvidia.com/gpu")]; ok {
			t.Fatal("gpu must not be emitted as a request")
		}
		disk := tmpl.Container.Resources.Limits[corev1.ResourceEphemeralStorage]
		if got := disk.String(); got != "50Gi" {
			t.Fatalf("disk limit = %q, want 50Gi", got)
		}
		return
	}
	t.Fatal("step-gpu-step template not found")
}

func TestTranspileRejectsBareMemoryAndDiskQuantities(t *testing.T) {
	p := &Pipeline{
		Name: "bad-resources",
		Nodes: []Node{{
			ID: "bad-step",
			Component: Component{
				Name:  "bad",
				Image: "busybox:latest",
				Resources: &ResourceRequirements{
					CPU:    "1",
					Memory: "1",
					Disk:   "1",
					GPU:    "0",
				},
			},
		}},
	}

	_, err := Transpile(p, &Options{Name: "bad-resources"})
	if err == nil {
		t.Fatal("expected resource validation error")
	}
	if !strings.Contains(err.Error(), "Memory") && !strings.Contains(err.Error(), "内存") {
		t.Fatalf("error = %q, want memory detail", err.Error())
	}
	if !strings.Contains(err.Error(), "Disk") && !strings.Contains(err.Error(), "磁盘") {
		t.Fatalf("error = %q, want disk detail", err.Error())
	}
}

func TestTranspileAcceptsConsumedOutputWithoutFileWrite(t *testing.T) {
	p := &Pipeline{
		Name: "missing-output",
		Nodes: []Node{
			{
				ID: "a",
				Component: Component{
					Name:    "a",
					Image:   "busybox:latest",
					Command: []string{"sh", "-c"},
					Args:    []Argument{{Name: "script", Value: "echo a"}},
				},
				Outputs: []Port{{Name: "output", Type: "string"}},
			},
			{
				ID: "b",
				Component: Component{
					Name:    "b",
					Image:   "busybox:latest",
					Command: []string{"sh", "-c"},
					Args:    []Argument{{Name: "script", Value: "echo b"}},
				},
				Inputs: []Port{{Name: "input", Type: "string"}},
			},
		},
		Edges: []Edge{{Source: "a.output", Target: "b.input"}},
	}

	_, err := Transpile(p, &Options{Name: "missing-output"})
	if err != nil {
		t.Fatalf("expected pipeline to transpile without output file check, got error = %q", err.Error())
	}
}

func TestTranspileExplicitArgumentFromCreatesDataBinding(t *testing.T) {
	p := &Pipeline{
		Name: "explicit-binding",
		Nodes: []Node{
			{
				ID: "a",
				Component: Component{
					Name:    "a",
					Image:   "busybox:latest",
					Command: []string{"sh", "-c"},
					Args:    []Argument{{Name: "script", Value: "echo value > /tmp/outputs/output"}},
				},
				Outputs: []Port{{Name: "output", Type: "string"}},
			},
			{
				ID: "b",
				Component: Component{
					Name:    "b",
					Image:   "busybox:latest",
					Command: []string{"sh", "-c"},
					Args: []Argument{
						{Name: "input", From: "a.output"},
					},
				},
			},
		},
	}

	wf, err := Transpile(p, &Options{Name: "explicit-binding"})
	if err != nil {
		t.Fatal(err)
	}
	task := findDAGTask(t, wf, "step-b")
	if got := task.Dependencies; len(got) != 1 || got[0] != "step-a" {
		t.Fatalf("dependencies = %#v, want [step-a]", got)
	}
	if len(task.Arguments.Parameters) != 1 {
		t.Fatalf("task arguments = %+v, want one parameter", task.Arguments.Parameters)
	}
	param := task.Arguments.Parameters[0]
	if param.Name != "input" || param.Value == nil || param.Value.String() != "{{tasks.step-a.outputs.parameters.output}}" {
		t.Fatalf("argument = %+v, want explicit output binding", param)
	}
	tmpl := findTemplate(t, wf, "step-a")
	if len(tmpl.Outputs.Parameters) != 1 || tmpl.Outputs.Parameters[0].Name != "output" {
		t.Fatalf("upstream outputs = %+v, want explicit output parameter", tmpl.Outputs.Parameters)
	}
}

func TestTranspileSkipOutputArtifactsDisablesExplicitArgumentFrom(t *testing.T) {
	p := &Pipeline{
		Name: "deploy-binding",
		Nodes: []Node{
			{
				ID: "step-6",
				Component: Component{
					Name:    "producer",
					Image:   "busybox:latest",
					Command: []string{"sh", "-c"},
					Args:    []Argument{{Name: "script", Value: "echo value"}},
				},
				Outputs: []Port{{Name: "output", Type: "string"}},
			},
			{
				ID: "step-7",
				Component: Component{
					Name:    "consumer",
					Image:   "busybox:latest",
					Command: []string{"sh", "-c"},
					Args: []Argument{
						{Name: "input", From: "step-6.output"},
						{Name: "script", Value: "echo consumer"},
					},
				},
				Inputs: []Port{{Name: "input", Type: "string"}},
			},
		},
		Edges: []Edge{{Source: "step-6.output", Target: "step-7.input"}},
	}

	wf, err := Transpile(p, &Options{Name: "deploy-binding", SkipOutputArtifacts: true})
	if err != nil {
		t.Fatal(err)
	}
	task := findDAGTask(t, wf, "step-step-7")
	if got := task.Dependencies; len(got) != 1 || got[0] != "step-step-6" {
		t.Fatalf("dependencies = %#v, want [step-step-6]", got)
	}
	if len(task.Arguments.Parameters) != 0 {
		t.Fatalf("task arguments = %+v, want none in output-skipping mode", task.Arguments.Parameters)
	}
	producer := findTemplate(t, wf, "step-step-6")
	if len(producer.Outputs.Parameters) != 0 {
		t.Fatalf("producer outputs = %+v, want none in output-skipping mode", producer.Outputs.Parameters)
	}
	consumer := findTemplate(t, wf, "step-step-7")
	if len(consumer.Inputs.Parameters) != 0 {
		t.Fatalf("consumer inputs = %+v, want none in output-skipping mode", consumer.Inputs.Parameters)
	}
	for _, arg := range consumer.Container.Args {
		if arg == "{{inputs.parameters.input}}" {
			t.Fatalf("consumer args = %+v, must not reference skipped input parameter", consumer.Container.Args)
		}
	}
}

func TestTranspileNormalizesDuplicatedShellArgs(t *testing.T) {
	p := &Pipeline{
		Name: "normalized-shell",
		Nodes: []Node{{
			ID: "n1",
			Component: Component{
				Name:    "n",
				Image:   "busybox:latest",
				Command: []string{"sh", "-c"},
				Args: []Argument{
					{Name: "sh", Value: "sh"},
					{Name: "-c", Value: "-c"},
					{Name: "script", Value: "echo ok"},
				},
			},
		}},
	}

	wf, err := Transpile(p, &Options{Name: "normalized-shell"})
	if err != nil {
		t.Fatal(err)
	}
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == "step-n1" {
			if got := tmpl.Container.Args; len(got) != 1 || got[0] != "echo ok" {
				t.Fatalf("args = %#v, want single script body", got)
			}
			return
		}
	}
	t.Fatal("step-n1 template not found")
}

func TestTranspileSkipsUnconsumedOutputFileContract(t *testing.T) {
	p := &Pipeline{
		Name: "unconsumed-output",
		Nodes: []Node{{
			ID: "n1",
			Component: Component{
				Name:    "n",
				Image:   "busybox:latest",
				Command: []string{"sh", "-c"},
				Args:    []Argument{{Name: "script", Value: "echo ok"}},
			},
			Outputs: []Port{{Name: "output", Type: "string"}},
		}},
	}

	wf, err := Transpile(p, &Options{Name: "unconsumed-output"})
	if err != nil {
		t.Fatal(err)
	}
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == "step-n1" {
			if len(tmpl.Outputs.Parameters) != 0 {
				t.Fatalf("outputs = %+v, want none for unconsumed output port", tmpl.Outputs.Parameters)
			}
			if len(tmpl.Container.Args) > 0 && tmpl.Container.Args[len(tmpl.Container.Args)-1] != "echo ok" {
				t.Fatalf("container args = %v, want command unchanged", tmpl.Container.Args)
			}
			return
		}
	}
	t.Fatal("step-n1 template not found")
}

func TestTranspileDeclaresOutputWhenCommandWritesOutputFile(t *testing.T) {
	p := &Pipeline{
		Name: "file-output",
		Nodes: []Node{{
			ID: "n1",
			Component: Component{
				Name:    "n",
				Image:   "busybox:latest",
				Command: []string{"sh", "-c"},
				Args:    []Argument{{Name: "script", Value: "echo ok > /tmp/outputs/output"}},
			},
			Outputs: []Port{{Name: "output", Type: "string"}},
		}},
	}

	wf, err := Transpile(p, &Options{Name: "file-output"})
	if err != nil {
		t.Fatal(err)
	}
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == "step-n1" {
			if len(tmpl.Outputs.Parameters) != 1 || tmpl.Outputs.Parameters[0].Name != "output" {
				t.Fatalf("outputs = %+v, want output parameter", tmpl.Outputs.Parameters)
			}
			if got := tmpl.Container.Args[len(tmpl.Container.Args)-1]; got != "mkdir -p /tmp/outputs && echo ok > /tmp/outputs/output" {
				t.Fatalf("script arg = %q", got)
			}
			return
		}
	}
	t.Fatal("step-n1 template not found")
}

func TestTranspileDefaultTTL(t *testing.T) {
	p := &Pipeline{
		Name: "ttl",
		Nodes: []Node{{
			ID:        "n1",
			Component: Component{Name: "n", Image: "busybox:latest"},
		}},
	}
	wf, err := Transpile(p, &Options{Name: "ttl-test"})
	if err != nil {
		t.Fatal(err)
	}
	if wf.Spec.TTLStrategy == nil || wf.Spec.TTLStrategy.SecondsAfterCompletion == nil {
		t.Fatal("expected TTL strategy")
	}
	if *wf.Spec.TTLStrategy.SecondsAfterCompletion != DefaultTTLSecondsAfterCompletion {
		t.Fatalf("ttl = %d, want %d", *wf.Spec.TTLStrategy.SecondsAfterCompletion, DefaultTTLSecondsAfterCompletion)
	}
}

func TestTranspileRetryStrategy(t *testing.T) {
	p := &Pipeline{
		Name: "retry-test",
		Nodes: []Node{{
			ID:        "n1",
			Component: Component{Name: "n", Image: "busybox:latest"},
		}},
	}
	opts := &Options{
		Name:          "retry-test",
		RetryStrategy: &RetryStrategy{Limit: 3},
	}
	wf, err := Transpile(p, opts)
	if err != nil {
		t.Fatal(err)
	}

	// Find the node template (not the DAG template)
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == "step-n1" {
			if tmpl.RetryStrategy == nil {
				t.Fatal("expected retry strategy")
			}
			if tmpl.RetryStrategy.Limit.IntValue() != 3 {
				t.Fatalf("retry limit = %d, want 3", tmpl.RetryStrategy.Limit.IntValue())
			}
			return
		}
	}
	t.Fatal("step-n1 template not found")
}

func TestTranspileActiveDeadlineSeconds(t *testing.T) {
	p := &Pipeline{
		Name: "timeout-test",
		Nodes: []Node{{
			ID:        "n1",
			Component: Component{Name: "n", Image: "busybox:latest"},
		}},
	}
	opts := &Options{
		Name:                  "timeout-test",
		ActiveDeadlineSeconds: 600,
	}
	wf, err := Transpile(p, opts)
	if err != nil {
		t.Fatal(err)
	}

	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == "step-n1" {
			if tmpl.ActiveDeadlineSeconds == nil {
				t.Fatal("expected activeDeadlineSeconds")
			}
			if tmpl.ActiveDeadlineSeconds.IntValue() != 600 {
				t.Fatalf("activeDeadlineSeconds = %d, want 600", tmpl.ActiveDeadlineSeconds.IntValue())
			}
			return
		}
	}
	t.Fatal("step-n1 template not found")
}

func TestTranspileRetryAndTimeout(t *testing.T) {
	p := &Pipeline{
		Name: "combined",
		Nodes: []Node{{
			ID:        "n1",
			Component: Component{Name: "n", Image: "busybox:latest"},
		}},
	}
	opts := &Options{
		Name:                  "combined",
		RetryStrategy:         &RetryStrategy{Limit: 2},
		ActiveDeadlineSeconds: 300,
	}
	wf, err := Transpile(p, opts)
	if err != nil {
		t.Fatal(err)
	}

	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == "step-n1" {
			if tmpl.RetryStrategy == nil || tmpl.RetryStrategy.Limit.IntValue() != 2 {
				t.Fatal("expected retry limit 2")
			}
			if tmpl.ActiveDeadlineSeconds == nil || tmpl.ActiveDeadlineSeconds.IntValue() != 300 {
				t.Fatal("expected timeout 300s")
			}
			return
		}
	}
	t.Fatal("step-n1 template not found")
}

func TestTranspileDefaultActiveDeadlineSeconds(t *testing.T) {
	p := &Pipeline{
		Name: "default-timeout-test",
		Nodes: []Node{{
			ID:        "n1",
			Component: Component{Name: "n", Image: "busybox:latest"},
		}},
	}
	opts := &Options{Name: "default-timeout-test"}
	wf, err := Transpile(p, opts)
	if err != nil {
		t.Fatal(err)
	}

	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == "step-n1" {
			if tmpl.ActiveDeadlineSeconds == nil {
				t.Fatal("expected default activeDeadlineSeconds")
			}
			if tmpl.ActiveDeadlineSeconds.IntValue() != int(DefaultActiveDeadlineSeconds) {
				t.Fatalf("activeDeadlineSeconds = %d, want %d", tmpl.ActiveDeadlineSeconds.IntValue(), DefaultActiveDeadlineSeconds)
			}
			return
		}
	}
	t.Fatal("step-n1 template not found")
}

func TestTranspileWorkflowParams(t *testing.T) {
	p := &Pipeline{
		Name: "wf-params",
		Nodes: []Node{{
			ID:        "n1",
			Component: Component{Name: "n", Image: "busybox:latest"},
		}},
	}
	opts := &Options{
		Name: "wf-params",
		WorkflowParams: []Param{
			{Name: "asset_ids", Value: "[\"V4ftKjqX\",\"kK9qIAqG\"]"},
			{Name: "threshold", Value: "0.5"},
		},
	}
	wf, err := Transpile(p, opts)
	if err != nil {
		t.Fatal(err)
	}

	if len(wf.Spec.Arguments.Parameters) != 2 {
		t.Fatalf("got %d params, want 2", len(wf.Spec.Arguments.Parameters))
	}

	var names []string
	for _, p := range wf.Spec.Arguments.Parameters {
		names = append(names, p.Name)
		if p.Value == nil || p.Value.String() == "" {
			t.Fatalf("param %q has nil or empty value", p.Name)
		}
	}

	if names[0] != "asset_ids" || names[1] != "threshold" {
		t.Fatalf("params = %v, want [asset_ids threshold]", names)
	}
}

func TestTranspileParallelism(t *testing.T) {
	p := &Pipeline{
		Name:        "parallel-test",
		Parallelism: 3,
		Nodes: []Node{{
			ID:        "n1",
			Component: Component{Name: "n", Image: "busybox:latest"},
		}},
	}
	wf, err := Transpile(p, &Options{Name: "parallel-test"})
	if err != nil {
		t.Fatal(err)
	}
	if wf.Spec.Parallelism == nil {
		t.Fatal("expected Parallelism to be set")
	}
	if *wf.Spec.Parallelism != 3 {
		t.Fatalf("Parallelism = %d, want 3", *wf.Spec.Parallelism)
	}
}

func TestTranspileParallelismZero(t *testing.T) {
	p := &Pipeline{
		Name: "no-parallel-limit",
		Nodes: []Node{{
			ID:        "n1",
			Component: Component{Name: "n", Image: "busybox:latest"},
		}},
	}
	wf, err := Transpile(p, &Options{Name: "no-parallel"})
	if err != nil {
		t.Fatal(err)
	}
	if wf.Spec.Parallelism != nil {
		t.Fatal("expected Parallelism to be nil when not set")
	}
}

func findTemplate(t *testing.T, wf *wfv1.Workflow, name string) wfv1.Template {
	t.Helper()
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == name {
			return tmpl
		}
	}
	t.Fatalf("template %q not found", name)
	return wfv1.Template{}
}

func findDAGTask(t *testing.T, wf *wfv1.Workflow, name string) wfv1.DAGTask {
	t.Helper()
	dag := findTemplate(t, wf, "dag")
	if dag.DAG == nil {
		t.Fatal("dag template has nil DAG")
	}
	for _, task := range dag.DAG.Tasks {
		if task.Name == name {
			return task
		}
	}
	t.Fatalf("dag task %q not found", name)
	return wfv1.DAGTask{}
}
