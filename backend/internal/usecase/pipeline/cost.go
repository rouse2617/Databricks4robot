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

// PricingConfig maps GCP machine types to hourly USD rates.
type PricingConfig struct {
	Region            string         `yaml:"region"`
	CalibrationFactor float64        `yaml:"calibration_factor"`
	Prices            map[string]any `yaml:"prices"`
}

var (
	pricingMu     sync.Mutex
	pricingByPath = map[string]*PricingConfig{}
)

// LoadPricing reads the GCP pricing YAML and caches it process-wide, keyed by
// path. Successful loads are cached so subsequent calls for the same path skip
// the file read; transient read/parse failures are not cached so a later fixed
// file can still load. Different paths are cached independently.
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
	if cfg.Prices == nil {
		cfg.Prices = map[string]any{}
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

// resourcesDurationToCost computes estimated cost in USD from Argo's
// resourcesDuration map. The function is deterministic: same inputs always
// produce the same cost.
//
// The resourcesDuration keys are resource names ("cpu", "memory",
// "nvidia.com/gpu") with values in seconds of usage. Pricing lookup uses
// the instance_type and provisioning_mode keys when present; when absent
// it falls back to the "g2-standard-16" / "nvidia-l4" / "standard" path
// as the default GPU pipeline configuration.
//
// Costing model and its known approximations (intentional — see decision A1):
//   - cost = dominantResourceSeconds * instanceHourlyRate / 3600. We bill the
//     whole instance by wall-clock, using one resource's duration (GPU > CPU >
//     max) as a wall-clock proxy rather than summing cpu+memory+gpu durations.
//   - When the node carries no embedded instance_type/gpu_type, we default to
//     the GPU node pool rate. A CPU-only step therefore gets priced at the GPU
//     instance hourly rate, which OVERESTIMATES non-GPU work. This is accepted
//     for now; tighten by embedding per-node instance info during Argo refresh.
func resourcesDurationToCost(rd map[string]any, pricing *PricingConfig) *float64 {
	if pricing == nil || len(rd) == 0 {
		return nil
	}

	// Default instance specs for the primary GPU node pool.
	instanceType := "g2-standard-16"
	gpuType := "nvidia-l4"
	provisioning := "standard"

	// If the node has explicit instance info embedded (set during Argo refresh),
	// use that instead. For now we default to the GPU pool.
	if v, ok := rd["instance_type"].(string); ok && v != "" {
		instanceType = v
	}
	if v, ok := rd["gpu_type"].(string); ok && v != "" {
		gpuType = v
	}
	if v, ok := rd["provisioning"].(string); ok && v != "" {
		provisioning = v
	}

	hourlyRate := lookupHourlyRate(pricing, instanceType, gpuType, provisioning)
	if hourlyRate == nil {
		return nil
	}

	// Argo tracks each resource independently. Use the dominant resource
	// duration as wall-clock runtime; short pods can report cpu=0 while memory
	// still has a positive duration.
	var totalSec float64
	if gpuSec, ok := toFloat64(rd["nvidia.com/gpu"]); ok && gpuSec > 0 {
		totalSec = gpuSec
	} else if cpuSec, ok := toFloat64(rd["cpu"]); ok && cpuSec > 0 {
		totalSec = cpuSec
	} else if maxSec := maxResourceDurationSeconds(rd); maxSec > 0 {
		totalSec = maxSec
	} else {
		return nil
	}

	cost := totalSec * (*hourlyRate) / 3600.0
	if math.IsNaN(cost) || math.IsInf(cost, 0) {
		return nil
	}
	return &cost
}

func maxResourceDurationSeconds(rd map[string]any) float64 {
	var maxSec float64
	for key, raw := range rd {
		switch key {
		case "instance_type", "gpu_type", "provisioning":
			continue
		}
		sec, ok := toFloat64(raw)
		if ok && sec > maxSec {
			maxSec = sec
		}
	}
	return maxSec
}

// lookupHourlyRate resolves (instance_type, gpu_type, provisioning) → $/hr.
// Returns nil when the combination is not found in pricing.
func lookupHourlyRate(pricing *PricingConfig, instanceType, gpuType, provisioning string) *float64 {
	prices := pricing.Prices
	if prices == nil {
		return nil
	}

	instBlockRaw, ok := prices[instanceType]
	if !ok {
		return nil
	}
	instBlock, ok := instBlockRaw.(map[string]any)
	if !ok {
		return nil
	}

	gpuBlockRaw, ok := instBlock[gpuType]
	if !ok {
		gpuBlockRaw, ok = instBlock["none"]
		if !ok {
			return nil
		}
	}
	gpuBlock, ok := gpuBlockRaw.(map[string]any)
	if !ok {
		return nil
	}

	rateRaw, ok := gpuBlock[provisioning]
	if !ok {
		return nil
	}
	rate, ok := toFloat64(rateRaw)
	if !ok {
		return nil
	}

	result := rate * pricing.CalibrationFactor
	return &result
}

// ComputeRunCost sums the estimated cost across all nodes in a pipeline run.
// Uses the stored cost on each node (computed during Argo status refresh)
// rather than recomputing from resourcesDuration so historical costs are stable.
func ComputeRunCost(run *models.PipelineRun, pricing *PricingConfig) *float64 {
	if run == nil || len(run.Nodes) == 0 {
		return nil
	}
	var total float64
	hasCost := false
	for _, n := range run.Nodes {
		cost := n.EstimatedCostUSD
		if cost != nil {
			total += *cost
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
