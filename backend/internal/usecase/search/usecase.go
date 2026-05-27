package search

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/CyberOrigin2077/cyber-databrew/internal/elasticsearch"
)

const (
	DirectionUpstream   = "upstream"
	DirectionDownstream = "downstream"
	DirectionBoth       = "both"
)

var ErrLineageSeedNotFound = errors.New("lineage seed asset not found")

type ElasticsearchRepository interface {
	Search(ctx context.Context, req elasticsearch.SearchRequest) (*elasticsearch.SearchResponse, error)
	GetDocumentSource(ctx context.Context, id string) (map[string]any, bool, error)
}

type Usecase struct {
	es ElasticsearchRepository
}

type SearchAssetsRequest struct {
	Mode             string
	Query            string
	Filters          []elasticsearch.FilterOp
	Page             int
	PageSize         int
	LineageWith      string
	LineageDirection string
	LineageDepth     int
	RelationTypes    []string
}

func New(es ElasticsearchRepository) *Usecase {
	return &Usecase{es: es}
}

func (u *Usecase) SearchAssets(ctx context.Context, req SearchAssetsRequest) (*elasticsearch.SearchResponse, error) {
	if u == nil || u.es == nil {
		return nil, fmt.Errorf("search usecase: elasticsearch repository is not configured")
	}

	esReq := elasticsearch.SearchRequest{
		Mode:     req.Mode,
		Query:    req.Query,
		Filters:  req.Filters,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	if strings.TrimSpace(req.LineageWith) != "" {
		ids, err := u.resolveLineageAssetIDs(ctx, req)
		if err != nil {
			return nil, err
		}
		if len(ids) == 0 {
			return &elasticsearch.SearchResponse{Hits: []elasticsearch.SearchHit{}, Total: 0}, nil
		}
		esReq.AssetIDs = ids
	}

	resp, err := u.es.Search(ctx, esReq)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.LineageWith) != "" {
		u.annotateLineage(resp, req)
	}
	return resp, nil
}

func (u *Usecase) resolveLineageAssetIDs(ctx context.Context, req SearchAssetsRequest) ([]string, error) {
	seed := strings.TrimSpace(req.LineageWith)
	depth := req.LineageDepth
	if depth < 1 {
		depth = 1
	}
	if depth > 3 {
		depth = 3
	}
	direction := normalizeDirection(req.LineageDirection)
	relationTypeSet := stringSet(req.RelationTypes)

	if _, ok, err := u.es.GetDocumentSource(ctx, seed); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrLineageSeedNotFound
	}

	visited := map[string]struct{}{seed: {}}
	result := map[string]struct{}{}
	frontier := []string{seed}
	for level := 0; level < depth && len(frontier) > 0; level++ {
		next := make([]string, 0)
		for _, id := range frontier {
			doc, ok, err := u.es.GetDocumentSource(ctx, id)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			if len(relationTypeSet) > 0 && !intersects(stringSlice(doc["lineage_relation_types"]), relationTypeSet) {
				continue
			}
			for _, relatedID := range relatedIDs(doc, direction) {
				if _, seen := visited[relatedID]; seen {
					continue
				}
				visited[relatedID] = struct{}{}
				result[relatedID] = struct{}{}
				next = append(next, relatedID)
			}
		}
		frontier = next
	}

	out := make([]string, 0, len(result))
	for id := range result {
		out = append(out, id)
	}
	sort.Strings(out)
	return out, nil
}

func (u *Usecase) annotateLineage(resp *elasticsearch.SearchResponse, req SearchAssetsRequest) {
	if resp == nil {
		return
	}
	relationTypeSet := stringSet(req.RelationTypes)
	depth := req.LineageDepth
	if depth < 1 {
		depth = 1
	}
	if depth > 3 {
		depth = 3
	}
	for i := range resp.Hits {
		if resp.Hits[i].Source == nil {
			resp.Hits[i].Source = map[string]any{}
		}
		relationTypes := stringSlice(resp.Hits[i].Source["lineage_relation_types"])
		if len(relationTypeSet) > 0 {
			relationTypes = filterStrings(relationTypes, relationTypeSet)
		}
		resp.Hits[i].Source["lineage_relation"] = map[string]any{
			"asset_id":       strings.TrimSpace(req.LineageWith),
			"direction":      normalizeDirection(req.LineageDirection),
			"depth":          depth,
			"relation_types": relationTypes,
		}
	}
}

func normalizeDirection(direction string) string {
	switch strings.ToLower(strings.TrimSpace(direction)) {
	case DirectionUpstream:
		return DirectionUpstream
	case DirectionDownstream:
		return DirectionDownstream
	default:
		return DirectionBoth
	}
}

func relatedIDs(doc map[string]any, direction string) []string {
	var out []string
	if direction == DirectionUpstream || direction == DirectionBoth {
		out = append(out, stringSlice(doc["lineage_upstream_ids"])...)
	}
	if direction == DirectionDownstream || direction == DirectionBoth {
		out = append(out, stringSlice(doc["lineage_downstream_ids"])...)
	}
	return out
}

func stringSlice(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, raw := range t {
			if s, ok := raw.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func stringSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, v := range values {
		v = strings.ToLower(strings.TrimSpace(v))
		if v != "" {
			out[v] = struct{}{}
		}
	}
	return out
}

func intersects(values []string, allowed map[string]struct{}) bool {
	for _, v := range values {
		if _, ok := allowed[strings.ToLower(strings.TrimSpace(v))]; ok {
			return true
		}
	}
	return false
}

func filterStrings(values []string, allowed map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if _, ok := allowed[strings.ToLower(strings.TrimSpace(v))]; ok {
			out = append(out, v)
		}
	}
	return out
}
