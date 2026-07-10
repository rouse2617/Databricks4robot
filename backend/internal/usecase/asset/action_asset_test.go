package asset

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// CYB-3268: action first-class asset usecase behavior.

func TestActionLabelsFromMetadata(t *testing.T) {
	cases := []struct {
		name        string
		md          map[string]interface{}
		wantPrimary string
		wantLabels  []string
	}{
		{"nil", nil, "", nil},
		{"go strings", map[string]interface{}{"primary_label": "pickup", "labels": []string{"pickup", "left_hand"}}, "pickup", []string{"pickup", "left_hand"}},
		{"json arrays", map[string]interface{}{"primary_label": "place", "labels": []interface{}{"place", "right_hand"}}, "place", []string{"place", "right_hand"}},
		{"missing labels", map[string]interface{}{"primary_label": "pickup"}, "pickup", nil},
		{"wrong types dropped", map[string]interface{}{"primary_label": 123, "labels": []interface{}{1, "ok", 2}}, "", []string{"ok"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotP, gotL := actionLabelsFromMetadata(tc.md)
			if gotP != tc.wantPrimary {
				t.Errorf("primary = %q, want %q", gotP, tc.wantPrimary)
			}
			if len(gotL) != len(tc.wantLabels) {
				t.Fatalf("labels = %v, want %v", gotL, tc.wantLabels)
			}
			for i := range gotL {
				if gotL[i] != tc.wantLabels[i] {
					t.Errorf("labels[%d] = %q, want %q", i, gotL[i], tc.wantLabels[i])
				}
			}
		})
	}
}

// writeTestActionRegistry writes a minimal action-label registry to a temp file
// and loads it, so tests don't depend on the repo-root config path.
func writeTestActionRegistry(t *testing.T) *config.ActionLabelRegistry {
	t.Helper()
	path := filepath.Join(t.TempDir(), "reg.yaml")
	body := "primary_labels:\n  - pickup\n  - place\nlabels:\n  - pickup\n  - place\n  - left_hand\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write registry: %v", err)
	}
	reg, err := config.LoadActionLabelRegistry(path)
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	return reg
}

func TestCreateChildAsset_ActionLabelRegistryRejectsUnknown(t *testing.T) {
	uc := New(&readModelAssetRepo{})
	uc.SetActionLabelRegistry(writeTestActionRegistry(t))

	// Unregistered primary_label → ErrInvalidActionLabel, before any repo call.
	_, err := uc.CreateChildAsset(context.Background(), CreateChildAssetInput{
		AssetType:        "action",
		ParentAssetID:    "seg00001",
		StartTimestampNs: 0,
		EndTimestampNs:   2_000_000,
		Metadata:         map[string]interface{}{"primary_label": "not_a_label"},
	})
	if !errors.Is(err, ErrInvalidActionLabel) {
		t.Fatalf("want ErrInvalidActionLabel, got %v", err)
	}
}

func TestGetActionForParent_RejectsMissingWrongTypeAndCrossParent(t *testing.T) {
	cases := []struct {
		name string
		row  *models.Asset
	}{
		{"missing", nil},
		{"wrong type", &models.Asset{AssetID: "act00001", AssetType: "segment", ParentAssetID: "seg00001"}},
		{"cross parent", &models.Asset{AssetID: "act00001", AssetType: "action", ParentAssetID: "other_seg"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &readModelAssetRepo{
				getFn: func(context.Context, string) (*models.Asset, error) { return tc.row, nil },
			}
			uc := New(repo)
			_, err := uc.getActionForParent(context.Background(), "seg00001", "act00001")
			if !errors.Is(err, ErrNotFound) {
				t.Fatalf("want ErrNotFound, got %v", err)
			}
		})
	}
}

func TestListActionsByParent_QueriesActionsUnderParent(t *testing.T) {
	var gotWhere string
	var gotArgs []interface{}
	repo := &readModelAssetRepo{
		listWithFiltersFn: func(_ context.Context, where string, args []interface{}, _, _ int, _ filter.OrderByClause) ([]*models.Asset, int64, error) {
			gotWhere = where
			gotArgs = args
			return []*models.Asset{{AssetID: "act00001", AssetType: "action", ParentAssetID: "seg00001"}}, 1, nil
		},
	}
	uc := New(repo)
	items, total, err := uc.ListActionsByParent(context.Background(), "seg00001", 50, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].AssetType != "action" {
		t.Fatalf("unexpected result: total=%d items=%v", total, items)
	}
	if gotWhere != "parent_asset_id = $1 AND asset_type = $2" {
		t.Errorf("where clause = %q", gotWhere)
	}
	if len(gotArgs) != 2 || gotArgs[0] != "seg00001" || gotArgs[1] != "action" {
		t.Errorf("args = %v, want [seg00001 action]", gotArgs)
	}
}
