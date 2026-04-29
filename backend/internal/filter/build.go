package filter

import (
	"fmt"
	"strings"
	"time"
)

// WhereClause represents a built SQL WHERE clause with parameterized values.
type WhereClause struct {
	SQL  string        // e.g. "status = $1 AND EXISTS (SELECT 1 FROM asset_algo_latest ...)"
	Args []interface{} // Parameterized values
}

// BuildWhereClause converts a list of filters into a parameterized SQL WHERE clause.
// startParam specifies the starting positional parameter number (e.g. 1 for $1).
func BuildWhereClause(filters []Filter, startParam int) (*WhereClause, error) {
	if len(filters) == 0 {
		return &WhereClause{SQL: "", Args: nil}, nil
	}

	paramIdx := startParam
	conditions := make([]string, 0, len(filters))
	args := make([]interface{}, 0, len(filters))

	for _, f := range filters {
		cond, newArgs, nextParam, err := buildCondition(f, paramIdx)
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, cond)
		args = append(args, newArgs...)
		paramIdx = nextParam
	}

	return &WhereClause{
		SQL:  strings.Join(conditions, " AND "),
		Args: args,
	}, nil
}

// buildCondition builds a single SQL condition from a filter.
// Returns the condition string, args, and the next parameter index.
func buildCondition(f Filter, paramIdx int) (string, []interface{}, int, error) {
	// Handle virtual fields by delegating to their handler.
	if f.IsVirtual {
		handler, ok := VirtualFields[f.Field]
		if !ok {
			return "", nil, paramIdx, fmt.Errorf("filter: unknown virtual field %q", f.Field)
		}
		return handler.BuildSQL(f.Value, paramIdx)
	}

	if f.McapColumn != "" {
		return buildMcapLookupCondition(f, paramIdx)
	}

	if strings.HasPrefix(f.StorageField, "asset_tags.") {
		return buildAssetTagCondition(f, paramIdx)
	}
	if strings.HasPrefix(f.StorageField, "asset_algo_latest.") {
		return buildAssetAlgoLatestCondition(f, paramIdx)
	}

	colExpr := resolveColumnExpr(f)

	switch f.Op {
	case "IN", "NOT IN":
		return buildInCondition(f, colExpr, paramIdx)
	case "@>":
		return buildContainsCondition(f, colExpr, paramIdx)
	default:
		return buildSimpleCondition(f, colExpr, paramIdx)
	}
}

func existsCondition(base, expr string, f Filter, paramIdx int) (string, []interface{}, int, error) {
	switch f.Op {
	case "IN", "NOT IN":
		arr, ok := f.Value.([]interface{})
		if !ok {
			sql := fmt.Sprintf("EXISTS (SELECT 1 FROM %s AND %s %s ($%d))", base, expr, f.Op, paramIdx)
			return sql, []interface{}{f.Value}, paramIdx + 1, nil
		}
		if len(arr) == 0 {
			if f.Op == "IN" {
				return "FALSE", nil, paramIdx, nil
			}
			return "TRUE", nil, paramIdx, nil
		}
		placeholders := make([]string, len(arr))
		args := make([]interface{}, len(arr))
		for i, v := range arr {
			placeholders[i] = fmt.Sprintf("$%d", paramIdx+i)
			args[i] = v
		}
		sql := fmt.Sprintf("EXISTS (SELECT 1 FROM %s AND %s %s (%s))", base, expr, f.Op, strings.Join(placeholders, ", "))
		return sql, args, paramIdx + len(arr), nil
	case "BETWEEN":
		lo, hi, err := parseBetweenValuePair(f.Value)
		if err != nil {
			return "", nil, paramIdx, err
		}
		sql := fmt.Sprintf("EXISTS (SELECT 1 FROM %s AND %s >= $%d AND %s <= $%d)", base, expr, paramIdx, expr, paramIdx+1)
		return sql, []interface{}{lo, hi}, paramIdx + 2, nil
	default:
		if f.Value == nil {
			switch f.Op {
			case "=":
				return fmt.Sprintf("EXISTS (SELECT 1 FROM %s AND %s IS NULL)", base, expr), nil, paramIdx, nil
			case "!=":
				return fmt.Sprintf("EXISTS (SELECT 1 FROM %s AND %s IS NOT NULL)", base, expr), nil, paramIdx, nil
			}
		}
		sql := fmt.Sprintf("EXISTS (SELECT 1 FROM %s AND %s %s $%d)", base, expr, f.Op, paramIdx)
		return sql, []interface{}{f.Value}, paramIdx + 1, nil
	}
}

func tagValueColumn(v interface{}) string {
	switch v.(type) {
	case bool:
		return "tag_value_bool"
	case int, int8, int16, int32, int64, float32, float64:
		return "tag_value_num"
	default:
		return "tag_value"
	}
}

func buildAssetTagCondition(f Filter, paramIdx int) (string, []interface{}, int, error) {
	key := strings.TrimPrefix(f.StorageField, "asset_tags.")
	base := fmt.Sprintf("asset_tags t WHERE t.asset_id = assets.asset_id AND t.tag_key = '%s'", key)
	expr := "t." + tagValueColumn(f.Value)
	return existsCondition(base, expr, f, paramIdx)
}

func algoLatestAttrColumn(attr string) (string, error) {
	switch attr {
	case "status":
		return "status", nil
	case "run_id":
		return "run_id", nil
	case "method":
		return "method", nil
	case "output_uri":
		return "output_uri", nil
	case "error_message", "reason":
		return "error_message", nil
	case "started_at":
		return "started_at", nil
	case "finished_at":
		return "finished_at", nil
	case "updated_at":
		return "updated_at", nil
	default:
		return "", fmt.Errorf("filter: unsupported algo attribute %q", attr)
	}
}

func buildAssetAlgoLatestCondition(f Filter, paramIdx int) (string, []interface{}, int, error) {
	key := strings.TrimPrefix(f.StorageField, "asset_algo_latest.")
	colon := strings.LastIndex(key, ":")
	if colon <= 0 || colon >= len(key)-1 {
		return "", nil, paramIdx, fmt.Errorf("filter: invalid algo field %q", f.Field)
	}
	algoKey := key[:colon]
	attr := key[colon+1:]
	at := strings.LastIndex(algoKey, "@")
	if at <= 0 || at >= len(algoKey)-1 {
		return "", nil, paramIdx, fmt.Errorf("filter: invalid algo key %q", algoKey)
	}
	algoName, algoVersion := algoKey[:at], algoKey[at+1:]
	col, err := algoLatestAttrColumn(attr)
	if err != nil {
		return "", nil, paramIdx, err
	}
	base := fmt.Sprintf(
		"asset_algo_latest al WHERE al.asset_id = assets.asset_id AND al.algo_name = '%s' AND al.algo_version = '%s'",
		algoName, algoVersion,
	)
	return existsCondition(base, "al."+col, f, paramIdx)
}

// resolveColumnExpr returns the SQL column expression for a filter.
// For JSONB-backed map fields, everything after the first dot is treated as one key.
func resolveColumnExpr(f Filter) string {
	field := f.Field
	if f.StorageField != "" {
		field = f.StorageField
	}

	if !f.IsJsonb {
		return field
	}

	// Split at first dot: metadata.priority → metadata#>>'{priority}'
	dotIdx := strings.Index(field, ".")
	if dotIdx < 0 {
		return field
	}

	column := field[:dotIdx]
	key := field[dotIdx+1:]
	return fmt.Sprintf("%s#>>'{%s}'", column, key)
}

// buildSimpleCondition builds a simple comparison condition (=, !=, <, >, etc.).
func buildSimpleCondition(f Filter, colExpr string, paramIdx int) (string, []interface{}, int, error) {
	// Handle NULL values
	if f.Value == nil {
		switch f.Op {
		case "=":
			return fmt.Sprintf("%s IS NULL", colExpr), nil, paramIdx, nil
		case "!=":
			return fmt.Sprintf("%s IS NOT NULL", colExpr), nil, paramIdx, nil
		default:
			return fmt.Sprintf("%s %s $%d", colExpr, f.Op, paramIdx), []interface{}{f.Value}, paramIdx + 1, nil
		}
	}

	if f.Op == "BETWEEN" {
		lo, hi, err := parseBetweenValuePair(f.Value)
		if err != nil {
			return "", nil, paramIdx, err
		}
		expr := colExpr
		if f.IsJsonb {
			expr = addTypeCast(colExpr, lo)
		}
		sql := fmt.Sprintf("(%s >= $%d AND %s <= $%d)", expr, paramIdx, expr, paramIdx+1)
		return sql, []interface{}{lo, hi}, paramIdx + 2, nil
	}

	// For JSONB numeric/timestamp comparisons, add type cast
	if f.IsJsonb && isNumericOp(f.Op) {
		castExpr := addTypeCast(colExpr, f.Value)
		return fmt.Sprintf("%s %s $%d", castExpr, f.Op, paramIdx), []interface{}{f.Value}, paramIdx + 1, nil
	}

	return fmt.Sprintf("%s %s $%d", colExpr, f.Op, paramIdx), []interface{}{f.Value}, paramIdx + 1, nil
}

// buildInCondition builds an IN or NOT IN condition with parameterized placeholder list.
func buildInCondition(f Filter, colExpr string, paramIdx int) (string, []interface{}, int, error) {
	arr, ok := f.Value.([]interface{})
	if !ok {
		// Single value — treat as a one-element list
		cond := fmt.Sprintf("%s %s ($%d)", colExpr, f.Op, paramIdx)
		return cond, []interface{}{f.Value}, paramIdx + 1, nil
	}

	if len(arr) == 0 {
		if f.Op == "IN" {
			return "FALSE", nil, paramIdx, nil
		}
		return "TRUE", nil, paramIdx, nil
	}

	placeholders := make([]string, len(arr))
	args := make([]interface{}, len(arr))
	for i, v := range arr {
		placeholders[i] = fmt.Sprintf("$%d", paramIdx+i)
		args[i] = v
	}

	cond := fmt.Sprintf("%s %s (%s)", colExpr, f.Op, strings.Join(placeholders, ", "))
	return cond, args, paramIdx + len(arr), nil
}

// buildContainsCondition builds a @> containment condition for JSONB.
func buildContainsCondition(f Filter, colExpr string, paramIdx int) (string, []interface{}, int, error) {
	if f.IsJsonb {
		// For JSONB contains, use the column (not the extraction expression)
		field := f.Field
		if f.StorageField != "" {
			field = f.StorageField
		}
		dotIdx := strings.Index(field, ".")
		if dotIdx >= 0 {
			column := field[:dotIdx]
			cond := fmt.Sprintf("%s @> $%d::jsonb", column, paramIdx)
			return cond, []interface{}{f.Value}, paramIdx + 1, nil
		}
	}
	cond := fmt.Sprintf("%s @> $%d::jsonb", colExpr, paramIdx)
	return cond, []interface{}{f.Value}, paramIdx + 1, nil
}

// isNumericOp returns true for numeric comparison operators.
func isNumericOp(op string) bool {
	switch op {
	case "<", ">", "<=", ">=":
		return true
	}
	return false
}

// addTypeCast adds ::NUMERIC or ::TIMESTAMPTZ cast to a JSONB extraction expression
// based on the value type.
func addTypeCast(colExpr string, value interface{}) string {
	switch value.(type) {
	case time.Time:
		return fmt.Sprintf("(%s)::TIMESTAMPTZ", colExpr)
	default:
		// For int, float64, and other numeric types
		return fmt.Sprintf("(%s)::NUMERIC", colExpr)
	}
}

// parseBetweenValuePair parses BETWEEN bounds from a "lo,hi" string or a
// two-element JSON array (as produced by inferValueType).
func parseBetweenValuePair(v interface{}) (interface{}, interface{}, error) {
	switch val := v.(type) {
	case string:
		parts := strings.SplitN(val, ",", 2)
		if len(parts) != 2 {
			return nil, nil, fmt.Errorf("filter: between value must be two comma-separated parts")
		}
		lo := strings.TrimSpace(parts[0])
		hi := strings.TrimSpace(parts[1])
		if lo == "" || hi == "" {
			return nil, nil, fmt.Errorf("filter: between bounds must be non-empty")
		}
		return inferValueType(lo), inferValueType(hi), nil
	case []interface{}:
		if len(val) != 2 {
			return nil, nil, fmt.Errorf("filter: between value must be a two-element array")
		}
		return val[0], val[1], nil
	default:
		return nil, nil, fmt.Errorf("filter: invalid between value type %T", v)
	}
}

// buildMcapLookupCondition turns mcap.<col> filters into EXISTS subqueries
// against mcap_files (column names were validated in ResolveField).
func buildMcapLookupCondition(f Filter, paramIdx int) (string, []interface{}, int, error) {
	col := f.McapColumn
	prefix := fmt.Sprintf(
		`EXISTS (SELECT 1 FROM mcap_files mf WHERE mf.mcap_file_id = assets.mcap_file_id AND COALESCE(mf.is_deleted, FALSE) = FALSE AND mf.%s `,
		col,
	)

	switch f.Op {
	case "BETWEEN":
		lo, hi, err := parseBetweenValuePair(f.Value)
		if err != nil {
			return "", nil, paramIdx, err
		}
		sql := fmt.Sprintf("%sBETWEEN $%d AND $%d)", prefix, paramIdx, paramIdx+1)
		return sql, []interface{}{lo, hi}, paramIdx + 2, nil
	case "ILIKE", "LIKE", "=", "!=", "<", ">", "<=", ">=":
		sql := fmt.Sprintf("%s%s $%d)", prefix, f.Op, paramIdx)
		return sql, []interface{}{f.Value}, paramIdx + 1, nil
	default:
		return "", nil, paramIdx, fmt.Errorf("filter: operator %q not supported for mcap fields", f.Op)
	}
}

// ResolveSortBy converts a validated sort field to a SQL ORDER BY expression.
// Supports "-" prefix for DESC and approved JSONB-backed aliases.
func ResolveSortBy(sortBy string) (string, error) {
	if sortBy == "" {
		return "created_at DESC", nil
	}

	direction := "ASC"
	field := sortBy

	if strings.HasPrefix(sortBy, "-") {
		direction = "DESC"
		field = sortBy[1:]
	}

	if field == "" {
		return "created_at DESC", nil
	}

	spec, err := ResolveField(field)
	if err != nil {
		return "", err
	}

	if spec.IsJSONB {
		if strings.HasPrefix(spec.StorageField, "asset_tags.") {
			key := strings.TrimPrefix(spec.StorageField, "asset_tags.")
			expr := fmt.Sprintf("(SELECT t.tag_value FROM asset_tags t WHERE t.asset_id = assets.asset_id AND t.tag_key = '%s' LIMIT 1)", key)
			return fmt.Sprintf("%s %s", castJsonbSortExpr(spec.Canonical, expr), direction), nil
		}
		if strings.HasPrefix(spec.StorageField, "asset_algo_latest.") {
			key := strings.TrimPrefix(spec.StorageField, "asset_algo_latest.")
			colon := strings.LastIndex(key, ":")
			if colon <= 0 || colon >= len(key)-1 {
				return "", fmt.Errorf("filter: invalid algo sort field %q", field)
			}
			algoKey := key[:colon]
			attr := key[colon+1:]
			at := strings.LastIndex(algoKey, "@")
			if at <= 0 || at >= len(algoKey)-1 {
				return "", fmt.Errorf("filter: invalid algo sort field %q", field)
			}
			algoName, algoVersion := algoKey[:at], algoKey[at+1:]
			col, err := algoLatestAttrColumn(attr)
			if err != nil {
				return "", err
			}
			expr := fmt.Sprintf("(SELECT al.%s FROM asset_algo_latest al WHERE al.asset_id = assets.asset_id AND al.algo_name = '%s' AND al.algo_version = '%s' LIMIT 1)", col, algoName, algoVersion)
			return fmt.Sprintf("%s %s", castJsonbSortExpr(spec.Canonical, expr), direction), nil
		}
		dotIdx := strings.Index(spec.StorageField, ".")
		if dotIdx < 0 {
			return "", fmt.Errorf("filter: invalid JSONB sort field %q", field)
		}
		column := spec.StorageField[:dotIdx]
		key := spec.StorageField[dotIdx+1:]
		expr := fmt.Sprintf("%s#>>'{%s}'", column, key)
		return fmt.Sprintf("%s %s", castJsonbSortExpr(spec.Canonical, expr), direction), nil
	}

	return fmt.Sprintf("%s %s", spec.StorageField, direction), nil
}

func castJsonbSortExpr(field, expr string) string {
	switch field {
	case "duration_sec", "duration_ms", "delivery_count", "archive_after_days", "delete_after_days", "total_size_bytes":
		return fmt.Sprintf("(%s)::NUMERIC", expr)
	case "last_delivered_at", "last_accessed_at":
		return fmt.Sprintf("(%s)::TIMESTAMPTZ", expr)
	default:
		return expr
	}
}
