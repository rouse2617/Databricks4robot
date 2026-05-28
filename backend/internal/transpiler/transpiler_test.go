package transpiler

import (
	"testing"
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
	if *wf.Spec.TTLStrategy.SecondsAfterCompletion != 3600 {
		t.Fatalf("ttl = %d, want 3600", *wf.Spec.TTLStrategy.SecondsAfterCompletion)
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
