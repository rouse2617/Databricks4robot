package elasticsearch

import (
	"context"

	corees "data-platform/internal/elasticsearch"
	"data-platform/internal/queryir"
)

func (e *Executor) Execute(ctx context.Context, body map[string]any) (*queryir.CompiledQuery, error) {
	if e == nil || e.client == nil {
		return &queryir.CompiledQuery{}, nil
	}
	searchResp, err := e.client.SearchBody(ctx, body)
	if err != nil {
		return nil, err
	}
	ids, err := e.client.ListMatchingDocumentIDs(ctx, body, 1000)
	if err != nil {
		return nil, err
	}
	facets := make(map[string][]queryir.FacetBucket, len(searchResp.Aggregations))
	for name, buckets := range searchResp.Aggregations {
		facets[name] = mapBuckets(buckets)
	}
	return &queryir.CompiledQuery{
		CandidateAssetIDs: ids,
		Facets:            facets,
	}, nil
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
