package config

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// AlgoOutput defines the output constraints for an algorithm.
type AlgoOutput struct {
	RequiredFields []string `yaml:"required_fields"`
	URIRequired    bool     `yaml:"uri_required"`
	ReportSize     bool     `yaml:"report_size"`
}

// AlgoDefinition defines a single algorithm's registration info.
type AlgoDefinition struct {
	Description string     `yaml:"description"`
	Versions    []string   `yaml:"versions"`
	DependsOn   []string   `yaml:"depends_on"`
	Output      AlgoOutput `yaml:"output"`
}

// AlgoRegistry manages all registered algorithm definitions.
type AlgoRegistry struct {
	mu         sync.RWMutex
	algorithms map[string]AlgoDefinition
	path       string
}

// algoRegistryFile is the top-level YAML structure.
type algoRegistryFile struct {
	Algorithms map[string]AlgoDefinition `yaml:"algorithms"`
}

// LoadAlgoRegistry loads the algorithm registry from a YAML file.
// Returns an error if the file is missing or contains invalid syntax.
func LoadAlgoRegistry(path string) (*AlgoRegistry, error) {
	algos, err := loadAlgosFromFile(path)
	if err != nil {
		return nil, err
	}
	return &AlgoRegistry{algorithms: algos, path: path}, nil
}

func loadAlgosFromFile(path string) (map[string]AlgoDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("algo_registry: read file %s: %w", path, err)
	}

	var f algoRegistryFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("algo_registry: parse yaml: %w", err)
	}

	// Validate: reject entries with empty version lists.
	for name, def := range f.Algorithms {
		if len(def.Versions) == 0 {
			return nil, fmt.Errorf("algo_registry: algorithm %q has empty version list", name)
		}
	}

	return f.Algorithms, nil
}

// Reload re-reads the YAML file and swaps the internal map atomically.
func (r *AlgoRegistry) Reload() error {
	algos, err := loadAlgosFromFile(r.path)
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.algorithms = algos
	r.mu.Unlock()
	return nil
}

// Validate checks that algoKey has the format "<name>@<version>" and that
// both the algorithm name and version are registered.
func (r *AlgoRegistry) Validate(algoKey string) error {
	name, version, ok := parseAlgoKey(algoKey)
	if !ok {
		return fmt.Errorf("algo_registry: invalid algo_key format %q, expected <name>@<version>", algoKey)
	}

	r.mu.RLock()
	def, exists := r.algorithms[name]
	r.mu.RUnlock()
	if !exists {
		return fmt.Errorf("algo_registry: algorithm %q not registered", name)
	}

	for _, v := range def.Versions {
		if v == version {
			return nil
		}
	}
	return fmt.Errorf("algo_registry: version %q not valid for algorithm %q", version, name)
}

// GetDefinition returns the algorithm definition for the given name.
func (r *AlgoRegistry) GetDefinition(name string) (*AlgoDefinition, bool) {
	r.mu.RLock()
	def, ok := r.algorithms[name]
	r.mu.RUnlock()
	if !ok {
		return nil, false
	}
	return &def, true
}

// GetRequiredFields returns the required output fields and URI requirement
// for the given algo_key. Returns an error if the key format is invalid or
// the algorithm is not registered.
func (r *AlgoRegistry) GetRequiredFields(algoKey string) ([]string, bool, error) {
	name, _, ok := parseAlgoKey(algoKey)
	if !ok {
		return nil, false, fmt.Errorf("algo_registry: invalid algo_key format %q", algoKey)
	}

	r.mu.RLock()
	def, exists := r.algorithms[name]
	r.mu.RUnlock()
	if !exists {
		return nil, false, fmt.Errorf("algo_registry: algorithm %q not registered", name)
	}

	return def.Output.RequiredFields, def.Output.URIRequired, nil
}

// GetAllAlgorithms returns a copy of all registered algorithm definitions.
// Useful for iterating over all algorithms (e.g., depends_on checking).
func (r *AlgoRegistry) GetAllAlgorithms() map[string]AlgoDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cp := make(map[string]AlgoDefinition, len(r.algorithms))
	for k, v := range r.algorithms {
		cp[k] = v
	}
	return cp
}

// parseAlgoKey splits "name@version" into its components.
func parseAlgoKey(algoKey string) (name, version string, ok bool) {
	idx := strings.LastIndex(algoKey, "@")
	if idx <= 0 || idx == len(algoKey)-1 {
		return "", "", false
	}
	return algoKey[:idx], algoKey[idx+1:], true
}
