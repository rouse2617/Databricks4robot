package elasticsearch

import (
	"context"
	"fmt"

	corees "github.com/CyberOrigin2077/cyber-databrew/internal/elasticsearch"
	"github.com/CyberOrigin2077/cyber-databrew/internal/queryir"
)

const maxRecallCandidateIDs = 10000

func (e *Executor) Execute(ctx context.Context, body map[string]any, collectCandidates bool) (*queryir.CompiledQuery, error) {
	if e == nil || e.client == nil {
		return &queryir.CompiledQuery{}, nil
	}

	// For match_all queries (common assets landing page), scrolling all IDs adds
	// heavy ES overhead but does not improve correctness versus PG refine.
	if collectCandidates && isMatchAllQuery(body) {
		collectCandidates = false
	}
	// For very broad recall, skip candidate transfer altogether.
	if collectCandidates {
		// We only know the total after the first ES call below; this flag
		// will be checked after we get the total.
	}

	var searchResp *corees.SearchResponse
	var scrollID string
	var err error

	if collectCandidates {
		// One ES call for facets + first page hits + scroll context
		searchResp, scrollID, err = e.client.SearchBodyScroll(ctx, body)
	} else {
		searchResp, err = e.client.SearchBody(ctx, body)
	}
	if err != nil {
		return nil, err
	}

	facets := make(map[string][]queryir.FacetBucket, len(searchResp.Aggregations))
	for name, buckets := range searchResp.Aggregations {
		facets[name] = mapBuckets(buckets)
	}
	out := &queryir.CompiledQuery{
		Facets:     facets,
		MatchTotal: searchResp.Total,
	}

	if !collectCandidates {
		return out, nil
	}
	defer func() {
		if scrollID != "" {
			e.client.ClearScroll(ctx, scrollID)
		}
	}()

	// If the recall matched too many docs, skip candidate transfer (PG refine is enough).
	if searchResp.Total > maxRecallCandidateIDs {
		out.Warnings = append(out.Warnings, fmt.Sprintf(
			"elasticsearch recall matched %d docs (> %d); skipped candidate transfer and returned elasticsearch results directly",
			searchResp.Total, maxRecallCandidateIDs,
		))
		if len(searchResp.Hits) > 0 {
			hits := make([]map[string]any, 0, len(searchResp.Hits))
			for _, hit := range searchResp.Hits {
				hits = append(hits, hit.Source)
			}
			out.ESResults = hits
		}
		out.MatchTotal = searchResp.Total
		return out, nil
	}

	// Extract IDs from first page hits (already part of the same ES call).
	ids := make([]string, 0, searchResp.Total)
	for _, hit := range searchResp.Hits {
		if hit.ID != "" {
			ids = append(ids, hit.ID)
		}
	}

	// Continue scrolling for remaining IDs.
	for scrollID != "" && len(ids) < int(searchResp.Total) {
		var page []string
		page, scrollID, err = e.client.ScrollNext(ctx, scrollID)
		if err != nil {
			return nil, fmt.Errorf("elasticsearch: scroll IDs: %w", err)
		}
		if len(page) == 0 {
			break
		}
		ids = append(ids, page...)
	}

	out.CandidateAssetIDs = ids
	return out, nil
}

func mapBuckets(buckets []corees.AggBucket) []queryir.FacetBucket {
	out := make([]queryir.FacetBucket, 0, len(buckets))
	for _, bucket := range buckets {
		out = append(out, queryir.FacetBucket{
			Value: bucket.Key,
			Count: bucket.DocCount,
		})
	}
	return out
}

func isMatchAllQuery(body map[string]any) bool {
	query, ok := body["query"].(map[string]any)
	if !ok || len(query) != 1 {
		return false
	}
	_, ok = query["match_all"]
	return ok
}
