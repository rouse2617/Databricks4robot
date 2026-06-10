package action

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// stubAssetRepo only implements Get + the no-op repo surface usecase.New touches.
type stubAssetRepo struct {
	parent *models.Asset
	err    error
}

func (r *stubAssetRepo) Get(ctx context.Context, id string) (*models.Asset, error) {
	return r.GetAll(ctx, id)
}
func (r *stubAssetRepo) GetAll(_ context.Context, _ string) (*models.Asset, error) {
	return r.parent, r.err
}
func (r *stubAssetRepo) InsertNew(context.Context, *models.Asset) error { return nil }
func (r *stubAssetRepo) Set(context.Context, *models.Asset) error       { return nil }
func (r *stubAssetRepo) SoftDelete(context.Context, string) error       { return nil }
func (r *stubAssetRepo) ListByMcapFile(context.Context, string) ([]*models.Asset, error) {
	return nil, nil
}
func (r *stubAssetRepo) ListByLogicalAssetID(context.Context, string) ([]*models.Asset, error) {
	return nil, nil
}
func (r *stubAssetRepo) WriteSegmentIndex(context.Context, *models.Asset) error { return nil }
func (r *stubAssetRepo) ListDescendants(_ context.Context, _ string) ([]*models.Asset, error) {
	return nil, nil
}
func (r *stubAssetRepo) ListWithFilters(context.Context, string, []interface{}, int, int, filter.OrderByClause) ([]*models.Asset, int64, error) {
	return nil, 0, nil
}

// fakeActionRepo records inserts and replays a single canned list result.
type fakeActionRepo struct {
	inserts   []*models.Action
	insertErr error

	listResult []*models.Action
	listOpts   repository.ActionListOptions

	getResult    *models.Action
	updateRow    *models.Action
	updateErr    error
	updatePatch  repository.ActionPatch
	deleteRow    *models.Action
	deleteErr    error
	lastDeleteID string
}

func (r *fakeActionRepo) Insert(_ context.Context, a *models.Action) error {
	if r.insertErr != nil {
		return r.insertErr
	}
	if a.ActionID == "" {
		a.ActionID = "action-1"
	}
	a.Version = 1
	r.inserts = append(r.inserts, a)
	return nil
}
func (r *fakeActionRepo) Get(context.Context, string) (*models.Action, error) {
	return r.getResult, nil
}
func (r *fakeActionRepo) ListByAsset(_ context.Context, _ string, opts repository.ActionListOptions) ([]*models.Action, error) {
	r.listOpts = opts
	return r.listResult, nil
}
func (r *fakeActionRepo) Update(_ context.Context, _ string, _ int64, patch repository.ActionPatch) (*models.Action, error) {
	r.updatePatch = patch
	if r.updateErr != nil {
		return nil, r.updateErr
	}
	return r.updateRow, nil
}
func (r *fakeActionRepo) SoftDelete(_ context.Context, actionID string, _ int64) (*models.Action, error) {
	r.lastDeleteID = actionID
	if r.deleteErr != nil {
		return nil, r.deleteErr
	}
	return r.deleteRow, nil
}

// fakeEventRepo records every Append call so we can assert same-tx event emission.
type fakeEventRepo struct {
	appended []repository.AssetEventAppendInput
}

func (r *fakeEventRepo) Append(_ context.Context, in repository.AssetEventAppendInput) error {
	r.appended = append(r.appended, in)
	return nil
}
func (r *fakeEventRepo) ListPending(context.Context, int) ([]*models.AssetEvent, error) {
	return nil, nil
}
func (r *fakeEventRepo) ListPendingSafe(context.Context, time.Duration, int) ([]*models.AssetEvent, error) {
	return nil, nil
}
func (r *fakeEventRepo) ListByAsset(context.Context, string, repository.AssetEventListOptions) ([]*models.AssetEvent, error) {
	return nil, nil
}
func (r *fakeEventRepo) ListVersionPromotedByLogical(context.Context, string) ([]*models.AssetEvent, error) {
	return nil, nil
}

func (m *fakeEventRepo) ListGlobal(_ context.Context, opts repository.AssetEventListOptions) ([]*models.AssetEvent, error) {
	return nil, nil
}

func (r *fakeEventRepo) MarkPublished(context.Context, []int64) error { return nil }
func (r *fakeEventRepo) MarkFailed(context.Context, int64, string) error {
	return nil
}
func (r *fakeEventRepo) CountPending(context.Context) (int64, error) { return 0, nil }
func (r *fakeEventRepo) CountPendingClaimable(context.Context, time.Duration) (int64, error) {
	return 0, nil
}
func (r *fakeEventRepo) CountProcessing(context.Context) (int64, error)    { return 0, nil }
func (r *fakeEventRepo) OldestPendingAge(context.Context) (float64, error) { return 0, nil }
func (r *fakeEventRepo) ListBetweenSeq(context.Context, int64, int64, int) ([]*models.AssetEvent, error) {
	return nil, nil
}

func (r *fakeEventRepo) PublishStateCounts(context.Context) (map[string]int64, error) {
	return map[string]int64{}, nil
}

func newSegParent() *models.Asset {
	return &models.Asset{
		AssetID:          "asset-1",
		AssetType:        "segment",
		StartTimestampNs: 1,
		EndTimestampNs:   10_000_000_000,
		TenantID:         "t1",
		ProjectID:        "p1",
	}
}

func TestCreate_PersistsAndAppendsEvent(t *testing.T) {
	parent := newSegParent()
	actions := &fakeActionRepo{}
	events := &fakeEventRepo{}
	uc := New(nil, actions, &stubAssetRepo{parent: parent}, events)

	row, err := uc.Create(context.Background(), CreateInput{
		AssetID:      parent.AssetID,
		StartNs:      1_000_000,
		EndNs:        2_000_000,
		PrimaryLabel: "pickup",
		Labels:       []string{"pickup", "left_hand"},
		SourceType:   models.ActionSourceHuman,
		SourceName:   "annotator-001",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if row.ActionID == "" {
		t.Fatalf("expected ActionID assigned by repo")
	}
	if got, want := row.TenantID, parent.TenantID; got != want {
		t.Fatalf("tenant_id should be inherited from parent: got %q want %q", got, want)
	}
	if len(actions.inserts) != 1 {
		t.Fatalf("expected 1 insert, got %d", len(actions.inserts))
	}
	if len(events.appended) != 1 {
		t.Fatalf("expected 1 event append, got %d", len(events.appended))
	}
	got := events.appended[0]
	if got.EventType != "action_upserted" {
		t.Fatalf("event_type = %q want action_upserted", got.EventType)
	}
	if got.AssetID != parent.AssetID {
		t.Fatalf("event asset_id = %q want %q", got.AssetID, parent.AssetID)
	}
	if got.PayloadSchemaVersion != "v1" {
		t.Fatalf("payload schema version = %q want v1", got.PayloadSchemaVersion)
	}
	if len(got.EventPayload) == 0 {
		t.Fatalf("event payload should be populated")
	}
}

func TestCreate_RejectsNonSegParent(t *testing.T) {
	parent := newSegParent()
	parent.AssetType = "frame_set"
	uc := New(nil, &fakeActionRepo{}, &stubAssetRepo{parent: parent}, &fakeEventRepo{})

	_, err := uc.Create(context.Background(), CreateInput{
		AssetID: parent.AssetID,
		StartNs: 0, EndNs: 1,
		SourceType: models.ActionSourceHuman,
	})
	if !errors.Is(err, ErrParentNotSeg) {
		t.Fatalf("expected ErrParentNotSeg, got %v", err)
	}
}

func TestCreate_RejectsRangeOutsideSeg(t *testing.T) {
	parent := newSegParent()
	uc := New(nil, &fakeActionRepo{}, &stubAssetRepo{parent: parent}, &fakeEventRepo{})

	_, err := uc.Create(context.Background(), CreateInput{
		AssetID: parent.AssetID,
		StartNs: parent.EndTimestampNs - 1,
		EndNs:   parent.EndTimestampNs + 1,
	})
	if !errors.Is(err, ErrRangeOutsideSeg) {
		t.Fatalf("expected ErrRangeOutsideSeg, got %v", err)
	}
}

func TestCreate_RejectsInvalidRange(t *testing.T) {
	uc := New(nil, &fakeActionRepo{}, &stubAssetRepo{parent: newSegParent()}, &fakeEventRepo{})
	_, err := uc.Create(context.Background(), CreateInput{
		AssetID: "asset-1",
		StartNs: 100,
		EndNs:   50,
	})
	if !errors.Is(err, ErrInvalidRange) {
		t.Fatalf("expected ErrInvalidRange, got %v", err)
	}
}

func TestCreate_MissingSeg(t *testing.T) {
	uc := New(nil, &fakeActionRepo{}, &stubAssetRepo{parent: nil}, &fakeEventRepo{})
	_, err := uc.Create(context.Background(), CreateInput{
		AssetID: "missing", StartNs: 0, EndNs: 1,
	})
	if !errors.Is(err, ErrSegNotFound) {
		t.Fatalf("expected ErrSegNotFound, got %v", err)
	}
}

func TestList_NonSegmentParent_ReturnsEmpty(t *testing.T) {
	parent := newSegParent()
	parent.AssetType = "task_demo"
	repo := &fakeActionRepo{listResult: []*models.Action{{ActionID: "should-not-return"}}}
	uc := New(nil, repo, &stubAssetRepo{parent: parent}, &fakeEventRepo{})

	items, err := uc.List(context.Background(), ListInput{AssetID: parent.AssetID, Limit: 50})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty list for non-segment asset, got %d items", len(items))
	}
}

func TestList_ForwardsOptions(t *testing.T) {
	at := int64(500)
	from := int64(100)
	to := int64(900)
	repo := &fakeActionRepo{listResult: []*models.Action{{ActionID: "a1"}}}
	uc := New(nil, repo, &stubAssetRepo{parent: newSegParent()}, &fakeEventRepo{})

	items, err := uc.List(context.Background(), ListInput{
		AssetID:   "asset-1",
		PointAtNs: &at, FromNs: &from, ToNs: &to,
		Label: "pickup", Limit: 50,
	})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if repo.listOpts.PointAtNs == nil || *repo.listOpts.PointAtNs != at {
		t.Fatalf("PointAtNs not forwarded: %+v", repo.listOpts.PointAtNs)
	}
	if repo.listOpts.Label != "pickup" || repo.listOpts.Limit != 50 {
		t.Fatalf("opts not forwarded: %+v", repo.listOpts)
	}
}

func TestUpdate_AppliesPatchAndEmitsUpsertEvent(t *testing.T) {
	parent := newSegParent()
	existing := &models.Action{
		ActionID: "act00001", AssetID: parent.AssetID,
		StartNs: 100, EndNs: 200, Version: 3,
		TenantID: parent.TenantID, ProjectID: parent.ProjectID,
	}
	updated := &models.Action{
		ActionID: existing.ActionID, AssetID: parent.AssetID,
		StartNs: 150, EndNs: 250, Version: 4,
		TenantID: parent.TenantID, ProjectID: parent.ProjectID,
	}
	repo := &fakeActionRepo{getResult: existing, updateRow: updated}
	events := &fakeEventRepo{}
	uc := New(nil, repo, &stubAssetRepo{parent: parent}, events)

	newStart := int64(150)
	newEnd := int64(250)
	row, err := uc.Update(context.Background(), UpdateInput{
		AssetID:         parent.AssetID,
		ActionID:        existing.ActionID,
		ExpectedVersion: 3,
		Patch:           repository.ActionPatch{StartNs: &newStart, EndNs: &newEnd},
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if row.Version != 4 {
		t.Fatalf("expected version 4, got %d", row.Version)
	}
	if len(events.appended) != 1 || events.appended[0].EventType != "action_upserted" {
		t.Fatalf("expected 1 action_upserted event, got %+v", events.appended)
	}
}

func TestUpdate_RangeOutsideSegRejected(t *testing.T) {
	parent := newSegParent()
	existing := &models.Action{ActionID: "act00001", AssetID: parent.AssetID, StartNs: 100, EndNs: 200, Version: 1}
	repo := &fakeActionRepo{getResult: existing}
	uc := New(nil, repo, &stubAssetRepo{parent: parent}, &fakeEventRepo{})
	bad := parent.EndTimestampNs + 1
	_, err := uc.Update(context.Background(), UpdateInput{
		AssetID: parent.AssetID, ActionID: existing.ActionID,
		Patch: repository.ActionPatch{EndNs: &bad},
	})
	if !errors.Is(err, ErrRangeOutsideSeg) {
		t.Fatalf("expected ErrRangeOutsideSeg, got %v", err)
	}
}

func TestUpdate_OptimisticLockSurfaced(t *testing.T) {
	parent := newSegParent()
	existing := &models.Action{ActionID: "act00001", AssetID: parent.AssetID, StartNs: 100, EndNs: 200, Version: 5}
	repo := &fakeActionRepo{getResult: existing, updateErr: repository.ErrOptimisticLock}
	uc := New(nil, repo, &stubAssetRepo{parent: parent}, &fakeEventRepo{})
	desc := "hi"
	_, err := uc.Update(context.Background(), UpdateInput{
		AssetID: parent.AssetID, ActionID: existing.ActionID, ExpectedVersion: 1,
		Patch: repository.ActionPatch{Description: &desc},
	})
	if !errors.Is(err, repository.ErrOptimisticLock) {
		t.Fatalf("expected ErrOptimisticLock, got %v", err)
	}
}

func TestUpdate_ActionMissingReturnsNotFound(t *testing.T) {
	parent := newSegParent()
	repo := &fakeActionRepo{getResult: nil}
	uc := New(nil, repo, &stubAssetRepo{parent: parent}, &fakeEventRepo{})
	desc := "hi"
	_, err := uc.Update(context.Background(), UpdateInput{
		AssetID: parent.AssetID, ActionID: "missing01",
		Patch: repository.ActionPatch{Description: &desc},
	})
	if !errors.Is(err, ErrActionNotFound) {
		t.Fatalf("expected ErrActionNotFound, got %v", err)
	}
}

func TestDelete_SoftDeletesAndEmitsEvent(t *testing.T) {
	parent := newSegParent()
	existing := &models.Action{ActionID: "act00001", AssetID: parent.AssetID, Version: 2,
		TenantID: parent.TenantID, ProjectID: parent.ProjectID}
	deleted := &models.Action{ActionID: existing.ActionID, AssetID: parent.AssetID, Version: 3,
		TenantID: parent.TenantID, ProjectID: parent.ProjectID, IsDeleted: true}
	repo := &fakeActionRepo{getResult: existing, deleteRow: deleted}
	events := &fakeEventRepo{}
	uc := New(nil, repo, &stubAssetRepo{parent: parent}, events)

	if err := uc.Delete(context.Background(), DeleteInput{
		AssetID: parent.AssetID, ActionID: existing.ActionID,
	}); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if repo.lastDeleteID != existing.ActionID {
		t.Fatalf("expected SoftDelete called with %q, got %q", existing.ActionID, repo.lastDeleteID)
	}
	if len(events.appended) != 1 || events.appended[0].EventType != "action_deleted" {
		t.Fatalf("expected 1 action_deleted event, got %+v", events.appended)
	}
}

func TestDelete_MissingActionReturnsNotFound(t *testing.T) {
	parent := newSegParent()
	repo := &fakeActionRepo{getResult: nil}
	uc := New(nil, repo, &stubAssetRepo{parent: parent}, &fakeEventRepo{})
	err := uc.Delete(context.Background(), DeleteInput{AssetID: parent.AssetID, ActionID: "missing01"})
	if !errors.Is(err, ErrActionNotFound) {
		t.Fatalf("expected ErrActionNotFound, got %v", err)
	}
}

func writeActionLabelRegistry(t *testing.T) *config.ActionLabelRegistry {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "action_label_registry.yaml")
	content := `primary_labels:
  - pickup
  - place
labels:
  - pickup
  - place
  - left_hand
  - right_hand
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write registry: %v", err)
	}
	reg, err := config.LoadActionLabelRegistry(path)
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	return reg
}

func TestCreate_RejectsUnregisteredPrimaryLabel(t *testing.T) {
	reg := writeActionLabelRegistry(t)
	uc := NewWithLabelRegistry(nil, &fakeActionRepo{}, &stubAssetRepo{parent: newSegParent()}, &fakeEventRepo{}, reg)
	_, err := uc.Create(context.Background(), CreateInput{
		AssetID:      "asset-1",
		StartNs:      100,
		EndNs:        200,
		PrimaryLabel: "unknown",
		Labels:       []string{"pickup"},
	})
	if !errors.Is(err, ErrInvalidLabel) {
		t.Fatalf("expected ErrInvalidLabel, got %v", err)
	}
}

func TestCreate_RejectsUnregisteredLabel(t *testing.T) {
	reg := writeActionLabelRegistry(t)
	uc := NewWithLabelRegistry(nil, &fakeActionRepo{}, &stubAssetRepo{parent: newSegParent()}, &fakeEventRepo{}, reg)
	_, err := uc.Create(context.Background(), CreateInput{
		AssetID:      "asset-1",
		StartNs:      100,
		EndNs:        200,
		PrimaryLabel: "pickup",
		Labels:       []string{"pickup", "not_allowed"},
	})
	if !errors.Is(err, ErrInvalidLabel) {
		t.Fatalf("expected ErrInvalidLabel, got %v", err)
	}
}
