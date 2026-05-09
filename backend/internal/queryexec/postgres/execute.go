package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/queryir"
	assetUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/asset"
)

func (e *Executor) Execute(ctx context.Context, assetUsecase *assetUC.Usecase, compiled *queryir.CompiledQuery) ([]*models.Asset, int64, error) {
	whereClause, _, err := BuildExprWhereClause(compiled.NormalizedQuery.Where, 1)
	if err != nil {
		return nil, 0, err
	}
	whereSQL := whereClause.SQL
	args := append([]interface{}{}, whereClause.Args...)
	if len(compiled.CandidateAssetIDs) == 0 && compiled.CandidateAssetIDs != nil {
		return []*models.Asset{}, 0, nil
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
	orderBy, err := filter.ResolveSortBy(compiled.SortBy)
	if err != nil {
		return nil, 0, err
	}
	return assetUsecase.ListWithFilters(ctx, whereSQL, args, compiled.Page, compiled.PageSize, orderBy)
}

func buildCandidateIDsClause(ids []string, startParam int) (string, []interface{}) {
	placeholders := make([]string, 0, len(ids))
	args := make([]interface{}, 0, len(ids))
	for i, id := range ids {
		placeholders = append(placeholders, fmt.Sprintf("$%d", startParam+i))
		args = append(args, id)
	}
	return fmt.Sprintf("asset_id IN (%s)", strings.Join(placeholders, ", ")), args
}
