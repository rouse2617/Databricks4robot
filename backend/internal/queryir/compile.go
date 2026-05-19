package queryir

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	SchemaVersionV1 = "v1"
	ResourceAssets  = "assets"
)

var supportedLeafOperators = map[string]struct{}{
	"eq":       {},
	"ne":       {},
	"lt":       {},
	"gt":       {},
	"lte":      {},
	"gte":      {},
	"like":     {},
	"ilike":    {},
	"in":       {},
	"nin":      {},
	"contains": {},
	"between":  {},
}

func Compile(req QueryRequest) (*CompiledQuery, error) {
	normalized := Normalize(req)
	if normalized.SchemaVersion != SchemaVersionV1 {
		return nil, fmt.Errorf("unsupported schema_version %q", normalized.SchemaVersion)
	}
	if normalized.Scope.Resource != ResourceAssets {
		return nil, fmt.Errorf("unsupported scope.resource %q", normalized.Scope.Resource)
	}
	if err := validateExpr(normalized.Where); err != nil {
		return nil, err
	}

	sortBy, err := compileSort(normalized.Sort)
	if err != nil {
		return nil, err
	}
	if normalized.Page.Limit > 0 && normalized.Page.Offset%normalized.Page.Limit != 0 {
		return nil, fmt.Errorf("invalid page: offset must be a multiple of limit in bridge mode")
	}
	return &CompiledQuery{
		FilterStrings:     nil,
		SortBy:            sortBy,
		Page:              normalized.Page.Page,
		PageSize:          normalized.Page.PageSize,
		NormalizedQuery:   normalized,
		FieldCapabilities: collectFieldCapabilities(normalized),
		DebugPlan: DebugPlan{
			Steps: []DebugPlanStep{{Engine: "postgres", Mode: "filter"}},
		},
		Warnings: []string{},
	}, nil
}

func compileLeaf(p *QueryPredicate) (string, error) {
	field := strings.TrimSpace(p.Field)
	operator := strings.ToLower(strings.TrimSpace(p.Op))
	if field == "" {
		return "", fmt.Errorf("missing field in predicate")
	}
	if operator == "" {
		return "", fmt.Errorf("missing operator in predicate")
	}
	if _, ok := supportedLeafOperators[operator]; !ok {
		return "", fmt.Errorf("unsupported operator %q", p.Op)
	}
	valueEncoded, err := encodeFilterValue(p.Value)
	if err != nil {
		return "", fmt.Errorf("encode value: %w", err)
	}
	return fmt.Sprintf("%s:%s:%s", field, operator, valueEncoded), nil
}

func validateExpr(expr *QueryExpr) error {
	if expr == nil {
		return nil
	}
	kindCount := 0
	if len(expr.And) > 0 {
		kindCount++
	}
	if len(expr.Or) > 0 {
		kindCount++
	}
	if expr.Not != nil {
		kindCount++
	}
	if expr.Pred != nil {
		kindCount++
	}
	if kindCount == 0 {
		return fmt.Errorf("invalid where expression: empty node")
	}
	if kindCount > 1 {
		return fmt.Errorf("invalid where expression: exactly one of and/or/not/pred must be set")
	}

	switch {
	case expr.Pred != nil:
		_, err := compileLeaf(expr.Pred)
		return err
	case len(expr.And) > 0:
		for i := range expr.And {
			if err := validateExpr(&expr.And[i]); err != nil {
				return err
			}
		}
		return nil
	case len(expr.Or) > 0:
		for i := range expr.Or {
			if err := validateExpr(&expr.Or[i]); err != nil {
				return err
			}
		}
		return nil
	case expr.Not != nil:
		return validateExpr(expr.Not)
	default:
		return fmt.Errorf("invalid where expression: empty node")
	}
}

func encodeFilterValue(v any) (string, error) {
	switch x := v.(type) {
	case nil:
		return "null", nil
	case string:
		return x, nil
	case bool:
		if x {
			return "true", nil
		}
		return "false", nil
	case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		b, err := json.Marshal(x)
		return string(b), err
	default:
		b, err := json.Marshal(x)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}

func compileSort(sort []QuerySort) (string, error) {
	if len(sort) == 0 {
		return "-created_at", nil
	}
	first := sort[0]
	field := strings.TrimSpace(first.Field)
	if field == "" {
		return "", fmt.Errorf("sort[0].field is required")
	}
	dir := strings.ToLower(strings.TrimSpace(first.Direction))
	if dir == "" || dir == "asc" {
		return field, nil
	}
	if dir == "desc" {
		return "-" + field, nil
	}
	return "", fmt.Errorf("unsupported sort direction %q", first.Direction)
}

func normalizePage(p QueryPage) (int, int) {
	page := p.Page
	pageSize := p.PageSize

	// offset/limit bridge: convert into page/page_size for repo paging.
	if p.Limit > 0 {
		pageSize = p.Limit
		if p.Offset >= 0 {
			page = (p.Offset / p.Limit) + 1
		}
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}

func Normalize(req QueryRequest) QueryRequest {
	normalized := req
	normalized.SchemaVersion = strings.TrimSpace(normalized.SchemaVersion)
	if normalized.SchemaVersion == "" {
		normalized.SchemaVersion = SchemaVersionV1
	}
	normalized.Mode = strings.ToLower(strings.TrimSpace(normalized.Mode))
	normalized.Scope.Resource = strings.TrimSpace(normalized.Scope.Resource)
	normalized.Where = normalizeExpr(normalized.Where)
	normalized.Sort = normalizeSort(normalized.Sort)
	page, pageSize := normalizePage(normalized.Page)
	normalized.Page = QueryPage{
		Page:     page,
		PageSize: pageSize,
		Offset:   normalized.Page.Offset,
		Limit:    normalized.Page.Limit,
	}
	return normalized
}

func normalizeExpr(expr *QueryExpr) *QueryExpr {
	if expr == nil {
		return nil
	}
	out := &QueryExpr{}
	if len(expr.And) > 0 {
		out.And = make([]QueryExpr, 0, len(expr.And))
		for i := range expr.And {
			child := normalizeExpr(&expr.And[i])
			if child != nil {
				out.And = append(out.And, *child)
			}
		}
	}
	if len(expr.Or) > 0 {
		out.Or = make([]QueryExpr, 0, len(expr.Or))
		for i := range expr.Or {
			child := normalizeExpr(&expr.Or[i])
			if child != nil {
				out.Or = append(out.Or, *child)
			}
		}
	}
	if expr.Not != nil {
		out.Not = normalizeExpr(expr.Not)
	}
	if expr.Pred != nil {
		out.Pred = &QueryPredicate{
			Field: strings.TrimSpace(expr.Pred.Field),
			Op:    strings.ToLower(strings.TrimSpace(expr.Pred.Op)),
			Value: expr.Pred.Value,
		}
	}
	return out
}

func normalizeSort(sort []QuerySort) []QuerySort {
	if len(sort) == 0 {
		return nil
	}
	out := make([]QuerySort, 0, len(sort))
	for _, item := range sort {
		out = append(out, QuerySort{
			Field:     strings.TrimSpace(item.Field),
			Direction: strings.ToLower(strings.TrimSpace(item.Direction)),
		})
	}
	return out
}

func collectFieldCapabilities(req QueryRequest) []FieldCapabilityBrief {
	seen := map[string]struct{}{}
	fields := make([]string, 0)
	collectFieldsFromExpr(req.Where, seen, &fields)
	for _, item := range req.Sort {
		field := strings.TrimSpace(item.Field)
		if field == "" {
			continue
		}
		if _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}
		fields = append(fields, field)
	}
	for _, item := range req.Facets {
		field := strings.TrimSpace(item.Field)
		if field == "" {
			continue
		}
		if _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}
		fields = append(fields, field)
	}
	sort.Strings(fields)
	out := make([]FieldCapabilityBrief, 0, len(fields))
	for _, field := range fields {
		out = append(out, FieldCapabilityBrief{
			Field:   field,
			Engines: []string{"postgres"},
		})
	}
	return out
}

func collectFieldsFromExpr(expr *QueryExpr, seen map[string]struct{}, fields *[]string) {
	if expr == nil {
		return
	}
	if expr.Pred != nil {
		field := strings.TrimSpace(expr.Pred.Field)
		if field != "" {
			if _, ok := seen[field]; !ok {
				seen[field] = struct{}{}
				*fields = append(*fields, field)
			}
		}
	}
	for i := range expr.And {
		collectFieldsFromExpr(&expr.And[i], seen, fields)
	}
	for i := range expr.Or {
		collectFieldsFromExpr(&expr.Or[i], seen, fields)
	}
	if expr.Not != nil {
		collectFieldsFromExpr(expr.Not, seen, fields)
	}
}
