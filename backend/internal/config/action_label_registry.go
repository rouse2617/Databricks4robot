package config

import (
	"fmt"
	"os"
	"sort"
	"sync"

	"gopkg.in/yaml.v3"
)

type ActionLabelRegistry struct {
	mu            sync.RWMutex
	primaryLabels map[string]struct{}
	labels        map[string]struct{}
	path          string
}

type actionLabelRegistryFile struct {
	PrimaryLabels []string `yaml:"primary_labels"`
	Labels        []string `yaml:"labels"`
}

func LoadActionLabelRegistry(path string) (*ActionLabelRegistry, error) {
	primary, labels, err := loadActionLabelsFromFile(path)
	if err != nil {
		return nil, err
	}
	return &ActionLabelRegistry{
		primaryLabels: primary,
		labels:        labels,
		path:          path,
	}, nil
}

func loadActionLabelsFromFile(path string) (map[string]struct{}, map[string]struct{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("action_label_registry: read file %s: %w", path, err)
	}

	var f actionLabelRegistryFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, nil, fmt.Errorf("action_label_registry: parse yaml: %w", err)
	}
	if len(f.Labels) == 0 {
		return nil, nil, fmt.Errorf("action_label_registry: labels must not be empty")
	}

	primary := make(map[string]struct{}, len(f.PrimaryLabels))
	for _, v := range f.PrimaryLabels {
		primary[v] = struct{}{}
	}
	labels := make(map[string]struct{}, len(f.Labels))
	for _, v := range f.Labels {
		labels[v] = struct{}{}
	}
	return primary, labels, nil
}

func (r *ActionLabelRegistry) Reload() error {
	primary, labels, err := loadActionLabelsFromFile(r.path)
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.primaryLabels = primary
	r.labels = labels
	r.mu.Unlock()
	return nil
}

func (r *ActionLabelRegistry) Validate(primary string, labels []string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, label := range labels {
		if _, ok := r.labels[label]; !ok {
			return fmt.Errorf("action_label_registry: label %q not registered", label)
		}
	}
	if primary != "" {
		if _, ok := r.primaryLabels[primary]; !ok {
			return fmt.Errorf("action_label_registry: primary_label %q not registered", primary)
		}
	}
	return nil
}

func (r *ActionLabelRegistry) PrimaryLabels() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.primaryLabels))
	for k := range r.primaryLabels {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func (r *ActionLabelRegistry) Labels() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.labels))
	for k := range r.labels {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
