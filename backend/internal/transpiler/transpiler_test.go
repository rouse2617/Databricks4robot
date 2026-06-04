package transpiler

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestTranspileEdgePorts(t *testing.T) {
	p := &Pipeline{
		Name: "test",
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
					Args:    []Argument{{Name: "script", Value: "echo {{inputs.parameters.in}}"}},
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
}

func TestTranspileRejectsDuplicateTargetInputs(t *testing.T) {
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

	_, err := Transpile(p, &Options{Name: "dup-target"})
	if err == nil {
		t.Fatal("expected duplicate target input error")
	}
	if !strings.Contains(err.Error(), "join.input") {
		t.Fatalf("error = %q, want join.input detail", err.Error())
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

func TestTranspileRejectsConsumedOutputWithoutFileWrite(t *testing.T) {
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
	if err == nil {
		t.Fatal("expected consumed output file error")
	}
	if !strings.Contains(err.Error(), "/tmp/outputs/output") {
		t.Fatalf("error = %q, want output path detail", err.Error())
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
