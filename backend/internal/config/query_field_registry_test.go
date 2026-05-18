package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadQueryFieldRegistry_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "query_field_registry.yaml")
	content := `
schema_version: 1
resources:
  assets:
    fields:
      - field: owner
        filter_engines: [postgres, elasticsearch]
        sort_engines: [postgres]
      - field: tag.priority
        filter_engines: [postgres]
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	reg, err := LoadQueryFieldRegistry(path)
	if err != nil {
		t.Fatalf("LoadQueryFieldRegistry: %v", err)
	}
	def, ok := reg.Get("assets", "owner")
	if !ok {
		t.Fatalf("expected owner field")
	}
	if def.Field != "owner" {
		t.Fatalf("unexpected field: %+v", def)
	}
	if got, want := reg.EnginesFor("assets", "owner"), []string{"elasticsearch", "postgres"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("EnginesFor(owner) = %#v want %#v", got, want)
	}
}

func TestLoadQueryFieldRegistry_RejectsDuplicateField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "query_field_registry.yaml")
	content := `
schema_version: 1
resources:
  assets:
    fields:
      - field: owner
        filter_engines: [postgres]
      - field: owner
        sort_engines: [postgres]
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if _, err := LoadQueryFieldRegistry(path); err == nil {
		t.Fatal("expected error for duplicate field")
	}
}
