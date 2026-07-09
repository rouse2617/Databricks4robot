package asset

import (
	"context"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

type mockLogicalAssetRepo struct {
	byID      map[string]*models.LogicalAsset
	inserted  []*models.LogicalAsset
	maxRev    map[string]int64
	currentID map[string]string
}

func newMockLogicalAssetRepo() *mockLogicalAssetRepo {
	return &mockLogicalAssetRepo{
		byID:      map[string]*models.LogicalAsset{},
		maxRev:    map[string]int64{},
		currentID: map[string]string{},
	}
}

func (m *mockLogicalAssetRepo) Get(_ context.Context, id string) (*models.LogicalAsset, error) {
	return m.byID[id], nil
}

func (m *mockLogicalAssetRepo) Insert(_ context.Context, la *models.LogicalAsset) error {
	cp := *la
	m.byID[la.LogicalAssetID] = &cp
	m.inserted = append(m.inserted, &cp)
	m.maxRev[la.LogicalAssetID] = la.CurrentRevision
	return nil
}

func (m *mockLogicalAssetRepo) BumpRevision(_ context.Context, id string, rev int64) error {
	if la, ok := m.byID[id]; ok {
		la.CurrentRevision = rev
		la.TotalRevisions = rev
	}
	m.maxRev[id] = rev
	return nil
}

func (m *mockLogicalAssetRepo) MaxRevision(_ context.Context, id string) (int64, error) {
	return m.maxRev[id], nil
}

func (m *mockLogicalAssetRepo) CurrentAssetID(_ context.Context, id string) (string, error) {
	return m.currentID[id], nil
}

func (m *mockLogicalAssetRepo) ClearCurrentForLogical(_ context.Context, id string) error {
	delete(m.currentID, id)
	return nil
}

type versionedAssetRepo struct {
	*readModelAssetRepo
	inserted  []*models.Asset
	relations []string
}

func (r *versionedAssetRepo) InsertNew(_ context.Context, a *models.Asset) error {
	cp := *a
	r.inserted = append(r.inserted, &cp)
	return nil
}

var _ repository.AssetRelationWriter = (*versionedAssetRepo)(nil)

func (r *versionedAssetRepo) InsertRevisionOf(_ context.Context, newID, priorID, _ string) error {
	r.relations = append(r.relations, newID+"->"+priorID)
	return nil
}

func (r *versionedAssetRepo) InsertRelation(_ context.Context, parentID, childID, relType, _ string) error {
	r.relations = append(r.relations, parentID+"->"+childID+":"+relType)
	return nil
}

func hasEventForAsset(events []*models.AssetEvent, eventType, assetID string) bool {
	for _, e := range events {
		if e.EventType == eventType && e.AssetID == assetID {
			return true
		}
	}
	return false
}

func TestCreate_FirstVersionSetsLogicalFields(t *testing.T) {
	assetRepo := &versionedAssetRepo{readModelAssetRepo: &readModelAssetRepo{}}
	logicalRepo := newMockLogicalAssetRepo()
	eventRepo := newMockAssetEventRepo()
	uc := NewWithProjections(noopTxRunner{}, assetRepo, newMockAssetTagRepo(), nil, eventRepo, nil, nil)
	uc.SetLogicalAssetRepo(logicalRepo)

	a, err := uc.Create(context.Background(), CreateInput{
		AssetID:          "aaaaaaaa",
		McapFileID:       "bbbbbbbb",
		StartTimestampNs: 1,
		EndTimestampNs:   1000001,
		Reviewer:         "r",
		AssetType:        "clip",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if a.LogicalAssetID != "aaaaaaaa" || a.Revision != 1 || !a.IsCurrent {
		t.Fatalf("version fields: logical=%s rev=%d current=%v", a.LogicalAssetID, a.Revision, a.IsCurrent)
	}
	if len(logicalRepo.inserted) != 1 {
		t.Fatalf("expected 1 logical_assets insert, got %d", len(logicalRepo.inserted))
	}
	if !hasEventType(eventRepo.events, "asset_created") {
		t.Fatal("expected asset_created event")
	}
}

func TestCreate_PromoteVersion(t *testing.T) {
	assetRepo := &versionedAssetRepo{readModelAssetRepo: &readModelAssetRepo{}}
	logicalRepo := newMockLogicalAssetRepo()
	logicalRepo.byID["aaaaaaaa"] = &models.LogicalAsset{
		LogicalAssetID:  "aaaaaaaa",
		AssetType:       "clip",
		CurrentRevision: 1,
		TotalRevisions:  1,
	}
	logicalRepo.maxRev["aaaaaaaa"] = 1
	logicalRepo.currentID["aaaaaaaa"] = "cccccccc"
	eventRepo := newMockAssetEventRepo()
	uc := NewWithProjections(noopTxRunner{}, assetRepo, newMockAssetTagRepo(), nil, eventRepo, nil, nil)
	uc.SetLogicalAssetRepo(logicalRepo)

	a, err := uc.Create(context.Background(), CreateInput{
		AssetID:          "dddddddd",
		LogicalAssetID:   "aaaaaaaa",
		McapFileID:       "bbbbbbbb",
		StartTimestampNs: 10,
		EndTimestampNs:   1000010,
		Reviewer:         "r",
		AssetType:        "clip",
	})
	if err != nil {
		t.Fatalf("Create promote: %v", err)
	}
	if a.Revision != 2 || !a.IsCurrent || a.LogicalAssetID != "aaaaaaaa" {
		t.Fatalf("promoted asset: %+v", a)
	}
	if !hasEventType(eventRepo.events, "version_promoted") {
		t.Fatal("expected version_promoted event")
	}
	if !hasEventForAsset(eventRepo.events, "asset_updated", "cccccccc") {
		t.Fatal("expected asset_updated for demoted prior revision")
	}
	if len(assetRepo.relations) != 1 {
		t.Fatalf("expected revision_of relation, got %v", assetRepo.relations)
	}
}

func TestCreate_PromoteVersionWithGeneratedAssetID(t *testing.T) {
	assetRepo := &versionedAssetRepo{readModelAssetRepo: &readModelAssetRepo{}}
	logicalRepo := newMockLogicalAssetRepo()
	logicalRepo.byID["aaaaaaaa"] = &models.LogicalAsset{
		LogicalAssetID:  "aaaaaaaa",
		AssetType:       "clip",
		CurrentRevision: 1,
		TotalRevisions:  1,
	}
	logicalRepo.maxRev["aaaaaaaa"] = 1
	logicalRepo.currentID["aaaaaaaa"] = "cccccccc"
	eventRepo := newMockAssetEventRepo()
	uc := NewWithProjections(noopTxRunner{}, assetRepo, newMockAssetTagRepo(), nil, eventRepo, nil, nil)
	uc.SetLogicalAssetRepo(logicalRepo)

	a, err := uc.Create(context.Background(), CreateInput{
		LogicalAssetID:   "aaaaaaaa",
		McapFileID:       "bbbbbbbb",
		StartTimestampNs: 10,
		EndTimestampNs:   1000010,
		Reviewer:         "r",
		AssetType:        "clip",
	})
	if err != nil {
		t.Fatalf("Create promote with generated asset_id: %v", err)
	}
	if a.AssetID == "" || a.AssetID == "aaaaaaaa" {
		t.Fatalf("expected generated asset id, got %q", a.AssetID)
	}
	if a.Revision != 2 || !a.IsCurrent || a.LogicalAssetID != "aaaaaaaa" {
		t.Fatalf("promoted asset: %+v", a)
	}
	if !hasEventType(eventRepo.events, "version_promoted") {
		t.Fatal("expected version_promoted event")
	}
}
