package pipeline

import (
	"math"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// testPricing returns a per-resource unit-rate config mirroring gcp_pricing.yaml.
func testPricing() *PricingConfig {
	return &PricingConfig{
		CalibrationFactor: 1,
		Units: map[string]map[string]float64{
			"cpu":            {"standard": 0.0300, "spot": 0.0153},
			"nvidia.com/gpu": {"standard": 0.5600, "spot": 0.2862},
			"memory":         {"standard": 0.0004, "spot": 0.0002},
		},
	}
}

func approxEqual(got, want float64) bool {
	return math.Abs(got-want) <= 1e-9
}

// TestResourcesDurationToCostCPUOnlyPricedByVCPURate is the CYB-3705 regression
// guard: a CPU-only step whose machine type cannot be resolved must be priced by
// the vCPU rate, NOT the GPU rate, and cpu core-seconds must not be treated as
// wall-clock.
func TestResourcesDurationToCostCPUOnlyPricedByVCPURate(t *testing.T) {
	pricing := testPricing()

	// 4-core delivery pod, ~5033s wall → ~20132 cpu core-seconds. No instance
	// metadata embedded (unresolved cross-cluster node).
	got := resourcesDurationToCost(map[string]any{"cpu": 20132.0}, pricing)
	if got == nil {
		t.Fatal("expected a cost for a cpu-only node")
	}
	want := 20132.0 * 0.0300 / 3600.0 // ≈ $0.1678
	if !approxEqual(*got, want) {
		t.Fatalf("cost = %v, want %v (vCPU rate, no GPU fallback)", *got, want)
	}
	// Must be an order of magnitude below the old GPU-rate result ($6.71).
	oldGPURate := 20132.0 * 1.20 / 3600.0
	if *got >= oldGPURate/5 {
		t.Fatalf("cost %v not far enough below old GPU-rate estimate %v", *got, oldGPURate)
	}
}

// TestResourcesDurationToCostGPUStepAddsAcceleratorCost verifies a GPU step is
// priced as cpu core-seconds + gpu-seconds, each at its own unit rate.
func TestResourcesDurationToCostGPUStepAddsAcceleratorCost(t *testing.T) {
	pricing := testPricing()

	// 8 vCPU × 500s = 4000 core-sec; 1 GPU × 500s = 500 gpu-sec.
	got := resourcesDurationToCost(map[string]any{
		"cpu":            4000.0,
		"nvidia.com/gpu": 500.0,
		"memory":         1000.0,
	}, pricing)
	if got == nil {
		t.Fatal("expected a cost for a gpu node")
	}
	want := 4000.0*0.0300/3600.0 + 500.0*0.5600/3600.0 + 1000.0*0.0004/3600.0
	if !approxEqual(*got, want) {
		t.Fatalf("cost = %v, want %v (cpu + gpu + memory)", *got, want)
	}
}

// TestResourcesDurationToCostSpotCheaperThanStandard verifies the provisioning
// axis discounts spot work.
func TestResourcesDurationToCostSpotCheaperThanStandard(t *testing.T) {
	pricing := testPricing()

	standard := resourcesDurationToCost(map[string]any{"cpu": 1000.0, "provisioning": "standard"}, pricing)
	spot := resourcesDurationToCost(map[string]any{"cpu": 1000.0, "provisioning": "spot"}, pricing)
	if standard == nil || spot == nil {
		t.Fatal("expected costs for both provisioning tiers")
	}
	if !(*spot < *standard) {
		t.Fatalf("spot cost %v should be below standard cost %v", *spot, *standard)
	}
	if !approxEqual(*spot, 1000.0*0.0153/3600.0) {
		t.Fatalf("spot cost = %v, want %v", *spot, 1000.0*0.0153/3600.0)
	}
}

// TestResourcesDurationToCostProvisioningDefaultsStandard verifies an unresolved
// node (no provisioning key) is priced at the standard tier.
func TestResourcesDurationToCostProvisioningDefaultsStandard(t *testing.T) {
	pricing := testPricing()

	got := resourcesDurationToCost(map[string]any{"cpu": 1000.0}, pricing)
	if got == nil {
		t.Fatal("expected a cost")
	}
	if !approxEqual(*got, 1000.0*0.0300/3600.0) {
		t.Fatalf("cost = %v, want standard-tier %v", *got, 1000.0*0.0300/3600.0)
	}
}

// TestResourcesDurationToCostMemoryOnlyStillPriced verifies a short pod that
// reports cpu=0 but positive memory still gets a (tiny) non-nil cost, so the
// cost-snapshot-missing check does not loop.
func TestResourcesDurationToCostMemoryOnlyStillPriced(t *testing.T) {
	pricing := testPricing()

	got := resourcesDurationToCost(map[string]any{"cpu": 0, "memory": 3.0}, pricing)
	if got == nil {
		t.Fatal("expected a non-nil cost for a memory-only node")
	}
	if !approxEqual(*got, 3.0*0.0004/3600.0) {
		t.Fatalf("cost = %v, want %v", *got, 3.0*0.0004/3600.0)
	}
}

// TestResourcesDurationToCostUnknownResourceContributesZero verifies pricing
// degrades gracefully: an unpriced resource adds nothing rather than erroring or
// falling back to a wrong rate.
func TestResourcesDurationToCostUnknownResourceContributesZero(t *testing.T) {
	pricing := testPricing()

	got := resourcesDurationToCost(map[string]any{
		"cpu":               100.0,
		"ephemeral-storage": 9999.0,
	}, pricing)
	if got == nil {
		t.Fatal("expected a cost")
	}
	if !approxEqual(*got, 100.0*0.0300/3600.0) {
		t.Fatalf("cost = %v, want cpu-only %v (unknown resource priced at 0)", *got, 100.0*0.0300/3600.0)
	}
}

// TestResourcesDurationToCostCalibrationFactor verifies the calibration factor
// scales the total (used to fold in CUD/SUD from a real bill).
func TestResourcesDurationToCostCalibrationFactor(t *testing.T) {
	pricing := testPricing()
	pricing.CalibrationFactor = 0.8

	got := resourcesDurationToCost(map[string]any{"cpu": 1000.0}, pricing)
	if got == nil {
		t.Fatal("expected a cost")
	}
	if !approxEqual(*got, 1000.0*0.0300/3600.0*0.8) {
		t.Fatalf("cost = %v, want calibrated %v", *got, 1000.0*0.0300/3600.0*0.8)
	}
}

func TestResourcesDurationToCostNilOrEmpty(t *testing.T) {
	if resourcesDurationToCost(map[string]any{"cpu": 100.0}, nil) != nil {
		t.Fatal("expected nil cost when pricing is nil")
	}
	if resourcesDurationToCost(map[string]any{}, testPricing()) != nil {
		t.Fatal("expected nil cost for empty resourcesDuration")
	}
	if resourcesDurationToCost(map[string]any{"cpu": 100.0}, &PricingConfig{CalibrationFactor: 1}) != nil {
		t.Fatal("expected nil cost when no units are configured")
	}
	// Only metadata keys, no billable resource → nil.
	if resourcesDurationToCost(map[string]any{"provisioning": "spot", "instance_type": "c3d"}, testPricing()) != nil {
		t.Fatal("expected nil cost when only metadata keys are present")
	}
}

// TestShippedPricingYAMLPricesDeliveryNodeSanely loads the actual shipped
// gcp_pricing.yaml and prices the exact resourcesDuration of the CYB-3705 sample
// run (run 35961808-…, step-sea-sky-delivery on delivery-clust-t2d-pool: a 4-core
// pod, ~5033s wall, no GPU). Before the fix this node was priced at $6.7567 (GPU
// rate × cpu core-seconds); it must now land in a sane CPU range around $0.26.
func TestShippedPricingYAMLPricesDeliveryNodeSanely(t *testing.T) {
	resetPricingCache()
	t.Cleanup(resetPricingCache)

	pricing, err := LoadPricing("../../../config/gcp_pricing.yaml")
	if err != nil {
		t.Fatalf("load shipped pricing yaml: %v", err)
	}
	if pricing == nil || len(pricing.Units) == 0 {
		t.Fatal("shipped pricing yaml has no units")
	}

	// Real resourcesDuration pulled from the live dev run (no gpu key; the node's
	// machine type is unresolvable cross-cluster so provisioning defaults to
	// standard). cpu is core-seconds (20270 = ~4.03 cores × 5033s wall).
	got := resourcesDurationToCost(map[string]any{"cpu": 20270.0, "memory": 814970.0}, pricing)
	if got == nil {
		t.Fatal("expected a cost from shipped pricing")
	}
	if *got < 0.15 || *got > 0.40 {
		t.Fatalf("delivery node cost = $%.4f, want ~$0.26 CPU-tier range (pre-fix was $6.7567)", *got)
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
