package filter

import (
	"fmt"
	"strings"
	"time"
)

// WhereClause represents a built SQL WHERE clause with parameterized values.
type WhereClause struct {
	SQL  string        // e.g. "status = $1 AND cf_algo#>>'{hand_tracking@1.2.0,status}' = $2"
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

	// Split at first dot: cf_tag.priority → cf_tag#>>'{priority}'
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
	case "duration_sec", "delivery_count", "archive_after_days", "delete_after_days", "total_size_bytes":
		return fmt.Sprintf("(%s)::NUMERIC", expr)
	case "last_delivered_at", "last_accessed_at":
		return fmt.Sprintf("(%s)::TIMESTAMPTZ", expr)
	default:
		return expr
	}
}
