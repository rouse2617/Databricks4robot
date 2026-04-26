package filter

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ParseFilter parses a single filter string in "<field>:<op>:<value>" format.
// Uses sort.Slice to sort operators by length descending before matching,
// so "nin" isn't matched by "in", "lte" isn't matched by "lt", etc.
func ParseFilter(filterStr string) (*Filter, error) {
	if filterStr == "" {
		return nil, fmt.Errorf("filter: empty filter string")
	}

	// Find the operator by trying each known operator name.
	// Sort by length descending so longer operators match first.
	opNames := make([]string, 0, len(OperatorMap))
	for name := range OperatorMap {
		opNames = append(opNames, name)
	}
	sort.Slice(opNames, func(i, j int) bool {
		return len(opNames[i]) > len(opNames[j])
	})

	var field, opName, valueStr string
	matched := false

	for _, op := range opNames {
		// Look for ":<op>:" pattern in the string
		sep := ":" + op + ":"
		idx := strings.Index(filterStr, sep)
		if idx > 0 {
			field = filterStr[:idx]
			valueStr = filterStr[idx+len(sep):]
			opName = op
			matched = true
			break
		}
	}

	if !matched {
		return nil, fmt.Errorf("filter: unrecognized operator in %q, expected <field>:<op>:<value>", filterStr)
	}

	if field == "" {
		return nil, fmt.Errorf("filter: empty field name in %q", filterStr)
	}

	// Validate field against the searchable fields whitelist.
	if err := ValidateFieldWhitelist(field); err != nil {
		// Fall through to ResolveField for fields not in the whitelist
		// but recognized by the existing dynamic prefix system (e.g. algo.hand_tracking@1.2.0:status).
		if _, resolveErr := ResolveField(field); resolveErr != nil {
			return nil, err // return the whitelist error
		}
	}

	sqlOp := OperatorMap[opName]
	value := inferValueType(valueStr)

	// Check if this is a virtual field and delegate to its handler.
	if meta, ok := SearchableFields[field]; ok && meta.Type == "virtual" {
		if _, exists := VirtualFields[field]; exists {
			return &Filter{
				Field:     field,
				Op:        sqlOp,
				Value:     value,
				IsVirtual: true,
			}, nil
		}
	}

	// Apply alias prefix replacement for business-friendly field names.
	aliasedField := ResolveFieldAlias(field)

	spec, err := ResolveField(aliasedField)
	if err != nil {
		// If aliased field fails, try the original field directly.
		spec, err = ResolveField(field)
		if err != nil {
			return nil, err
		}
	}

	// like/ilike: auto-wrap with % wildcards if not already present
	if opName == "like" || opName == "ilike" {
		if strVal, ok := value.(string); ok {
			if !strings.Contains(strVal, "%") && !strings.Contains(strVal, "_") {
				value = "%" + strVal + "%"
			}
		}
	}

	return &Filter{
		Field:        spec.Canonical,
		StorageField: spec.StorageField,
		Op:           sqlOp,
		Value:        value,
		IsJsonb:      spec.IsJSONB,
	}, nil
}

// ParseFilters parses multiple filter strings.
func ParseFilters(filterStrs []string) ([]Filter, error) {
	filters := make([]Filter, 0, len(filterStrs))
	for _, s := range filterStrs {
		f, err := ParseFilter(s)
		if err != nil {
			return nil, err
		}
		filters = append(filters, *f)
	}
	return filters, nil
}

// ValidateFilters validates filters against the field whitelist, operator whitelist, and value length.
func ValidateFilters(filters []Filter) error {
	for _, f := range filters {
		// Virtual fields (algo_status, has:delivery) are handled by VirtualFieldHandler,
		// not by ResolveField. Skip the field resolution check for them.
		if f.IsVirtual {
			continue
		}
		if _, err := ResolveField(f.Field); err != nil {
			return err
		}

		// Operator whitelist check — Op should already be a valid SQL operator
		found := false
		for _, sqlOp := range OperatorMap {
			if f.Op == sqlOp {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("filter: operator %q is not allowed", f.Op)
		}

		// Value length check (max 255 for string values)
		if strVal, ok := f.Value.(string); ok {
			if len(strVal) > 255 {
				return fmt.Errorf("filter: value for field %q exceeds maximum length of 255", f.Field)
			}
		}
	}
	return nil
}

// inferValueType attempts to parse the value string into the most specific type:
// integer, float, boolean, null, RFC3339 timestamp, JSON array, or falls back to string.
func inferValueType(s string) interface{} {
	// null
	if s == "null" {
		return nil
	}

	// boolean
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}

	// JSON array
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		var arr []interface{}
		if err := json.Unmarshal([]byte(s), &arr); err == nil {
			return arr
		}
	}

	// integer (try before float to preserve int type)
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}

	// float
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		// Only treat as float if it contains a decimal point or scientific notation
		if strings.Contains(s, ".") || strings.ContainsAny(s, "eE") {
			return f
		}
	}

	// RFC3339 timestamp
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}

	// fallback: string
	return s
}
