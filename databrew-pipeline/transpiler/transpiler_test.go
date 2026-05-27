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
					Name:  "a",
					Image: "busybox:latest",
					Command: []string{"sh", "-c"},
					Args: []Argument{{Name: "script", Value: "echo a"}},
				},
				Outputs: []Port{{Name: "out", Type: "string"}},
			},
			{
				ID: "b",
				Component: Component{
					Name:  "b",
					Image: "busybox:latest",
					Command: []string{"sh", "-c"},
					Args: []Argument{{Name: "script", Value: "echo {{inputs.parameters.in}}"}},
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
			ID: "n1",
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
