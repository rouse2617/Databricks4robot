package elasticsearch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	corees "github.com/CyberOrigin2077/cyber-databrew/internal/elasticsearch"
)

func TestExecute_CollectsCandidateIDsAndFacets(t *testing.T) {
	scrollCalls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/assets/_search":
			// doScrollSearch: one POST returns aggregations + first-page hits + _scroll_id
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"_scroll_id":"scroll-1",
				"hits":{"total":{"value":2},"hits":[{"_id":"aset0001","_score":1.0,"_source":{"asset_id":"aset0001"}}]},
				"aggregations":{"owner":{"buckets":[{"key":"alice","doc_count":2}]}}
			}`))
		case r.Method == http.MethodPost && r.URL.Path == "/_search/scroll":
			scrollCalls++
			w.Header().Set("Content-Type", "application/json")
			if scrollCalls == 1 {
				// Return remaining hit (page 2 of scroll)
				_, _ = w.Write([]byte(`{"_scroll_id":"scroll-1","hits":{"hits":[{"_id":"aset0002"}]}}`))
				return
			}
			// Empty page terminates the scroll loop
			_, _ = w.Write([]byte(`{"_scroll_id":"scroll-1","hits":{"hits":[]}}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/_search/scroll":
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer srv.Close()

	exec := New(corees.New(srv.URL, "assets", "", ""))
	result, err := exec.Execute(context.Background(), map[string]any{
		"query": map[string]any{"term": map[string]any{"owner.keyword": "alice"}},
		"aggs": map[string]any{
			"owner": map[string]any{"terms": map[string]any{"field": "owner.keyword", "size": 20}},
		},
	}, true)
	if err != nil {
		t.Fatalf("Execute() err = %v", err)
	}
	if len(result.CandidateAssetIDs) != 2 {
		t.Fatalf("unexpected candidate IDs: %#v", result.CandidateAssetIDs)
	}
	if len(result.Facets["owner"]) != 1 || result.Facets["owner"][0].Value != "alice" {
		t.Fatalf("unexpected facets: %#v", result.Facets)
	}
	if scrollCalls == 0 {
		t.Fatalf("expected ScrollNext call(s), got 0")
	}
}

func TestExecute_SkipsCandidateIDsForMatchAll(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/assets/_search":
			// match_all → SearchBody (no scroll)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"hits":{"total":{"value":115400},"hits":[]},
				"aggregations":{"owner":{"buckets":[{"key":"collector-import","doc_count":115400}]}}
			}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer srv.Close()

	exec := New(corees.New(srv.URL, "assets", "", ""))
	result, err := exec.Execute(context.Background(), map[string]any{
		"query": map[string]any{"match_all": map[string]any{}},
		"aggs": map[string]any{
			"owner": map[string]any{"terms": map[string]any{"field": "owner.keyword", "size": 20}},
		},
	}, true)
	if err != nil {
		t.Fatalf("Execute() err = %v", err)
	}
	if len(result.CandidateAssetIDs) != 0 {
		t.Fatalf("expected no candidate IDs, got: %#v", result.CandidateAssetIDs)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got: %#v", result.Warnings)
	}
}

func TestExecute_SkipsBroadRecallCandidateTransfer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/assets/_search":
			// doScrollSearch with total > 10000 → skips candidate transfer
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"_scroll_id":"scroll-1",
				"hits":{"total":{"value":20001},"hits":[]},
				"aggregations":{"owner":{"buckets":[{"key":"collector-import","doc_count":20001}]}}
			}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/_search/scroll":
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer srv.Close()

	exec := New(corees.New(srv.URL, "assets", "", ""))
	result, err := exec.Execute(context.Background(), map[string]any{
		"query": map[string]any{"term": map[string]any{"owner.keyword": "collector-import"}},
	}, true)
	if err != nil {
		t.Fatalf("Execute() err = %v", err)
	}
	if len(result.CandidateAssetIDs) != 0 {
		t.Fatalf("expected no candidate IDs, got: %#v", result.CandidateAssetIDs)
	}
	if len(result.Warnings) == 0 {
		t.Fatalf("expected warning for broad recall skip")
	}
}

func TestExecute_FacetOnlySkipsCandidateCollection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/assets/_search":
			// collectCandidates=false → SearchBody (no scroll)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"hits":{"total":{"value":2},"hits":[]},
				"aggregations":{"owner":{"buckets":[{"key":"alice","doc_count":2}]}}
			}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer srv.Close()

	exec := New(corees.New(srv.URL, "assets", "", ""))
	result, err := exec.Execute(context.Background(), map[string]any{
		"query": map[string]any{"term": map[string]any{"owner.keyword": "alice"}},
		"aggs": map[string]any{
			"owner": map[string]any{"terms": map[string]any{"field": "owner.keyword", "size": 20}},
		},
	}, false)
	if err != nil {
		t.Fatalf("Execute() err = %v", err)
	}
	if len(result.CandidateAssetIDs) != 0 {
		t.Fatalf("expected no candidate IDs, got: %#v", result.CandidateAssetIDs)
	}
	if len(result.Facets["owner"]) != 1 || result.Facets["owner"][0].Value != "alice" {
		t.Fatalf("unexpected facets: %#v", result.Facets)
	}
}
