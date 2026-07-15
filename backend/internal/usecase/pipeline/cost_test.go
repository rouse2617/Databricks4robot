package pipeline

import (
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

func TestResourcesDurationToCostUsesMemoryWhenCPUIsZero(t *testing.T) {
	pricing := &PricingConfig{
		CalibrationFactor: 1,
		Prices: map[string]any{
			"g2-standard-16": map[string]any{
				"nvidia-l4": map[string]any{
					"standard": 1.20,
				},
			},
		},
	}

	got := resourcesDurationToCost(map[string]any{
		"cpu":    0,
		"memory": 3,
	}, pricing)
	if got == nil {
		t.Fatal("expected cost for memory-only resourcesDuration")
	}
	want := 3.0 * 1.20 / 3600.0
	if diff := *got - want; diff < -0.0000001 || diff > 0.0000001 {
		t.Fatalf("cost = %v, want %v", *got, want)
	}
}

func TestResourcesDurationToCostUsesExplicitInstanceMetadata(t *testing.T) {
	pricing := &PricingConfig{
		CalibrationFactor: 1,
		Prices: map[string]any{
			"t2d-standard-8": map[string]any{
				"none": map[string]any{
					"spot": 0.09,
				},
			},
		},
	}

	got := resourcesDurationToCost(map[string]any{
		"cpu":           10,
		"instance_type": "t2d-standard-8",
		"gpu_type":      "none",
		"provisioning":  "spot",
	}, pricing)
	if got == nil {
		t.Fatal("expected cost")
	}
	want := 10.0 * 0.09 / 3600.0
	if diff := *got - want; diff < -0.0000001 || diff > 0.0000001 {
		t.Fatalf("cost = %v, want %v", *got, want)
	}
}

func TestComputeRunCostSumsLeafPodsOnly(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	run := &models.PipelineRun{Nodes: []models.PipelineRunNode{
		{Type: "DAG", EstimatedCostUSD: f(0.16)},               // aggregate rollup — must be skipped
		{Type: "Pod", EstimatedCostUSD: f(0.10)},               // leaf
		{Type: "Pod", EstimatedCostUSD: f(0.13)},               // leaf
		{Type: "Steps", EstimatedCostUSD: f(0.05)},             // aggregate — skipped
		{Type: "", PodName: "wf.x", EstimatedCostUSD: f(0.02)}, // blank-typed pod — counted
	}}
	got := ComputeRunCost(run, &PricingConfig{})
	if got == nil {
		t.Fatal("expected a cost")
	}
	want := 0.25 // 0.10 + 0.13 + 0.02, DAG/Steps excluded
	if *got < want-1e-9 || *got > want+1e-9 {
		t.Fatalf("ComputeRunCost = %v, want %v (leaf pods only)", *got, want)
	}
}
