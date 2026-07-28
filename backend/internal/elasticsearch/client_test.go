package elasticsearch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/queryir"
)

func TestBuildSearchBody_NestedTagsAndDurationBetween(t *testing.T) {
	req := SearchRequest{
		Query: "highway",
		Filters: []FilterOp{
			{Field: "duration_ms", Op: "between", Value: "9000,11000"},
			{Field: "tags.scene", Op: "eq", Value: "highway"},
			{Field: "tags.source_type", Op: "eq", Value: "algo"},
			{Field: "tags_flat.priority", Op: "eq", Value: "high"},
			{Field: "mcap.vendor_id", Op: "eq", Value: "acme"},
		},
		Page:     1,
		PageSize: 20,
	}
	body := buildSearchBody(req)
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	for _, want := range []string{
		`"nested"`,
		`"path":"tags"`,
		`"tags.key"`,
		`"scene"`,
		`"tags.value"`,
		`"highway"`,
		`"minimum_should_match"`,
		`"duration_ms"`,
		`"gte":"9000"`,
		`"lte":"11000"`,
		`tags_flat.priority`,
		`mcap.vendor_id`,
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in body: %s", want, s)
		}
	}
}

func TestBuildSearchBody_EnvFilterMapsToMetadataEnv(t *testing.T) {
	req := SearchRequest{
		Filters: []FilterOp{
			{Field: "env", Op: "eq", Value: "outdoor"},
		},
		Page:     1,
		PageSize: 20,
	}
	body := buildSearchBody(req)
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, `"metadata.env"`) || !strings.Contains(s, `"outdoor"`) {
		t.Fatalf("expected env filter to target metadata.env, got: %s", s)
	}
}

func TestBuildSearchBody_NestedAlgosScore(t *testing.T) {
	req := SearchRequest{
		Filters: []FilterOp{
			{Field: "algos.hand_tracking.score", Op: "gt", Value: "0.8"},
		},
		Page:     2,
		PageSize: 10,
	}
	body := buildSearchBody(req)
	raw, _ := json.Marshal(body)
	s := string(raw)
	if !strings.Contains(s, `"algos.name"`) || !strings.Contains(s, `"hand_tracking"`) {
		t.Fatalf("expected algos name clause: %s", s)
	}
	if !strings.Contains(s, `"algos.result_score"`) || !strings.Contains(s, `"gt"`) {
		t.Fatalf("expected result_score range: %s", s)
	}
	if body["from"] != 10 {
		t.Fatalf("expected from=10, got %v", body["from"])
	}
}

func TestBuildSearchBody_SemanticModeUsesFuzziness(t *testing.T) {
	req := SearchRequest{
		Mode:     "semantic",
		Query:    "forklift",
		Page:     1,
		PageSize: 20,
	}
	body := buildSearchBody(req)
	raw, _ := json.Marshal(body)
	s := string(raw)
	if !strings.Contains(s, `"fuzziness":"AUTO"`) {
		t.Fatalf("expected semantic fuzziness in body: %s", s)
	}
}

func TestBuildSearchBody_SimilarModeUsesMoreLikeThis(t *testing.T) {
	req := SearchRequest{
		Mode:     "similar",
		Query:    "aset0001",
		Page:     1,
		PageSize: 20,
	}
	body := buildSearchBody(req)
	raw, _ := json.Marshal(body)
	s := string(raw)
	if !strings.Contains(s, `"more_like_this"`) || !strings.Contains(s, `"aset0001"`) {
		t.Fatalf("expected more_like_this body: %s", s)
	}
}

func TestBuildSearchBody_NestedActionsFilter(t *testing.T) {
	req := SearchRequest{
		Filters: []FilterOp{
			{Field: "actions.source_type", Op: "eq", Value: "algo"},
			{Field: "actions.confidence", Op: "gte", Value: "0.9"},
		},
		Page:     1,
		PageSize: 20,
	}
	body := buildSearchBody(req)
	raw, _ := json.Marshal(body)
	s := string(raw)
	for _, want := range []string{
		`"path":"actions"`,
		`"actions.source_type"`,
		`"actions.confidence"`,
		`"gte":"0.9"`,
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in body: %s", want, s)
		}
	}
}

func TestBuildSearchBody_AssetIDsTermsFilter(t *testing.T) {
	req := SearchRequest{
		AssetIDs: []string{"asset-b", "asset-a", "asset-b"},
		Page:     1,
		PageSize: 20,
	}
	body := buildSearchBody(req)
	query := body["query"].(map[string]any)
	boolQuery := query["bool"].(map[string]any)
	filters := boolQuery["filter"].([]map[string]any)
	found := false
	for _, clause := range filters {
		terms, ok := clause["terms"].(map[string]any)
		if !ok {
			continue
		}
		values, ok := terms["asset_id"].([]any)
		if !ok {
			continue
		}
		if len(values) == 2 && values[0] == "asset-b" && values[1] == "asset-a" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing asset_id terms filter: %#v", body)
	}
}

func TestBuildSearchBody_MixedNestedEqAndNeStayInSameTagGroup(t *testing.T) {
	req := SearchRequest{
		Filters: []FilterOp{
			{Field: "tags.quality", Op: "ne", Value: "good"},
			{Field: "tags.source_type", Op: "eq", Value: "algo"},
		},
		Page:     1,
		PageSize: 20,
	}

	body := buildSearchBody(req)
	boolQuery := body["query"].(map[string]any)["bool"].(map[string]any)
	filters := boolQuery["filter"].([]map[string]any)
	if len(filters) != 1 {
		t.Fatalf("expected 1 grouped nested filter, got %d", len(filters))
	}

	inner := filters[0]["nested"].(map[string]any)["query"].(map[string]any)["bool"].(map[string]any)
	must := inner["must"].([]map[string]any)
	mustNot := inner["must_not"].([]map[string]any)

	if !containsTermClause(must, "tags.key", "quality") {
		t.Fatalf("expected tags.key=quality in must, got %#v", must)
	}
	if !containsTermClause(must, "tags.source_type", "algo") {
		t.Fatalf("expected tags.source_type=algo in must, got %#v", must)
	}
	if !containsTermClause(mustNot, "tags.value", "good") {
		t.Fatalf("expected tags.value=good in must_not, got %#v", mustNot)
	}
}

func TestBuildSearchBody_PureNegativeInnerTagClauseStaysOuterMustNot(t *testing.T) {
	req := SearchRequest{
		Filters: []FilterOp{
			{Field: "tags.source_type", Op: "ne", Value: "algo"},
		},
		Page:     1,
		PageSize: 20,
	}

	body := buildSearchBody(req)
	boolQuery := body["query"].(map[string]any)["bool"].(map[string]any)
	if _, ok := boolQuery["filter"]; ok {
		t.Fatalf("did not expect top-level filter for pure negative nested clause: %#v", boolQuery)
	}
	mustNot := boolQuery["must_not"].([]map[string]any)
	if len(mustNot) != 1 {
		t.Fatalf("expected 1 top-level must_not clause, got %d", len(mustNot))
	}

	inner := mustNot[0]["nested"].(map[string]any)["query"].(map[string]any)["bool"].(map[string]any)
	must := inner["must"].([]map[string]any)
	if !containsTermClause(must, "tags.source_type", "algo") {
		t.Fatalf("expected tags.source_type=algo in outer must_not nested clause, got %#v", must)
	}
}

func containsTermClause(clauses []map[string]any, field, value string) bool {
	for _, clause := range clauses {
		term, ok := clause["term"].(map[string]any)
		if !ok {
			continue
		}
		if got, exists := term[field]; exists && got == value {
			return true
		}
	}
	return false
}

func TestDeleteAllDocuments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/assets/_delete_by_query" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("conflicts"); got != "proceed" {
			t.Fatalf("unexpected conflicts query: %q", got)
		}
		if got := r.URL.Query().Get("refresh"); got != "true" {
			t.Fatalf("unexpected refresh query: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"deleted": 7})
	}))
	defer srv.Close()

	client := New(srv.URL, "assets", "", "")
	deleted, err := client.DeleteAllDocuments(context.Background())
	if err != nil {
		t.Fatalf("DeleteAllDocuments() error: %v", err)
	}
	if deleted != 7 {
		t.Fatalf("expected deleted=7, got %d", deleted)
	}
}

func TestDeleteAllDocuments_IndexMissingReturnsZero(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/assets/_delete_by_query" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"type":"index_not_found_exception"}}`))
	}))
	defer srv.Close()

	client := New(srv.URL, "assets", "", "")
	deleted, err := client.DeleteAllDocuments(context.Background())
	if err != nil {
		t.Fatalf("DeleteAllDocuments() error: %v", err)
	}
	if deleted != 0 {
		t.Fatalf("expected deleted=0 for missing index, got %d", deleted)
	}
}

func TestDeleteAllDocuments_IndexNotFoundReturnsZero(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/assets/_delete_by_query" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"type":"index_not_found_exception"}}`))
	}))
	defer srv.Close()

	client := New(srv.URL, "assets", "", "")
	deleted, err := client.DeleteAllDocuments(context.Background())
	if err != nil {
		t.Fatalf("DeleteAllDocuments() error: %v", err)
	}
	if deleted != 0 {
		t.Fatalf("expected deleted=0 for missing index, got %d", deleted)
	}
}

func TestNormalizeScalarField_TagSingularMapsToTagsFlat(t *testing.T) {
	tests := []struct {
		field string
		want  string
	}{
		{"tag.priority", "tags_flat.priority"},
		{"tag.quality", "tags_flat.quality"},
		{"tag.custom_key", "tags_flat.custom_key"},
		{"env", "metadata.env"},
		{"task", "metadata.task"},
		{"type", "asset_type"},
		{"tags_flat.priority", "tags_flat.priority"},
		{"unknown_field", "unknown_field"},
	}
	for _, tt := range tests {
		got := normalizeScalarField(tt.field)
		if got != tt.want {
			t.Errorf("normalizeScalarField(%q) = %q, want %q", tt.field, got, tt.want)
		}
	}
}

func TestBuildFilterClause_TagSingularFieldUsesTagsFlat(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		op      string
		value   string
		wantKey string
	}{
		{
			name:    "tag.priority scalar filter",
			field:   "tag.priority",
			op:      "eq",
			value:   "high",
			wantKey: "tags_flat.priority",
		},
		{
			name:    "tag.quality scalar filter",
			field:   "tag.quality",
			op:      "eq",
			value:   "verified",
			wantKey: "tags_flat.quality",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := FilterOp{Field: tt.field, Op: tt.op, Value: tt.value}
			positive, negative := buildFilterClause(f)
			if tt.op == "ne" && negative == nil {
				t.Fatalf("ne operator should produce negative clause")
			}
			if tt.op != "ne" && positive == nil {
				t.Fatalf("non-ne operator should produce positive clause")
			}
			clause := positive
			if tt.op == "ne" {
				clause = negative
			}
			raw, _ := json.Marshal(clause)
			s := string(raw)
			if !strings.Contains(s, tt.wantKey) {
				t.Errorf("expected %q in clause, got: %s", tt.wantKey, s)
			}
		})
	}
}

// TestBuildSearchModeQuery_IncludesSharedFulltextFields pins CYB-4011: the
// keyword/top-search query must match every field in
// queryir.FulltextExtraFields (grace_video_id, device_id, …) via a match clause
// with operator=and (whole-id match, no partial/fuzzy). Iterating the source
// list keeps the test honest as fields are added.
func TestBuildSearchModeQuery_IncludesSharedFulltextFields(t *testing.T) {
	q := buildSearchModeQuery("keyword", "019f9ede-0256-7a4c-a5b3-3b26fe82e554")
	raw, err := json.Marshal(q)
	if err != nil {
		t.Fatalf("marshal query: %v", err)
	}
	s := string(raw)
	// Baseline core fields still present.
	if !strings.Contains(s, `"asset_id"`) {
		t.Fatalf("core asset_id clause missing: %s", s)
	}
	// Every shared field appears as a match clause with operator=and.
	for _, f := range queryir.FulltextExtraFields {
		wantField := `"match":{"` + f + `":`
		if !strings.Contains(s, wantField) {
			t.Fatalf("expected match clause for %q, got: %s", f, s)
		}
	}
	if !strings.Contains(s, `"operator":"and"`) {
		t.Fatalf("expected operator=and for id fields, got: %s", s)
	}
	// Empty query must not panic and must degrade to a valid bool query.
	if _, err := json.Marshal(buildSearchModeQuery("keyword", "")); err != nil {
		t.Fatalf("empty query marshal: %v", err)
	}
}
