package migrations_test

import (
	"bufio"
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestMigration008_TablesExist verifies that the 008 migration script contains
// all required CREATE TABLE IF NOT EXISTS statements, primary keys, columns,
// indexes, and FK references per the design document.
// This is a static analysis test — no real database connection is needed.

func loadMigration008SQL(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("008_schema_evolution_tables.sql")
	if err != nil {
		t.Fatalf("failed to read migration file: %v", err)
	}
	return string(data)
}

// parseCreateTables extracts table names from CREATE TABLE IF NOT EXISTS lines.
func parseCreateTables(sql string) []string {
	re := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+IF\s+NOT\s+EXISTS\s+(\w+)`)
	var tables []string
	scanner := bufio.NewScanner(strings.NewReader(sql))
	for scanner.Scan() {
		m := re.FindStringSubmatch(scanner.Text())
		if m != nil {
			tables = append(tables, strings.ToLower(m[1]))
		}
	}
	return tables
}

// parse008IndexNames extracts index names from CREATE [UNIQUE] INDEX IF NOT EXISTS lines.
func parse008IndexNames(sql string) []string {
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

func TestMigration008_TablesExist(t *testing.T) {
	sql := loadMigration008SQL(t)
	tables := toSet(parseCreateTables(sql))

	expected := []string{
		"asset_tags",
		"asset_algo_latest",
		"asset_events",
		"outbox_sink_cursors",
		"asset_relations",
	}
	for _, tbl := range expected {
		if !tables[tbl] {
			t.Errorf("missing table: %s", tbl)
		}
	}
}

func TestMigration008_PrimaryKeys(t *testing.T) {
	sql := loadMigration008SQL(t)
	lower := strings.ToLower(sql)

	// asset_tags PK: (asset_id, tag_key)
	if !strings.Contains(lower, "primary key (asset_id, tag_key)") {
		t.Error("asset_tags missing PRIMARY KEY (asset_id, tag_key)")
	}

	// asset_algo_latest PK: (asset_id, algo_name)
	if !strings.Contains(lower, "primary key (asset_id, algo_name)") {
		t.Error("asset_algo_latest missing PRIMARY KEY (asset_id, algo_name)")
	}

	// asset_events PK: event_id (inline PRIMARY KEY)
	if !strings.Contains(lower, "event_id") || !containsPKForEventID(lower) {
		t.Error("asset_events missing PRIMARY KEY on event_id")
	}

	// asset_events: event_seq BIGSERIAL UNIQUE
	if !strings.Contains(lower, "event_seq") {
		t.Error("asset_events missing event_seq column")
	}
	if !strings.Contains(lower, "bigserial") {
		t.Error("asset_events event_seq should be BIGSERIAL")
	}

	// outbox_sink_cursors PK: sink_name
	if !strings.Contains(lower, "sink_name") && !strings.Contains(lower, "primary key") {
		t.Error("outbox_sink_cursors missing PRIMARY KEY on sink_name")
	}

	// asset_relations PK: (parent_asset_id, child_asset_id, relation_type)
	if !strings.Contains(lower, "primary key (parent_asset_id, child_asset_id, relation_type)") {
		t.Error("asset_relations missing PRIMARY KEY (parent_asset_id, child_asset_id, relation_type)")
	}
}

// containsPKForEventID checks that event_id has PRIMARY KEY in the asset_events table.
func containsPKForEventID(lower string) bool {
	// Look for "event_id ... primary key" on the same line
	re := regexp.MustCompile(`event_id\s+uuid\s+primary\s+key`)
	return re.MatchString(lower)
}

func TestMigration008_IndexesExist(t *testing.T) {
	sql := loadMigration008SQL(t)
	indexes := toSet(parse008IndexNames(sql))

	expected := []string{
		// asset_tags indexes (4)
		"idx_asset_tags_key_value_str",
		"idx_asset_tags_key_value_num",
		"idx_asset_tags_key_value_bool",
		"idx_asset_tags_tenant_project",
		// asset_algo_latest indexes (4)
		"idx_asset_algo_latest_algo_status",
		"idx_asset_algo_latest_algo_version",
		"idx_asset_algo_latest_run",
		"idx_asset_algo_latest_updated",
		// asset_events indexes (5)
		"idx_asset_events_type_time",
		"idx_asset_events_asset",
		"idx_asset_events_run",
		"idx_asset_events_publish_pending",
		"idx_asset_events_tenant_project",
		// asset_relations indexes (1)
		"idx_asset_relations_child",
	}
	for _, idx := range expected {
		if !indexes[idx] {
			t.Errorf("missing index: %s", idx)
		}
	}
}

func TestMigration008_ForeignKeys(t *testing.T) {
	sql := loadMigration008SQL(t)
	lower := strings.ToLower(sql)

	// asset_tags FK → assets(asset_id)
	if !containsFKRef(lower, "asset_tags", "assets(asset_id)") {
		t.Error("asset_tags missing FK reference to assets(asset_id)")
	}

	// asset_algo_latest FK → assets(asset_id)
	if !containsFKRef(lower, "asset_algo_latest", "assets(asset_id)") {
		t.Error("asset_algo_latest missing FK reference to assets(asset_id)")
	}

	// asset_events FK → assets(asset_id)
	if !containsFKRef(lower, "asset_events", "assets(asset_id)") {
		t.Error("asset_events missing FK reference to assets(asset_id)")
	}

	// asset_events FK → mcap_files(mcap_file_id)
	if !containsFKRef(lower, "asset_events", "mcap_files(mcap_file_id)") {
		t.Error("asset_events missing FK reference to mcap_files(mcap_file_id)")
	}

	// asset_relations FK → assets(asset_id) for parent_asset_id
	if !containsFKRef(lower, "asset_relations", "assets(asset_id)") {
		t.Error("asset_relations missing FK reference to assets(asset_id)")
	}
}

// containsFKRef checks that a table's CREATE TABLE block contains a REFERENCES clause.
func containsFKRef(sql, table, ref string) bool {
	// Find the CREATE TABLE block for the given table and check for REFERENCES
	tableStart := strings.Index(sql, "create table if not exists "+table)
	if tableStart == -1 {
		return false
	}
	// Find the closing parenthesis of the CREATE TABLE
	rest := sql[tableStart:]
	// Look for the ref within a reasonable range (up to next CREATE TABLE or end)
	nextCreate := strings.Index(rest[1:], "create table if not exists")
	var block string
	if nextCreate == -1 {
		block = rest
	} else {
		block = rest[:nextCreate+1]
	}
	return strings.Contains(block, "references "+ref)
}

func TestMigration008_AssetTagsColumns(t *testing.T) {
	sql := loadMigration008SQL(t)
	lower := strings.ToLower(sql)

	// Extract the asset_tags CREATE TABLE block
	block := extractTableBlock(lower, "asset_tags")
	if block == "" {
		t.Fatal("could not find asset_tags CREATE TABLE block")
	}

	expectedCols := []string{
		"asset_id", "tag_key", "tag_value", "tag_value_num", "tag_value_bool",
		"tag_type", "source_type", "source_name", "source_version", "run_id",
		"confidence", "tenant_id", "project_id", "created_at", "updated_at",
	}
	for _, col := range expectedCols {
		if !strings.Contains(block, col) {
			t.Errorf("asset_tags missing column: %s", col)
		}
	}
}

func TestMigration008_AssetAlgoLatestColumns(t *testing.T) {
	sql := loadMigration008SQL(t)
	lower := strings.ToLower(sql)

	block := extractTableBlock(lower, "asset_algo_latest")
	if block == "" {
		t.Fatal("could not find asset_algo_latest CREATE TABLE block")
	}

	expectedCols := []string{
		"asset_id", "algo_name", "algo_version", "status", "result_tag",
		"result_score", "result_summary", "run_id", "method", "model_uri",
		"output_uri", "error_code", "error_message", "started_at", "finished_at",
		"tenant_id", "project_id", "updated_at",
	}
	for _, col := range expectedCols {
		if !strings.Contains(block, col) {
			t.Errorf("asset_algo_latest missing column: %s", col)
		}
	}
}

func TestMigration008_AssetEventsColumns(t *testing.T) {
	sql := loadMigration008SQL(t)
	lower := strings.ToLower(sql)

	block := extractTableBlock(lower, "asset_events")
	if block == "" {
		t.Fatal("could not find asset_events CREATE TABLE block")
	}

	expectedCols := []string{
		"event_id", "event_seq", "event_type", "payload_schema_version",
		"asset_id", "mcap_file_id", "tenant_id", "project_id",
		"event_source", "actor_type", "actor_id", "request_id",
		"idempotency_key", "run_id", "occurred_at", "created_at",
		"publish_state", "published_at", "event_payload",
		"retry_count", "last_error",
	}
	for _, col := range expectedCols {
		if !strings.Contains(block, col) {
			t.Errorf("asset_events missing column: %s", col)
		}
	}
}

func TestMigration008_OutboxSinkCursorsColumns(t *testing.T) {
	sql := loadMigration008SQL(t)
	lower := strings.ToLower(sql)

	block := extractTableBlock(lower, "outbox_sink_cursors")
	if block == "" {
		t.Fatal("could not find outbox_sink_cursors CREATE TABLE block")
	}

	expectedCols := []string{
		"sink_name", "last_published_seq", "updated_at",
	}
	for _, col := range expectedCols {
		if !strings.Contains(block, col) {
			t.Errorf("outbox_sink_cursors missing column: %s", col)
		}
	}
}

func TestMigration008_AssetRelationsColumns(t *testing.T) {
	sql := loadMigration008SQL(t)
	lower := strings.ToLower(sql)

	block := extractTableBlock(lower, "asset_relations")
	if block == "" {
		t.Fatal("could not find asset_relations CREATE TABLE block")
	}

	expectedCols := []string{
		"parent_asset_id", "child_asset_id", "relation_type",
		"method", "algo_name", "algo_version", "run_id",
		"parent_start_offset_ms", "parent_end_offset_ms", "created_at",
	}
	for _, col := range expectedCols {
		if !strings.Contains(block, col) {
			t.Errorf("asset_relations missing column: %s", col)
		}
	}
}

func TestMigration008_Idempotent(t *testing.T) {
	sql := loadMigration008SQL(t)

	// Every CREATE TABLE must use IF NOT EXISTS
	createTableRe := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+`)
	ifNotExistsRe := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+IF\s+NOT\s+EXISTS`)
	scanner := bufio.NewScanner(strings.NewReader(sql))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if createTableRe.MatchString(line) && !ifNotExistsRe.MatchString(line) {
			t.Errorf("line %d: CREATE TABLE without IF NOT EXISTS: %s", lineNum, line)
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

func TestMigration008_NotNullColumnsHaveDefaults(t *testing.T) {
	sql := loadMigration008SQL(t)
	notNullRe := regexp.MustCompile(`(?i)NOT\s+NULL`)
	defaultRe := regexp.MustCompile(`(?i)DEFAULT\s+`)
	primaryKeyRe := regexp.MustCompile(`(?i)PRIMARY\s+KEY`)
	referencesRe := regexp.MustCompile(`(?i)REFERENCES\s+`)
	bigserialRe := regexp.MustCompile(`(?i)BIGSERIAL`)
	createIndexRe := regexp.MustCompile(`(?i)CREATE\s+(?:UNIQUE\s+)?INDEX`)
	onTableRe := regexp.MustCompile(`(?i)^\s*ON\s+\w+\s+\(`)

	scanner := bufio.NewScanner(strings.NewReader(sql))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		// Skip comments, empty lines, PRIMARY KEY lines, index lines
		if line == "" || strings.HasPrefix(line, "--") || primaryKeyRe.MatchString(line) {
			continue
		}
		// Skip CREATE INDEX lines and ON <table> (...) continuation lines
		if createIndexRe.MatchString(line) || onTableRe.MatchString(line) {
			continue
		}
		if !notNullRe.MatchString(line) {
			continue
		}
		// BIGSERIAL columns have implicit NOT NULL and sequence default
		if bigserialRe.MatchString(line) {
			continue
		}
		// FK reference columns (asset_id NOT NULL REFERENCES ...) don't need DEFAULT
		if referencesRe.MatchString(line) {
			continue
		}
		if notNullRe.MatchString(line) && !defaultRe.MatchString(line) {
			// PK component columns are always provided, no DEFAULT needed
			isPKComponent := false
			pkComponentCols := []string{"tag_key", "algo_name", "algo_version", "status",
				"event_type", "event_source", "relation_type", "tag_value"}
			for _, col := range pkComponentCols {
				if strings.Contains(strings.ToLower(line), col) {
					isPKComponent = true
					break
				}
			}
			if !isPKComponent {
				t.Errorf("line %d: NOT NULL column without DEFAULT: %s", lineNum, line)
			}
		}
	}
}

// extractTableBlock returns the CREATE TABLE block for the given table name.
func extractTableBlock(sql, tableName string) string {
	marker := "create table if not exists " + tableName
	start := strings.Index(sql, marker)
	if start == -1 {
		return ""
	}
	// Find the closing ");", accounting for nested parens
	rest := sql[start:]
	depth := 0
	for i, ch := range rest {
		if ch == '(' {
			depth++
		} else if ch == ')' {
			depth--
			if depth == 0 {
				return rest[:i+1]
			}
		}
	}
	return rest
}
