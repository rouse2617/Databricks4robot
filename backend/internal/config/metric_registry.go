package config

import (
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// MetricDefinition defines a single metric entry in the registry.
type MetricDefinition struct {
	Key                string `yaml:"key" json:"key"`
	DisplayName        string `yaml:"display_name" json:"display_name"`
	MetricType         string `yaml:"metric_type" json:"metric_type"`
	MetricUnit         string `yaml:"metric_unit" json:"metric_unit"`
	TargetType         string `yaml:"target_type" json:"target_type"`
	HigherIsBetter     bool   `yaml:"higher_is_better" json:"higher_is_better"`
	DefaultAggregation string `yaml:"default_aggregation" json:"default_aggregation"`
	Queryable          bool   `yaml:"queryable" json:"queryable"`
	Description        string `yaml:"description" json:"description"`
}

// MetricRegistry holds the registered metrics loaded from YAML.
type MetricRegistry struct {
	mu      sync.RWMutex
	metrics map[string]MetricDefinition
	path    string
}

type metricRegistryFile struct {
	Metrics []MetricDefinition `yaml:"metrics"`
}

// LoadMetricRegistry loads the metric registry from a YAML file.
func LoadMetricRegistry(path string) (*MetricRegistry, error) {
	m, err := loadMetricsFromFile(path)
	if err != nil {
		return nil, err
	}
	return &MetricRegistry{metrics: m, path: path}, nil
}

func loadMetricsFromFile(path string) (map[string]MetricDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("metric_registry: read %s: %w", path, err)
	}
	var f metricRegistryFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("metric_registry: parse %s: %w", path, err)
	}
	m := make(map[string]MetricDefinition, len(f.Metrics))
	for _, def := range f.Metrics {
		m[def.Key] = def
	}
	return m, nil
}

// IsQueryable returns true if the key is registered and queryable=true.
func (r *MetricRegistry) IsQueryable(key string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	def, ok := r.metrics[key]
	return ok && def.Queryable
}

// Get returns the definition for a metric key.
func (r *MetricRegistry) Get(key string) (MetricDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	def, ok := r.metrics[key]
	return def, ok
}

// All returns a copy of all metric definitions, sorted by key.
func (r *MetricRegistry) All() []MetricDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]MetricDefinition, 0, len(r.metrics))
	for _, def := range r.metrics {
		out = append(out, def)
	}
	return out
}
