package asset

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	assetUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/asset"
)

type mockMcapRepoForAsset struct {
	getFn func(ctx context.Context, id string) (*models.McapFile, error)
}

func (m *mockMcapRepoForAsset) Get(ctx context.Context, id string) (*models.McapFile, error) {
	if m.getFn != nil {
		return m.getFn(ctx, id)
	}
	return nil, nil
}
func (m *mockMcapRepoForAsset) Set(context.Context, *models.McapFile) error { return nil }
func (m *mockMcapRepoForAsset) UpdateIngestState(context.Context, string, models.IngestState) error {
	return nil
}
func (m *mockMcapRepoForAsset) List(context.Context, int, int, string, string) ([]*models.McapFile, int64, error) {
	return nil, 0, nil
}

func newLocatorHandler(assetRepo *mockAssetRepo, mcapRepo *mockMcapRepoForAsset) *Handler {
	h := New(assetUC.New(assetRepo), &mockDeliveryRepoForAsset{})
	if mcapRepo != nil {
		h.SetMcapRepo(mcapRepo)
	}
	return h
}

func TestMcapLocator_Success(t *testing.T) {
	updated := time.Date(2026, 5, 8, 15, 32, 52, 0, time.UTC)
	assetRepo := &mockAssetRepo{
		getFn: func(_ context.Context, id string) (*models.Asset, error) {
			return &models.Asset{
				AssetID:          id,
				McapFileID:       "z7zyx6sl",
				LifecycleState:   "ready",
				StartTimestampNs: 1775435036056908229,
				EndTimestampNs:   1775435056056908229,
				DurationMs:       20000,
				Version:          1,
				UpdatedAt:        updated,
			}, nil
		},
	}
	mcapRepo := &mockMcapRepoForAsset{
		getFn: func(_ context.Context, id string) (*models.McapFile, error) {
			return &models.McapFile{
				McapFileID: id,
				GCSPath:    "gs://cyber-databrew-dev/uploads/" + id + ".mcap",
				SizeBytes:  1334578288,
				RawHashMD5: "deadbeef",
			}, nil
		},
	}

	h := newLocatorHandler(assetRepo, mcapRepo)
	r := setupAssetRouter(http.MethodGet, "/assets/:id/mcap-locator", h.McapLocator)

	w := doReq(t, r, http.MethodGet, "/assets/S86trk85/mcap-locator", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var got McapLocatorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.AssetID != "S86trk85" || got.LifecycleState != "ready" {
		t.Fatalf("unexpected asset fields: %+v", got)
	}
	if got.Mcap.McapFileID != "z7zyx6sl" || got.Mcap.SizeBytes != 1334578288 {
		t.Fatalf("unexpected mcap fields: %+v", got.Mcap)
	}
	if got.Window.StartTimestampNs != 1775435036056908229 || got.Window.DurationMs != 20000 {
		t.Fatalf("unexpected window fields: %+v", got.Window)
	}
}

func TestMcapLocator_AssetNotFound(t *testing.T) {
	assetRepo := &mockAssetRepo{
		getFn: func(context.Context, string) (*models.Asset, error) { return nil, nil },
	}
	h := newLocatorHandler(assetRepo, &mockMcapRepoForAsset{})
	r := setupAssetRouter(http.MethodGet, "/assets/:id/mcap-locator", h.McapLocator)

	w := doReq(t, r, http.MethodGet, "/assets/aaaaaaaa/mcap-locator", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestMcapLocator_NotPreviewable(t *testing.T) {
	assetRepo := &mockAssetRepo{
		getFn: func(_ context.Context, id string) (*models.Asset, error) {
			return &models.Asset{AssetID: id, McapFileID: "z7zyx6sl", LifecycleState: "processing"}, nil
		},
	}
	h := newLocatorHandler(assetRepo, &mockMcapRepoForAsset{})
	r := setupAssetRouter(http.MethodGet, "/assets/:id/mcap-locator", h.McapLocator)

	w := doReq(t, r, http.MethodGet, "/assets/S86trk85/mcap-locator", nil)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestMcapLocator_CreatedWithMcapSuccess(t *testing.T) {
	updated := time.Date(2026, 5, 8, 15, 32, 52, 0, time.UTC)
	assetRepo := &mockAssetRepo{
		getFn: func(_ context.Context, id string) (*models.Asset, error) {
			return &models.Asset{
				AssetID:          id,
				McapFileID:       "z7zyx6sl",
				LifecycleState:   "created",
				StartTimestampNs: 1775435036056908229,
				EndTimestampNs:   1775435056056908229,
				DurationMs:       20000,
				Version:          1,
				UpdatedAt:        updated,
			}, nil
		},
	}
	mcapRepo := &mockMcapRepoForAsset{
		getFn: func(_ context.Context, id string) (*models.McapFile, error) {
			return &models.McapFile{
				McapFileID: id,
				GCSPath:    "gs://cyber-databrew-dev/uploads/" + id + ".mcap",
				SizeBytes:  1334578288,
				RawHashMD5: "deadbeef",
			}, nil
		},
	}

	h := newLocatorHandler(assetRepo, mcapRepo)
	r := setupAssetRouter(http.MethodGet, "/assets/:id/mcap-locator", h.McapLocator)

	w := doReq(t, r, http.MethodGet, "/assets/S86trk85/mcap-locator", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestFoxgloveSource_CreatedSuccess(t *testing.T) {
	assetRepo := &mockAssetRepo{
		getFn: func(_ context.Context, id string) (*models.Asset, error) {
			return &models.Asset{
				AssetID:        id,
				McapFileID:     "z7zyx6sl",
				LifecycleState: "created",
			}, nil
		},
	}
	mcapRepo := &mockMcapRepoForAsset{
		getFn: func(_ context.Context, id string) (*models.McapFile, error) {
			return &models.McapFile{
				McapFileID: id,
				GCSPath:    "gs://cyber-databrew-dev/uploads/" + id + ".mcap",
			}, nil
		},
	}
	h := newLocatorHandler(assetRepo, mcapRepo)
	r := setupAssetRouter(http.MethodGet, "/assets/:id/foxglove-source", h.FoxgloveSource)

	w := doReq(t, r, http.MethodGet, "/assets/S86trk85/foxglove-source", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var got FoxgloveSourceResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.DSParams["url"] != "/api/v1/mcap-files/z7zyx6sl/bytes" {
		t.Fatalf("unexpected source payload: %+v", got)
	}
}

func TestMcapLocator_McapMissing(t *testing.T) {
	assetRepo := &mockAssetRepo{
		getFn: func(_ context.Context, id string) (*models.Asset, error) {
			return &models.Asset{AssetID: id, McapFileID: "z7zyx6sl", LifecycleState: "ready"}, nil
		},
	}
	mcapRepo := &mockMcapRepoForAsset{
		getFn: func(context.Context, string) (*models.McapFile, error) { return nil, nil },
	}
	h := newLocatorHandler(assetRepo, mcapRepo)
	r := setupAssetRouter(http.MethodGet, "/assets/:id/mcap-locator", h.McapLocator)

	w := doReq(t, r, http.MethodGet, "/assets/S86trk85/mcap-locator", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestMcapLocator_RepoNotConfigured(t *testing.T) {
	assetRepo := &mockAssetRepo{
		getFn: func(_ context.Context, id string) (*models.Asset, error) {
			return &models.Asset{AssetID: id, McapFileID: "z7zyx6sl", LifecycleState: "ready"}, nil
		},
	}
	h := newLocatorHandler(assetRepo, nil) // no mcapRepo wired
	r := setupAssetRouter(http.MethodGet, "/assets/:id/mcap-locator", h.McapLocator)

	w := doReq(t, r, http.MethodGet, "/assets/S86trk85/mcap-locator", nil)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestMcapLocator_AssetRepoError(t *testing.T) {
	assetRepo := &mockAssetRepo{
		getFn: func(context.Context, string) (*models.Asset, error) { return nil, errors.New("boom") },
	}
	h := newLocatorHandler(assetRepo, &mockMcapRepoForAsset{})
	r := setupAssetRouter(http.MethodGet, "/assets/:id/mcap-locator", h.McapLocator)

	w := doReq(t, r, http.MethodGet, "/assets/S86trk85/mcap-locator", nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestFoxgloveSource_Success(t *testing.T) {
	assetRepo := &mockAssetRepo{
		getFn: func(_ context.Context, id string) (*models.Asset, error) {
			return &models.Asset{
				AssetID:        id,
				McapFileID:     "z7zyx6sl",
				LifecycleState: "ready",
			}, nil
		},
	}
	mcapRepo := &mockMcapRepoForAsset{
		getFn: func(_ context.Context, id string) (*models.McapFile, error) {
			return &models.McapFile{
				McapFileID: id,
				GCSPath:    "gs://cyber-databrew-dev/uploads/" + id + ".mcap",
			}, nil
		},
	}
	h := newLocatorHandler(assetRepo, mcapRepo)
	r := setupAssetRouter(http.MethodGet, "/assets/:id/foxglove-source", h.FoxgloveSource)

	w := doReq(t, r, http.MethodGet, "/assets/S86trk85/foxglove-source", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var got FoxgloveSourceResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.DS != "remote-file" || got.DSParams["url"] != "/api/v1/mcap-files/z7zyx6sl/bytes" {
		t.Fatalf("unexpected source payload: %+v", got)
	}
}
