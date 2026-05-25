package postgres

import (
	"fmt"
	"strings"

	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/queryir"
)

type SQLClause struct {
	SQL  string
	Args []interface{}
}

func BuildExprWhereClause(expr *queryir.QueryExpr, startParam int) (*SQLClause, int, error) {
	if expr == nil {
		return &SQLClause{}, startParam, nil
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
		return nil, startParam, fmt.Errorf("invalid where expression: empty node")
	}
	if kindCount > 1 {
		return nil, startParam, fmt.Errorf("invalid where expression: exactly one of and/or/not/pred must be set")
	}

	switch {
	case expr.Pred != nil:
		return buildPredicateClause(*expr.Pred, startParam)
	case len(expr.And) > 0:
		return buildLogicalClause("AND", expr.And, startParam)
	case len(expr.Or) > 0:
		return buildLogicalClause("OR", expr.Or, startParam)
	case expr.Not != nil:
		child, nextParam, err := BuildExprWhereClause(expr.Not, startParam)
		if err != nil {
			return nil, startParam, err
		}
		if child.SQL == "" {
			return &SQLClause{}, nextParam, nil
		}
		return &SQLClause{
			SQL:  fmt.Sprintf("NOT (%s)", child.SQL),
			Args: child.Args,
		}, nextParam, nil
	default:
		return nil, startParam, fmt.Errorf("invalid where expression: empty node")
	}
}

func buildLogicalClause(joiner string, children []queryir.QueryExpr, startParam int) (*SQLClause, int, error) {
	parts := make([]string, 0, len(children))
	args := make([]interface{}, 0)
	nextParam := startParam
	for i := range children {
		clause, after, err := BuildExprWhereClause(&children[i], nextParam)
		if err != nil {
			return nil, startParam, err
		}
		nextParam = after
		if clause == nil || clause.SQL == "" {
			continue
		}
		parts = append(parts, "("+clause.SQL+")")
		args = append(args, clause.Args...)
	}
	if len(parts) == 0 {
		return &SQLClause{}, nextParam, nil
	}
	return &SQLClause{
		SQL:  strings.Join(parts, " "+joiner+" "),
		Args: args,
	}, nextParam, nil
}

func buildPredicateClause(pred queryir.QueryPredicate, startParam int) (*SQLClause, int, error) {
	if strings.TrimSpace(pred.Field) == "_fulltext" {
		return buildFulltextClause(pred, startParam)
	}
	filterStr := serializePredicate(pred)
	parsed, err := filter.ParseFilter(filterStr)
	if err != nil {
		return nil, startParam, err
	}
	if err := filter.ValidateFilters([]filter.Filter{*parsed}); err != nil {
		return nil, startParam, err
	}
	wc, err := filter.BuildWhereClause([]filter.Filter{*parsed}, startParam)
	if err != nil {
		return nil, startParam, err
	}
	return &SQLClause{
		SQL:  wc.SQL,
		Args: wc.Args,
	}, startParam + len(wc.Args), nil
}

func serializePredicate(pred queryir.QueryPredicate) string {
	filterObj := filter.Filter{
		Field: pred.Field,
		Op:    filter.OperatorMap[strings.ToLower(strings.TrimSpace(pred.Op))],
		Value: pred.Value,
	}
	return filterObj.Serialize()
}

func buildFulltextClause(pred queryir.QueryPredicate, startParam int) (*SQLClause, int, error) {
	op := strings.ToLower(strings.TrimSpace(pred.Op))
	switch op {
	case "ilike", "like", "eq":
	default:
		return nil, startParam, fmt.Errorf("%w %q", queryir.ErrUnsupportedOperator, pred.Op)
	}
	value := fmt.Sprint(pred.Value)
	if strings.TrimSpace(value) == "" {
		return &SQLClause{}, startParam, nil
	}
	pattern := value
	if !strings.Contains(pattern, "%") && !strings.Contains(pattern, "_") {
		pattern = "%" + pattern + "%"
	}
	sql := fmt.Sprintf(
		"(assets.asset_id::text ILIKE $%d OR assets.mcap_file_id::text ILIKE $%d OR owner ILIKE $%d OR reviewer ILIKE $%d OR EXISTS (SELECT 1 FROM asset_tags t WHERE t.asset_id = assets.asset_id AND t.tag_key = 'notes' AND t.tag_value ILIKE $%d))",
		startParam, startParam,
		startParam, startParam, startParam,
	)
	return &SQLClause{
		SQL:  sql,
		Args: []interface{}{pattern},
	}, startParam + 1, nil
}
