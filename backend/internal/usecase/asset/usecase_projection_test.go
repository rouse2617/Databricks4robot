package asset

import (
	"context"
	"testing"

	"data-platform/internal/models"
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

func tagRowKey(assetID, tagKey string) string { return assetID + "|" + tagKey }

func (m *mockAssetTagRepo) Upsert(_ context.Context, assetID, tagKey, tagValue, tagType, sourceType string) error {
	m.rows[tagRowKey(assetID, tagKey)] = &models.AssetTag{
		AssetID:    assetID,
		TagKey:     tagKey,
		TagValue:   tagValue,
		TagType:    tagType,
		SourceType: sourceType,
	}
	return nil
}

func (m *mockAssetTagRepo) ListByAsset(_ context.Context, assetID string) ([]*models.AssetTag, error) {
	out := []*models.AssetTag{}
	for _, row := range m.rows {
		if row.AssetID == assetID {
			out = append(out, row)
		}
	}
	return out, nil
}

func (m *mockAssetTagRepo) Delete(_ context.Context, assetID, tagKey string) error {
	delete(m.rows, tagRowKey(assetID, tagKey))
	return nil
}

func ptrString(s string) *string { return &s }

func hasEventType(events []*models.AssetEvent, want string) bool {
	for _, e := range events {
		if e.EventType == want {
			return true
		}
	}
	return false
}

func TestCreate_WritesTagProjectionAndOutbox(t *testing.T) {
	repo := newMockAssetRepo()
	tagRepo := newMockAssetTagRepo()
	eventRepo := newMockAssetEventRepo()
	uc := NewWithProjections(noopTxRunner{}, repo, tagRepo, eventRepo, nil, nil)

	a, err := uc.Create(context.Background(), CreateInput{
		McapFileID:       "mcap-create-001",
		StartTimestampNs: 100,
		EndTimestampNs:   200,
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

func TestUpdate_WritesTagProjectionAndLifecycleEvents(t *testing.T) {
	repo := newMockAssetRepo()
	tagRepo := newMockAssetTagRepo()
	eventRepo := newMockAssetEventRepo()
	uc := NewWithProjections(noopTxRunner{}, repo, tagRepo, eventRepo, nil, nil)

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

func TestDelete_AppendsLifecycleEvent(t *testing.T) {
	repo := newMockAssetRepo()
	eventRepo := newMockAssetEventRepo()
	uc := NewWithProjections(noopTxRunner{}, repo, nil, eventRepo, nil, nil)

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
