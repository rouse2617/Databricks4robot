package searchindex

import (
	"context"
	"fmt"

	"github.com/CyberOrigin2077/cyber-databrew/internal/elasticsearch"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// lineageMgetChunk bounds one _mget round trip. ES defaults to 10k ids per
// batch (max_terms_count on the coordinating node). We stay well under so
// a 5000-id request from the batch lineage endpoint fits in a single call
// but a bigger call still stays safe by chunking.
const lineageMgetChunk = 5000

// lineageSourceFields is the _source projection asked for by the depth="all"
// path. Matches the three fields written into every asset doc by builder.go.
var lineageSourceFields = []string{
	"lineage_upstream_ids",
	"lineage_downstream_ids",
	"lineage_relation_types",
}

// LineageBatchClient is the minimal ES surface used by LineageBatchReader.
// Keeps the reader unit-testable without a live ES cluster — a fake that
// implements this one method drives the parse path.
type LineageBatchClient interface {
	MgetSource(ctx context.Context, ids []string, sourceFields []string) ([]elasticsearch.MgetSourceHit, error)
}

// LineageBatchReader implements repository.AssetLineageBatchRepository by
// projecting the three lineage fields off the assets index via _mget.
// Documents that don't exist in ES map to a zero-value projection (all three
// slice fields nil) so the caller can distinguish "no doc" from "doc with
// empty lineage".
type LineageBatchReader struct {
	es LineageBatchClient
}

// NewLineageBatchReader wires the ES-side reader used by the CYB-4305 batch
// lineage endpoint.
func NewLineageBatchReader(es LineageBatchClient) *LineageBatchReader {
	return &LineageBatchReader{es: es}
}

// LineageDocsByAssetID runs one or more `_mget` calls (chunked at
// lineageMgetChunk) and returns the projected upstream/downstream/relation
// slices keyed by asset_id. Missing docs are dropped from the map — callers
// look up by asset_id and treat "not present" as "no lineage projection".
func (r *LineageBatchReader) LineageDocsByAssetID(ctx context.Context, assetIDs []string) (map[string]repository.AssetLineageProjection, error) {
	out := make(map[string]repository.AssetLineageProjection, len(assetIDs))
	if len(assetIDs) == 0 {
		return out, nil
	}
	for start := 0; start < len(assetIDs); start += lineageMgetChunk {
		end := start + lineageMgetChunk
		if end > len(assetIDs) {
			end = len(assetIDs)
		}
		chunk := assetIDs[start:end]
		hits, err := r.es.MgetSource(ctx, chunk, lineageSourceFields)
		if err != nil {
			return nil, fmt.Errorf("searchindex LineageBatchReader.mget: %w", err)
		}
		for _, h := range hits {
			if !h.Found || h.Source == nil {
				continue
			}
			out[h.ID] = repository.AssetLineageProjection{
				UpstreamIDs:   extractStringSlice(h.Source["lineage_upstream_ids"]),
				DownstreamIDs: extractStringSlice(h.Source["lineage_downstream_ids"]),
				RelationTypes: extractStringSlice(h.Source["lineage_relation_types"]),
			}
		}
	}
	return out, nil
}

// extractStringSlice pulls a []string out of an ES _source array which
// unmarshals as []any. Returns nil for any other shape (missing key, wrong
// type). Never panics.
func extractStringSlice(v any) []string {
	if v == nil {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, elem := range arr {
		if s, ok := elem.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
