package postgres

import (
	"context"
	"strings"
	"testing"
)

// CYB-3291: InsertRelationWithMetadata SQL has 5 placeholders ($5 = metadata jsonb).
// It must pass 5 args or pgx errors "mismatched param and argument count" (500).
func TestInsertRelationWithMetadata_PassesMetadataArg(t *testing.T) {
	// with metadata
	db := &fakeDB{}
	repo := NewAssetRepo(&Client{db: db})
	if err := repo.InsertRelationWithMetadata(context.Background(),
		"parent1", "child1", "split_from", "run-1",
		map[string]any{"asset_type": "action"}); err != nil {
		t.Fatalf("InsertRelationWithMetadata err = %v", err)
	}
	if len(db.execArgs) != 1 {
		t.Fatalf("expected 1 exec call, got %d", len(db.execArgs))
	}
	args := db.execArgs[0]
	if len(args) != 5 {
		t.Fatalf("expected 5 args (incl metadata), got %d: %#v", len(args), args)
	}
	meta, ok := args[4].([]byte)
	if !ok || !strings.Contains(string(meta), "action") {
		t.Fatalf("5th arg = %#v, want metadata json bytes containing 'action'", args[4])
	}

	// nil metadata → still 5 args, defaults to {}
	db2 := &fakeDB{}
	repo2 := NewAssetRepo(&Client{db: db2})
	if err := repo2.InsertRelation(context.Background(), "p", "c", "revision_of", ""); err != nil {
		t.Fatalf("InsertRelation err = %v", err)
	}
	if len(db2.execArgs[0]) != 5 {
		t.Fatalf("InsertRelation: expected 5 args, got %d", len(db2.execArgs[0]))
	}
	if b, _ := db2.execArgs[0][4].([]byte); string(b) != "{}" {
		t.Fatalf("nil metadata should marshal to {}, got %q", string(b))
	}
}
