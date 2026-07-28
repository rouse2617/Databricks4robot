package algorun_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	algorunUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/algorun"
	"github.com/CyberOrigin2077/cyber-databrew/internal/usecase/assetvalidation"
)

type memAlgoRunRepo struct {
	byID map[string]*models.AlgoRun
}

func newMemAlgoRunRepo() *memAlgoRunRepo {
	return &memAlgoRunRepo{byID: map[string]*models.AlgoRun{}}
}

func (m *memAlgoRunRepo) Insert(_ context.Context, run *models.AlgoRun) error {
	if _, ok := m.byID[run.RunID]; ok {
		return repository.ErrDuplicateRunID
	}
	cp := *run
	m.byID[run.RunID] = &cp
	return nil
}

func (m *memAlgoRunRepo) Get(_ context.Context, runID string) (*models.AlgoRun, error) {
	r, ok := m.byID[runID]
	if !ok {
		return nil, nil
	}
	cp := *r
	return &cp, nil
}

func (m *memAlgoRunRepo) Exists(_ context.Context, runID string) (bool, error) {
	_, ok := m.byID[runID]
	return ok, nil
}

func (m *memAlgoRunRepo) Start(_ context.Context, runID string, _ time.Time) error {
	r, ok := m.byID[runID]
	if !ok {
		return repository.ErrAlgoRunNotFound
	}
	if r.Status != models.AlgoRunStatusPending {
		return repository.ErrAlgoRunBadState
	}
	r.Status = models.AlgoRunStatusRunning
	return nil
}

func (m *memAlgoRunRepo) Finish(_ context.Context, runID string, patch repository.AlgoRunFinishPatch) error {
	r, ok := m.byID[runID]
	if !ok {
		return repository.ErrAlgoRunNotFound
	}
	if r.Status != models.AlgoRunStatusRunning {
		return repository.ErrAlgoRunBadState
	}
	r.Status = patch.Status
	return nil
}

func (m *memAlgoRunRepo) Cancel(_ context.Context, runID, reason string, finishedAt time.Time) error {
	r, ok := m.byID[runID]
	if !ok {
		return repository.ErrAlgoRunNotFound
	}
	if r.Status == models.AlgoRunStatusOK || r.Status == models.AlgoRunStatusFailed || r.Status == "cancelled" {
		return repository.ErrAlgoRunBadState
	}
	r.Status = "cancelled"
	return nil
}

func (m *memAlgoRunRepo) List(_ context.Context, _ repository.AlgoRunListFilter) ([]*models.AlgoRun, int64, error) {
	var out []*models.AlgoRun
	for _, r := range m.byID {
		cp := *r
		out = append(out, &cp)
	}
	return out, int64(len(out)), nil
}

func (m *memAlgoRunRepo) GetAffectedAssets(_ context.Context, _ string) ([]*repository.AffectedAsset, error) {
	return nil, nil
}

type memAssetRepo struct {
	existing map[string]struct{}
}

func (m *memAssetRepo) FindExistingIDs(_ context.Context, assetIDs []string) (map[string]struct{}, error) {
	out := make(map[string]struct{})
	for _, assetID := range assetIDs {
		if _, ok := m.existing[assetID]; ok {
			out[assetID] = struct{}{}
		}
	}
	return out, nil
}
func (m *memAssetRepo) Get(context.Context, string) (*models.Asset, error)    { return nil, nil }
func (m *memAssetRepo) GetAll(context.Context, string) (*models.Asset, error) { return nil, nil }
func (m *memAssetRepo) InsertNew(context.Context, *models.Asset) error        { return nil }
func (m *memAssetRepo) Set(context.Context, *models.Asset) error              { return nil }
func (m *memAssetRepo) SoftDelete(context.Context, string) error              { return nil }
func (m *memAssetRepo) ListByMcapFile(context.Context, string) ([]*models.Asset, error) {
	return nil, nil
}
func (m *memAssetRepo) ListByLogicalAssetID(context.Context, string) ([]*models.Asset, error) {
	return nil, nil
}
func (m *memAssetRepo) WriteSegmentIndex(context.Context, *models.Asset) error { return nil }
func (m *memAssetRepo) ListWithFilters(context.Context, string, []interface{}, int, int, filter.OrderByClause) ([]*models.Asset, int64, error) {
	return nil, 0, nil
}
func (m *memAssetRepo) ListDescendants(context.Context, string) ([]*models.Asset, error) {
	return nil, nil
}

func TestAlgoRunLifecycle(t *testing.T) {
	repo := newMemAlgoRunRepo()
	uc := algorunUC.New(repo)
	ctx := context.Background()

	runID := "R001abc123def456"
	created, err := uc.Create(ctx, algorunUC.CreateInput{
		RunID:       runID,
		AlgoName:    "hand_track",
		AlgoVersion: "2.0",
		TriggeredBy: "manual:ops",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Status != models.AlgoRunStatusPending {
		t.Fatalf("status=%s", created.Status)
	}

	started, err := uc.Start(ctx, runID)
	if err != nil || started.Status != models.AlgoRunStatusRunning {
		t.Fatalf("start: err=%v status=%s", err, started.Status)
	}

	n := 10
	finished, err := uc.Finish(ctx, runID, algorunUC.FinishInput{
		Status:          models.AlgoRunStatusOK,
		AssetsProcessed: &n,
	})
	if err != nil || finished.Status != models.AlgoRunStatusOK {
		t.Fatalf("finish: err=%v status=%s", err, finished.Status)
	}
}

func TestCreate_ValidatesInputAssetIDs(t *testing.T) {
	ctx := context.Background()
	baseInput := algorunUC.CreateInput{
		RunID:       "R001abc123def456",
		AlgoName:    "hand_track",
		AlgoVersion: "2.0",
		TriggeredBy: "manual:ops",
	}

	t.Run("missing asset fails before insert", func(t *testing.T) {
		runRepo := newMemAlgoRunRepo()
		uc := algorunUC.New(runRepo)
		uc.SetAssetRepo(&memAssetRepo{existing: map[string]struct{}{"asset-1": {}}})

		in := baseInput
		in.InputAssetIDs = []string{"asset-1", "missing"}
		_, err := uc.Create(ctx, in)
		if err == nil {
			t.Fatal("expected missing asset error")
		}
		var validationErr *assetvalidation.ValidationError
		if !errors.As(err, &validationErr) {
			t.Fatalf("expected ValidationError, got %T %[1]v", err)
		}
		if len(validationErr.MissingIDs) != 1 || validationErr.MissingIDs[0] != "missing" {
			t.Fatalf("missing IDs = %v", validationErr.MissingIDs)
		}
		if len(runRepo.byID) != 0 {
			t.Fatalf("expected no run inserted, got %d", len(runRepo.byID))
		}
	})

	t.Run("duplicate asset ID fails with details", func(t *testing.T) {
		uc := algorunUC.New(newMemAlgoRunRepo())
		uc.SetAssetRepo(&memAssetRepo{existing: map[string]struct{}{"asset-1": {}}})

		in := baseInput
		in.InputAssetIDs = []string{"asset-1", "asset-1"}
		_, err := uc.Create(ctx, in)
		if err == nil {
			t.Fatal("expected duplicate asset error")
		}
		var validationErr *assetvalidation.ValidationError
		if !errors.As(err, &validationErr) {
			t.Fatalf("expected ValidationError, got %T %[1]v", err)
		}
		if len(validationErr.DuplicateIDs) != 1 || validationErr.DuplicateIDs[0] != "asset-1" {
			t.Fatalf("duplicate IDs = %v", validationErr.DuplicateIDs)
		}
	})

	t.Run("valid assets are normalized", func(t *testing.T) {
		uc := algorunUC.New(newMemAlgoRunRepo())
		uc.SetAssetRepo(&memAssetRepo{existing: map[string]struct{}{"asset-1": {}}})

		in := baseInput
		in.InputAssetIDs = []string{" asset-1 "}
		run, err := uc.Create(ctx, in)
		if err != nil {
			t.Fatalf("Create err=%v", err)
		}
		if len(run.InputAssetIDs) != 1 || run.InputAssetIDs[0] != "asset-1" {
			t.Fatalf("input asset IDs = %v", run.InputAssetIDs)
		}
	})
}

func TestCreate_DuplicateRunIDReturnsConflict(t *testing.T) {
	repo := newMemAlgoRunRepo()
	uc := algorunUC.New(repo)
	ctx := context.Background()

	in := algorunUC.CreateInput{
		RunID:       "R001abc123def456",
		AlgoName:    "hand_track",
		AlgoVersion: "2.0",
		TriggeredBy: "manual:ops",
	}
	if _, err := uc.Create(ctx, in); err != nil {
		t.Fatalf("first create err=%v", err)
	}
	if _, err := uc.Create(ctx, in); err != repository.ErrDuplicateRunID {
		t.Fatalf("second create err=%v, want ErrDuplicateRunID", err)
	}
}

func (m *memAssetRepo) LookupCosts(context.Context, []string, time.Time, time.Time, bool) ([]repository.AssetCostRow, error) {
	return nil, nil
}

func (m *memAssetRepo) LookupDurations(context.Context, []string, int64, int64) ([]repository.DurationRow, error) {
	return nil, nil
}

func (m *memAssetRepo) LookupLineage(context.Context, []string) ([]repository.LineageRow, error) {
	return nil, nil
}
