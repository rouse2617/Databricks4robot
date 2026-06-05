package pipeline

import "testing"

func TestResourcesDurationToCostRequiresExplicitPricingMetadata(t *testing.T) {
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

	if got := resourcesDurationToCost(map[string]any{
		"cpu": 25,
	}, pricing); got != nil {
		t.Fatalf("expected nil cost when instance metadata is missing, got %v", *got)
	}

	if got := resourcesDurationToCost(map[string]any{
		"memory": 3_000_000_000,
	}, pricing); got != nil {
		t.Fatalf("expected nil cost for memory-only resourcesDuration, got %v", *got)
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

func TestResourcesDurationToCostUsesExplicitGPUProfile(t *testing.T) {
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
		"cpu":            20,
		"nvidia.com/gpu": 25,
		"instance_type":  "g2-standard-16",
		"gpu_type":       "nvidia-l4",
		"provisioning":   "standard",
	}, pricing)
	if got == nil {
		t.Fatal("expected gpu cost")
	}
	want := 25.0 * 1.20 / 3600.0
	if diff := *got - want; diff < -0.0000001 || diff > 0.0000001 {
		t.Fatalf("cost = %v, want %v", *got, want)
	}
}
