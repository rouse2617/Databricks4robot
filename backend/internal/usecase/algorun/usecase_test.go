package algorun_test

import (
	"context"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	algorunUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/algorun"
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
