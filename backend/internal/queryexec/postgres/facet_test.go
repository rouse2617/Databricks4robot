package postgres

import (
	"context"
	"errors"
	"testing"

	pgrepo "github.com/CyberOrigin2077/cyber-databrew/internal/postgres"
	"github.com/CyberOrigin2077/cyber-databrew/internal/queryir"
)

type fakeSource struct {
	buckets map[string][]pgrepo.AssetFacetBucket
	errs    map[string]error
	seenWhere string
	seenArgs  []interface{}
	sizeSeen  map[string]int
	calls     int
}

func (f *fakeSource) FacetCounts(_ context.Context, field, whereSQL string, args []interface{}, size int) ([]pgrepo.AssetFacetBucket, error) {
	f.calls++
	f.seenWhere = whereSQL
	f.seenArgs = args
	if f.sizeSeen == nil {
		f.sizeSeen = map[string]int{}
	}
	f.sizeSeen[field] = size
	if err := f.errs[field]; err != nil {
		return nil, err
	}
	return f.buckets[field], nil
}

func newCompiled(fields ...string) *queryir.CompiledQuery {
	facets := make([]queryir.QueryFacet, 0, len(fields))
	for _, f := range fields {
		facets = append(facets, queryir.QueryFacet{Field: f, Size: 20})
	}
	return &queryir.CompiledQuery{
		NormalizedQuery: queryir.QueryRequest{
			SchemaVersion: queryir.SchemaVersionV1,
			Scope:         queryir.QueryScope{Resource: queryir.ResourceAssets},
			Facets:        facets,
		},
	}
}

func TestFacetExecutor_NilSourceReturnsEmpty(t *testing.T) {
	e := NewFacetExecutor(nil)
	out, warn, err := e.Execute(context.Background(), newCompiled("asset_type"))
	if err != nil || out != nil || warn != nil {
		t.Fatalf("nil source must return (nil, nil, nil); got %+v %+v %v", out, warn, err)
	}
}

func TestFacetExecutor_ExecutesSupportedFields(t *testing.T) {
	fake := &fakeSource{
		buckets: map[string][]pgrepo.AssetFacetBucket{
			"asset_type": {{Value: "segment", Count: 154}, {Value: "action", Count: 2}},
			"owner":      {{Value: "alice", Count: 3}},
		},
	}
	e := NewFacetExecutor(fake)
	out, warn, err := e.Execute(context.Background(), newCompiled("asset_type", "owner"))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(warn) != 0 {
		t.Fatalf("no warnings expected, got %v", warn)
	}
	if got := out["asset_type"]; len(got) != 2 || got[0].Value != "segment" || got[0].Count != 154 {
		t.Fatalf("asset_type buckets wrong: %+v", got)
	}
	if got := out["owner"]; len(got) != 1 || got[0].Value != "alice" || got[0].Count != 3 {
		t.Fatalf("owner buckets wrong: %+v", got)
	}
	if fake.calls != 2 {
		t.Fatalf("expected 2 source calls, got %d", fake.calls)
	}
}

func TestFacetExecutor_DropsUnsupportedFields(t *testing.T) {
	fake := &fakeSource{
		buckets: map[string][]pgrepo.AssetFacetBucket{
			"asset_type": {{Value: "segment", Count: 12}},
		},
	}
	e := NewFacetExecutor(fake)
	out, warn, err := e.Execute(context.Background(), newCompiled("asset_type", "mcap.vendor_id", "tag.priority"))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if _, ok := out["mcap.vendor_id"]; ok {
		t.Fatal("unsupported field mcap.vendor_id must be dropped")
	}
	if _, ok := out["tag.priority"]; ok {
		t.Fatal("unsupported field tag.priority must be dropped")
	}
	if len(warn) != 2 {
		t.Fatalf("expected 2 warnings for dropped fields, got %v", warn)
	}
}

func TestFacetExecutor_PerFieldErrorSurfacesAsWarningNotFailure(t *testing.T) {
	fake := &fakeSource{
		buckets: map[string][]pgrepo.AssetFacetBucket{
			"asset_type": {{Value: "segment", Count: 12}},
		},
		errs: map[string]error{
			"owner": errors.New("owner boom"),
		},
	}
	e := NewFacetExecutor(fake)
	out, warn, err := e.Execute(context.Background(), newCompiled("asset_type", "owner"))
	if err != nil {
		t.Fatalf("execute must not fail on per-field error, got %v", err)
	}
	if _, ok := out["owner"]; ok {
		t.Fatal("errored field owner must not appear in output")
	}
	if len(warn) != 1 {
		t.Fatalf("expected 1 warning, got %v", warn)
	}
}

func TestFacetExecutor_NoFacetsRequestedIsNoop(t *testing.T) {
	e := NewFacetExecutor(&fakeSource{})
	compiled := &queryir.CompiledQuery{
		NormalizedQuery: queryir.QueryRequest{
			SchemaVersion: queryir.SchemaVersionV1,
			Scope:         queryir.QueryScope{Resource: queryir.ResourceAssets},
		},
	}
	out, warn, err := e.Execute(context.Background(), compiled)
	if err != nil || out != nil || warn != nil {
		t.Fatalf("no facets → no work; got %+v %+v %v", out, warn, err)
	}
}
