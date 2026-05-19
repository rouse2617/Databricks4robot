package admin

import (
	"context"
	"errors"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// fakeBatchOps captures calls and returns scripted responses.
type fakeBatchOps struct {
	idsByBatch      map[string][]string
	mcapsByBatch    map[string][]string
	dryRunAssets    repository.PurgeCounts
	dryRunMcap      repository.PurgeCounts
	purgeAssets     repository.PurgeCounts
	purgeMcap       repository.PurgeCounts
	purgeAssetsCall struct {
		ids       []string
		chunkSize int
		called    bool
	}
	purgeMcapCall struct {
		ids       []string
		chunkSize int
		called    bool
	}
	dryRunAssetsCall struct {
		ids    []string
		called bool
	}
	dryRunMcapCall struct {
		ids    []string
		called bool
	}
	listMcapErr error
}

func (f *fakeBatchOps) ListAssetIDsByImportBatch(_ context.Context, b string) ([]string, error) {
	return f.idsByBatch[b], nil
}
func (f *fakeBatchOps) ListMcapFileIDsByImportBatch(_ context.Context, b string) ([]string, error) {
	if f.listMcapErr != nil {
		return nil, f.listMcapErr
	}
	return f.mcapsByBatch[b], nil
}
func (f *fakeBatchOps) CountAssetReferencesToMcapFiles(_ context.Context, ids []string) (map[string]int64, error) {
	out := map[string]int64{}
	for _, id := range ids {
		out[id] = 0
	}
	return out, nil
}
func (f *fakeBatchOps) PurgeAssetsDryRun(_ context.Context, ids []string) (repository.PurgeCounts, error) {
	f.dryRunAssetsCall.ids = ids
	f.dryRunAssetsCall.called = true
	return f.dryRunAssets, nil
}
func (f *fakeBatchOps) PurgeAssets(_ context.Context, ids []string, chunkSize int) (repository.PurgeCounts, error) {
	f.purgeAssetsCall.ids = ids
	f.purgeAssetsCall.chunkSize = chunkSize
	f.purgeAssetsCall.called = true
	return f.purgeAssets, nil
}
func (f *fakeBatchOps) PurgeMcapFilesDryRun(_ context.Context, ids []string) (repository.PurgeCounts, error) {
	f.dryRunMcapCall.ids = ids
	f.dryRunMcapCall.called = true
	return f.dryRunMcap, nil
}
func (f *fakeBatchOps) PurgeMcapFiles(_ context.Context, ids []string, chunkSize int) (repository.PurgeCounts, error) {
	f.purgeMcapCall.ids = ids
	f.purgeMcapCall.chunkSize = chunkSize
	f.purgeMcapCall.called = true
	return f.purgeMcap, nil
}

func TestPurgeOne_NotFoundWhenAssetsCountZero(t *testing.T) {
	f := &fakeBatchOps{dryRunAssets: repository.PurgeCounts{Assets: 0}}
	u := New(f)

	_, err := u.PurgeOne(context.Background(), "AbCd1234")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if f.purgeAssetsCall.called {
		t.Fatalf("PurgeAssets should not run when dry-run says assets=0")
	}
}

func TestPurgeOne_HappyPathDelegates(t *testing.T) {
	f := &fakeBatchOps{
		dryRunAssets: repository.PurgeCounts{Assets: 1},
		purgeAssets:  repository.PurgeCounts{Assets: 1, AssetTags: 2},
	}
	u := New(f)

	got, err := u.PurgeOne(context.Background(), "AbCd1234")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Assets != 1 || got.AssetTags != 2 {
		t.Fatalf("unexpected counts: %+v", got)
	}
	if !f.purgeAssetsCall.called || len(f.purgeAssetsCall.ids) != 1 || f.purgeAssetsCall.ids[0] != "AbCd1234" {
		t.Fatalf("PurgeAssets not called as expected: %+v", f.purgeAssetsCall)
	}
	if f.purgeAssetsCall.chunkSize != DefaultChunkSize {
		t.Fatalf("expected default chunk size %d, got %d", DefaultChunkSize, f.purgeAssetsCall.chunkSize)
	}
}

func TestPurgeOne_EmptyAssetIDInvalid(t *testing.T) {
	u := New(&fakeBatchOps{})
	if _, err := u.PurgeOne(context.Background(), "   "); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for blank id, got %v", err)
	}
}

func TestPurgeBatch_RequiresExactlyOneSelector(t *testing.T) {
	u := New(&fakeBatchOps{})
	cases := []BatchInput{
		{},
		{AssetIDs: []string{"a"}, ImportBatch: "x"},
	}
	for i, in := range cases {
		if _, err := u.PurgeBatch(context.Background(), in); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("case %d: expected ErrInvalidInput, got %v", i, err)
		}
	}
}

func TestPurgeBatch_DryRunByAssetIDsDoesNotPurge(t *testing.T) {
	f := &fakeBatchOps{
		dryRunAssets: repository.PurgeCounts{Assets: 2, AssetTags: 5},
	}
	u := New(f)

	out, err := u.PurgeBatch(context.Background(), BatchInput{
		AssetIDs: []string{"a1", "a2"},
		DryRun:   true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.DryRun || out.Resolved.AssetIDs != 2 || out.Resolved.McapFileIDs != 0 {
		t.Fatalf("unexpected output: %+v", out)
	}
	if out.Deleted.Assets != 2 || out.Deleted.AssetTags != 5 {
		t.Fatalf("expected counts to come from dry-run: %+v", out.Deleted)
	}
	if !f.dryRunAssetsCall.called {
		t.Fatal("expected PurgeAssetsDryRun to be called")
	}
	if f.purgeAssetsCall.called || f.purgeMcapCall.called {
		t.Fatal("dry-run should not invoke physical purge")
	}
}

func TestPurgeBatch_ImportBatchExpandsAndCascadesToMcap(t *testing.T) {
	f := &fakeBatchOps{
		idsByBatch:   map[string][]string{"b1": {"a1", "a2", "a3"}},
		mcapsByBatch: map[string][]string{"b1": {"m1", "m2"}},
		purgeAssets:  repository.PurgeCounts{Assets: 3, AssetTags: 7, AssetEventsByAsset: 9},
		purgeMcap:    repository.PurgeCounts{McapFiles: 2, AssetEventsByMcap: 4, EvalResultsByMcap: 1},
	}
	u := New(f)

	out, err := u.PurgeBatch(context.Background(), BatchInput{
		ImportBatch:      " b1 ",
		IncludeMcapFiles: true,
		ChunkSize:        250,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.Resolved.ImportBatch != "b1" || out.Resolved.AssetIDs != 3 || out.Resolved.McapFileIDs != 2 {
		t.Fatalf("unexpected resolved: %+v", out.Resolved)
	}
	if !f.purgeAssetsCall.called || f.purgeAssetsCall.chunkSize != 250 {
		t.Fatalf("PurgeAssets should run with caller-supplied chunk size, got %+v", f.purgeAssetsCall)
	}
	if !f.purgeMcapCall.called || f.purgeMcapCall.chunkSize != 250 {
		t.Fatalf("PurgeMcapFiles should run with caller-supplied chunk size, got %+v", f.purgeMcapCall)
	}

	// Merged counts include both asset and mcap-side deletions.
	if out.Deleted.Assets != 3 || out.Deleted.AssetTags != 7 {
		t.Fatalf("asset counts missing in merged output: %+v", out.Deleted)
	}
	if out.Deleted.McapFiles != 2 || out.Deleted.AssetEventsByMcap != 4 || out.Deleted.EvalResultsByMcap != 1 {
		t.Fatalf("mcap counts missing in merged output: %+v", out.Deleted)
	}
}

// fakeESDeleter records the asset_ids that hard-delete tries to remove from
// the search index, and lets a test inject a per-call error to validate that
// ES failures don't fail the caller (best-effort contract).
type fakeESDeleter struct {
	calls   []string
	errOnce error // returned exactly once on first call, then cleared
}

func (f *fakeESDeleter) DeleteDocument(_ context.Context, id string) error {
	f.calls = append(f.calls, id)
	if f.errOnce != nil {
		err := f.errOnce
		f.errOnce = nil
		return err
	}
	return nil
}

func TestPurgeOne_CleansESAfterSuccess(t *testing.T) {
	f := &fakeBatchOps{
		dryRunAssets: repository.PurgeCounts{Assets: 1},
		purgeAssets:  repository.PurgeCounts{Assets: 1},
	}
	es := &fakeESDeleter{}
	u := New(f).WithES(es)

	if _, err := u.PurgeOne(context.Background(), "AbCd1234"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(es.calls) != 1 || es.calls[0] != "AbCd1234" {
		t.Fatalf("expected ES.DeleteDocument called once with the purged id, got %v", es.calls)
	}
}

func TestPurgeOne_SkipsESCleanupWhenNotFound(t *testing.T) {
	f := &fakeBatchOps{dryRunAssets: repository.PurgeCounts{Assets: 0}}
	es := &fakeESDeleter{}
	u := New(f).WithES(es)

	if _, err := u.PurgeOne(context.Background(), "AbCd1234"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if len(es.calls) != 0 {
		t.Fatalf("ES.DeleteDocument must not run on ErrNotFound, got %v", es.calls)
	}
}

func TestPurgeOne_ESFailureDoesNotFailCaller(t *testing.T) {
	f := &fakeBatchOps{
		dryRunAssets: repository.PurgeCounts{Assets: 1},
		purgeAssets:  repository.PurgeCounts{Assets: 1},
	}
	es := &fakeESDeleter{errOnce: errors.New("es unavailable")}
	u := New(f).WithES(es)

	if _, err := u.PurgeOne(context.Background(), "AbCd1234"); err != nil {
		t.Fatalf("ES failure must not propagate; got %v", err)
	}
	if len(es.calls) != 1 {
		t.Fatalf("expected ES delete attempted exactly once, got %v", es.calls)
	}
}

func TestPurgeBatch_CleansESForResolvedAssetIDs(t *testing.T) {
	f := &fakeBatchOps{
		purgeAssets: repository.PurgeCounts{Assets: 2},
	}
	es := &fakeESDeleter{}
	u := New(f).WithES(es)

	if _, err := u.PurgeBatch(context.Background(), BatchInput{
		AssetIDs: []string{"a1", "a2"},
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(es.calls) != 2 || es.calls[0] != "a1" || es.calls[1] != "a2" {
		t.Fatalf("expected ES.DeleteDocument called for each purged id in order, got %v", es.calls)
	}
}

func TestPurgeBatch_DryRunDoesNotTouchES(t *testing.T) {
	f := &fakeBatchOps{
		dryRunAssets: repository.PurgeCounts{Assets: 2},
	}
	es := &fakeESDeleter{}
	u := New(f).WithES(es)

	if _, err := u.PurgeBatch(context.Background(), BatchInput{
		AssetIDs: []string{"a1", "a2"},
		DryRun:   true,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(es.calls) != 0 {
		t.Fatalf("dry-run must not touch ES, got %v", es.calls)
	}
}

func TestPurgeBatch_NoESConfiguredIsNoop(t *testing.T) {
	f := &fakeBatchOps{
		purgeAssets: repository.PurgeCounts{Assets: 1},
	}
	u := New(f) // no WithES

	if _, err := u.PurgeBatch(context.Background(), BatchInput{
		AssetIDs: []string{"a1"},
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Nothing to assert beyond "did not panic"; the cleanES nil-guard is what
	// keeps this safe and we want a regression test for it.
}

func TestPurgeBatch_AssetIDsModeIgnoresMcapFlag(t *testing.T) {
	f := &fakeBatchOps{
		purgeAssets: repository.PurgeCounts{Assets: 2},
	}
	u := New(f)

	out, err := u.PurgeBatch(context.Background(), BatchInput{
		AssetIDs:         []string{"a1", "a2"},
		IncludeMcapFiles: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Resolved.McapFileIDs != 0 || f.purgeMcapCall.called {
		t.Fatalf("asset-id mode must not resolve / purge mcap_files; resolved=%+v call=%+v", out.Resolved, f.purgeMcapCall)
	}
	if out.Deleted.Assets != 2 {
		t.Fatalf("unexpected counts: %+v", out.Deleted)
	}
}
