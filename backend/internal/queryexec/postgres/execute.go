package postgres

import (
	"context"
	"fmt"

	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/queryir"
	assetUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/asset"
)

func (e *Executor) Execute(ctx context.Context, assetUsecase *assetUC.Usecase, compiled *queryir.CompiledQuery) ([]*models.Asset, int64, error) {
	whereSQL, args, orderByClause, empty, err := e.buildListParams(compiled)
	if err != nil {
		return nil, 0, err
	}
	if empty {
		return []*models.Asset{}, 0, nil
	}
	return assetUsecase.ListWithFilters(ctx, whereSQL, args, compiled.Page, compiled.PageSize, orderByClause)
}

// ExecutePage runs the list query without COUNT(*).
func (e *Executor) ExecutePage(ctx context.Context, assetUsecase *assetUC.Usecase, compiled *queryir.CompiledQuery) ([]*models.Asset, error) {
	whereSQL, args, orderByClause, empty, err := e.buildListParams(compiled)
	if err != nil {
		return nil, err
	}
	if empty {
		return []*models.Asset{}, nil
	}
	return assetUsecase.ListWithFiltersPage(ctx, whereSQL, args, compiled.Page, compiled.PageSize, orderByClause)
}

func (e *Executor) buildListParams(compiled *queryir.CompiledQuery) (whereSQL string, args []interface{}, orderBy filter.OrderByClause, empty bool, err error) {
	whereClause, _, err := BuildExprWhereClause(compiled.NormalizedQuery.Where, 1)
	if err != nil {
		return "", nil, filter.OrderByClause{}, false, err
	}
	whereSQL = whereClause.SQL
	args = append([]interface{}{}, whereClause.Args...)
	if len(compiled.CandidateAssetIDs) == 0 && compiled.CandidateAssetIDs != nil {
		return "", nil, filter.OrderByClause{}, true, nil
	}
	if len(compiled.CandidateAssetIDs) > 0 {
		candidateSQL, candidateArgs := buildCandidateIDsClause(compiled.CandidateAssetIDs, len(args)+1)
		if whereSQL == "" {
			whereSQL = candidateSQL
		} else {
			whereSQL = fmt.Sprintf("(%s) AND (%s)", whereSQL, candidateSQL)
		}
		args = append(args, candidateArgs...)
	}
	if queryir.ApplyCurrentOnlyFilter(compiled.NormalizedQuery) {
		if whereSQL == "" {
			whereSQL = queryir.CurrentRevisionOnlySQL
		} else {
			whereSQL = fmt.Sprintf("%s AND %s", queryir.CurrentRevisionOnlySQL, whereSQL)
		}
	}
	orderByClause, _, err := filter.ResolveSortBy(compiled.SortBy, len(args)+1)
	if err != nil {
		return "", nil, filter.OrderByClause{}, false, err
	}
	return whereSQL, args, orderByClause, false, nil
}

func buildCandidateIDsClause(ids []string, startParam int) (string, []interface{}) {
	// Use a single array bind to avoid hitting PostgreSQL's 65535 parameter limit
	// when candidate ID sets are large.
	return fmt.Sprintf("asset_id = ANY($%d::text[])", startParam), []interface{}{ids}
}
