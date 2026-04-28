package migrations_test

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// TestMigration009_TriggersExist verifies that the 009 migration script contains
// the set_updated_at() trigger function and BEFORE UPDATE triggers on
// asset_tags and asset_algo_latest.
// This is a static analysis test — no real database connection is needed.

func loadMigration009SQL(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("009_schema_evolution_triggers.sql")
	if err != nil {
		t.Fatalf("failed to read migration file: %v", err)
	}
	return string(data)
}

func TestMigration009_TriggersExist(t *testing.T) {
	sql := loadMigration009SQL(t)
	lower := strings.ToLower(sql)

	// 1. Verify set_updated_at() function is created
	funcRe := regexp.MustCompile(`(?i)CREATE\s+OR\s+REPLACE\s+FUNCTION\s+set_updated_at\s*\(\)`)
	if !funcRe.MatchString(sql) {
		t.Error("missing CREATE OR REPLACE FUNCTION set_updated_at()")
	}

	// 2. Verify function returns TRIGGER
	if !strings.Contains(lower, "returns trigger") {
		t.Error("set_updated_at() should RETURN TRIGGER")
	}

	// 3. Verify function body sets NEW.updated_at = now()
	if !strings.Contains(lower, "new.updated_at") || !strings.Contains(lower, "now()") {
		t.Error("set_updated_at() body should contain NEW.updated_at = now()")
	}

	// 4. Verify function language is plpgsql
	if !strings.Contains(lower, "language plpgsql") {
		t.Error("set_updated_at() should use LANGUAGE plpgsql")
	}

	// 5. Verify trigger on asset_tags
	triggerAssetTagsRe := regexp.MustCompile(
		`(?i)CREATE\s+TRIGGER\s+trg_asset_tags_updated_at\s+BEFORE\s+UPDATE\s+ON\s+asset_tags`)
	if !triggerAssetTagsRe.MatchString(sql) {
		t.Error("missing BEFORE UPDATE trigger trg_asset_tags_updated_at on asset_tags")
	}

	// 6. Verify trigger on asset_algo_latest
	triggerAlgoRe := regexp.MustCompile(
		`(?i)CREATE\s+TRIGGER\s+trg_asset_algo_latest_updated_at\s+BEFORE\s+UPDATE\s+ON\s+asset_algo_latest`)
	if !triggerAlgoRe.MatchString(sql) {
		t.Error("missing BEFORE UPDATE trigger trg_asset_algo_latest_updated_at on asset_algo_latest")
	}

	// 7. Verify both triggers execute set_updated_at()
	executeRe := regexp.MustCompile(`(?i)EXECUTE\s+FUNCTION\s+set_updated_at\s*\(\)`)
	matches := executeRe.FindAllString(sql, -1)
	if len(matches) < 2 {
		t.Errorf("expected at least 2 EXECUTE FUNCTION set_updated_at() clauses, got %d", len(matches))
	}

	// 8. Verify FOR EACH ROW on both triggers
	forEachRowRe := regexp.MustCompile(`(?i)FOR\s+EACH\s+ROW`)
	forEachMatches := forEachRowRe.FindAllString(sql, -1)
	if len(forEachMatches) < 2 {
		t.Errorf("expected at least 2 FOR EACH ROW clauses, got %d", len(forEachMatches))
	}
}

func TestMigration009_Idempotent(t *testing.T) {
	sql := loadMigration009SQL(t)
	lower := strings.ToLower(sql)

	// CREATE OR REPLACE FUNCTION is inherently idempotent
	if !strings.Contains(lower, "create or replace function") {
		t.Error("trigger function should use CREATE OR REPLACE for idempotency")
	}

	// Triggers use DROP IF EXISTS + CREATE pattern for idempotency
	// (PostgreSQL doesn't support CREATE TRIGGER IF NOT EXISTS)
	dropTriggerRe := regexp.MustCompile(`(?i)DROP\s+TRIGGER\s+IF\s+EXISTS`)
	dropMatches := dropTriggerRe.FindAllString(sql, -1)
	if len(dropMatches) < 2 {
		t.Errorf("expected at least 2 DROP TRIGGER IF EXISTS for idempotency, got %d", len(dropMatches))
	}
}

func TestMigration009_TriggersAttachedToCorrectTables(t *testing.T) {
	sql := loadMigration009SQL(t)

	// Map trigger names to expected tables
	expected := map[string]string{
		"trg_asset_tags_updated_at":        "asset_tags",
		"trg_asset_algo_latest_updated_at": "asset_algo_latest",
	}

	for triggerName, tableName := range expected {
		// Build regex: CREATE TRIGGER <name> BEFORE UPDATE ON <table>
		pattern := `(?i)CREATE\s+TRIGGER\s+` + triggerName + `\s+BEFORE\s+UPDATE\s+ON\s+` + tableName
		re := regexp.MustCompile(pattern)
		if !re.MatchString(sql) {
			t.Errorf("trigger %s not attached to table %s", triggerName, tableName)
		}
	}
}

// TestMigration009_Property14_UpdatedAtAdvancesOnUpdate is a property-based test
// verifying that the trigger function body contains the logic to set
// updated_at = now(), and that triggers are attached to the correct tables.
//
// Feature: schema-evolution-v2, Property 14: Updated_at trigger fires on update
// **Validates: Requirements 20.1**
func TestMigration009_Property14_UpdatedAtAdvancesOnUpdate(t *testing.T) {
	sql := loadMigration009SQL(t)
	lower := strings.ToLower(sql)

	// Tables that must have the updated_at trigger
	targetTables := []string{"asset_tags", "asset_algo_latest"}

	rapid.Check(t, func(t *rapid.T) {
		// Pick a random target table
		idx := rapid.IntRange(0, len(targetTables)-1).Draw(t, "tableIndex")
		table := targetTables[idx]

		// Property: The trigger function body sets NEW.updated_at = now()
		// This guarantees that for ANY update on the table, updated_at advances.
		if !strings.Contains(lower, "new.updated_at") {
			t.Fatalf("trigger function does not set NEW.updated_at")
		}
		if !strings.Contains(lower, "now()") {
			t.Fatalf("trigger function does not use now()")
		}

		// Property: A BEFORE UPDATE trigger is attached to this table
		triggerPattern := `(?i)CREATE\s+TRIGGER\s+\w+\s+BEFORE\s+UPDATE\s+ON\s+` + table
		re := regexp.MustCompile(triggerPattern)
		if !re.MatchString(sql) {
			t.Fatalf("no BEFORE UPDATE trigger attached to %s", table)
		}

		// Property: The trigger executes set_updated_at()
		executePattern := `(?i)ON\s+` + table + `\s+FOR\s+EACH\s+ROW\s+EXECUTE\s+FUNCTION\s+set_updated_at\s*\(\)`
		execRe := regexp.MustCompile(executePattern)
		if !execRe.MatchString(sql) {
			t.Fatalf("trigger on %s does not EXECUTE FUNCTION set_updated_at()", table)
		}

		// Property: The function returns NEW (ensuring the updated row is saved)
		if !strings.Contains(lower, "return new") {
			t.Fatalf("trigger function does not RETURN NEW — updated row would be discarded")
		}
	})
}
