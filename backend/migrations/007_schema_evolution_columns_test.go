package migrations_test

import (
	"bufio"
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestMigration007_ColumnsExist verifies that the 007 migration script contains
// all required ADD COLUMN IF NOT EXISTS statements and CREATE INDEX IF NOT EXISTS
// statements per the design document. This is a static analysis test — no real
// database connection is needed.

func loadMigrationSQL(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("007_schema_evolution_columns.sql")
	if err != nil {
		t.Fatalf("failed to read migration file: %v", err)
	}
	return string(data)
}

// parseALTERColumns extracts (table, column) pairs from ADD COLUMN IF NOT EXISTS lines.
func parseALTERColumns(sql string) map[string][]string {
	re := regexp.MustCompile(`(?i)ALTER\s+TABLE\s+(\w+)\s+ADD\s+COLUMN\s+IF\s+NOT\s+EXISTS\s+(\w+)`)
	result := make(map[string][]string)
	scanner := bufio.NewScanner(strings.NewReader(sql))
	for scanner.Scan() {
		m := re.FindStringSubmatch(scanner.Text())
		if m != nil {
			table := strings.ToLower(m[1])
			col := strings.ToLower(m[2])
			result[table] = append(result[table], col)
		}
	}
	return result
}

// parseIndexNames extracts index names from CREATE [UNIQUE] INDEX IF NOT EXISTS lines.
func parseIndexNames(sql string) []string {
	re := regexp.MustCompile(`(?i)CREATE\s+(?:UNIQUE\s+)?INDEX\s+IF\s+NOT\s+EXISTS\s+(\w+)`)
	var names []string
	scanner := bufio.NewScanner(strings.NewReader(sql))
	for scanner.Scan() {
		m := re.FindStringSubmatch(scanner.Text())
		if m != nil {
			names = append(names, strings.ToLower(m[1]))
		}
	}
	return names
}

func TestMigration007_ColumnsExist(t *testing.T) {
	sql := loadMigrationSQL(t)
	columns := parseALTERColumns(sql)

	// ---- assets table columns ----
	assetsExpected := []string{
		"asset_type", "lifecycle_state", "duration_ms", "owner", "reviewer",
		"storage_uri", "thumb_uri", "retention_tier", "expire_at",
		"asset_level", "parent_asset_id", "root_asset_id",
		"delivery_count", "last_delivered_at", "last_delivered_to",
		"segment_index", "parent_start_offset_ms", "parent_end_offset_ms",
		"split_method", "split_algo_name", "split_algo_version",
		"split_run_id", "split_reason",
		"metadata", "files", "tenant_id", "project_id",
	}
	assetCols := toSet(columns["assets"])
	for _, col := range assetsExpected {
		if !assetCols[col] {
			t.Errorf("assets table missing column: %s", col)
		}
	}

	// ---- mcap_files table columns ----
	mcapExpected := []string{
		"mcap_uri", "size_bytes", "file_duration_ms",
		"start_timestamp_ns", "end_timestamp_ns",
		"channel_count", "chunk_count", "ingest_state",
		"vendor_id", "collector_id", "task_id", "device_id",
		"camera_model", "data_source", "location_id", "scene_id",
		"environment_id", "collection_method",
		"owner", "retention_tier", "expire_at",
		"metadata", "process_state", "raw_hash_sha256",
		"tenant_id", "project_id",
	}
	mcapCols := toSet(columns["mcap_files"])
	for _, col := range mcapExpected {
		if !mcapCols[col] {
			t.Errorf("mcap_files table missing column: %s", col)
		}
	}

	// ---- deliveries table columns ----
	deliveriesExpected := []string{
		"contract_id", "delivery_type", "requested_by", "approved_by",
		"delivered_by", "manifest_uri", "replay_manifest_uri",
		"item_count", "total_size_bytes", "completed_at",
		"metadata", "tenant_id", "project_id",
	}
	deliveryCols := toSet(columns["deliveries"])
	for _, col := range deliveriesExpected {
		if !deliveryCols[col] {
			t.Errorf("deliveries table missing column: %s", col)
		}
	}
}

func TestMigration007_IndexesExist(t *testing.T) {
	sql := loadMigrationSQL(t)
	indexes := toSet(parseIndexNames(sql))

	expected := []string{
		// assets indexes
		"idx_assets_lifecycle",
		"idx_assets_asset_type",
		"idx_assets_parent",
		"idx_assets_root",
		"idx_assets_tenant_project",
		"idx_assets_metadata_gin",
		// mcap_files indexes
		"uq_mcap_files_hash_md5",
		"idx_mcap_files_tenant_project",
		"idx_mcap_files_ingest_state",
		"idx_mcap_files_metadata_gin",
		// deliveries indexes
		"idx_deliveries_tenant_project",
	}
	for _, idx := range expected {
		if !indexes[idx] {
			t.Errorf("missing index: %s", idx)
		}
	}
}

func TestMigration007_Idempotent(t *testing.T) {
	sql := loadMigrationSQL(t)

	// Every ALTER TABLE must use ADD COLUMN IF NOT EXISTS
	alterRe := regexp.MustCompile(`(?i)ALTER\s+TABLE\s+\w+\s+ADD\s+COLUMN\s+`)
	ifNotExistsRe := regexp.MustCompile(`(?i)ADD\s+COLUMN\s+IF\s+NOT\s+EXISTS`)
	scanner := bufio.NewScanner(strings.NewReader(sql))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if alterRe.MatchString(line) && !ifNotExistsRe.MatchString(line) {
			t.Errorf("line %d: ALTER TABLE ADD COLUMN without IF NOT EXISTS: %s", lineNum, line)
		}
	}

	// Every CREATE INDEX must use IF NOT EXISTS
	createIdxRe := regexp.MustCompile(`(?i)CREATE\s+(?:UNIQUE\s+)?INDEX\s+`)
	idxIfNotExistsRe := regexp.MustCompile(`(?i)CREATE\s+(?:UNIQUE\s+)?INDEX\s+IF\s+NOT\s+EXISTS`)
	scanner = bufio.NewScanner(strings.NewReader(sql))
	lineNum = 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if createIdxRe.MatchString(line) && !idxIfNotExistsRe.MatchString(line) {
			t.Errorf("line %d: CREATE INDEX without IF NOT EXISTS: %s", lineNum, line)
		}
	}
}

func TestMigration007_NotNullColumnsHaveDefaults(t *testing.T) {
	sql := loadMigrationSQL(t)
	notNullRe := regexp.MustCompile(`(?i)NOT\s+NULL`)
	defaultRe := regexp.MustCompile(`(?i)DEFAULT\s+`)
	scanner := bufio.NewScanner(strings.NewReader(sql))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if !strings.Contains(strings.ToUpper(line), "ADD COLUMN") {
			continue
		}
		if notNullRe.MatchString(line) && !defaultRe.MatchString(line) {
			t.Errorf("line %d: NOT NULL column without DEFAULT: %s", lineNum, line)
		}
	}
}

func TestMigration007_LegacyColumnsUntouched(t *testing.T) {
	sql := loadMigrationSQL(t)
	upper := strings.ToUpper(sql)

	// Must NOT drop or alter existing cf_* columns
	legacyCols := []string{"CF_META", "CF_ALGO", "CF_TAG", "CF_FILES", "CF_PROCESS"}
	for _, col := range legacyCols {
		dropPattern := "DROP COLUMN " + col
		alterTypePattern := "ALTER COLUMN " + col
		if strings.Contains(upper, dropPattern) {
			t.Errorf("migration drops legacy column %s", col)
		}
		if strings.Contains(upper, alterTypePattern) {
			t.Errorf("migration alters legacy column %s type", col)
		}
	}
}

func toSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[item] = true
	}
	return s
}
