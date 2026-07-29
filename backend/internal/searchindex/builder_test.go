package searchindex

import (
	"context"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// ── stub repositories ────────────────────────────────────────────────────────

type stubAssetRepo struct {
	asset *models.Asset
}

func (s *stubAssetRepo) Get(ctx context.Context, id string) (*models.Asset, error) {
	return s.GetAll(ctx, id)
}
func (s *stubAssetRepo) GetAll(_ context.Context, _ string) (*models.Asset, error) {
	return s.asset, nil
}
func (s *stubAssetRepo) FindExistingIDs(_ context.Context, assetIDs []string) (map[string]struct{}, error) {
	out := make(map[string]struct{})
	if s.asset == nil {
		return out, nil
	}
	for _, assetID := range assetIDs {
		if assetID == s.asset.AssetID {
			out[assetID] = struct{}{}
		}
	}
	return out, nil
}
func (s *stubAssetRepo) InsertNew(context.Context, *models.Asset) error { return nil }
func (s *stubAssetRepo) Set(context.Context, *models.Asset) error       { return nil }
func (s *stubAssetRepo) SoftDelete(context.Context, string) error       { return nil }
func (s *stubAssetRepo) ListByMcapFile(context.Context, string) ([]*models.Asset, error) {
	return nil, nil
}
func (s *stubAssetRepo) ListByLogicalAssetID(context.Context, string) ([]*models.Asset, error) {
	return nil, nil
}
func (s *stubAssetRepo) WriteSegmentIndex(context.Context, *models.Asset) error { return nil }
func (s *stubAssetRepo) ListDescendants(_ context.Context, _ string) ([]*models.Asset, error) {
	return nil, nil
}
func (s *stubAssetRepo) ListWithFilters(_ context.Context, _ string, _ []interface{}, page, pageSize int, _ filter.OrderByClause) ([]*models.Asset, int64, error) {
	return nil, 0, nil
}

type stubTagRepo struct {
	tags []*models.AssetTag
}

func (s *stubTagRepo) Upsert(context.Context, repository.AssetTagUpsertInput) error {
	return nil
}
func (s *stubTagRepo) ListByAsset(_ context.Context, _ string) ([]*models.AssetTag, error) {
	return s.tags, nil
}
func (s *stubTagRepo) Delete(context.Context, string, string, string) error { return nil }

type stubAlgoRepo struct {
	algos []*models.AssetAlgoLatest
}

func (s *stubAlgoRepo) Upsert(context.Context, *models.AssetAlgoLatest) error { return nil }
func (s *stubAlgoRepo) GetByAlgo(context.Context, string, string) (*models.AssetAlgoLatest, error) {
	return nil, nil
}
func (s *stubAlgoRepo) ListByAsset(_ context.Context, _ string) ([]*models.AssetAlgoLatest, error) {
	return s.algos, nil
}

type stubLineageRepo struct {
	projection *repository.AssetLineageProjection
}

func (s *stubLineageRepo) GetLineageProjection(context.Context, string) (*repository.AssetLineageProjection, error) {
	return s.projection, nil
}

type stubMcapRepo struct {
	file *models.McapFile
}

func (s *stubMcapRepo) Get(_ context.Context, _ string) (*models.McapFile, error) {
	return s.file, nil
}
func (s *stubMcapRepo) Set(context.Context, *models.McapFile) error { return nil }
func (s *stubMcapRepo) UpdateIngestState(context.Context, string, models.IngestState) error {
	return nil
}
func (s *stubMcapRepo) List(context.Context, int, int, string, string, string) ([]*models.McapFile, int64, error) {
	return nil, 0, nil
}

// ── tests ────────────────────────────────────────────────────────────────────

func TestBuild_LifecycleStateIsPrimary(t *testing.T) {
	now := time.Now().UTC()
	b := &Builder{
		Assets: &stubAssetRepo{asset: &models.Asset{
			AssetID:        "abcd1234",
			McapFileID:     "m-1",
			AssetType:      "segment",
			LifecycleState: "ready",
			Status:         models.AssetStatusApproved,
			CreatedAt:      now,
			UpdatedAt:      now,
		}},
		Tags:  &stubTagRepo{},
		Algos: &stubAlgoRepo{},
	}

	doc, ok, err := b.Build(context.Background(), "abcd1234")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true")
	}

	// lifecycle_state should use the model's LifecycleState value.
	if got := doc["lifecycle_state"]; got != "ready" {
		t.Errorf("lifecycle_state = %q, want %q", got, "ready")
	}
	// status should still be present (dual-write window).
	if got := doc["status"]; got != "approved" {
		t.Errorf("status = %q, want %q", got, "approved")
	}
}

// CYB-3268: action assets are first-class but never traverse the lifecycle, so
// the ES doc must omit lifecycle_state (no meaningless facet bucket) and must not
// carry a nested actions[] array (actions are top-level docs now).
func TestBuild_ActionAssetOmitsLifecycleState(t *testing.T) {
	now := time.Now().UTC()
	b := &Builder{
		Assets: &stubAssetRepo{asset: &models.Asset{
			AssetID:        "act00001",
			McapFileID:     "m-1",
			AssetType:      "action",
			LifecycleState: "ready",
			ParentAssetID:  "seg00001",
			Metadata:       map[string]interface{}{"primary_label": "pickup"},
			CreatedAt:      now,
			UpdatedAt:      now,
		}},
		Tags:  &stubTagRepo{},
		Algos: &stubAlgoRepo{},
	}

	doc, ok, err := b.Build(context.Background(), "act00001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true")
	}
	if _, present := doc["lifecycle_state"]; present {
		t.Errorf("action doc must omit lifecycle_state, got %v", doc["lifecycle_state"])
	}
	if _, present := doc["actions"]; present {
		t.Errorf("action doc must not carry nested actions[], got %v", doc["actions"])
	}
	// asset_type must still be present so actions remain facetable.
	if doc["asset_type"] != "action" {
		t.Errorf("asset_type = %v, want action", doc["asset_type"])
	}
}

func TestBuild_LifecycleStateFallsBackToStatus(t *testing.T) {
	now := time.Now().UTC()
	b := &Builder{
		Assets: &stubAssetRepo{asset: &models.Asset{
			AssetID:        "zzzz8888",
			McapFileID:     "m-2",
			AssetType:      "segment",
			LifecycleState: "", // not yet backfilled
			Status:         models.AssetStatusRejected,
			CreatedAt:      now,
			UpdatedAt:      now,
		}},
		Tags:  &stubTagRepo{},
		Algos: &stubAlgoRepo{},
	}

	doc, ok, err := b.Build(context.Background(), "zzzz8888")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true")
	}

	// When lifecycle_state is empty, fall back to status for aggregation safety.
	if got := doc["lifecycle_state"]; got != "rejected" {
		t.Errorf("lifecycle_state = %q, want %q (fallback from status)", got, "rejected")
	}
	if got := doc["status"]; got != "rejected" {
		t.Errorf("status = %q, want %q", got, "rejected")
	}
}

func TestBuild_VersionFieldsWhenLogicalAssetSet(t *testing.T) {
	now := time.Now().UTC()
	b := &Builder{
		Assets: &stubAssetRepo{asset: &models.Asset{
			AssetID:        "newrev01",
			LogicalAssetID: "logical1",
			Revision:       2,
			IsCurrent:      true,
			McapFileID:     "m-1",
			AssetType:      "clip",
			LifecycleState: "ready",
			CreatedAt:      now,
			UpdatedAt:      now,
		}},
		Tags:  &stubTagRepo{},
		Algos: &stubAlgoRepo{},
	}
	doc, ok, err := b.Build(context.Background(), "newrev01")
	if err != nil || !ok {
		t.Fatalf("Build: ok=%v err=%v", ok, err)
	}
	if doc["logical_asset_id"] != "logical1" || doc["revision"] != int64(2) || doc["is_current"] != true {
		t.Fatalf("version fields: %#v", doc)
	}
}

func TestBuild_DatasetProjection(t *testing.T) {
	now := time.Now().UTC()
	b := &Builder{
		Assets: &stubAssetRepo{asset: &models.Asset{
			AssetID:        "data0001",
			AssetType:      "dataset",
			LifecycleState: "ready",
			Metadata: map[string]interface{}{
				"format":            "parquet",
				"record_count":      float64(42),
				"size_bytes":        float64(2048),
				"annotation_status": "raw",
				"source":            "ignored in typed projection",
			},
			CreatedAt: now,
			UpdatedAt: now,
		}},
		Tags:  &stubTagRepo{},
		Algos: &stubAlgoRepo{},
	}
	doc, ok, err := b.Build(context.Background(), "data0001")
	if err != nil || !ok {
		t.Fatalf("Build: ok=%v err=%v", ok, err)
	}
	dataset, ok := doc["dataset"].(map[string]any)
	if !ok {
		t.Fatalf("expected dataset projection, got %#v", doc["dataset"])
	}
	if dataset["format"] != "parquet" || dataset["annotation_status"] != "raw" {
		t.Fatalf("unexpected dataset projection: %#v", dataset)
	}
	if _, exists := dataset["source"]; exists {
		t.Fatalf("source should not be in typed ES projection: %#v", dataset)
	}
}

func TestBuild_AnnotationResultProjection(t *testing.T) {
	now := time.Now().UTC()
	b := &Builder{
		Assets: &stubAssetRepo{asset: &models.Asset{
			AssetID:        "anno0001",
			AssetType:      "annotation_result",
			LifecycleState: "ready",
			Metadata: map[string]interface{}{
				"tool":          "label-studio",
				"quality_score": 0.8,
				"coverage":      0.9,
				"artifact_uri":  "ignored in typed projection",
			},
			CreatedAt: now,
			UpdatedAt: now,
		}},
		Tags:  &stubTagRepo{},
		Algos: &stubAlgoRepo{},
	}
	doc, ok, err := b.Build(context.Background(), "anno0001")
	if err != nil || !ok {
		t.Fatalf("Build: ok=%v err=%v", ok, err)
	}
	projection, ok := doc["annotation_result"].(map[string]any)
	if !ok {
		t.Fatalf("expected annotation_result projection, got %#v", doc["annotation_result"])
	}
	if projection["tool"] != "label-studio" || projection["coverage"] != 0.9 {
		t.Fatalf("unexpected annotation_result projection: %#v", projection)
	}
	if _, exists := projection["artifact_uri"]; exists {
		t.Fatalf("artifact_uri should not be in typed ES projection: %#v", projection)
	}
}

func TestBuild_MLModelProjection(t *testing.T) {
	now := time.Now().UTC()
	b := &Builder{
		Assets: &stubAssetRepo{asset: &models.Asset{
			AssetID:        "model001",
			AssetType:      "ml_model",
			LifecycleState: "ready",
			Metadata: map[string]interface{}{
				"framework":    "pytorch",
				"architecture": "resnet50",
				"metrics": map[string]any{
					"accuracy": 0.94,
					"f1":       0.91,
				},
				"quantization": "int8",
				"artifact_uri": "gs://models/resnet50",
				"owner_note":   "ignored in typed projection",
			},
			CreatedAt: now,
			UpdatedAt: now,
		}},
		Tags:  &stubTagRepo{},
		Algos: &stubAlgoRepo{},
	}
	doc, ok, err := b.Build(context.Background(), "model001")
	if err != nil || !ok {
		t.Fatalf("Build: ok=%v err=%v", ok, err)
	}
	projection, ok := doc["ml_model"].(map[string]any)
	if !ok {
		t.Fatalf("expected ml_model projection, got %#v", doc["ml_model"])
	}
	if projection["framework"] != "pytorch" ||
		projection["architecture"] != "resnet50" ||
		projection["quantization"] != "int8" ||
		projection["artifact_uri"] != "gs://models/resnet50" {
		t.Fatalf("unexpected ml_model projection: %#v", projection)
	}
	metrics, ok := projection["metrics"].(map[string]any)
	if !ok || metrics["accuracy"] != 0.94 || metrics["f1"] != 0.91 {
		t.Fatalf("unexpected ml_model metrics projection: %#v", projection["metrics"])
	}
	if _, exists := projection["owner_note"]; exists {
		t.Fatalf("owner_note should not be in typed ES projection: %#v", projection)
	}
}

func TestBuild_EvaluationReportProjection(t *testing.T) {
	now := time.Now().UTC()
	b := &Builder{
		Assets: &stubAssetRepo{asset: &models.Asset{
			AssetID:        "eval001",
			AssetType:      "evaluation_report",
			LifecycleState: "ready",
			Metadata: map[string]interface{}{
				"model_id":   "model001",
				"dataset_id": "dataset001",
				"metrics": map[string]any{
					"precision": 0.88,
					"recall":    0.86,
				},
				"tool":       "evaluator",
				"report_uri": "ignored in typed projection",
			},
			CreatedAt: now,
			UpdatedAt: now,
		}},
		Tags:  &stubTagRepo{},
		Algos: &stubAlgoRepo{},
	}
	doc, ok, err := b.Build(context.Background(), "eval001")
	if err != nil || !ok {
		t.Fatalf("Build: ok=%v err=%v", ok, err)
	}
	projection, ok := doc["evaluation_report"].(map[string]any)
	if !ok {
		t.Fatalf("expected evaluation_report projection, got %#v", doc["evaluation_report"])
	}
	if projection["model_id"] != "model001" ||
		projection["dataset_id"] != "dataset001" ||
		projection["tool"] != "evaluator" {
		t.Fatalf("unexpected evaluation_report projection: %#v", projection)
	}
	metrics, ok := projection["metrics"].(map[string]any)
	if !ok || metrics["precision"] != 0.88 || metrics["recall"] != 0.86 {
		t.Fatalf("unexpected evaluation_report metrics projection: %#v", projection["metrics"])
	}
	if _, exists := projection["report_uri"]; exists {
		t.Fatalf("report_uri should not be in typed ES projection: %#v", projection)
	}
}

func TestBuild_NilAssetReturnsNotOk(t *testing.T) {
	b := &Builder{
		Assets: &stubAssetRepo{asset: nil},
		Tags:   &stubTagRepo{},
		Algos:  &stubAlgoRepo{},
	}

	_, ok, err := b.Build(context.Background(), "missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected ok=false for nil asset")
	}
}

func TestBuild_LineageProjection(t *testing.T) {
	now := time.Now().UTC()
	b := &Builder{
		Assets: &stubAssetRepo{asset: &models.Asset{
			AssetID:        "asset-mid",
			McapFileID:     "m-1",
			AssetType:      "segment",
			LifecycleState: "ready",
			CreatedAt:      now,
			UpdatedAt:      now,
		}},
		Tags:  &stubTagRepo{},
		Algos: &stubAlgoRepo{},
		Lineage: &stubLineageRepo{projection: &repository.AssetLineageProjection{
			UpstreamIDs:   []string{"asset-parent"},
			DownstreamIDs: []string{"asset-child"},
			RelationTypes: []string{"derived_from"},
		}},
	}
	doc, ok, err := b.Build(context.Background(), "asset-mid")
	if err != nil || !ok {
		t.Fatalf("Build: ok=%v err=%v", ok, err)
	}
	if got := doc["lineage_upstream_ids"].([]string); len(got) != 1 || got[0] != "asset-parent" {
		t.Fatalf("lineage_upstream_ids = %#v", doc["lineage_upstream_ids"])
	}
	if got := doc["lineage_downstream_ids"].([]string); len(got) != 1 || got[0] != "asset-child" {
		t.Fatalf("lineage_downstream_ids = %#v", doc["lineage_downstream_ids"])
	}
}

// CYB-3297 (Phase C): the builder must denormalize the mcap capture fields so
// the vendor/device/scene/... filters and facets the mapping+UI advertise
// actually resolve. Regression guard: before this, only mcap_uri/recorded_at
// were emitted and vendor_agg/scene_agg were permanently empty.
func TestBuild_EmitsMcapCaptureFields(t *testing.T) {
	now := time.Now().UTC()
	b := &Builder{
		Assets: &stubAssetRepo{asset: &models.Asset{
			AssetID:    "seg00001",
			McapFileID: "m-1",
			AssetType:  "segment",
			CreatedAt:  now,
			UpdatedAt:  now,
		}},
		Tags:  &stubTagRepo{},
		Algos: &stubAlgoRepo{},
		Mcap: &stubMcapRepo{file: &models.McapFile{
			McapFileID:     "m-1",
			GCSPath:        "gs://bucket/m-1.mcap",
			VendorID:       "vendor-x",
			DeviceID:       "device-y",
			CameraModel:    "CyberCap2",
			GraceVideoID:   "019f9893-3456-7376-ae68-30a89227eb46",
			DataSource:     "vendor",
			SceneID:        "scene-z",
			EnvironmentID:  "warehouse",
			TaskID:         "task-1",
			FileDurationMs: 1234,
		}},
	}

	doc, ok, err := b.Build(context.Background(), "seg00001")
	if err != nil || !ok {
		t.Fatalf("Build: ok=%v err=%v", ok, err)
	}
	mcap, _ := doc["mcap"].(map[string]any)
	if mcap == nil {
		t.Fatal("doc[mcap] missing or wrong type")
	}
	want := map[string]any{
		"vendor_id":        "vendor-x",
		"device_id":        "device-y",
		"camera_model":     "CyberCap2",
		"grace_video_id":   "019f9893-3456-7376-ae68-30a89227eb46",
		"data_source":      "vendor",
		"scene_id":         "scene-z",
		"environment_id":   "warehouse",
		"task_id":          "task-1",
		"mcap_uri":         "gs://bucket/m-1.mcap",
		"file_duration_ms": int64(1234),
	}
	for k, v := range want {
		if got := mcap[k]; got != v {
			t.Errorf("mcap[%q] = %#v, want %#v", k, got, v)
		}
	}
}

// CYB-3715: 7 flatten mirror columns on the Asset struct project to
// top-level string fields in the ES doc so filter/facet paths that address
// them by direct name (instead of mcap.<col>) actually resolve.
func TestBuild_EmitsCYB3715TopLevelFlattenFields(t *testing.T) {
	now := time.Now().UTC()
	b := &Builder{
		Assets: &stubAssetRepo{asset: &models.Asset{
			AssetID:          "seg37150",
			McapFileID:       "m-3715",
			AssetType:        "raw_mcap",
			CameraModel:      "CyberCap2",
			GraceVideoID:     "019f9893-3456-7376-ae68-30a89227eb46",
			DeviceID:         "11111111-1111-1111-1111-111111111111",
			CollectorID:      "22222222-2222-2222-2222-222222222222",
			SceneID:          "33333333-3333-3333-3333-333333333333",
			DataSource:       "vibecap",
			CollectionMethod: "manual",
			SourcePlatform:   "vibecap",
			CreatedAt:        now,
			UpdatedAt:        now,
		}},
		Tags:  &stubTagRepo{},
		Algos: &stubAlgoRepo{},
	}
	doc, ok, err := b.Build(context.Background(), "seg37150")
	if err != nil || !ok {
		t.Fatalf("Build: ok=%v err=%v", ok, err)
	}
	wantTop := map[string]any{
		"camera_model":      "CyberCap2",
		"grace_video_id":    "019f9893-3456-7376-ae68-30a89227eb46",
		"device_id":         "11111111-1111-1111-1111-111111111111",
		"collector_id":      "22222222-2222-2222-2222-222222222222",
		"scene_id":          "33333333-3333-3333-3333-333333333333",
		"data_source":       "vibecap",
		"collection_method": "manual",
		"source_platform":   "vibecap",
	}
	for k, v := range wantTop {
		if got, present := doc[k]; !present {
			t.Errorf("doc top-level %q missing", k)
		} else if got != v {
			t.Errorf("doc[%q] = %#v, want %#v", k, got, v)
		}
	}
}

// CYB-3715: empty flatten mirror columns stay absent so ES dynamic mapping
// doesn't create empty-string keyword buckets for every asset.
func TestBuild_OmitsEmptyCYB3715FlattenFields(t *testing.T) {
	now := time.Now().UTC()
	b := &Builder{
		Assets: &stubAssetRepo{asset: &models.Asset{
			AssetID: "seg37151", AssetType: "raw_mcap",
			CreatedAt: now, UpdatedAt: now,
		}},
		Tags:  &stubTagRepo{},
		Algos: &stubAlgoRepo{},
	}
	doc, ok, err := b.Build(context.Background(), "seg37151")
	if err != nil || !ok {
		t.Fatalf("Build: ok=%v err=%v", ok, err)
	}
	for _, k := range []string{
		"camera_model", "grace_video_id", "device_id", "collector_id", "scene_id",
		"data_source", "collection_method", "source_platform",
	} {
		if _, present := doc[k]; present {
			t.Errorf("doc top-level %q should be absent when Asset field is empty", k)
		}
	}
}

// Empty mcap fields must stay absent (additive, no empty-string buckets).
func TestBuild_OmitsEmptyMcapFields(t *testing.T) {
	now := time.Now().UTC()
	b := &Builder{
		Assets: &stubAssetRepo{asset: &models.Asset{
			AssetID: "seg00002", McapFileID: "m-2", AssetType: "segment",
			CreatedAt: now, UpdatedAt: now,
		}},
		Tags:  &stubTagRepo{},
		Algos: &stubAlgoRepo{},
		Mcap:  &stubMcapRepo{file: &models.McapFile{McapFileID: "m-2", GCSPath: "gs://b/x.mcap"}},
	}
	doc, ok, err := b.Build(context.Background(), "seg00002")
	if err != nil || !ok {
		t.Fatalf("Build: ok=%v err=%v", ok, err)
	}
	mcap, _ := doc["mcap"].(map[string]any)
	for _, k := range []string{"vendor_id", "device_id", "scene_id", "environment_id", "task_id", "file_duration_ms"} {
		if _, present := mcap[k]; present {
			t.Errorf("mcap[%q] should be absent when source is empty", k)
		}
	}
}

func (s *stubAssetRepo) LookupDurations(context.Context, []string, int64, int64) ([]repository.DurationRow, error) {
	return nil, nil
}

func (s *stubAssetRepo) LookupLineage(context.Context, []string) ([]repository.LineageRow, error) {
	return nil, nil
}

func (s *stubAssetRepo) LookupCosts(context.Context, []string, time.Time, time.Time, bool) ([]repository.AssetCostRow, error) {
	return nil, nil
}
