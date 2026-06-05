package pipeline

import (
	"fmt"
	"math"
	"os"
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
	pricingCache     *PricingConfig
	pricingCacheOnce sync.Once
	pricingCacheErr  error
)

// LoadPricing reads the GCP pricing YAML and caches it process-wide.
// Returns the cached config on subsequent calls without re-reading the file.
func LoadPricing(path string) (*PricingConfig, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	pricingCacheOnce.Do(func() {
		data, err := os.ReadFile(path)
		if err != nil {
			pricingCacheErr = fmt.Errorf("read pricing file: %w", err)
			return
		}
		var cfg PricingConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			pricingCacheErr = fmt.Errorf("parse pricing yaml: %w", err)
			return
		}
		if cfg.CalibrationFactor == 0 {
			cfg.CalibrationFactor = 1.0
		}
		if cfg.Prices == nil {
			cfg.Prices = map[string]any{}
		}
		pricingCache = &cfg
	})
	return pricingCache, pricingCacheErr
}

// resetPricingCache clears the cached pricing (used in tests).
func resetPricingCache() {
	pricingCacheOnce = sync.Once{}
	pricingCache = nil
	pricingCacheErr = nil
}

// resourcesDurationToCost computes estimated cost in USD from Argo's
// resourcesDuration map. The function is deterministic: same inputs always
// produce the same cost.
//
// The resourcesDuration keys are resource names ("cpu", "memory",
// "nvidia.com/gpu") plus optional pricing metadata fields. Argo's
// resourcesDuration is only an indicative runtime proxy, so we estimate cost
// conservatively:
//   - only CPU / GPU durations are treated as runtime seconds
//   - pricing requires explicit instance metadata
//   - memory-only durations do not produce a node cost estimate
func resourcesDurationToCost(rd map[string]any, pricing *PricingConfig) *float64 {
	if pricing == nil || len(rd) == 0 {
		return nil
	}

	instanceType, gpuType, provisioning, ok := pricingProfileFromResourceDuration(rd)
	if !ok {
		return nil
	}

	hourlyRate := lookupHourlyRate(pricing, instanceType, gpuType, provisioning)
	if hourlyRate == nil {
		return nil
	}

	totalSec, ok := runtimeSecondsFromResourceDuration(rd, gpuType)
	if !ok || totalSec <= 0 {
		return nil
	}

	cost := totalSec * (*hourlyRate) / 3600.0
	if math.IsNaN(cost) || math.IsInf(cost, 0) {
		return nil
	}
	return &cost
}

func pricingProfileFromResourceDuration(rd map[string]any) (instanceType, gpuType, provisioning string, ok bool) {
	instanceType, _ = rd["instance_type"].(string)
	instanceType = strings.TrimSpace(instanceType)
	if instanceType == "" {
		return "", "", "", false
	}

	gpuType, _ = rd["gpu_type"].(string)
	gpuType = strings.TrimSpace(gpuType)
	if gpuType == "" {
		gpuType = "none"
	}

	provisioning, _ = rd["provisioning"].(string)
	provisioning = strings.TrimSpace(provisioning)
	if provisioning == "" {
		provisioning = "standard"
	}

	return instanceType, gpuType, provisioning, true
}

func runtimeSecondsFromResourceDuration(rd map[string]any, gpuType string) (float64, bool) {
	cpuSec, cpuOK := positiveFloat64(rd["cpu"])
	gpuSec, gpuOK := positiveFloat64(rd["nvidia.com/gpu"])

	if gpuType != "" && gpuType != "none" {
		switch {
		case gpuOK && cpuOK:
			if gpuSec > cpuSec {
				return gpuSec, true
			}
			return cpuSec, true
		case gpuOK:
			return gpuSec, true
		case cpuOK:
			return cpuSec, true
		default:
			return 0, false
		}
	}

	if cpuOK {
		return cpuSec, true
	}
	return 0, false
}

func positiveFloat64(v any) (float64, bool) {
	value, ok := toFloat64(v)
	if !ok || value <= 0 {
		return 0, false
	}
	return value, true
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
	default:
		return 0, false
	}
}
