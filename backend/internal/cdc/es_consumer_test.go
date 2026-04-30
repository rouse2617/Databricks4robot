package cdc

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	espkg "data-platform/internal/elasticsearch"
	"data-platform/internal/models"
)

type fakeAssetRepo struct {
	byMcap map[string][]*models.Asset
}

func (f *fakeAssetRepo) Get(context.Context, string) (*models.Asset, error)     { return nil, nil }
func (f *fakeAssetRepo) Set(context.Context, *models.Asset) error               { return nil }
func (f *fakeAssetRepo) SoftDelete(context.Context, string) error               { return nil }
func (f *fakeAssetRepo) WriteSegmentIndex(context.Context, *models.Asset) error { return nil }
func (f *fakeAssetRepo) ListWithFilters(context.Context, string, []interface{}, int, int, string) ([]*models.Asset, int64, error) {
	return nil, 0, nil
}
func (f *fakeAssetRepo) ListByMcapFile(_ context.Context, mcapFileID string) ([]*models.Asset, error) {
	return f.byMcap[mcapFileID], nil
}

type fakeBuilder struct {
	docs map[string]map[string]any
	ok   map[string]bool
	errs map[string]error
}

func (f *fakeBuilder) Build(_ context.Context, assetID string) (map[string]any, bool, error) {
	if err := f.errs[assetID]; err != nil {
		return nil, false, err
	}
	return f.docs[assetID], f.ok[assetID], nil
}

func newTestESServer(t *testing.T) (*httptest.Server, *[]string, *[]string) {
	t.Helper()

	var bulkIDs []string
	var deletedIDs []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/_bulk":
			body, _ := io.ReadAll(r.Body)
			lines := strings.Split(strings.TrimSpace(string(body)), "\n")
			for i := 0; i < len(lines)-1; i += 2 {
				var action struct {
					Index struct {
						ID string `json:"_id"`
					} `json:"index"`
				}
				if err := json.Unmarshal([]byte(lines[i]), &action); err != nil {
					t.Fatalf("unmarshal bulk action: %v", err)
				}
				bulkIDs = append(bulkIDs, action.Index.ID)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"errors": false,
				"items":  []map[string]any{},
			})
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/assets/_doc/"):
			deletedIDs = append(deletedIDs, strings.TrimPrefix(r.URL.Path, "/assets/_doc/"))
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))

	return server, &bulkIDs, &deletedIDs
}

func TestESConsumer_HandleAssetsEvent_RebuildsOneDocument(t *testing.T) {
	server, bulkIDs, deletedIDs := newTestESServer(t)
	defer server.Close()

	consumer := &ESConsumer{
		Assets: &fakeAssetRepo{},
		Builder: &fakeBuilder{
			docs: map[string]map[string]any{"a1": {"asset_id": "a1"}},
			ok:   map[string]bool{"a1": true},
			errs: map[string]error{},
		},
		ES: espkg.New(server.URL, "assets"),
	}

	err := consumer.Handle(context.Background(), ChangeEvent{
		Table: "assets",
		Op:    OperationUpdate,
		After: map[string]any{"asset_id": "a1"},
	})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if len(*bulkIDs) != 1 || (*bulkIDs)[0] != "a1" {
		t.Fatalf("expected bulk index for a1, got %v", *bulkIDs)
	}
	if len(*deletedIDs) != 0 {
		t.Fatalf("expected no deletes, got %v", *deletedIDs)
	}
}

func TestESConsumer_HandleMcapEvent_RebuildsAllAffectedAssets(t *testing.T) {
	server, bulkIDs, _ := newTestESServer(t)
	defer server.Close()

	consumer := &ESConsumer{
		Assets: &fakeAssetRepo{
			byMcap: map[string][]*models.Asset{
				"m1": {
					{AssetID: "a1"},
					{AssetID: "a2"},
				},
			},
		},
		Builder: &fakeBuilder{
			docs: map[string]map[string]any{
				"a1": {"asset_id": "a1"},
				"a2": {"asset_id": "a2"},
			},
			ok:   map[string]bool{"a1": true, "a2": true},
			errs: map[string]error{},
		},
		ES: espkg.New(server.URL, "assets"),
	}

	err := consumer.Handle(context.Background(), ChangeEvent{
		Table: "mcap_files",
		Op:    OperationUpdate,
		After: map[string]any{"mcap_file_id": "m1"},
	})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if len(*bulkIDs) != 2 {
		t.Fatalf("expected 2 indexed docs, got %v", *bulkIDs)
	}
}

func TestESConsumer_HandleDelete_BuildMissingDeletesDoc(t *testing.T) {
	server, bulkIDs, deletedIDs := newTestESServer(t)
	defer server.Close()

	consumer := &ESConsumer{
		Assets: &fakeAssetRepo{},
		Builder: &fakeBuilder{
			docs: map[string]map[string]any{},
			ok:   map[string]bool{"a1": false},
			errs: map[string]error{},
		},
		ES: espkg.New(server.URL, "assets"),
	}

	err := consumer.Handle(context.Background(), ChangeEvent{
		Table:  "assets",
		Op:     OperationDelete,
		Before: map[string]any{"asset_id": "a1"},
	})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if len(*bulkIDs) != 0 {
		t.Fatalf("expected no bulk indexing, got %v", *bulkIDs)
	}
	if len(*deletedIDs) != 1 || (*deletedIDs)[0] != "a1" {
		t.Fatalf("expected delete for a1, got %v", *deletedIDs)
	}
}

func TestESConsumer_HandleBatch_DeduplicatesAffectedAssetIDs(t *testing.T) {
	server, bulkIDs, _ := newTestESServer(t)
	defer server.Close()

	consumer := &ESConsumer{
		Assets: &fakeAssetRepo{
			byMcap: map[string][]*models.Asset{
				"m1": {{AssetID: "a1"}},
			},
		},
		Builder: &fakeBuilder{
			docs: map[string]map[string]any{"a1": {"asset_id": "a1"}},
			ok:   map[string]bool{"a1": true},
			errs: map[string]error{},
		},
		ES: espkg.New(server.URL, "assets"),
	}

	err := consumer.HandleBatch(context.Background(), []ChangeEvent{
		{Table: "assets", Op: OperationUpdate, After: map[string]any{"asset_id": "a1"}},
		{Table: "asset_tags", Op: OperationUpdate, After: map[string]any{"asset_id": "a1"}},
		{Table: "mcap_files", Op: OperationUpdate, After: map[string]any{"mcap_file_id": "m1"}},
	})
	if err != nil {
		t.Fatalf("HandleBatch returned error: %v", err)
	}
	if len(*bulkIDs) != 1 || (*bulkIDs)[0] != "a1" {
		t.Fatalf("expected one deduplicated rebuild for a1, got %v", *bulkIDs)
	}
}
