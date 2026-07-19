package transpiler

import (
	"fmt"
	"strings"
	"testing"
)

// CYB-3681: step-count guardrails.

func pipelineWithNSteps(n int) *Pipeline {
	p := &Pipeline{Name: "big"}
	for i := 0; i < n; i++ {
		p.Nodes = append(p.Nodes, Node{
			ID:        fmt.Sprintf("n%d", i),
			Component: Component{Name: fmt.Sprintf("c%d", i), Image: "img"},
		})
	}
	return p
}

func TestValidateStepCount_UnderWarnIsSilent(t *testing.T) {
	var warned bool
	orig := stepCountWarnHook
	stepCountWarnHook = func(string, int) { warned = true }
	defer func() { stepCountWarnHook = orig }()

	if problems := validateStepCount(pipelineWithNSteps(stepCountWarn)); len(problems) != 0 {
		t.Fatalf("problems = %v, want none", problems)
	}
	if warned {
		t.Fatal("must not warn at exactly the warn threshold")
	}
}

func TestValidateStepCount_OverWarnFiresHook(t *testing.T) {
	var gotName string
	var gotSteps int
	orig := stepCountWarnHook
	stepCountWarnHook = func(name string, steps int) { gotName, gotSteps = name, steps }
	defer func() { stepCountWarnHook = orig }()

	if problems := validateStepCount(pipelineWithNSteps(stepCountWarn + 1)); len(problems) != 0 {
		t.Fatalf("problems = %v, want none (warn only)", problems)
	}
	if gotName != "big" || gotSteps != stepCountWarn+1 {
		t.Fatalf("warn hook got (%q, %d)", gotName, gotSteps)
	}
}

func TestValidateStepCount_OverRejectFails(t *testing.T) {
	problems := validateStepCount(pipelineWithNSteps(stepCountReject + 1))
	if len(problems) != 1 || !strings.Contains(problems[0], "exceeding the maximum") {
		t.Fatalf("problems = %v, want one reject", problems)
	}
}

// Sub-graph nodes count toward the total (flattenNodes semantics).
func TestValidateStepCount_CountsSubNodes(t *testing.T) {
	p := &Pipeline{Name: "nested"}
	sub := make([]Node, stepCountReject)
	for i := range sub {
		sub[i] = Node{ID: fmt.Sprintf("s%d", i), Component: Component{Name: "c", Image: "img"}}
	}
	p.Nodes = []Node{{ID: "parent", Component: Component{Name: "p", Image: "img"}, SubNodes: sub}}
	if problems := validateStepCount(p); len(problems) != 1 {
		t.Fatalf("problems = %v, want one reject (parent + %d subnodes)", problems, stepCountReject)
	}
}

// The reject surfaces through ValidatePipeline as a ValidationError, so batch
// dispatch classifies it permanent ("invalid pipeline" marker).
func TestValidatePipeline_StepCountReject(t *testing.T) {
	err := ValidatePipeline(pipelineWithNSteps(stepCountReject + 1))
	if err == nil || !strings.Contains(err.Error(), "invalid pipeline") {
		t.Fatalf("err = %v, want ValidationError", err)
	}
}
