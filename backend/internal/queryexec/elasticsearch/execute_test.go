package elasticsearch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	corees "data-platform/internal/elasticsearch"
)

func TestExecute_CollectsCandidateIDsAndFacets(t *testing.T) {
	scrollCalls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/assets/_search":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"hits":{"total":{"value":2},"hits":[{"_id":"aset0001","_score":1.0,"_source":{"asset_id":"aset0001"}}]},
				"aggregations":{"owner":{"buckets":[{"key":"alice","doc_count":2}]}}
			}`))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/assets/_search"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"_scroll_id":"scroll-1",
				"hits":{"hits":[{"_id":"aset0001"},{"_id":"aset0002"}]}
			}`))
		case r.Method == http.MethodPost && r.URL.Path == "/_search/scroll":
			scrollCalls++
			w.Header().Set("Content-Type", "application/json")
			if scrollCalls == 1 {
				_, _ = w.Write([]byte(`{"_scroll_id":"scroll-1","hits":{"hits":[]}}`))
				return
			}
			_, _ = w.Write([]byte(`{"_scroll_id":"scroll-1","hits":{"hits":[]}}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/_search/scroll":
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer srv.Close()

	exec := New(corees.New(srv.URL, "assets"))
	result, err := exec.Execute(context.Background(), map[string]any{
		"query": map[string]any{"match_all": map[string]any{}},
		"aggs": map[string]any{
			"owner": map[string]any{"terms": map[string]any{"field": "owner.keyword", "size": 20}},
		},
	})
	if err != nil {
		t.Fatalf("Execute() err = %v", err)
	}
	if len(result.CandidateAssetIDs) != 2 {
		t.Fatalf("unexpected candidate IDs: %#v", result.CandidateAssetIDs)
	}
	if len(result.Facets["owner"]) != 1 || result.Facets["owner"][0].Value != "alice" {
		t.Fatalf("unexpected facets: %#v", result.Facets)
	}
}
