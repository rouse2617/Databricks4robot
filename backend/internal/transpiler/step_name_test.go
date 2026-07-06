package transpiler

import (
	"regexp"
	"testing"
)

var rfc1123Label = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

func TestStepSlug(t *testing.T) {
	cases := map[string]string{
		"head-track-pycuvslam":   "head-track-pycuvslam",
		"Head Track (pyCUVSLAM)": "head-track-pycuvslam",
		"  Transcode__Video  ":   "transcode-video",
		"":                       "",
		"a-really-long-component-name-that-goes-way-past-thirty": "a-really-long-component-name-t",
	}
	for in, want := range cases {
		if got := stepSlug(in); got != want {
			t.Errorf("stepSlug(%q) = %q, want %q", in, got, want)
		}
		if got := stepSlug(in); got != "" && !rfc1123Label.MatchString(got) {
			t.Errorf("stepSlug(%q) = %q is not an RFC1123 label", in, got)
		}
	}
}

func TestNodeUUID8AndHex(t *testing.T) {
	id := "node-8242f006-694e-4c17-945c-fdc485c50ff1"
	if got := nodeUUID8(id); got != "8242f006" {
		t.Errorf("nodeUUID8 = %q, want 8242f006", got)
	}
	if got := nodeUUIDHex(id); got != "8242f006694e4c17945cfdc485c50ff1" {
		t.Errorf("nodeUUIDHex = %q", got)
	}
	if got := nodeUUID8("abc"); got != "abc" {
		t.Errorf("nodeUUID8(short) = %q, want abc", got)
	}
}

func TestBuildStepTemplateNames(t *testing.T) {
	nodes := []Node{
		{ID: "node-8242f006-694e-4c17-945c-fdc485c50ff1", Component: Component{Name: "head-track-pycuvslam"}},
		{ID: "node-11111111-1111-1111-1111-111111111111", Component: Component{Name: "transcode"}},
		{ID: "node-22222222-2222-2222-2222-222222222222", Component: Component{Name: "transcode"}}, // dup component
		{ID: "node-33333333-3333-3333-3333-333333333333", Component: Component{Name: ""}},          // empty -> legacy
	}
	names := buildStepTemplateNames(nodes)

	if got := names[nodes[0].ID]; got != "step-head-track-pycuvslam" {
		t.Errorf("unique component name = %q, want step-head-track-pycuvslam", got)
	}
	// Duplicate component names must each get a distinct uuid8-suffixed name.
	a, b := names[nodes[1].ID], names[nodes[2].ID]
	if a == b {
		t.Errorf("duplicate components got the same template name %q", a)
	}
	if a != "step-transcode-11111111" || b != "step-transcode-22222222" {
		t.Errorf("dup names = %q, %q", a, b)
	}
	if got := names[nodes[3].ID]; got != "step-node-33333333-3333-3333-3333-333333333333" {
		t.Errorf("empty component fallback = %q", got)
	}
	for id, name := range names {
		if !rfc1123Label.MatchString(name) {
			t.Errorf("template name %q for %q is not an RFC1123 label", name, id)
		}
	}
}

func TestStepCandidateKeys(t *testing.T) {
	id := "node-8242f006-694e-4c17-945c-fdc485c50ff1"
	keys := StepCandidateKeys("head-track-pycuvslam", id)
	want := map[string]bool{
		"step-node-8242f006-694e-4c17-945c-fdc485c50ff1":             true, // legacy
		"step-head-track-pycuvslam":                                  true, // unique
		"step-head-track-pycuvslam-8242f006":                         true, // dup
		"step-head-track-pycuvslam-8242f006694e4c17945cfdc485c50ff1": true, // full-hex fallback
	}
	if len(keys) != len(want) {
		t.Fatalf("keys = %v", keys)
	}
	for _, k := range keys {
		if !want[k] {
			t.Errorf("unexpected candidate key %q", k)
		}
	}

	// Empty component -> only the legacy key.
	if got := StepCandidateKeys("", id); len(got) != 1 || got[0] != "step-node-8242f006-694e-4c17-945c-fdc485c50ff1" {
		t.Errorf("empty-component candidates = %v", got)
	}
}

// TestTranspileReadableNamesAndOutputWiring verifies that transpiled templates
// carry the readable step-<component> names, that a downstream task references
// its upstream by the same readable name (the transpiler.go:620 fix), and that
// two nodes sharing a component name still transpile with distinct names.
func TestTranspileReadableNamesAndOutputWiring(t *testing.T) {
	p := &Pipeline{
		Name: "readable",
		Nodes: []Node{
			{
				ID:        "node-aaaaaaaa-0000-0000-0000-000000000001",
				Component: Component{Name: "extract", Image: "busybox:latest", Command: []string{"sh", "-c"}, Args: []Argument{{Name: "script", Value: "echo hi > /tmp/outputs/out"}}},
				Outputs:   []Port{{Name: "out", Type: "string"}},
			},
			{
				ID:        "node-bbbbbbbb-0000-0000-0000-000000000002",
				Component: Component{Name: "extract", Image: "busybox:latest", Command: []string{"sh", "-c"}, Args: []Argument{{Name: "script", Value: "echo hi > /tmp/outputs/out"}}},
				Outputs:   []Port{{Name: "out", Type: "string"}},
			},
			{
				ID:        "node-cccccccc-0000-0000-0000-000000000003",
				Component: Component{Name: "join", Image: "busybox:latest", Command: []string{"sh", "-c"}, Args: []Argument{{Name: "script", Value: "echo {{inputs.parameters.left}}"}}},
				Inputs:    []Port{{Name: "left", Type: "string"}},
			},
		},
		Edges: []Edge{{Source: "node-aaaaaaaa-0000-0000-0000-000000000001.out", Target: "node-cccccccc-0000-0000-0000-000000000003.left"}},
	}

	wf, err := Transpile(p, &Options{Name: "readable", Namespace: "default"})
	if err != nil {
		t.Fatalf("transpile: %v", err)
	}

	names := map[string]bool{}
	for _, tmpl := range wf.Spec.Templates {
		names[tmpl.Name] = true
	}
	if !names["step-extract-aaaaaaaa"] || !names["step-extract-bbbbbbbb"] {
		t.Fatalf("expected distinct dup-component templates, got %v", names)
	}
	if !names["step-join"] {
		t.Fatalf("expected step-join template, got %v", names)
	}

	// The join task must reference extract's output by the readable name.
	for i := range wf.Spec.Templates {
		if wf.Spec.Templates[i].DAG == nil {
			continue
		}
		for _, task := range wf.Spec.Templates[i].DAG.Tasks {
			if task.Name != "step-join" {
				continue
			}
			if len(task.Arguments.Parameters) != 1 {
				t.Fatalf("join args = %+v", task.Arguments.Parameters)
			}
			got := task.Arguments.Parameters[0].Value.String()
			want := "{{tasks.step-extract-aaaaaaaa.outputs.parameters.out}}"
			if got != want {
				t.Fatalf("output-param ref = %q, want %q", got, want)
			}
			return
		}
	}
	t.Fatal("step-join DAG task not found")
}
