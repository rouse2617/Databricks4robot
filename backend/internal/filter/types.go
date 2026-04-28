package filter

import (
	"fmt"
	"strings"
	"time"
)

// FieldAliasMap maps business-friendly field prefixes to storage column prefixes.
// The frontend sends "tag.notes" and the backend resolves it to "cf_tag.notes".
var FieldAliasMap = map[string]string{
	"tag.":  "cf_tag.",
	"file.": "cf_files.",
	"algo.": "cf_algo.",
}

// FieldMeta describes a searchable field's type and allowed values.
type FieldMeta struct {
	Type   string   // "string", "enum", "numeric", "timestamp", "virtual"
	Values []string // for enum types, the allowed values
}

// SearchableFields is the whitelist of fields the frontend may filter on.
var SearchableFields = map[string]FieldMeta{
	"asset_id":          {Type: "string"},
	"mcap_file_id":      {Type: "string"},
	"owner":             {Type: "string"},
	"reviewer":          {Type: "string"},
	"status":            {Type: "enum", Values: []string{"approved", "rejected", "superseded", "archived"}},
	"env":               {Type: "enum", Values: []string{"kitchen", "outdoor", "warehouse", "office", "factory"}},
	"scene":             {Type: "enum"},
	"task":              {Type: "string"},
	"batch":             {Type: "string"},
	"duration_sec":      {Type: "numeric"},
	"created_at":        {Type: "timestamp"},
	"updated_at":        {Type: "timestamp"},
	"delivery_count":    {Type: "numeric"},
	"lifecycle_state":   {Type: "enum", Values: []string{"created", "processing", "ready", "rejected", "delivered", "archived", "superseded"}},
	"asset_type":        {Type: "enum", Values: []string{"segment", "clip", "frame_set", "derived_asset"}},
	"expire_at":         {Type: "timestamp"},
	"retention_tier":    {Type: "string"},
	"last_delivered_to": {Type: "string"},
	"last_delivered_at": {Type: "timestamp"},
	"tag.priority":      {Type: "enum", Values: []string{"critical", "high", "medium", "low"}},
	"tag.quality":       {Type: "enum", Values: []string{"excellent", "good", "acceptable", "poor", "unusable"}},
	"tag.notes":         {Type: "string"},
	"tag.scene":         {Type: "enum"},
	"tag.task":          {Type: "string"},
	"tag.batch":         {Type: "string"},
	"algo_status":       {Type: "virtual"},
	"has:delivery":      {Type: "virtual"},
}

// ResolveFieldAlias applies the FieldAliasMap prefix replacement.
// If the field starts with a key in FieldAliasMap, the prefix is replaced
// with the corresponding storage prefix. Otherwise the field is returned as-is.
func ResolveFieldAlias(field string) string {
	for prefix, replacement := range FieldAliasMap {
		if strings.HasPrefix(field, prefix) {
			return replacement + strings.TrimPrefix(field, prefix)
		}
	}
	return field
}

// ValidateFieldWhitelist checks whether a field is in the SearchableFields whitelist.
// For prefixed fields (e.g. "tag.anything"), it checks if the exact field or
// any matching prefix pattern is present. Returns an error for unknown fields.
func ValidateFieldWhitelist(field string) error {
	// Direct match
	if _, ok := SearchableFields[field]; ok {
		return nil
	}
	return fmt.Errorf("filter: unknown field %q", field)
}

// VirtualFieldHandler defines how a virtual field is resolved for both
// Postgres SQL generation and Bigtable in-memory matching.
type VirtualFieldHandler interface {
	// BuildSQL generates a SQL condition fragment, returning the SQL string,
	// bound args, the next parameter index, and any error.
	BuildSQL(value interface{}, paramIdx int) (string, []interface{}, int, error)
	// MatchBigtable checks whether an asset matches the virtual field condition in-memory.
	MatchBigtable(algoResults map[string]string, deliveryCount int, value interface{}) bool
}

// VirtualFields maps virtual field names to their handlers.
var VirtualFields = map[string]VirtualFieldHandler{
	"algo_status":  algoStatusVirtualHandler{},
	"has:delivery": hasDeliveryVirtualHandler{},
}

// algoStatusVirtualHandler matches any cf_algo key ending in ":status".
type algoStatusVirtualHandler struct{}

func (h algoStatusVirtualHandler) BuildSQL(value interface{}, paramIdx int) (string, []interface{}, int, error) {
	sql := fmt.Sprintf("asset_algo_statuses(cf_algo) @> ARRAY[$%d]::text[]", paramIdx)
	return sql, []interface{}{value}, paramIdx + 1, nil
}

func (h algoStatusVirtualHandler) MatchBigtable(algoResults map[string]string, _ int, value interface{}) bool {
	target := fmt.Sprintf("%v", value)
	for k, v := range algoResults {
		if strings.HasSuffix(k, ":status") && v == target {
			return true
		}
	}
	return false
}

// hasDeliveryVirtualHandler translates "has:delivery" to the JSONB-backed delivery count.
type hasDeliveryVirtualHandler struct{}

func (h hasDeliveryVirtualHandler) BuildSQL(value interface{}, paramIdx int) (string, []interface{}, int, error) {
	// has:delivery is a boolean check: delivery_count > 0
	// delivery_count lives inside cf_meta JSONB, so use JSONB extraction.
	boolVal := true
	switch v := value.(type) {
	case bool:
		boolVal = v
	case string:
		boolVal = v != "false" && v != "0"
	}
	if boolVal {
		return "COALESCE((cf_meta->>'delivery_count')::int, 0) > 0", nil, paramIdx, nil
	}
	return "COALESCE((cf_meta->>'delivery_count')::int, 0) = 0", nil, paramIdx, nil
}

func (h hasDeliveryVirtualHandler) MatchBigtable(_ map[string]string, deliveryCount int, value interface{}) bool {
	boolVal := true
	switch v := value.(type) {
	case bool:
		boolVal = v
	case string:
		boolVal = v != "false" && v != "0"
	}
	if boolVal {
		return deliveryCount > 0
	}
	return deliveryCount == 0
}

// Filter represents a parsed filter condition.
type Filter struct {
	Field        string      // Canonical public field name, e.g. "status" or "tag.priority"
	StorageField string      // Resolved storage field, e.g. "cf_meta.owner" or "cf_tag.priority"
	Op           string      // SQL operator, e.g. "=", "!=", "LIKE"
	Value        interface{} // Typed value
	IsJsonb      bool        // Whether the resolved storage field is a JSONB path query
	IsVirtual    bool        // Whether this is a virtual field handled by VirtualFieldHandler
}

// OperatorMap defines allowed query operators and their SQL mappings.
var OperatorMap = map[string]string{
	"eq":       "=",
	"ne":       "!=",
	"lt":       "<",
	"gt":       ">",
	"lte":      "<=",
	"gte":      ">=",
	"like":     "LIKE",
	"ilike":    "ILIKE",
	"in":       "IN",
	"nin":      "NOT IN",
	"contains": "@>",
}

// ReverseOperatorMap maps SQL operators back to filter operator names.
var ReverseOperatorMap = map[string]string{
	"=":      "eq",
	"!=":     "ne",
	"<":      "lt",
	">":      "gt",
	"<=":     "lte",
	">=":     "gte",
	"LIKE":   "like",
	"ILIKE":  "ilike",
	"IN":     "in",
	"NOT IN": "nin",
	"@>":     "contains",
}

// Serialize converts a Filter back to its canonical string form: <field>:<op>:<value>.
func (f *Filter) Serialize() string {
	opName, ok := ReverseOperatorMap[f.Op]
	if !ok {
		opName = f.Op
	}
	return fmt.Sprintf("%s:%s:%s", f.Field, opName, serializeValue(f.Value))
}

// serializeValue converts a typed value back to its string representation.
func serializeValue(v interface{}) string {
	if v == nil {
		return "null"
	}
	switch val := v.(type) {
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case float64:
		return fmt.Sprintf("%g", val)
	case time.Time:
		return val.Format(time.RFC3339)
	case []interface{}:
		return serializeArray(val)
	case string:
		return val
	default:
		return fmt.Sprintf("%v", val)
	}
}

// serializeArray converts a slice back to JSON array string form.
func serializeArray(arr []interface{}) string {
	parts := make([]string, len(arr))
	for i, v := range arr {
		switch val := v.(type) {
		case string:
			parts[i] = fmt.Sprintf("%q", val)
		case float64:
			// Check if it's actually an integer
			if val == float64(int64(val)) {
				parts[i] = fmt.Sprintf("%d", int64(val))
			} else {
				parts[i] = fmt.Sprintf("%g", val)
			}
		default:
			parts[i] = fmt.Sprintf("%v", val)
		}
	}
	return "[" + strings.Join(parts, ",") + "]"
}
