package deliveryrules

import (
	"context"
	"fmt"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// ─── Error types ────────────────────────────────────────────────────────────

// HierarchyViolation is returned when an asset write violates a hierarchy invariant.
type HierarchyViolation struct {
	Invariant string // e.g. "L1"
	AssetType string
	ParentID  string
	ParentType string
	Expected  string
}

func (v *HierarchyViolation) Error() string {
	return fmt.Sprintf("hierarchy invariant %s violated: %s", v.Invariant, v.Expected)
}

// ─── Parent info query ──────────────────────────────────────────────────────

// ParentInfo holds metadata needed for hierarchy validation.
type ParentInfo struct {
	AssetType string
}

// ParentGetter abstracts parent asset lookup for testability.
type ParentGetter interface {
	GetParentInfo(ctx context.Context, parentAssetID string) (*ParentInfo, error)
}

// ─── Validator ──────────────────────────────────────────────────────────────

// AssetWriteValidator checks L1-L7 hierarchy invariants before asset writes.
type AssetWriteValidator struct {
	parents ParentGetter
}

func NewAssetWriteValidator(parents ParentGetter) *AssetWriteValidator {
	return &AssetWriteValidator{parents: parents}
}

// ValidateCreate checks all invariants for a new asset INSERT.
func (v *AssetWriteValidator) ValidateCreate(ctx context.Context, a *models.Asset) error {
	if v.parents == nil {
		return nil
	}

	// raw_mcap is always a root — no parent needed.
	if a.AssetType == "raw_mcap" {
		return nil
	}

	// L7: non-root assets must have a parent (derived_asset may use asset_relations).
	if a.AssetType != "derived_asset" && a.ParentAssetID == "" {
		return &HierarchyViolation{
			Invariant: "L7",
			AssetType: a.AssetType,
			Expected:  fmt.Sprintf("%s must have a parent_asset_id", a.AssetType),
		}
	}
	if a.ParentAssetID == "" {
		// derived_asset with no parent — skip remaining parent checks.
		return nil
	}

	// L₀: parent must exist.
	parent, err := v.parents.GetParentInfo(ctx, a.ParentAssetID)
	if err != nil {
		return fmt.Errorf("lookup parent %s: %w", a.ParentAssetID, err)
	}
	if parent == nil {
		return &HierarchyViolation{
			Invariant: "L0",
			AssetType: a.AssetType,
			ParentID:  a.ParentAssetID,
			Expected:  fmt.Sprintf("parent asset %s not found", a.ParentAssetID),
		}
	}

	// Dispatch to type-specific checks.
	switch a.AssetType {
	case "segment":
		return v.checkSegment(parent)
	case "clip", "frame":
		return v.checkChildOfSegment(a.AssetType, parent)
	case "task":
		return v.checkChildOfSegment(a.AssetType, parent)
	case "action":
		return v.checkAction(parent)
	case "derived_asset":
		return v.checkDerived(parent)
	}
	return nil
}

// ValidateUpdate checks invariants that apply to updates.
func (v *AssetWriteValidator) ValidateUpdate(_ context.Context, existing, updated *models.Asset) error {
	if existing.AssetType != updated.AssetType {
		return &HierarchyViolation{
			Invariant:  "L4b",
			AssetType:  existing.AssetType,
			Expected:   fmt.Sprintf("asset_type is immutable: cannot change %q to %q", existing.AssetType, updated.AssetType),
			ParentType: updated.AssetType,
		}
	}
	return nil
}

// ─── Type-specific checks ───────────────────────────────────────────────────

// L1 + L4 combined: segment's parent must be raw_mcap (not another segment).
func (v *AssetWriteValidator) checkSegment(parent *ParentInfo) error {
	if parent.AssetType != "raw_mcap" {
		return &HierarchyViolation{
			Invariant:  "L1",
			AssetType:  "segment",
			ParentType: parent.AssetType,
			Expected:   "segment parent must be raw_mcap",
		}
	}
	return nil
}

// L2: clip/frame/task parent must be segment.
func (v *AssetWriteValidator) checkChildOfSegment(childType string, parent *ParentInfo) error {
	if parent.AssetType != "segment" {
		return &HierarchyViolation{
			Invariant:  "L2",
			AssetType:  childType,
			ParentType: parent.AssetType,
			Expected:   fmt.Sprintf("%s parent must be segment", childType),
		}
	}
	return nil
}

// L2 + L3 + L5: action can be L2 (parent=segment) or L3 (parent=task).
func (v *AssetWriteValidator) checkAction(parent *ParentInfo) error {
	switch parent.AssetType {
	case "segment":
		return nil // L2 action — normal
	case "task":
		return nil // L3 action — normal (time-window check deferred to P1.5)
	default:
		return &HierarchyViolation{
			Invariant:  "L2",
			AssetType:  "action",
			ParentType: parent.AssetType,
			Expected:   "action parent must be segment (L2) or task (L3)",
		}
	}
}

// derived_asset with a parent_asset_id — just verify parent exists (L₀ already checked).
func (v *AssetWriteValidator) checkDerived(parent *ParentInfo) error {
	// No additional constraints — parent is informational; real multi-parent
	// relationships are modelled in asset_relations with type=merged_from.
	return nil
}

// ValidateCreateBatch checks L1-L7 for a batch of assets, sharing parent lookups.
// Returns the first violation found.
func (v *AssetWriteValidator) ValidateCreateBatch(ctx context.Context, assets []*models.Asset) error {
	// Simple P1 implementation: validate one at a time.
	// P1.5: add cachingParentGetter to avoid N+1 when siblings share a parent.
	for _, a := range assets {
		if err := v.ValidateCreate(ctx, a); err != nil {
			return err
		}
	}
	return nil
}

// ─── repo-backed ParentGetter ───────────────────────────────────────────────

// AssetRepoParentGetter implements ParentGetter via AssetRepository.
type AssetRepoParentGetter struct {
	repo interface {
		Get(ctx context.Context, assetID string) (*models.Asset, error)
	}
}

func NewAssetRepoParentGetter(repo interface {
	Get(ctx context.Context, assetID string) (*models.Asset, error)
}) *AssetRepoParentGetter {
	return &AssetRepoParentGetter{repo: repo}
}

func (g *AssetRepoParentGetter) GetParentInfo(ctx context.Context, parentAssetID string) (*ParentInfo, error) {
	a, err := g.repo.Get(ctx, parentAssetID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, nil
	}
		return &ParentInfo{AssetType: a.AssetType}, nil
}
