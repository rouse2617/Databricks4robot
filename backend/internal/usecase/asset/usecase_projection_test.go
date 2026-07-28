package asset

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

type noopTxRunner struct{}

func (noopTxRunner) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type mockAssetTagRepo struct {
	rows map[string]*models.AssetTag
}

func newMockAssetTagRepo() *mockAssetTagRepo {
	return &mockAssetTagRepo{rows: map[string]*models.AssetTag{}}
}

// tagRowKey for the mock uses the legacy (asset_id, tag_key) shape so older
// assertions still work. Multi-source assertions use tagRowKeyFull below.
func tagRowKey(assetID, tagKey string) string { return assetID + "|" + tagKey }

func tagRowKeyFull(assetID, tagKey, sourceType, sourceVersion string) string {
	if sourceType == "" {
		return tagRowKey(assetID, tagKey)
	}
	return assetID + "|" + tagKey + "|" + sourceType + "|" + sourceVersion
}

func (m *mockAssetTagRepo) Upsert(_ context.Context, in repository.AssetTagUpsertInput) error {
	row := &models.AssetTag{
		AssetID:       in.AssetID,
		TagKey:        in.TagKey,
		TagValue:      in.TagValue,
		TagType:       in.TagType,
		SourceType:    in.SourceType,
		SourceName:    in.SourceName,
		SourceVersion: in.SourceVersion,
		RunID:         in.RunID,
		AppliedAt:     time.Now().UTC(),
	}
	m.rows[tagRowKey(in.AssetID, in.TagKey)] = row
	m.rows[tagRowKeyFull(in.AssetID, in.TagKey, in.SourceType, in.SourceVersion)] = row
	return nil
}

func (m *mockAssetTagRepo) ListByAsset(_ context.Context, assetID string) ([]*models.AssetTag, error) {
	out := []*models.AssetTag{}
	seen := map[*models.AssetTag]bool{}
	for _, row := range m.rows {
		if row.AssetID != assetID || seen[row] {
			continue
		}
		seen[row] = true
		out = append(out, row)
	}
	return out, nil
}

func (m *mockAssetTagRepo) Delete(_ context.Context, assetID, tagKey, sourceType string) error {
	for k, row := range m.rows {
		if row.AssetID != assetID || row.TagKey != tagKey {
			continue
		}
		if sourceType != "" && row.SourceType != sourceType {
			continue
		}
		delete(m.rows, k)
	}
	return nil
}

func ptrString(s string) *string { return &s }

func buildTestTagRegistry(t *testing.T) *config.TagRegistry {
	t.Helper()
	reg, err := config.LoadTagRegistry("../../../config/tag_registry.yaml")
	if err != nil {
		t.Fatalf("failed to load tag registry: %v", err)
	}
	return reg
}

func hasEventType(events []*models.AssetEvent, want string) bool {
	for _, e := range events {
		if e.EventType == want {
			return true
		}
	}
	return false
}

type readModelAssetRepo struct {
	getFn             func(context.Context, string) (*models.Asset, error)
	listByMcapFileFn  func(context.Context, string) ([]*models.Asset, error)
	listWithFiltersFn func(context.Context, string, []interface{}, int, int, filter.OrderByClause) ([]*models.Asset, int64, error)
}

func (r *readModelAssetRepo) InsertNew(context.Context, *models.Asset) error { return nil }
func (r *readModelAssetRepo) Get(ctx context.Context, assetID string) (*models.Asset, error) {
	return r.GetAll(ctx, assetID)
}
func (r *readModelAssetRepo) GetAll(ctx context.Context, assetID string) (*models.Asset, error) {
	return r.getFn(ctx, assetID)
}
func (r *readModelAssetRepo) FindExistingIDs(ctx context.Context, assetIDs []string) (map[string]struct{}, error) {
	out := make(map[string]struct{})
	for _, assetID := range assetIDs {
		a, err := r.Get(ctx, assetID)
		if err != nil {
			return nil, err
		}
		if a != nil {
			out[assetID] = struct{}{}
		}
	}
	return out, nil
}
func (r *readModelAssetRepo) Set(context.Context, *models.Asset) error { return nil }
func (r *readModelAssetRepo) SoftDelete(context.Context, string) error { return nil }
func (r *readModelAssetRepo) ListByMcapFile(ctx context.Context, mcapFileID string) ([]*models.Asset, error) {
	return r.listByMcapFileFn(ctx, mcapFileID)
}
func (r *readModelAssetRepo) ListByLogicalAssetID(context.Context, string) ([]*models.Asset, error) {
	return nil, nil
}
func (r *readModelAssetRepo) WriteSegmentIndex(context.Context, *models.Asset) error { return nil }
func (r *readModelAssetRepo) ListDescendants(_ context.Context, _ string) ([]*models.Asset, error) {
	return nil, nil
}
func (r *readModelAssetRepo) ListWithFilters(ctx context.Context, whereSQL string, args []interface{}, page, pageSize int, orderBy filter.OrderByClause) ([]*models.Asset, int64, error) {
	return r.listWithFiltersFn(ctx, whereSQL, args, page, pageSize, orderBy)
}
func (r *readModelAssetRepo) LookupDurations(context.Context, []string, int64, int64) ([]repository.DurationRow, error) {
	return nil, nil
}

func TestCreate_WritesTagProjectionAndOutbox(t *testing.T) {
	repo := newMockAssetRepo()
	tagRepo := newMockAssetTagRepo()
	eventRepo := newMockAssetEventRepo()
	uc := NewWithProjections(noopTxRunner{}, repo, tagRepo, nil, eventRepo, nil, nil)

	a, err := uc.Create(context.Background(), CreateInput{
		McapFileID:       "mcap-create-001",
		StartTimestampNs: 100,
		EndTimestampNs:   1000100,
		Reviewer:         "alice",
		Owner:            "team-a",
		Tags:             map[string]string{"quality": "good"},
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if _, ok := tagRepo.rows[tagRowKey(a.AssetID, "quality")]; !ok {
		t.Fatalf("expected asset_tags upsert for quality")
	}

	events := eventRepo.all()
	if !hasEventType(events, "asset_created") {
		t.Fatalf("expected asset_created event, got %#v", events)
	}
	if !hasEventType(events, "tag_upserted") {
		t.Fatalf("expected tag_upserted event, got %#v", events)
	}
}

func TestUpsertTag_WritesProjectionAndEvent(t *testing.T) {
	repo := newMockAssetRepo()
	repo.assets["a1"] = &models.Asset{AssetID: "a1", McapFileID: "m1"}
	tagRepo := newMockAssetTagRepo()
	eventRepo := newMockAssetEventRepo()
	uc := NewWithProjections(noopTxRunner{}, repo, tagRepo, nil, eventRepo, buildTestTagRegistry(t), nil)

	a, err := uc.UpsertTag(context.Background(), "a1", UpsertTagInput{
		Key:        "quality",
		Value:      "good",
		SourceType: "human",
		SourceName: "tester",
	})
	if err != nil {
		t.Fatalf("UpsertTag failed: %v", err)
	}
	if got := a.Tags["quality"]; got != "good" {
		t.Fatalf("expected hydrated tag quality=good, got %q", got)
	}
	if _, ok := tagRepo.rows[tagRowKey("a1", "quality")]; !ok {
		t.Fatalf("expected asset_tags upsert for quality")
	}
	if !hasEventType(eventRepo.all(), "tag_upserted") {
		t.Fatalf("expected tag_upserted event")
	}
}

func TestUpdate_WritesTagProjectionAndLifecycleEvents(t *testing.T) {
	repo := newMockAssetRepo()
	tagRepo := newMockAssetTagRepo()
	eventRepo := newMockAssetEventRepo()
	uc := NewWithProjections(noopTxRunner{}, repo, tagRepo, nil, eventRepo, nil, nil)

	repo.assets["a1"] = &models.Asset{
		AssetID:    "a1",
		McapFileID: "mcap-update-001",
		Status:     models.AssetStatusApproved,
		Tags:       map[string]string{},
	}

	_, err := uc.Update(context.Background(), "a1", UpdateInput{
		Status: ptrString("rejected"),
		Tags:   map[string]string{"quality": "poor"},
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if _, ok := tagRepo.rows[tagRowKey("a1", "quality")]; !ok {
		t.Fatalf("expected asset_tags upsert for updated tag")
	}

	events := eventRepo.all()
	if !hasEventType(events, "asset_updated") {
		t.Fatalf("expected asset_updated event, got %#v", events)
	}
	if !hasEventType(events, "asset_lifecycle_changed") {
		t.Fatalf("expected asset_lifecycle_changed event, got %#v", events)
	}
	if !hasEventType(events, "tag_upserted") {
		t.Fatalf("expected tag_upserted event, got %#v", events)
	}
}

// CYB-3232: PATCH files must merge into FilesJSON (the field the write path
// serializes), preserving existing file keys and setting storage_uri/thumb_uri.
func TestUpdate_MergesFilesAndPointers(t *testing.T) {
	repo := newMockAssetRepo()
	eventRepo := newMockAssetEventRepo()
	uc := NewWithProjections(noopTxRunner{}, repo, newMockAssetTagRepo(), nil, eventRepo, nil, nil)

	repo.assets["a1"] = &models.Asset{
		AssetID:    "a1",
		McapFileID: "mcap-files-001",
		Status:     models.AssetStatusApproved,
		Files:      map[string]string{"raw_mcap": "mcap-files-001"},
		FilesJSON:  map[string]interface{}{"raw_mcap": "mcap-files-001"},
	}

	_, err := uc.Update(context.Background(), "a1", UpdateInput{
		Files:      map[string]string{"algo_input_forward_stereo": "gs://b/x.mp4"},
		StorageURI: ptrString("gs://b/seg.mcap"),
		ThumbURI:   ptrString("gs://b/t.jpg"),
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got := repo.assets["a1"]
	// FilesJSON is what bindAssetJSONAndRefs serializes — the new key must be there,
	// and the pre-existing key must be preserved (merge semantics).
	if got.FilesJSON["algo_input_forward_stereo"] != "gs://b/x.mp4" {
		t.Fatalf("FilesJSON missing merged key: %#v", got.FilesJSON)
	}
	if got.FilesJSON["raw_mcap"] != "mcap-files-001" {
		t.Fatalf("FilesJSON dropped existing key: %#v", got.FilesJSON)
	}
	if got.StorageURI != "gs://b/seg.mcap" {
		t.Fatalf("StorageURI = %q, want gs://b/seg.mcap", got.StorageURI)
	}
	if got.ThumbURI != "gs://b/t.jpg" {
		t.Fatalf("ThumbURI = %q, want gs://b/t.jpg", got.ThumbURI)
	}
}

func TestUpdate_WithLifecycleStateOnly_AppendsLifecycleChangedEvent(t *testing.T) {
	repo := newMockAssetRepo()
	eventRepo := newMockAssetEventRepo()
	uc := NewWithProjections(noopTxRunner{}, repo, nil, nil, eventRepo, nil, nil)

	repo.assets["a1"] = &models.Asset{
		AssetID:        "a1",
		McapFileID:     "mcap-update-lifecycle-only",
		Status:         models.AssetStatusApproved,
		LifecycleState: "ready",
		Tags:           map[string]string{},
	}

	_, err := uc.Update(context.Background(), "a1", UpdateInput{
		LifecycleState: ptrString("rejected"),
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	events := eventRepo.all()
	if !hasEventType(events, "asset_updated") {
		t.Fatalf("expected asset_updated event, got %#v", events)
	}
	if !hasEventType(events, "asset_lifecycle_changed") {
		t.Fatalf("expected asset_lifecycle_changed event, got %#v", events)
	}
}

func TestDelete_AppendsLifecycleEvent(t *testing.T) {
	repo := newMockAssetRepo()
	eventRepo := newMockAssetEventRepo()
	uc := NewWithProjections(noopTxRunner{}, repo, nil, nil, eventRepo, nil, nil)

	repo.assets["a1"] = &models.Asset{
		AssetID:        "a1",
		McapFileID:     "mcap-delete-001",
		Status:         models.AssetStatusApproved,
		LifecycleState: "ready",
	}

	if err := uc.Delete(context.Background(), "a1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if _, ok := repo.assets["a1"]; ok {
		t.Fatalf("expected asset to be soft-deleted in mock repo")
	}

	events := eventRepo.all()
	if !hasEventType(events, "asset_lifecycle_changed") {
		t.Fatalf("expected asset_lifecycle_changed event, got %#v", events)
	}
}

func TestDeleteTag_RemovesProjectionAndAppendsEvent(t *testing.T) {
	repo := newMockAssetRepo()
	repo.assets["a1"] = &models.Asset{AssetID: "a1", McapFileID: "m1"}
	tagRepo := newMockAssetTagRepo()
	_ = tagRepo.Upsert(context.Background(), repository.AssetTagUpsertInput{AssetID: "a1", TagKey: "quality", TagValue: "good", TagType: "enum", SourceType: "human", SourceName: "test"})
	eventRepo := newMockAssetEventRepo()
	uc := NewWithProjections(noopTxRunner{}, repo, tagRepo, nil, eventRepo, nil, nil)

	a, err := uc.DeleteTag(context.Background(), "a1", "quality", "")
	if err != nil {
		t.Fatalf("DeleteTag failed: %v", err)
	}
	if _, ok := tagRepo.rows[tagRowKey("a1", "quality")]; ok {
		t.Fatalf("expected tag to be deleted from projection")
	}
	if _, ok := a.Tags["quality"]; ok {
		t.Fatalf("expected hydrated asset tags to omit deleted key")
	}
	if !hasEventType(eventRepo.all(), "tag_deleted") {
		t.Fatalf("expected tag_deleted event")
	}
}

func TestDeleteTag_MissingIsIdempotent(t *testing.T) {
	repo := newMockAssetRepo()
	repo.assets["a1"] = &models.Asset{AssetID: "a1", McapFileID: "m1"}
	tagRepo := newMockAssetTagRepo()
	eventRepo := newMockAssetEventRepo()
	uc := NewWithProjections(noopTxRunner{}, repo, tagRepo, nil, eventRepo, nil, nil)

	a, err := uc.DeleteTag(context.Background(), "a1", "missing", "")
	if err != nil {
		t.Fatalf("DeleteTag failed: %v", err)
	}
	if len(a.Tags) != 0 {
		t.Fatalf("expected no tags after idempotent delete, got %#v", a.Tags)
	}
	if hasEventType(eventRepo.all(), "tag_deleted") {
		t.Fatalf("did not expect tag_deleted event when tag was absent")
	}
}

// countEventsFor returns the number of recorded events of eventType for the
// given assetID.
func countEventsFor(events []*models.AssetEvent, assetID, eventType string) int {
	n := 0
	for _, e := range events {
		if e.AssetID == assetID && e.EventType == eventType {
			n++
		}
	}
	return n
}

// TestTagPropagation_EmitsEventsAndUnpropagates covers CYB-1068: a propagating
// tag (compliance.status) written to an ancestor must (1) copy onto descendants
// AND emit a tag_upserted event per descendant, and (2) on DeleteTag remove the
// descendant copies AND emit a tag_deleted event per descendant.
func TestTagPropagation_EmitsEventsAndUnpropagates(t *testing.T) {
	repo := newMockAssetRepo()
	repo.assets["parent"] = &models.Asset{AssetID: "parent", McapFileID: "m1"}
	repo.assets["child"] = &models.Asset{AssetID: "child", McapFileID: "m1"}
	repo.descendants = map[string][]*models.Asset{
		"parent": {{AssetID: "child", McapFileID: "m1"}},
	}
	tagRepo := newMockAssetTagRepo()
	eventRepo := newMockAssetEventRepo()
	reg := buildTestTagRegistry(t)
	uc := NewWithProjections(noopTxRunner{}, repo, tagRepo, nil, eventRepo, reg, nil)

	// compliance.status is registered with propagation=descendants; source
	// "compliance" requires a source_name.
	if _, err := uc.UpsertTag(context.Background(), "parent", UpsertTagInput{
		Key: "compliance.status", Value: "approved",
		SourceType: "compliance", SourceName: "reviewer-1",
	}); err != nil {
		t.Fatalf("UpsertTag failed: %v", err)
	}

	if _, ok := tagRepo.rows[tagRowKey("child", "compliance.status")]; !ok {
		t.Fatalf("expected tag to propagate to descendant")
	}
	if got := countEventsFor(eventRepo.all(), "child", "tag_upserted"); got != 1 {
		t.Fatalf("expected 1 tag_upserted event on descendant, got %d", got)
	}

	// DeleteTag must un-propagate: remove the descendant copy + emit tag_deleted.
	if _, err := uc.DeleteTag(context.Background(), "parent", "compliance.status", ""); err != nil {
		t.Fatalf("DeleteTag failed: %v", err)
	}
	if _, ok := tagRepo.rows[tagRowKey("child", "compliance.status")]; ok {
		t.Fatalf("expected descendant tag copy to be un-propagated on delete")
	}
	if got := countEventsFor(eventRepo.all(), "child", "tag_deleted"); got != 1 {
		t.Fatalf("expected 1 tag_deleted event on descendant, got %d", got)
	}
}

func TestGet_HydratesTagsAndAlgoResultsFromProjections(t *testing.T) {
	repo := &readModelAssetRepo{
		getFn: func(context.Context, string) (*models.Asset, error) {
			return &models.Asset{AssetID: "a1", McapFileID: "m1"}, nil
		},
	}
	tagRepo := newMockAssetTagRepo()
	tagRepo.Upsert(context.Background(), repository.AssetTagUpsertInput{AssetID: "a1", TagKey: "quality", TagValue: "good", TagType: "string", SourceType: "system"})
	algoRepo := newMockAlgoLatestRepo()
	now := time.Now().UTC()
	_ = algoRepo.Upsert(context.Background(), &models.AssetAlgoLatest{
		AssetID:     "a1",
		AlgoName:    "hand_tracking",
		AlgoVersion: "1.2.0",
		Status:      "ok",
		RunID:       "run-1",
		StartedAt:   &now,
		OutputURI:   "gs://bucket/out",
	})

	uc := NewWithProjections(noopTxRunner{}, repo, tagRepo, algoRepo, nil, nil, nil)

	a, err := uc.Get(context.Background(), "a1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got := a.Tags["quality"]; got != "good" {
		t.Fatalf("expected hydrated tag quality=good, got %q", got)
	}
	if got := a.AlgoResults["hand_tracking@1.2.0:status"]; got != "ok" {
		t.Fatalf("expected hydrated algo status ok, got %q", got)
	}
	if got := a.AlgoResults["hand_tracking@1.2.0:run_id"]; got != "run-1" {
		t.Fatalf("expected hydrated algo run_id, got %q", got)
	}
}

func TestListWithFilters_HydratesProjectionData(t *testing.T) {
	repo := &readModelAssetRepo{
		listWithFiltersFn: func(context.Context, string, []interface{}, int, int, filter.OrderByClause) ([]*models.Asset, int64, error) {
			return []*models.Asset{
				{AssetID: "a1", McapFileID: "m1"},
				{AssetID: "a2", McapFileID: "m2"},
			}, 2, nil
		},
	}
	tagRepo := newMockAssetTagRepo()
	tagRepo.Upsert(context.Background(), repository.AssetTagUpsertInput{AssetID: "a1", TagKey: "quality", TagValue: "good", TagType: "string", SourceType: "system"})
	tagRepo.Upsert(context.Background(), repository.AssetTagUpsertInput{AssetID: "a2", TagKey: "quality", TagValue: "poor", TagType: "string", SourceType: "system"})

	uc := NewWithProjections(noopTxRunner{}, repo, tagRepo, nil, nil, nil, nil)

	items, total, err := uc.ListWithFilters(context.Background(), "", nil, 1, 20, filter.OrderByClause{SQL: "created_at DESC"})
	if err != nil {
		t.Fatalf("ListWithFilters failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total=2, got %d", total)
	}
	if items[0].Tags["quality"] == "" || items[1].Tags["quality"] == "" {
		t.Fatalf("expected hydrated tags on all list items, got %#v", items)
	}
}

func TestCreate_SeedsInitialAlgoProjectionRows(t *testing.T) {
	repo := newMockAssetRepo()
	algoRepo := newMockAlgoLatestRepo()
	algoReg := buildTestAlgoRegistry(t)
	uc := NewWithProjections(noopTxRunner{}, repo, nil, algoRepo, nil, nil, algoReg)

	a, err := uc.Create(context.Background(), CreateInput{
		McapFileID:       "mcap-seed-001",
		StartTimestampNs: 100,
		EndTimestampNs:   1000100,
		Reviewer:         "alice",
		Owner:            "team-a",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	row, _ := algoRepo.GetByAlgo(context.Background(), a.AssetID, "env_analysis")
	if row == nil || row.Status != string(models.AlgoStatusPending) {
		t.Fatalf("expected env_analysis pending row in asset_algo_latest, got %#v", row)
	}
	row, _ = algoRepo.GetByAlgo(context.Background(), a.AssetID, "action_annotation")
	if row == nil || row.Status != string(models.AlgoStatusBlocked) {
		t.Fatalf("expected action_annotation blocked row in asset_algo_latest, got %#v", row)
	}
}

func TestListEvents_GenericTimelineWithCursor(t *testing.T) {
	repo := &readModelAssetRepo{
		getFn: func(context.Context, string) (*models.Asset, error) {
			return &models.Asset{AssetID: "a1", McapFileID: "m1"}, nil
		},
	}
	eventRepo := newMockAssetEventRepo()

	appendEvent := func(eventType string, payload map[string]any) {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		if err := eventRepo.Append(context.Background(), repository.AssetEventAppendInput{
			EventType:    eventType,
			AssetID:      "a1",
			EventPayload: raw,
		}); err != nil {
			t.Fatalf("append event: %v", err)
		}
	}

	appendEvent("asset_created", map[string]any{"asset_id": "a1"})
	appendEvent("algo_started", map[string]any{"algo_key": "env_analysis@1.0.0"})
	appendEvent("algo_finished", map[string]any{"algo_key": "env_analysis@1.0.0"})

	uc := NewWithProjections(noopTxRunner{}, repo, nil, nil, eventRepo, nil, nil)
	res, err := uc.ListEvents(context.Background(), "a1", ListEventsInput{
		EventTypePatterns: []string{"algo_%"},
		Limit:             1,
	})
	if err != nil {
		t.Fatalf("ListEvents failed: %v", err)
	}
	if len(res.Items) != 1 || res.Items[0].EventType != "algo_finished" {
		t.Fatalf("unexpected first page: %+v", res)
	}
	if res.NextCursor == nil || *res.NextCursor != res.Items[0].EventSeq {
		t.Fatalf("expected next cursor from first page, got %+v", res.NextCursor)
	}

	res, err = uc.ListEvents(context.Background(), "a1", ListEventsInput{
		EventTypePatterns: []string{"algo_%"},
		AlgoKey:           "env_analysis@1.0.0",
		BeforeEventSeq:    res.NextCursor,
		Limit:             10,
	})
	if err != nil {
		t.Fatalf("ListEvents follow-up failed: %v", err)
	}
	if len(res.Items) != 1 || res.Items[0].EventType != "algo_started" {
		t.Fatalf("unexpected follow-up page: %+v", res)
	}
}

func TestListTagHistory_FiltersTagEventsOnly(t *testing.T) {
	repo := &readModelAssetRepo{
		getFn: func(context.Context, string) (*models.Asset, error) {
			return &models.Asset{AssetID: "a1", McapFileID: "m1"}, nil
		},
	}
	eventRepo := newMockAssetEventRepo()

	appendEvent := func(eventType string, payload map[string]any) {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		if err := eventRepo.Append(context.Background(), repository.AssetEventAppendInput{
			EventType:    eventType,
			AssetID:      "a1",
			EventPayload: raw,
		}); err != nil {
			t.Fatalf("append event: %v", err)
		}
	}

	appendEvent("asset_created", map[string]any{"asset_id": "a1"})
	appendEvent("tag_upserted", map[string]any{"tag_key": "quality", "tag_value": "good"})
	appendEvent("algo_started", map[string]any{"algo_key": "env_analysis@1.0.0"})
	appendEvent("tag_deleted", map[string]any{"tag_key": "quality", "tag_value": "good"})

	uc := NewWithProjections(noopTxRunner{}, repo, nil, nil, eventRepo, nil, nil)
	res, err := uc.ListTagHistory(context.Background(), "a1", ListEventsInput{Limit: 10})
	if err != nil {
		t.Fatalf("ListTagHistory failed: %v", err)
	}
	if len(res.Items) != 2 {
		t.Fatalf("expected 2 tag events, got %d", len(res.Items))
	}
	for _, item := range res.Items {
		if item.EventType != "tag_upserted" && item.EventType != "tag_deleted" {
			t.Fatalf("unexpected event type in tag history: %s", item.EventType)
		}
	}
}

func TestUpsertTag_MultiSourceCoexists(t *testing.T) {
	repo := newMockAssetRepo()
	repo.assets["a1"] = &models.Asset{AssetID: "a1", McapFileID: "m1"}
	tagRepo := newMockAssetTagRepo()
	eventRepo := newMockAssetEventRepo()
	uc := NewWithProjections(noopTxRunner{}, repo, tagRepo, nil, eventRepo, buildTestTagRegistry(t), nil)

	if _, err := uc.UpsertTag(context.Background(), "a1", UpsertTagInput{
		Key: "quality", Value: "good",
		SourceType: "human", SourceName: "labeler_007",
	}); err != nil {
		t.Fatalf("human upsert: %v", err)
	}
	if _, err := uc.UpsertTag(context.Background(), "a1", UpsertTagInput{
		Key: "quality", Value: "good",
		SourceType: "rule_engine", SourceName: "qc_check", SourceVersion: "1.0",
	}); err != nil {
		t.Fatalf("rule_engine upsert: %v", err)
	}

	rows, _ := tagRepo.ListByAsset(context.Background(), "a1")
	sources := map[string]bool{}
	for _, r := range rows {
		if r.TagKey == "quality" {
			sources[r.SourceType] = true
		}
	}
	if !sources["human"] || !sources["rule_engine"] {
		t.Fatalf("expected both human and rule_engine sources, got %v", sources)
	}
}

func TestUpsertTag_RejectsMissingSourceName(t *testing.T) {
	repo := newMockAssetRepo()
	repo.assets["a1"] = &models.Asset{AssetID: "a1", McapFileID: "m1"}
	uc := NewWithProjections(noopTxRunner{}, repo, newMockAssetTagRepo(), nil, newMockAssetEventRepo(), buildTestTagRegistry(t), nil)

	_, err := uc.UpsertTag(context.Background(), "a1", UpsertTagInput{
		Key: "quality", Value: "good",
		SourceType: "human", // missing source_name
	})
	if err == nil {
		t.Fatalf("expected ErrTagSourceInvalid, got nil")
	}
	if !errors.Is(err, ErrTagSourceInvalid) {
		t.Fatalf("expected ErrTagSourceInvalid, got %v", err)
	}
}

func TestUpsertTag_ImmutableSourceCannotBeRewritten(t *testing.T) {
	repo := newMockAssetRepo()
	repo.assets["a1"] = &models.Asset{AssetID: "a1", McapFileID: "m1"}
	uc := NewWithProjections(noopTxRunner{}, repo, newMockAssetTagRepo(), nil, newMockAssetEventRepo(), buildTestTagRegistry(t), nil)

	if _, err := uc.UpsertTag(context.Background(), "a1", UpsertTagInput{
		Key: "quality", Value: "good",
		SourceType: "algo_sdk", SourceName: "hand_track", SourceVersion: "2.0",
	}); err != nil {
		t.Fatalf("first algo upsert: %v", err)
	}
	_, err := uc.UpsertTag(context.Background(), "a1", UpsertTagInput{
		Key: "quality", Value: "acceptable",
		SourceType: "algo_sdk", SourceName: "hand_track", SourceVersion: "2.0",
	})
	if err == nil || !errors.Is(err, ErrTagImmutable) {
		t.Fatalf("expected ErrTagImmutable on second write, got %v", err)
	}
}

func TestDeleteTag_SourceScopedRemovesOnlyMatching(t *testing.T) {
	repo := newMockAssetRepo()
	repo.assets["a1"] = &models.Asset{AssetID: "a1", McapFileID: "m1"}
	tagRepo := newMockAssetTagRepo()
	uc := NewWithProjections(noopTxRunner{}, repo, tagRepo, nil, newMockAssetEventRepo(), buildTestTagRegistry(t), nil)

	_, _ = uc.UpsertTag(context.Background(), "a1", UpsertTagInput{
		Key: "quality", Value: "good",
		SourceType: "human", SourceName: "alice",
	})
	_, _ = uc.UpsertTag(context.Background(), "a1", UpsertTagInput{
		Key: "quality", Value: "good",
		SourceType: "rule_engine", SourceName: "qc", SourceVersion: "1.0",
	})

	if _, err := uc.DeleteTag(context.Background(), "a1", "quality", "human"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	rows, _ := tagRepo.ListByAsset(context.Background(), "a1")
	for _, r := range rows {
		if r.SourceType == "human" {
			t.Fatalf("expected human row removed, still present: %#v", r)
		}
	}
	hasRule := false
	for _, r := range rows {
		if r.SourceType == "rule_engine" {
			hasRule = true
		}
	}
	if !hasRule {
		t.Fatalf("expected rule_engine row to remain")
	}
}
