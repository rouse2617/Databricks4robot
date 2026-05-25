package deliveryrules

import (
	"context"
	"errors"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// ─── mock ParentGetter ──────────────────────────────────────────────────────

type mockParentGetter struct {
	infos map[string]*ParentInfo // parentAssetID → info; empty = not found
	err   error
}

func (m *mockParentGetter) GetParentInfo(_ context.Context, parentAssetID string) (*ParentInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	info, ok := m.infos[parentAssetID]
	if !ok {
		return nil, nil
	}
	return info, nil
}

// ─── helpers ────────────────────────────────────────────────────────────────

func assetWithParent(assetType, parentID string) *models.Asset {
	return &models.Asset{AssetType: assetType, ParentAssetID: parentID}
}

func parentInfo(assetType string) *ParentInfo {
	return &ParentInfo{AssetType: assetType}
}

var nilValidator = NewAssetWriteValidator(nil) // validates nothing when no getter

// ─── L0: parent must exist ──────────────────────────────────────────────────

func TestValidateCreate_L0_parentNotFound(t *testing.T) {
	v := NewAssetWriteValidator(&mockParentGetter{infos: map[string]*ParentInfo{}})
	a := assetWithParent("segment", "missing")
	err := v.ValidateCreate(context.Background(), a)
	var hv *HierarchyViolation
	if !errors.As(err, &hv) || hv.Invariant != "L0" {
		t.Fatalf("expected L0 violation, got %v", err)
	}
}

func TestValidateCreate_L0_parentExistsOK(t *testing.T) {
	v := NewAssetWriteValidator(&mockParentGetter{infos: map[string]*ParentInfo{
		"raw001": parentInfo("raw_mcap"),
	}})
	err := v.ValidateCreate(context.Background(), assetWithParent("segment", "raw001"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// ─── L1: segment parent must be raw_mcap ─────────────────────────────────────

func TestValidateCreate_L1_segmentParentMustBeRawMcap(t *testing.T) {
	v := NewAssetWriteValidator(&mockParentGetter{infos: map[string]*ParentInfo{
		"seg001": parentInfo("segment"), // wrong type
	}})
	err := v.ValidateCreate(context.Background(), assetWithParent("segment", "seg001"))
	var hv *HierarchyViolation
	if !errors.As(err, &hv) || hv.Invariant != "L1" {
		t.Fatalf("expected L1 violation, got %v", err)
	}
}

func TestValidateCreate_L1_segmentOnRawMcapOK(t *testing.T) {
	v := NewAssetWriteValidator(&mockParentGetter{infos: map[string]*ParentInfo{
		"raw001": parentInfo("raw_mcap"),
	}})
	err := v.ValidateCreate(context.Background(), assetWithParent("segment", "raw001"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// ─── L2: clip/frame parent must be segment; L5: task parent must be segment or task ─

func TestValidateCreate_L2_clipParentMustBeSegment(t *testing.T) {
	v := NewAssetWriteValidator(&mockParentGetter{infos: map[string]*ParentInfo{
		"raw001": parentInfo("raw_mcap"),
	}})
	err := v.ValidateCreate(context.Background(), assetWithParent("clip", "raw001"))
	var hv *HierarchyViolation
	if !errors.As(err, &hv) || hv.Invariant != "L2" {
		t.Fatalf("expected L2 violation, got %v", err)
	}
}

func TestValidateCreate_L2_clipOnSegmentOK(t *testing.T) {
	v := NewAssetWriteValidator(&mockParentGetter{infos: map[string]*ParentInfo{
		"seg001": parentInfo("segment"),
	}})
	err := v.ValidateCreate(context.Background(), assetWithParent("clip", "seg001"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateCreate_L2_frameParentMustBeSegment(t *testing.T) {
	v := NewAssetWriteValidator(&mockParentGetter{infos: map[string]*ParentInfo{
		"raw001": parentInfo("raw_mcap"),
	}})
	err := v.ValidateCreate(context.Background(), assetWithParent("frame", "raw001"))
	var hv *HierarchyViolation
	if !errors.As(err, &hv) || hv.Invariant != "L2" {
		t.Fatalf("expected L2 violation, got %v", err)
	}
}

func TestValidateCreate_L5_taskParentInvalid(t *testing.T) {
	v := NewAssetWriteValidator(&mockParentGetter{infos: map[string]*ParentInfo{
		"raw001": parentInfo("raw_mcap"),
	}})
	err := v.ValidateCreate(context.Background(), assetWithParent("task", "raw001"))
	var hv *HierarchyViolation
	if !errors.As(err, &hv) || hv.Invariant != "L5" {
		t.Fatalf("expected L5 violation, got %v", err)
	}
}

func TestValidateCreate_L5_taskOnTaskOK(t *testing.T) {
	v := NewAssetWriteValidator(&mockParentGetter{infos: map[string]*ParentInfo{
		"task001": parentInfo("task"),
	}})
	err := v.ValidateCreate(context.Background(), assetWithParent("task", "task001"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// ─── L3: action can be L2 (parent=segment) or L3 (parent=task) ───────────────

func TestValidateCreate_L3_actionOnSegmentOK(t *testing.T) {
	v := NewAssetWriteValidator(&mockParentGetter{infos: map[string]*ParentInfo{
		"seg001": parentInfo("segment"),
	}})
	err := v.ValidateCreate(context.Background(), assetWithParent("action", "seg001"))
	if err != nil {
		t.Fatalf("expected no error for L2 action, got %v", err)
	}
}

func TestValidateCreate_L3_actionOnTaskOK(t *testing.T) {
	v := NewAssetWriteValidator(&mockParentGetter{infos: map[string]*ParentInfo{
		"task001": parentInfo("task"),
	}})
	err := v.ValidateCreate(context.Background(), assetWithParent("action", "task001"))
	if err != nil {
		t.Fatalf("expected no error for L3 action, got %v", err)
	}
}

func TestValidateCreate_L3_actionOnRawMcapRejected(t *testing.T) {
	v := NewAssetWriteValidator(&mockParentGetter{infos: map[string]*ParentInfo{
		"raw001": parentInfo("raw_mcap"),
	}})
	err := v.ValidateCreate(context.Background(), assetWithParent("action", "raw001"))
	var hv *HierarchyViolation
	if !errors.As(err, &hv) || hv.Invariant != "L2" {
		t.Fatalf("expected L2 violation, got %v", err)
	}
	if hv.AssetType != "action" {
		t.Fatalf("expected asset_type=action, got %s", hv.AssetType)
	}
}

// ─── L5: task must not have clip/frame/task children ─────────────────────────

func TestValidateCreate_L5_taskChildClipRejected(t *testing.T) {
	// This is implicitly handled by L2: if parent is task, only action is allowed.
	// clip on task → L2 violation because clip expects segment.
	v := NewAssetWriteValidator(&mockParentGetter{infos: map[string]*ParentInfo{
		"task001": parentInfo("task"),
	}})
	err := v.ValidateCreate(context.Background(), assetWithParent("clip", "task001"))
	var hv *HierarchyViolation
	if !errors.As(err, &hv) || hv.Invariant != "L2" {
		t.Fatalf("expected L2 violation, got %v", err)
	}
}

func TestValidateCreate_L5_taskChildFrameRejected(t *testing.T) {
	v := NewAssetWriteValidator(&mockParentGetter{infos: map[string]*ParentInfo{
		"task001": parentInfo("task"),
	}})
	err := v.ValidateCreate(context.Background(), assetWithParent("frame", "task001"))
	var hv *HierarchyViolation
	if !errors.As(err, &hv) || hv.Invariant != "L2" {
		t.Fatalf("expected L2 violation, got %v", err)
	}
}

// ─── L7: non-root assets must have parent_asset_id ──────────────────────────

func TestValidateCreate_L7_segmentWithoutParent(t *testing.T) {
	v := NewAssetWriteValidator(&mockParentGetter{})
	a := &models.Asset{AssetType: "segment", ParentAssetID: ""}
	err := v.ValidateCreate(context.Background(), a)
	var hv *HierarchyViolation
	if !errors.As(err, &hv) || hv.Invariant != "L7" {
		t.Fatalf("expected L7 violation, got %v", err)
	}
}

// ─── Root types don't need parent ────────────────────────────────────────────

func TestValidateCreate_rawMcapNoParentOK(t *testing.T) {
	v := NewAssetWriteValidator(&mockParentGetter{})
	err := v.ValidateCreate(context.Background(), &models.Asset{AssetType: "raw_mcap"})
	if err != nil {
		t.Fatalf("expected no error for raw_mcap, got %v", err)
	}
}

func TestValidateCreate_derivedAssetNoParentOK(t *testing.T) {
	v := NewAssetWriteValidator(&mockParentGetter{})
	err := v.ValidateCreate(context.Background(), &models.Asset{AssetType: "derived_asset", ParentAssetID: ""})
	if err != nil {
		t.Fatalf("expected no error for derived_asset without parent, got %v", err)
	}
}

func TestValidateCreate_derivedAssetWithParentOK(t *testing.T) {
	v := NewAssetWriteValidator(&mockParentGetter{infos: map[string]*ParentInfo{
		"seg001": parentInfo("segment"),
	}})
	err := v.ValidateCreate(context.Background(), assetWithParent("derived_asset", "seg001"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// ─── nil validator (no getter) ──────────────────────────────────────────────

func TestValidateCreate_nilGetterSkipsValidation(t *testing.T) {
	err := nilValidator.ValidateCreate(context.Background(), &models.Asset{AssetType: "segment"})
	if err != nil {
		t.Fatalf("expected no error with nil getter, got %v", err)
	}
}

// ─── ValidateUpdate: asset_type immutable ───────────────────────────────────-

func TestValidateUpdate_L4b_rejectsAssetTypeChange(t *testing.T) {
	v := NewAssetWriteValidator(nil)
	existing := &models.Asset{AssetType: "segment"}
	updated := &models.Asset{AssetType: "clip"}
	err := v.ValidateUpdate(context.Background(), existing, updated)
	var hv *HierarchyViolation
	if !errors.As(err, &hv) || hv.Invariant != "L4b" {
		t.Fatalf("expected L4b violation, got %v", err)
	}
}

func TestValidateUpdate_L4b_sameTypeOK(t *testing.T) {
	v := NewAssetWriteValidator(nil)
	existing := &models.Asset{AssetType: "segment"}
	updated := &models.Asset{AssetType: "segment"}
	err := v.ValidateUpdate(context.Background(), existing, updated)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// ─── batch validation (P1: sequential) ───────────────────────────────────────

func TestValidateCreateBatch_firstViolation(t *testing.T) {
	v := NewAssetWriteValidator(&mockParentGetter{infos: map[string]*ParentInfo{
		"raw001": parentInfo("raw_mcap"),
	}})
	assets := []*models.Asset{
		{AssetType: "segment", ParentAssetID: "raw001"},       // OK
		{AssetType: "clip", ParentAssetID: "raw001"},          // L2 — clip needs segment
		{AssetType: "frame", ParentAssetID: "should_not_run"}, // should not be reached
	}
	err := v.ValidateCreateBatch(context.Background(), assets)
	var hv *HierarchyViolation
	if !errors.As(err, &hv) || hv.Invariant != "L2" || hv.AssetType != "clip" {
		t.Fatalf("expected L2 violation on clip, got %v", err)
	}
}

func TestValidateCreateBatch_allOK(t *testing.T) {
	v := NewAssetWriteValidator(&mockParentGetter{infos: map[string]*ParentInfo{
		"raw001": parentInfo("raw_mcap"),
		"seg001": parentInfo("segment"),
	}})
	assets := []*models.Asset{
		{AssetType: "segment", ParentAssetID: "raw001"},
		{AssetType: "clip", ParentAssetID: "seg001"},
		{AssetType: "frame", ParentAssetID: "seg001"},
		{AssetType: "task", ParentAssetID: "seg001"},
		{AssetType: "action", ParentAssetID: "seg001"},
	}
	err := v.ValidateCreateBatch(context.Background(), assets)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// ─── error from parent getter ────────────────────────────────────────────────

func TestValidateCreate_getterError(t *testing.T) {
	v := NewAssetWriteValidator(&mockParentGetter{err: errors.New("db conn lost")})
	err := v.ValidateCreate(context.Background(), assetWithParent("segment", "raw001"))
	if err == nil {
		t.Fatal("expected error from getter, got nil")
	}
}

// ─── unknown asset type ─────────────────────────────────────────────────────

func TestValidateCreate_unknownAssetType(t *testing.T) {
	v := NewAssetWriteValidator(&mockParentGetter{infos: map[string]*ParentInfo{
		"parent001": {AssetType: "segment"},
	}})
	err := v.ValidateCreate(context.Background(), assetWithParent("bogus_type", "parent001"))
	var hv *HierarchyViolation
	if !errors.As(err, &hv) || hv.Invariant != "L0" {
		t.Fatalf("expected L0 violation for unknown type, got %v", err)
	}
}

// ─── ParentInfo builder (repo-backed) ───────────────────────────────────────

func TestAssetRepoParentGetter_nilAsset(t *testing.T) {
	g := NewAssetRepoParentGetter(&mockAssetGet{asset: nil})
	info, err := g.GetParentInfo(context.Background(), "missing")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if info != nil {
		t.Fatal("expected nil info for missing asset")
	}
}

type mockAssetGet struct {
	asset *models.Asset
	err   error
}

func (m *mockAssetGet) Get(_ context.Context, _ string) (*models.Asset, error) {
	return m.asset, m.err
}
