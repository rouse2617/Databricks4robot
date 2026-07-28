package searchindex

import (
	"context"
	"errors"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/elasticsearch"
)

// stubMgetClient captures the ids/sourceFields it was called with and returns
// canned hits. Multiple `hits` slices exercise the chunk-loop path.
type stubMgetClient struct {
	hits [][]elasticsearch.MgetSourceHit
	err  error
	// Recorded per call.
	calls []struct {
		ids     []string
		sources []string
	}
}

func (s *stubMgetClient) MgetSource(_ context.Context, ids []string, sourceFields []string) ([]elasticsearch.MgetSourceHit, error) {
	s.calls = append(s.calls, struct {
		ids     []string
		sources []string
	}{ids: append([]string(nil), ids...), sources: append([]string(nil), sourceFields...)})
	if s.err != nil {
		return nil, s.err
	}
	if len(s.hits) == 0 {
		return nil, nil
	}
	out := s.hits[0]
	s.hits = s.hits[1:]
	return out, nil
}

func TestLineageBatchReaderMapsESDocsToProjections(t *testing.T) {
	stub := &stubMgetClient{
		hits: [][]elasticsearch.MgetSourceHit{
			{
				{
					ID:    "aaaaaaaa",
					Found: true,
					Source: map[string]any{
						"lineage_upstream_ids":   []any{"parent01"},
						"lineage_downstream_ids": []any{"child01", "child02"},
						"lineage_relation_types": []any{"derive", "split"},
					},
				},
				// Not found — should be dropped from the output map.
				{ID: "missing1", Found: false},
				// Found but nil Source — same drop path (defensive).
				{ID: "malformed", Found: true, Source: nil},
			},
		},
	}
	r := NewLineageBatchReader(stub)
	got, err := r.LineageDocsByAssetID(context.Background(), []string{"aaaaaaaa", "missing1", "malformed"})
	if err != nil {
		t.Fatalf("LineageDocsByAssetID err: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 hit, got %d: %#v", len(got), got)
	}
	proj := got["aaaaaaaa"]
	if len(proj.UpstreamIDs) != 1 || proj.UpstreamIDs[0] != "parent01" {
		t.Fatalf("upstream = %#v", proj.UpstreamIDs)
	}
	if len(proj.DownstreamIDs) != 2 {
		t.Fatalf("downstream = %#v", proj.DownstreamIDs)
	}
	if len(proj.RelationTypes) != 2 || proj.RelationTypes[0] != "derive" {
		t.Fatalf("relation_types = %#v", proj.RelationTypes)
	}
	// Verify the source-field projection was passed through.
	if len(stub.calls) != 1 {
		t.Fatalf("expected 1 mget call, got %d", len(stub.calls))
	}
	wantSrc := []string{"lineage_upstream_ids", "lineage_downstream_ids", "lineage_relation_types"}
	if len(stub.calls[0].sources) != 3 {
		t.Fatalf("sources = %#v, want %#v", stub.calls[0].sources, wantSrc)
	}
}

func TestLineageBatchReaderEmptyIDsSkipsCall(t *testing.T) {
	stub := &stubMgetClient{}
	r := NewLineageBatchReader(stub)
	got, err := r.LineageDocsByAssetID(context.Background(), nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty map, got %#v", got)
	}
	if len(stub.calls) != 0 {
		t.Fatalf("empty ids must not call MgetSource, got %d calls", len(stub.calls))
	}
}

func TestLineageBatchReaderPropagatesESError(t *testing.T) {
	stub := &stubMgetClient{err: errors.New("boom")}
	r := NewLineageBatchReader(stub)
	_, err := r.LineageDocsByAssetID(context.Background(), []string{"aaaaaaaa"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestExtractStringSliceSurvivesShapes(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want []string
	}{
		{"nil", nil, nil},
		{"wrong type", "not-a-slice", nil},
		{"empty slice", []any{}, nil},
		{"clean strings", []any{"a", "b"}, []string{"a", "b"}},
		{"drops non-strings and empties", []any{"a", 42, "", "b"}, []string{"a", "b"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := extractStringSlice(c.in)
			if len(got) != len(c.want) {
				t.Fatalf("len = %d, want %d (got=%#v)", len(got), len(c.want), got)
			}
			for i, v := range c.want {
				if got[i] != v {
					t.Fatalf("[%d] = %q, want %q", i, got[i], v)
				}
			}
		})
	}
}
