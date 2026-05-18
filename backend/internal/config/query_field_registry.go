package config

import (
	"fmt"
	"os"
	"slices"
	"sort"
	"sync"

	"gopkg.in/yaml.v3"
)

type QueryFieldDefinition struct {
	Field         string   `yaml:"field" json:"field"`
	FilterEngines []string `yaml:"filter_engines" json:"filter_engines"`
	SortEngines   []string `yaml:"sort_engines" json:"sort_engines"`
	FacetEngines  []string `yaml:"facet_engines" json:"facet_engines"`
}

func (d QueryFieldDefinition) SupportedEngines() []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(d.FilterEngines)+len(d.SortEngines)+len(d.FacetEngines))
	for _, group := range [][]string{d.FilterEngines, d.SortEngines, d.FacetEngines} {
		for _, engine := range group {
			if engine == "" {
				continue
			}
			if _, ok := seen[engine]; ok {
				continue
			}
			seen[engine] = struct{}{}
			out = append(out, engine)
		}
	}
	sort.Strings(out)
	return out
}

type QueryFieldRegistry struct {
	mu        sync.RWMutex
	resources map[string]map[string]QueryFieldDefinition
	path      string
}

type queryFieldRegistryFile struct {
	SchemaVersion int `yaml:"schema_version"`
	Resources     map[string]struct {
		Fields []QueryFieldDefinition `yaml:"fields"`
	} `yaml:"resources"`
}

func LoadQueryFieldRegistry(path string) (*QueryFieldRegistry, error) {
	resources, err := loadQueryFieldsFromFile(path)
	if err != nil {
		return nil, err
	}
	return &QueryFieldRegistry{resources: resources, path: path}, nil
}

func loadQueryFieldsFromFile(path string) (map[string]map[string]QueryFieldDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("query_field_registry: read file %s: %w", path, err)
	}

	var f queryFieldRegistryFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("query_field_registry: parse yaml: %w", err)
	}
	if len(f.Resources) == 0 {
		return nil, fmt.Errorf("query_field_registry: no resources configured")
	}

	resources := make(map[string]map[string]QueryFieldDefinition, len(f.Resources))
	for resource, resourceDef := range f.Resources {
		if len(resourceDef.Fields) == 0 {
			return nil, fmt.Errorf("query_field_registry: resource %q has no fields", resource)
		}
		fieldMap := make(map[string]QueryFieldDefinition, len(resourceDef.Fields))
		for _, field := range resourceDef.Fields {
			if field.Field == "" {
				return nil, fmt.Errorf("query_field_registry: resource %q has empty field entry", resource)
			}
			if _, exists := fieldMap[field.Field]; exists {
				return nil, fmt.Errorf("query_field_registry: duplicate field %q for resource %q", field.Field, resource)
			}
			if len(field.SupportedEngines()) == 0 {
				return nil, fmt.Errorf("query_field_registry: field %q for resource %q has no engines", field.Field, resource)
			}
			fieldMap[field.Field] = field
		}
		resources[resource] = fieldMap
	}
	return resources, nil
}

func (r *QueryFieldRegistry) Reload() error {
	resources, err := loadQueryFieldsFromFile(r.path)
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.resources = resources
	r.mu.Unlock()
	return nil
}

func (r *QueryFieldRegistry) Get(resource, field string) (QueryFieldDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	fields, ok := r.resources[resource]
	if !ok {
		return QueryFieldDefinition{}, false
	}
	def, ok := fields[field]
	return def, ok
}

func (r *QueryFieldRegistry) ResourceFields(resource string) []QueryFieldDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	fields, ok := r.resources[resource]
	if !ok {
		return nil
	}
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]QueryFieldDefinition, 0, len(names))
	for _, name := range names {
		out = append(out, fields[name])
	}
	return out
}

func (r *QueryFieldRegistry) EnginesFor(resource, field string) []string {
	def, ok := r.Get(resource, field)
	if !ok {
		return nil
	}
	return slices.Clone(def.SupportedEngines())
}
