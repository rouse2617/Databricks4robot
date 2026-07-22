package elasticsearch

import (
	"fmt"
	"strings"

	"github.com/CyberOrigin2077/cyber-databrew/internal/queryir"
)

func BuildQueryIRSearchBody(req queryir.QueryRequest, includeHits, includeFacets, trackTotalHits bool) (map[string]any, error) {
	query, err := compileExprQuery(req.Mode, req.Where)
	if err != nil {
		return nil, err
	}
	query = wrapCurrentRevisionOnlyQuery(req, query)
	body := map[string]any{
		"query": query,
	}
	if includeHits {
		from := 0
		size := 20
		if req.Page.Limit > 0 {
			size = req.Page.Limit
			if req.Page.Offset > 0 {
				from = req.Page.Offset
			}
		} else {
			if req.Page.Page > 1 && req.Page.PageSize > 0 {
				from = (req.Page.Page - 1) * req.Page.PageSize
			}
			if req.Page.PageSize > 0 {
				size = req.Page.PageSize
			}
		}
		body["from"] = from
		body["size"] = size
		body["sort"] = compileSortFields(req.Sort)
	} else {
		body["size"] = 0
		body["track_total_hits"] = trackTotalHits
	}
	if includeFacets {
		if aggs, err := buildFacetAggregations(req.Facets); err != nil {
			return nil, err
		} else if len(aggs) > 0 {
			body["aggs"] = aggs
		}
	}
	return body, nil
}

func compileExprQuery(mode string, expr *queryir.QueryExpr) (map[string]any, error) {
	if expr == nil {
		return map[string]any{"match_all": map[string]any{}}, nil
	}
	switch {
	case expr.Pred != nil:
		return compilePredicateQuery(mode, *expr.Pred)
	case len(expr.And) > 0:
		clauses := make([]map[string]any, 0, len(expr.And))
		for _, child := range expr.And {
			q, err := compileExprQuery(mode, &child)
			if err != nil {
				return nil, err
			}
			clauses = append(clauses, q)
		}
		return map[string]any{"bool": map[string]any{"must": clauses}}, nil
	case len(expr.Or) > 0:
		clauses := make([]map[string]any, 0, len(expr.Or))
		for _, child := range expr.Or {
			q, err := compileExprQuery(mode, &child)
			if err != nil {
				return nil, err
			}
			clauses = append(clauses, q)
		}
		return map[string]any{
			"bool": map[string]any{
				"should":               clauses,
				"minimum_should_match": 1,
			},
		}, nil
	case expr.Not != nil:
		q, err := compileExprQuery(mode, expr.Not)
		if err != nil {
			return nil, err
		}
		return map[string]any{"bool": map[string]any{"must_not": []map[string]any{q}}}, nil
	default:
		return nil, fmt.Errorf("invalid where expression: empty node")
	}
}

func compilePredicateQuery(mode string, pred queryir.QueryPredicate) (map[string]any, error) {
	if strings.TrimSpace(pred.Field) == "_fulltext" {
		return compileFulltextPredicate(mode, pred)
	}
	valueStr, err := encodeESValue(pred.Value)
	if err != nil {
		return nil, err
	}
	op := strings.ToLower(strings.TrimSpace(pred.Op))
	filterOp := FilterOp{
		Field: pred.Field,
		Op:    op,
		Value: valueStr,
	}
	if path, ok := nestedPath(filterOp.Field); ok {
		clause := buildNestedClause(path, filterOp)
		if op == "ne" {
			return map[string]any{"bool": map[string]any{"must_not": []map[string]any{clause}}}, nil
		}
		return clause, nil
	}
	positive, negative := buildFilterClause(filterOp)
	switch {
	case positive != nil && negative == nil:
		return positive, nil
	case positive == nil && negative != nil:
		return map[string]any{"bool": map[string]any{"must_not": []map[string]any{negative}}}, nil
	default:
		return nil, fmt.Errorf("unsupported predicate %s %s", pred.Field, pred.Op)
	}
}

func compileFulltextPredicate(mode string, pred queryir.QueryPredicate) (map[string]any, error) {
	value := strings.TrimSpace(fmt.Sprint(pred.Value))
	if value == "" {
		return map[string]any{"match_all": map[string]any{}}, nil
	}
	return buildSearchModeQuery(mode, value), nil
}

func buildFacetAggregations(facets []queryir.QueryFacet) (map[string]any, error) {
	if len(facets) == 0 {
		return nil, nil
	}
	aggs := make(map[string]any, len(facets))
	for _, facet := range facets {
		field := strings.TrimSpace(facet.Field)
		if field == "" {
			continue
		}
		size := facet.Size
		if size <= 0 {
			size = 20
		}
		esField, err := facetFieldPath(field)
		if err != nil {
			return nil, err
		}
		aggs[field] = map[string]any{
			"terms": map[string]any{
				"field": esField,
				"size":  size,
			},
		}
	}
	return aggs, nil
}

func facetFieldPath(field string) (string, error) {
	switch field {
	case "lifecycle_state", "asset_type", "owner", "mcap.vendor_id", "mcap.scene_id":
		return keywordAggField(field), nil
	// CYB-3715: flatten mirror columns are top-level string fields on the doc;
	// facet on the direct name so ES matches PG's PGSupportedFacetFields set.
	case "camera_model", "data_source", "collection_method", "source_platform":
		return keywordAggField(field), nil
	case "tag.priority":
		return "tags_flat.priority", nil
	case "tag.quality":
		return "tags_flat.quality", nil
	case "env":
		// CYB-3297 Phase B: env lives in the flattened metadata field; terms agg
		// on the subkey surfaces the real env values so the facet is discoverable
		// instead of a hardcoded list.
		return "metadata.env", nil
	default:
		return "", fmt.Errorf("unsupported facet field %q", field)
	}
}

func compileSortFields(sort []queryir.QuerySort) []map[string]any {
	if len(sort) == 0 {
		return []map[string]any{{"updated_at": map[string]any{"order": "desc"}}}
	}
	item := sort[0]
	order := "asc"
	if strings.ToLower(strings.TrimSpace(item.Direction)) == "desc" {
		order = "desc"
	}
	return []map[string]any{{strings.TrimSpace(item.Field): map[string]any{"order": order}}}
}

func encodeESValue(v any) (string, error) {
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
	case []string:
		out := make([]string, len(x))
		for i, item := range x {
			out[i] = fmt.Sprintf("%q", item)
		}
		return "[" + strings.Join(out, ",") + "]", nil
	case []any:
		out := make([]string, len(x))
		for i, item := range x {
			val, err := encodeESValue(item)
			if err != nil {
				return "", err
			}
			out[i] = val
		}
		return "[" + strings.Join(out, ",") + "]", nil
	default:
		return fmt.Sprint(x), nil
	}
}
