package config

import (
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/quick"

	"gopkg.in/yaml.v3"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func writeTagYAML(t *testing.T, f tagRegistryFile) string {
	t.Helper()
	data, err := yaml.Marshal(f)
	if err != nil {
		t.Fatalf("marshal yaml: %v", err)
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "tag_registry.yaml")
	if err := os.WriteFile(p, data, 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	return p
}

// ── Unit tests ───────────────────────────────────────────────────────────────

func TestLoadTagRegistry_MissingFile(t *testing.T) {
	_, err := LoadTagRegistry("/nonexistent/path.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadTagRegistry_BadSyntax(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(p, []byte("{{invalid yaml"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadTagRegistry(p)
	if err == nil {
		t.Fatal("expected error for bad yaml syntax")
	}
}

func TestLoadTagRegistry_Success(t *testing.T) {
	f := tagRegistryFile{
		Tags: map[string]TagDef{
			"priority": {Description: "test", Type: "enum", Values: []string{"high", "low"}},
			"notes":    {Description: "test", Type: "string", MaxLength: 100},
		},
	}
	p := writeTagYAML(t, f)
	reg, err := LoadTagRegistry(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := reg.Validate("priority", "high"); err != nil {
		t.Fatalf("validate failed: %v", err)
	}
}

func TestTagValidate_UnregisteredKey(t *testing.T) {
	reg := &TagRegistry{tags: map[string]TagDef{
		"priority": {Type: "enum", Values: []string{"high", "low"}},
	}}
	err := reg.Validate("unknown_key", "value")
	if err == nil {
		t.Fatal("expected error for unregistered key")
	}
	if !strings.Contains(err.Error(), "not registered") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestTagValidate_EnumValueValid(t *testing.T) {
	reg := &TagRegistry{tags: map[string]TagDef{
		"priority": {Type: "enum", Values: []string{"critical", "high", "medium", "low"}},
	}}
	for _, v := range []string{"critical", "high", "medium", "low"} {
		if err := reg.Validate("priority", v); err != nil {
			t.Errorf("expected valid for %q: %v", v, err)
		}
	}
}

func TestTagValidate_EnumValueInvalid(t *testing.T) {
	reg := &TagRegistry{tags: map[string]TagDef{
		"priority": {Type: "enum", Values: []string{"critical", "high", "medium", "low"}},
	}}
	err := reg.Validate("priority", "urgent")
	if err == nil {
		t.Fatal("expected error for invalid enum value")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestTagValidate_StringLengthValid(t *testing.T) {
	reg := &TagRegistry{tags: map[string]TagDef{
		"notes": {Type: "string", MaxLength: 10},
	}}
	if err := reg.Validate("notes", "short"); err != nil {
		t.Fatalf("expected valid: %v", err)
	}
	// Exactly at max length.
	if err := reg.Validate("notes", "1234567890"); err != nil {
		t.Fatalf("expected valid at max length: %v", err)
	}
}

func TestTagValidate_StringLengthExceeded(t *testing.T) {
	reg := &TagRegistry{tags: map[string]TagDef{
		"notes": {Type: "string", MaxLength: 10},
	}}
	err := reg.Validate("notes", "12345678901") // 11 chars
	if err == nil {
		t.Fatal("expected error for string exceeding max length")
	}
	if !strings.Contains(err.Error(), "exceeds max length") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestTagValidate_StringNoMaxLength(t *testing.T) {
	reg := &TagRegistry{tags: map[string]TagDef{
		"task": {Type: "string", MaxLength: 0},
	}}
	// Any length should be valid when max_length is 0 (unlimited).
	longStr := strings.Repeat("a", 10000)
	if err := reg.Validate("task", longStr); err != nil {
		t.Fatalf("expected valid for unlimited string: %v", err)
	}
}

// ── Property tests ───────────────────────────────────────────────────────────

// Property 24: tag_registry 校验拒绝未注册 key
// For any write to tags, when the tag key is not registered in tag_registry.yaml,
// the operation should return an error. When the key is registered and type is enum,
// the value must be in the values list, otherwise return an error.
// **Validates: Requirements 13.4, 13.5**
func TestProperty24_TagRegistryRejectsUnregisteredKeys(t *testing.T) {
	// Build a known registry.
	reg := &TagRegistry{tags: map[string]TagDef{
		"priority": {Type: "enum", Values: []string{"critical", "high", "medium", "low"}},
		"quality":  {Type: "enum", Values: []string{"excellent", "good", "poor"}},
		"notes":    {Type: "string", MaxLength: 100},
		"task":     {Type: "string", MaxLength: 0},
	}}

	registeredKeys := []string{"priority", "quality", "notes", "task"}
	enumValues := map[string][]string{
		"priority": {"critical", "high", "medium", "low"},
		"quality":  {"excellent", "good", "poor"},
	}

	cfg := &quick.Config{MaxCount: 200}
	err := quick.Check(func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		// 50% chance: use a registered key, 50% chance: use a random unregistered key.
		var key string
		useRegistered := r.Intn(2) == 0
		if useRegistered {
			key = registeredKeys[r.Intn(len(registeredKeys))]
		} else {
			// Generate a key that is NOT in the registry.
			key = "unregistered_" + randAlgoName(r)
		}

		// Generate a random value.
		value := randAlgoName(r)

		err := reg.Validate(key, value)

		if !useRegistered {
			// Unregistered key must always be rejected.
			if err == nil {
				t.Logf("expected error for unregistered key %q", key)
				return false
			}
			return true
		}

		// Registered key: check based on type.
		def := reg.tags[key]
		switch def.Type {
		case "enum":
			allowed := false
			for _, v := range enumValues[key] {
				if v == value {
					allowed = true
					break
				}
			}
			if allowed && err != nil {
				t.Logf("expected pass for enum key=%q value=%q: %v", key, value, err)
				return false
			}
			if !allowed && err == nil {
				t.Logf("expected error for invalid enum value key=%q value=%q", key, value)
				return false
			}
		case "string":
			if def.MaxLength > 0 && len(value) > def.MaxLength {
				if err == nil {
					t.Logf("expected error for string exceeding max_length key=%q", key)
					return false
				}
			} else {
				if err != nil {
					t.Logf("expected pass for valid string key=%q: %v", key, err)
					return false
				}
			}
		}
		return true
	}, cfg)
	if err != nil {
		t.Fatalf("Property 24 failed: %v", err)
	}
}
