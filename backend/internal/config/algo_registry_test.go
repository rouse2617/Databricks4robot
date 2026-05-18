package config

import (
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"testing/quick"

	"gopkg.in/yaml.v3"
)

// ── helpers ──────────────────────────────────────────────────────────────────

// randAlgoName generates a random algorithm name (lowercase letters, 3-12 chars).
func randAlgoName(r *rand.Rand) string {
	n := 3 + r.Intn(10)
	b := make([]byte, n)
	for i := range b {
		b[i] = 'a' + byte(r.Intn(26))
	}
	return string(b)
}

// randVersion generates a semver-like version string.
func randVersion(r *rand.Rand) string {
	return string(rune('0'+r.Intn(10))) + "." +
		string(rune('0'+r.Intn(10))) + "." +
		string(rune('0'+r.Intn(10)))
}

// writeYAML writes an algoRegistryFile to a temp file and returns the path.
func writeYAML(t *testing.T, f algoRegistryFile) string {
	t.Helper()
	data, err := yaml.Marshal(f)
	if err != nil {
		t.Fatalf("marshal yaml: %v", err)
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "algo_registry.yaml")
	if err := os.WriteFile(p, data, 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	return p
}

// ── Unit tests ───────────────────────────────────────────────────────────────

func TestLoadAlgoRegistry_MissingFile(t *testing.T) {
	_, err := LoadAlgoRegistry("/nonexistent/path.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadAlgoRegistry_BadSyntax(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(p, []byte("{{invalid yaml"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadAlgoRegistry(p)
	if err == nil {
		t.Fatal("expected error for bad yaml syntax")
	}
}

func TestLoadAlgoRegistry_EmptyVersions(t *testing.T) {
	f := algoRegistryFile{
		Algorithms: map[string]AlgoDefinition{
			"bad_algo": {Description: "test", Versions: []string{}},
		},
	}
	p := writeYAML(t, f)
	_, err := LoadAlgoRegistry(p)
	if err == nil {
		t.Fatal("expected error for empty version list")
	}
}

func TestLoadAlgoRegistry_Success(t *testing.T) {
	f := algoRegistryFile{
		Algorithms: map[string]AlgoDefinition{
			"hand_tracking": {
				Description: "test",
				Versions:    []string{"1.0.0", "1.2.0"},
				Output:      AlgoOutput{RequiredFields: []string{"output_uri"}, URIRequired: true},
			},
		},
	}
	p := writeYAML(t, f)
	reg, err := LoadAlgoRegistry(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := reg.Validate("hand_tracking@1.0.0"); err != nil {
		t.Fatalf("validate failed: %v", err)
	}
}

func TestValidate_InvalidFormat(t *testing.T) {
	reg := &AlgoRegistry{algorithms: map[string]AlgoDefinition{
		"x": {Versions: []string{"1.0.0"}},
	}}
	for _, key := range []string{"noversion", "@1.0.0", "name@", ""} {
		if err := reg.Validate(key); err == nil {
			t.Errorf("expected error for key %q", key)
		}
	}
}

func TestValidate_UnregisteredAlgo(t *testing.T) {
	reg := &AlgoRegistry{algorithms: map[string]AlgoDefinition{
		"x": {Versions: []string{"1.0.0"}},
	}}
	if err := reg.Validate("unknown@1.0.0"); err == nil {
		t.Fatal("expected error for unregistered algo")
	}
}

func TestValidate_InvalidVersion(t *testing.T) {
	reg := &AlgoRegistry{algorithms: map[string]AlgoDefinition{
		"x": {Versions: []string{"1.0.0"}},
	}}
	if err := reg.Validate("x@9.9.9"); err == nil {
		t.Fatal("expected error for invalid version")
	}
}

func TestGetDefinition(t *testing.T) {
	def := AlgoDefinition{Description: "test", Versions: []string{"1.0.0"}}
	reg := &AlgoRegistry{algorithms: map[string]AlgoDefinition{"x": def}}

	got, ok := reg.GetDefinition("x")
	if !ok || got.Description != "test" {
		t.Fatalf("expected definition, got ok=%v def=%+v", ok, got)
	}
	_, ok = reg.GetDefinition("missing")
	if ok {
		t.Fatal("expected not found")
	}
}

func TestGetRequiredFields(t *testing.T) {
	reg := &AlgoRegistry{algorithms: map[string]AlgoDefinition{
		"x": {
			Versions: []string{"1.0.0"},
			Output:   AlgoOutput{RequiredFields: []string{"output_uri", "type"}, URIRequired: true},
		},
	}}

	fields, uriReq, err := reg.GetRequiredFields("x@1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !uriReq || len(fields) != 2 {
		t.Fatalf("unexpected result: fields=%v uriReq=%v", fields, uriReq)
	}

	_, _, err = reg.GetRequiredFields("bad_format")
	if err == nil {
		t.Fatal("expected error for bad format")
	}

	_, _, err = reg.GetRequiredFields("unknown@1.0.0")
	if err == nil {
		t.Fatal("expected error for unknown algo")
	}
}

func TestGetAllAlgorithms(t *testing.T) {
	reg := &AlgoRegistry{algorithms: map[string]AlgoDefinition{
		"a": {Versions: []string{"1.0.0"}},
		"b": {Versions: []string{"2.0.0"}},
	}}
	all := reg.GetAllAlgorithms()
	if len(all) != 2 {
		t.Fatalf("expected 2 algorithms, got %d", len(all))
	}
}

// ── Property tests ───────────────────────────────────────────────────────────

// Property 11: 算法注册表往返一致性
// For any valid set of algorithm definitions, serializing to YAML and loading
// via LoadAlgoRegistry should preserve all algorithm names, version lists, and
// output field requirements.
// **Validates: Requirements 4.2**
func TestProperty11_AlgoRegistryRoundTrip(t *testing.T) {
	cfg := &quick.Config{MaxCount: 100}
	err := quick.Check(func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		// Generate 1-5 random algorithms, each with 1-3 versions.
		numAlgos := 1 + r.Intn(5)
		algos := make(map[string]AlgoDefinition, numAlgos)
		for i := 0; i < numAlgos; i++ {
			name := randAlgoName(r)
			numVersions := 1 + r.Intn(3)
			versions := make([]string, numVersions)
			for j := range versions {
				versions[j] = randVersion(r)
			}
			numFields := r.Intn(4)
			fields := make([]string, numFields)
			for j := range fields {
				fields[j] = randAlgoName(r)
			}
			algos[name] = AlgoDefinition{
				Description: "desc_" + name,
				Versions:    versions,
				Output: AlgoOutput{
					RequiredFields: fields,
					URIRequired:    r.Intn(2) == 1,
					ReportSize:     r.Intn(2) == 1,
				},
			}
		}

		// Serialize to YAML and load back.
		f := algoRegistryFile{Algorithms: algos}
		data, err := yaml.Marshal(f)
		if err != nil {
			t.Logf("marshal error: %v", err)
			return false
		}
		dir := t.TempDir()
		p := filepath.Join(dir, "test.yaml")
		if err := os.WriteFile(p, data, 0644); err != nil {
			t.Logf("write error: %v", err)
			return false
		}

		reg, err := LoadAlgoRegistry(p)
		if err != nil {
			t.Logf("load error: %v", err)
			return false
		}

		// Verify all algorithms are preserved.
		loaded := reg.GetAllAlgorithms()
		if len(loaded) != len(algos) {
			t.Logf("count mismatch: want %d got %d", len(algos), len(loaded))
			return false
		}
		for name, orig := range algos {
			got, ok := loaded[name]
			if !ok {
				t.Logf("missing algorithm %q", name)
				return false
			}
			if !reflect.DeepEqual(orig.Versions, got.Versions) {
				t.Logf("versions mismatch for %q: want %v got %v", name, orig.Versions, got.Versions)
				return false
			}
			if orig.Output.URIRequired != got.Output.URIRequired {
				t.Logf("uri_required mismatch for %q", name)
				return false
			}
			if !reflect.DeepEqual(orig.Output.RequiredFields, got.Output.RequiredFields) {
				// YAML may deserialize nil slice as nil vs empty slice
				if len(orig.Output.RequiredFields) == 0 && len(got.Output.RequiredFields) == 0 {
					continue
				}
				t.Logf("required_fields mismatch for %q: want %v got %v", name, orig.Output.RequiredFields, got.Output.RequiredFields)
				return false
			}
		}
		return true
	}, cfg)
	if err != nil {
		t.Fatalf("Property 11 failed: %v", err)
	}
}

// Property 12: 算法注册表验证正确性
// For any algo_key "<name>@<version>", Validate returns error iff the algorithm
// name is not registered OR the specified version is not in the valid version list.
// **Validates: Requirements 4.5**
func TestProperty12_AlgoRegistryValidation(t *testing.T) {
	cfg := &quick.Config{MaxCount: 200}
	err := quick.Check(func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		// Build a small registry with known algorithms.
		algos := map[string]AlgoDefinition{
			"alpha": {Versions: []string{"1.0.0", "2.0.0"}},
			"beta":  {Versions: []string{"3.0.0"}},
		}
		reg := &AlgoRegistry{algorithms: algos}

		// Generate a random algo_key.
		names := []string{"alpha", "beta", "gamma", "delta"}
		name := names[r.Intn(len(names))]
		versions := []string{"1.0.0", "2.0.0", "3.0.0", "9.9.9"}
		version := versions[r.Intn(len(versions))]
		algoKey := name + "@" + version

		err := reg.Validate(algoKey)

		// Determine expected result.
		def, nameExists := algos[name]
		versionValid := false
		if nameExists {
			for _, v := range def.Versions {
				if v == version {
					versionValid = true
					break
				}
			}
		}

		shouldPass := nameExists && versionValid
		if shouldPass && err != nil {
			t.Logf("expected pass for %q but got error: %v", algoKey, err)
			return false
		}
		if !shouldPass && err == nil {
			t.Logf("expected error for %q but got nil", algoKey)
			return false
		}
		return true
	}, cfg)
	if err != nil {
		t.Fatalf("Property 12 failed: %v", err)
	}
}
