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
	Type        string   `yaml:"type"`        // "enum" or "string"
	Values      []string `yaml:"values"`      // allowed values for enum type
	MaxLength   int      `yaml:"max_length"`  // max length for string type (0 = unlimited)
	Propagation string   `yaml:"propagation"` // "none" (default) or "descendants" (CYB-1068)
}

// TagSourceDef governs which sources are allowed to write into asset_tags,
// what identity fields they must provide, and whether their assertions are
// immutable. Land scope for CYB-1015 — writable_by ACL is P1.5 and
// deliberately not enforced here. Propagation is enforced since CYB-1068.
type TagSourceDef struct {
	Source                string `yaml:"source"`
	Description           string `yaml:"description"`
	RequiresSourceName    bool   `yaml:"requires_source_name"`
	RequiresSourceVersion bool   `yaml:"requires_source_version"`
	Immutable             bool   `yaml:"immutable"`
}

// TagRegistry manages all registered tag definitions and source contracts.
type TagRegistry struct {
	mu      sync.RWMutex
	tags    map[string]TagDef
	sources map[string]TagSourceDef
	path    string
}

// tagRegistryFile is the top-level YAML structure.
type tagRegistryFile struct {
	Tags       map[string]TagDef `yaml:"tags"`
	TagSources []TagSourceDef    `yaml:"tag_sources"`
}

// LoadTagRegistry loads the tag registry from a YAML file.
// Returns an error if the file is missing or contains invalid syntax.
func LoadTagRegistry(path string) (*TagRegistry, error) {
	tags, sources, err := loadTagsFromFile(path)
	if err != nil {
		return nil, err
	}
	return &TagRegistry{tags: tags, sources: sources, path: path}, nil
}

func loadTagsFromFile(path string) (map[string]TagDef, map[string]TagSourceDef, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("tag_registry: read file %s: %w", path, err)
	}

	var f tagRegistryFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, nil, fmt.Errorf("tag_registry: parse yaml: %w", err)
	}

	sources := make(map[string]TagSourceDef, len(f.TagSources))
	for _, s := range f.TagSources {
		if s.Source == "" {
			continue
		}
		sources[s.Source] = s
	}
	return f.Tags, sources, nil
}

// Reload re-reads the YAML file and swaps the internal maps atomically.
func (r *TagRegistry) Reload() error {
	tags, sources, err := loadTagsFromFile(r.path)
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.tags = tags
	r.sources = sources
	r.mu.Unlock()
	return nil
}

// DefaultUnregisteredTagMaxLength bounds the value length of open-vocabulary
// tags (CYB-3246). Keys not present in the registry are accepted as free-form
// string tags rather than rejected, but their values are still capped to avoid
// unbounded writes. Chosen to match the registered `notes` tag's max_length so
// ad-hoc notes and custom keys share a single ceiling; measured with len()
// (bytes) to stay consistent with the registered `string` validation below.
const DefaultUnregisteredTagMaxLength = 500

// ReplaceTags atomically swaps the in-memory tag definitions (CYB-3246 Phase 2).
// Sources are left untouched. Used to load definitions from the DB at startup
// and to refresh the validation hot-path map after an admin CRUD write, so
// managed-tag changes take effect without a service restart.
func (r *TagRegistry) ReplaceTags(tags map[string]TagDef) {
	r.mu.Lock()
	r.tags = tags
	r.mu.Unlock()
}

// Validate checks a tag write. A registered key is validated strictly against
// its definition (enum membership or string length). An unregistered key is
// accepted as a free-form string tag bounded by DefaultUnregisteredTagMaxLength
// (open vocabulary, CYB-3246) — this lets users tag assets with arbitrary
// semantic keys without editing tag_registry.yaml or restarting the service.
func (r *TagRegistry) Validate(key, value string) error {
	r.mu.RLock()
	def, ok := r.tags[key]
	r.mu.RUnlock()
	if !ok {
		if len(value) > DefaultUnregisteredTagMaxLength {
			return fmt.Errorf("tag_registry: value for unregistered key %q exceeds max length %d (got %d)", key, DefaultUnregisteredTagMaxLength, len(value))
		}
		return nil
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

// ErrUnknownSource is returned when a tag write references a source that is
// not declared in `tag_sources[]`. Callers translate this to HTTP 422.
type ErrUnknownSource struct{ Source string }

func (e ErrUnknownSource) Error() string {
	return fmt.Sprintf("tag_registry: source %q not registered", e.Source)
}

// ErrSourceContract signals that a write of a registered source violates an
// identity contract (`requires_source_name` / `requires_source_version`).
type ErrSourceContract struct {
	Source string
	Field  string
}

func (e ErrSourceContract) Error() string {
	return fmt.Sprintf("tag_registry: source %q requires %s", e.Source, e.Field)
}

// ValidateSource enforces the per-source contract for `asset_tags` writes.
// It does not enforce `writable_by` ACL (P1.5) — caller authentication is
// handled by middleware; only the identity-field requirements are checked
// here so that downstream queries can rely on those columns being populated.
//
// Returns nil when the source is unknown but the registry has no
// `tag_sources` block at all (back-compat for environments that have not
// configured sources yet).
func (r *TagRegistry) ValidateSource(sourceType, sourceName, sourceVersion string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.sources) == 0 {
		return nil
	}
	def, ok := r.sources[sourceType]
	if !ok {
		return ErrUnknownSource{Source: sourceType}
	}
	if def.RequiresSourceName && sourceName == "" {
		return ErrSourceContract{Source: sourceType, Field: "source_name"}
	}
	if def.RequiresSourceVersion && sourceVersion == "" {
		return ErrSourceContract{Source: sourceType, Field: "source_version"}
	}
	return nil
}

// SourceDef returns the source contract for the given source_type, if any.
// `ok=false` means the source is not registered (which under the back-compat
// rule in ValidateSource means writes are unrestricted).
func (r *TagRegistry) SourceDef(sourceType string) (TagSourceDef, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	def, ok := r.sources[sourceType]
	return def, ok
}

// ShouldPropagate returns true when the given tag key is registered with
// propagation=descendants (CYB-1068). Unregistered keys return false.
func (r *TagRegistry) ShouldPropagate(key string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	def, ok := r.tags[key]
	return ok && def.Propagation == "descendants"
}
