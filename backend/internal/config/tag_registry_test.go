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

// CYB-3246: open vocabulary — an unregistered key is accepted as a free-form
// string tag (no longer rejected), so users can tag assets with arbitrary
// semantic keys without editing tag_registry.yaml.
func TestTagValidate_UnregisteredKeyAcceptedAsFreeform(t *testing.T) {
	reg := &TagRegistry{tags: map[string]TagDef{
		"priority": {Type: "enum", Values: []string{"high", "low"}},
	}}
	if err := reg.Validate("unknown_key", "任意语义值"); err != nil {
		t.Fatalf("expected unregistered key to be accepted, got: %v", err)
	}
	// Empty value is also acceptable for an open-vocabulary key.
	if err := reg.Validate("another_key", ""); err != nil {
		t.Fatalf("expected empty value for unregistered key to be accepted, got: %v", err)
	}
}

// CYB-3246: open-vocabulary values are still bounded by the default max length.
func TestTagValidate_UnregisteredKeyExceedsDefaultMaxLength(t *testing.T) {
	reg := &TagRegistry{tags: map[string]TagDef{}}
	// Exactly at the default limit is allowed.
	if err := reg.Validate("freeform", strings.Repeat("a", DefaultUnregisteredTagMaxLength)); err != nil {
		t.Fatalf("expected value at default max length to be accepted, got: %v", err)
	}
	// One over the limit is rejected.
	err := reg.Validate("freeform", strings.Repeat("a", DefaultUnregisteredTagMaxLength+1))
	if err == nil {
		t.Fatal("expected error for unregistered value exceeding default max length")
	}
	if !strings.Contains(err.Error(), "exceeds max length") {
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

// CYB-3246: open-vocabulary keys must never auto-propagate to descendants —
// only keys registered with propagation=descendants do.
func TestShouldPropagate_UnregisteredKeyNeverPropagates(t *testing.T) {
	reg := &TagRegistry{tags: map[string]TagDef{
		"compliance.status": {Type: "enum", Values: []string{"approved"}, Propagation: "descendants"},
	}}
	if reg.ShouldPropagate("smoke_desc") {
		t.Fatal("unregistered key must not propagate")
	}
	if !reg.ShouldPropagate("compliance.status") {
		t.Fatal("registered descendants key should propagate")
	}
}

// ── Property tests ───────────────────────────────────────────────────────────

// Property 24: tag_registry 校验语义（CYB-3246 开放词汇后）
// For any write to tags: an unregistered key is accepted as a free-form string
// tag unless its value exceeds DefaultUnregisteredTagMaxLength. When the key is
// registered and type is enum, the value must be in the values list; when type
// is string it must respect max_length.
// **Validates: Requirements 13.4, 13.5 (as MODIFIED by CYB-3246 open vocabulary)**
func TestProperty24_TagRegistryOpenVocabularyAndEnumValidation(t *testing.T) {
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
			// Open vocabulary (CYB-3246): unregistered key is accepted as a
			// free-form string tag, unless it exceeds the default max length.
			tooLong := len(value) > DefaultUnregisteredTagMaxLength
			if tooLong && err == nil {
				t.Logf("expected error for over-long unregistered key %q (len=%d)", key, len(value))
				return false
			}
			if !tooLong && err != nil {
				t.Logf("expected pass for unregistered key %q (len=%d): %v", key, len(value), err)
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
