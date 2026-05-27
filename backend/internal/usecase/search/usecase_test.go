package search

import (
	"context"
	"reflect"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/elasticsearch"
)

type fakeESRepo struct {
	docs       map[string]map[string]any
	lastSearch elasticsearch.SearchRequest
}

func (f *fakeESRepo) Search(_ context.Context, req elasticsearch.SearchRequest) (*elasticsearch.SearchResponse, error) {
	f.lastSearch = req
	hits := make([]elasticsearch.SearchHit, 0, len(req.AssetIDs))
	for _, id := range req.AssetIDs {
		if doc, ok := f.docs[id]; ok {
			hits = append(hits, elasticsearch.SearchHit{ID: id, Source: cloneDoc(doc)})
		}
	}
	return &elasticsearch.SearchResponse{Total: int64(len(hits)), Hits: hits}, nil
}

func (f *fakeESRepo) GetDocumentSource(_ context.Context, id string) (map[string]any, bool, error) {
	doc, ok := f.docs[id]
	if !ok {
		return nil, false, nil
	}
	return cloneDoc(doc), true, nil
}

func cloneDoc(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func TestSearchAssets_LineageDownstreamDepthTwo(t *testing.T) {
	repo := &fakeESRepo{docs: map[string]map[string]any{
		"a": {
			"asset_id":                 "a",
			"lineage_downstream_ids":   []any{"b"},
			"lineage_relation_types":   []any{"derived_from"},
		},
		"b": {
			"asset_id":                 "b",
			"lineage_downstream_ids":   []any{"c"},
			"lineage_relation_types":   []any{"derived_from"},
		},
		"c": {
			"asset_id":               "c",
			"lineage_relation_types": []any{"derived_from"},
		},
	}}

	resp, err := New(repo).SearchAssets(context.Background(), SearchAssetsRequest{
		LineageWith:      "a",
		LineageDirection: DirectionDownstream,
		LineageDepth:     2,
		RelationTypes:    []string{"derived_from"},
	})
	if err != nil {
		t.Fatalf("SearchAssets: %v", err)
	}
	if !reflect.DeepEqual(repo.lastSearch.AssetIDs, []string{"b", "c"}) {
		t.Fatalf("asset ids = %#v, want [b c]", repo.lastSearch.AssetIDs)
	}
	if len(resp.Hits) != 2 {
		t.Fatalf("hits = %d, want 2", len(resp.Hits))
	}
	if _, ok := resp.Hits[0].Source["lineage_relation"].(map[string]any); !ok {
		t.Fatalf("missing lineage_relation annotation: %#v", resp.Hits[0].Source)
	}
}

func TestSearchAssets_LineageSeedMissing(t *testing.T) {
	_, err := New(&fakeESRepo{docs: map[string]map[string]any{}}).SearchAssets(
		context.Background(),
		SearchAssetsRequest{LineageWith: "missing"},
	)
	if err != ErrLineageSeedNotFound {
		t.Fatalf("err = %v, want ErrLineageSeedNotFound", err)
	}
}
