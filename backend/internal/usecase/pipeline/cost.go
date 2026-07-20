package pipeline

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// provisioning tiers used to key unit rates.
const (
	provisioningStandard = "standard"
	provisioningSpot     = "spot"
)

// PricingConfig holds per-resource unit rates used to price pipeline pods.
//
// Units maps an Argo resourcesDuration resource name to its unit rate per
// provisioning tier, e.g.:
//
//	units:
//	  cpu:            { standard: 0.0300, spot: 0.0100 }  # $/vCPU-hour
//	  nvidia.com/gpu: { standard: 0.9500, spot: 0.3000 }  # $/GPU-hour
//	  memory:         { standard: 0.0004, spot: 0.0001 }  # $/100Mi-hour
//
// The keys match Argo's resourcesDuration keys directly, so pricing degrades
// gracefully: an unknown/unpriced resource contributes $0 rather than a wrong
// fallback. Future accelerators (e.g. nvidia.com/h100) are added as new keys.
type PricingConfig struct {
	Region            string                        `yaml:"region"`
	CalibrationFactor float64                       `yaml:"calibration_factor"`
	Units             map[string]map[string]float64 `yaml:"units"`
}

var (
	pricingMu     sync.Mutex
	pricingByPath = map[string]*PricingConfig{}
)

// LoadPricing reads the GCP pricing YAML and caches it process-wide, keyed by
// path. Successful loads are cached so subsequent calls for the same path skip
// the file read; transient read/parse failures are not cached so a later fixed
// file can still load. Different paths are cached independently.
//
// This is the single load point for pricing; a future online-config source can
// populate the same PricingConfig struct without changing the cost math.
func LoadPricing(path string) (*PricingConfig, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	pricingMu.Lock()
	defer pricingMu.Unlock()
	if cfg, ok := pricingByPath[path]; ok {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read pricing file: %w", err)
	}
	var cfg PricingConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse pricing yaml: %w", err)
	}
	if cfg.CalibrationFactor == 0 {
		cfg.CalibrationFactor = 1.0
	}
	if cfg.Units == nil {
		cfg.Units = map[string]map[string]float64{}
	}
	pricingByPath[path] = &cfg
	return &cfg, nil
}

// resetPricingCache clears the cached pricing (used in tests).
func resetPricingCache() {
	pricingMu.Lock()
	defer pricingMu.Unlock()
	pricingByPath = map[string]*PricingConfig{}
}

// resourcesDurationToCost computes estimated cost in USD by summing each Argo
// resourcesDuration resource against its per-resource unit rate. Deterministic:
// same inputs always produce the same cost.
//
// Argo's resourcesDuration is already expressed as "resource-quantity × seconds"
// in a fixed base unit per resource:
//   - cpu            = cores × seconds        → priced by $/vCPU-hour
//   - nvidia.com/gpu = gpus  × seconds        → priced by $/GPU-hour
//   - memory         = (100Mi-units) × seconds → priced by $/100Mi-hour
//
// So cost = Σ_r resourcesDuration[r] × unit_rate[r][provisioning] / 3600. This
// is why we do NOT convert to wall-clock or multiply by a whole-instance rate:
// core-seconds is the correct quantity to multiply by a per-vCPU-hour rate.
//
// Design notes:
//   - GPU cost is added only when nvidia.com/gpu > 0, so a CPU-only step never
//     inherits GPU pricing even when the node's machine type is unresolved.
//   - provisioning comes from the resolved cloud.google.com/gke-provisioning
//     label (embedded during Argo refresh) and defaults to "standard" when
//     unavailable — an unresolved spot node is over-estimated by ≤3×, never the
//     ~10× a whole-instance GPU fallback produced.
//   - Per-vCPU rates vary only ~1.6× across machine types, so a single blended
//     cpu rate keeps the estimate best-effort without per-cluster node RBAC.
func resourcesDurationToCost(rd map[string]any, pricing *PricingConfig) *float64 {
	if pricing == nil || len(rd) == 0 || len(pricing.Units) == 0 {
		return nil
	}

	provisioning := provisioningStandard
	if v, ok := rd["provisioning"].(string); ok && strings.TrimSpace(v) == provisioningSpot {
		provisioning = provisioningSpot
	}

	var cost float64
	priced := false
	for resource, raw := range rd {
		switch resource {
		case "instance_type", "gpu_type", "provisioning":
			// Metadata keys embedded during Argo refresh, not billable resources.
			continue
		}
		sec, ok := toFloat64(raw)
		if !ok || sec <= 0 {
			continue
		}
		// A positive resource duration means the pod ran; record the cost even
		// when its unit rate is 0 so the snapshot is considered present and the
		// refresh loop does not keep retrying an unpriced node.
		priced = true
		cost += sec * unitRate(pricing, resource, provisioning) / 3600.0
	}
	if !priced {
		return nil
	}

	cost *= pricing.CalibrationFactor
	if math.IsNaN(cost) || math.IsInf(cost, 0) {
		return nil
	}
	return &cost
}

// unitRate returns the $/hour unit rate for a resource at a provisioning tier.
// Falls back to the standard tier when the specific tier is absent, and 0 when
// the resource is not priced (unknown resources contribute nothing).
func unitRate(pricing *PricingConfig, resource, provisioning string) float64 {
	tiers, ok := pricing.Units[resource]
	if !ok {
		return 0
	}
	if rate, ok := tiers[provisioning]; ok {
		return rate
	}
	if rate, ok := tiers[provisioningStandard]; ok {
		return rate
	}
	return 0
}

// isLeafPodNode reports whether a run node is a real executed pod (a leaf) whose
// cost should be counted, as opposed to an aggregate node (DAG, Steps, StepGroup,
// TaskGroup, Retry) whose resourcesDuration is a rollup of its children. Summing
// aggregate nodes on top of their pods double-counts the cost (CYB-3073).
func isLeafPodNode(n models.PipelineRunNode) bool {
	if n.Type == "Pod" {
		return true
	}
	// Tolerate older/blank-typed rows that still carry a pod name.
	return n.Type == "" && n.PodName != ""
}

// ComputeRunCost sums the estimated cost across the leaf (Pod) nodes of a run.
// Aggregate nodes (DAG/Steps/...) are skipped so the total is not double-counted
// and matches the per-step breakdown. Uses the stored per-node cost (computed
// during Argo status refresh) so historical costs are stable.
func ComputeRunCost(run *models.PipelineRun, pricing *PricingConfig) *float64 {
	if run == nil || len(run.Nodes) == 0 {
		return nil
	}
	var total float64
	hasCost := false
	for _, n := range run.Nodes {
		if !isLeafPodNode(n) {
			continue
		}
		if n.EstimatedCostUSD != nil {
			total += *n.EstimatedCostUSD
			hasCost = true
		}
	}
	if !hasCost {
		return nil
	}
	return &total
}

func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case float32:
		return float64(val), true
	case uint:
		return float64(val), true
	case int32:
		return float64(val), true
	case json.Number:
		f, err := val.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
		return f, err == nil
	default:
		return 0, false
	}
}
