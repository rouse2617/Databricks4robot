package elasticsearch

import (
	"encoding/json"
	"strings"
	"testing"
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
		`"multi_match"`,
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
