package config

import (
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// TagDef defines a single tag's registration info.
type TagDef struct {
	Description string   `yaml:"description"`
	Type        string   `yaml:"type"`       // "enum" or "string"
	Values      []string `yaml:"values"`     // allowed values for enum type
	MaxLength   int      `yaml:"max_length"` // max length for string type (0 = unlimited)
}

// TagRegistry manages all registered tag definitions.
type TagRegistry struct {
	mu   sync.RWMutex
	tags map[string]TagDef
	path string
}

// tagRegistryFile is the top-level YAML structure.
type tagRegistryFile struct {
	Tags map[string]TagDef `yaml:"tags"`
}

// LoadTagRegistry loads the tag registry from a YAML file.
// Returns an error if the file is missing or contains invalid syntax.
func LoadTagRegistry(path string) (*TagRegistry, error) {
	tags, err := loadTagsFromFile(path)
	if err != nil {
		return nil, err
	}
	return &TagRegistry{tags: tags, path: path}, nil
}

func loadTagsFromFile(path string) (map[string]TagDef, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("tag_registry: read file %s: %w", path, err)
	}

	var f tagRegistryFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("tag_registry: parse yaml: %w", err)
	}

	return f.Tags, nil
}

// Reload re-reads the YAML file and swaps the internal map atomically.
func (r *TagRegistry) Reload() error {
	tags, err := loadTagsFromFile(r.path)
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.tags = tags
	r.mu.Unlock()
	return nil
}

// Validate checks that the given key is registered and the value is valid
// according to the tag definition (enum membership or string length).
func (r *TagRegistry) Validate(key, value string) error {
	r.mu.RLock()
	def, ok := r.tags[key]
	r.mu.RUnlock()
	if !ok {
		return fmt.Errorf("tag_registry: key %q not registered", key)
	}

	switch def.Type {
	case "enum":
		for _, v := range def.Values {
			if v == value {
				return nil
			}
		}
		return fmt.Errorf("tag_registry: value %q not allowed for enum key %q (allowed: %v)", value, key, def.Values)
	case "string":
		if def.MaxLength > 0 && len(value) > def.MaxLength {
			return fmt.Errorf("tag_registry: value for key %q exceeds max length %d (got %d)", key, def.MaxLength, len(value))
		}
		return nil
	default:
		return fmt.Errorf("tag_registry: unknown type %q for key %q", def.Type, key)
	}
}

// GetAllTags returns a copy of all registered tag definitions.
func (r *TagRegistry) GetAllTags() map[string]TagDef {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cp := make(map[string]TagDef, len(r.tags))
	for k, v := range r.tags {
		cp[k] = v
	}
	return cp
}
