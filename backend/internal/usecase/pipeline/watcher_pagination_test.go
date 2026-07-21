package pipeline

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

func newMockRunRepoWith(runs []models.PipelineRun) *mockRunRepo {
	byID := make(map[string]*models.PipelineRun, len(runs))
	for i := range runs {
		r := runs[i]
		byID[r.ID] = &r
	}
	return &mockRunRepo{byID: byID}
}

// loadRunsForWatcherSync must advance its (createdAt, id) cursor after a
// full-cap page and reset after a short page (wrap to newest). CYB-3746: with
// enough active runs to exceed watcherActiveRunLoadCap, a static newest-only
// window would leave older active runs stale forever. Cursor rotation is what
// guarantees eventual coverage of every active run.
func TestLoadRunsForWatcherSync_CursorAdvancesAndWraps(t *testing.T) {
	// Build (cap + 2) active runs so the first page is full and the second
	// starts strictly older than the last (createdAt, id) of the first page.
	base := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	total := watcherActiveRunLoadCap + 2
	all := make([]models.PipelineRun, 0, total)
	for i := 0; i < total; i++ {
		all = append(all, models.PipelineRun{
			ID:        fmt.Sprintf("run-%05d", i),
			Status:    "Pending",
			CreatedAt: base.Add(-time.Duration(i) * time.Second), // newest at i=0
		})
	}
	uc := &Usecase{runRepo: newMockRunRepoWith(all)}

	// Page 1: fresh cursor, full cap → cursor must advance to the last row.
	if _, err := uc.loadRunsForWatcherSync(context.Background()); err != nil {
		t.Fatalf("page 1: %v", err)
	}
	uc.watcherLoadCursorMu.Lock()
	c1CreatedAt := uc.watcherLoadCursorCreatedAt
	c1ID := uc.watcherLoadCursorID
	uc.watcherLoadCursorMu.Unlock()
	wantLastID := fmt.Sprintf("run-%05d", watcherActiveRunLoadCap-1)
	if c1ID != wantLastID {
		t.Fatalf("cursor id after page 1 = %q, want %q", c1ID, wantLastID)
	}
	wantLastCreatedAt := base.Add(-time.Duration(watcherActiveRunLoadCap-1) * time.Second)
	if !c1CreatedAt.Equal(wantLastCreatedAt) {
		t.Fatalf("cursor createdAt after page 1 = %v, want %v", c1CreatedAt, wantLastCreatedAt)
	}

	// Page 2: only 2 rows left → short page → cursor must reset for wrap.
	if _, err := uc.loadRunsForWatcherSync(context.Background()); err != nil {
		t.Fatalf("page 2: %v", err)
	}
	uc.watcherLoadCursorMu.Lock()
	c2CreatedAt := uc.watcherLoadCursorCreatedAt
	c2ID := uc.watcherLoadCursorID
	uc.watcherLoadCursorMu.Unlock()
	if c2ID != "" || !c2CreatedAt.IsZero() {
		t.Fatalf("cursor must reset after a short page (wrap); got id=%q createdAt=%v", c2ID, c2CreatedAt)
	}

	// Page 3: cursor reset → next call must return the newest window. This
	// proves rotation is cyclic instead of one-way.
	got, err := uc.loadRunsForWatcherSync(context.Background())
	if err != nil {
		t.Fatalf("page 3: %v", err)
	}
	if len(got) == 0 || got[0].ID != "run-00000" {
		firstID := ""
		if len(got) > 0 {
			firstID = got[0].ID
		}
		t.Fatalf("after wrap, first row must be newest run-00000; got %d rows, first=%q", len(got), firstID)
	}
}

// A single scan-worth of active runs (below cap) must not stick the cursor —
// the small-batch path should keep behaving exactly as before the change.
func TestLoadRunsForWatcherSync_SmallSetLeavesCursorEmpty(t *testing.T) {
	base := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	all := make([]models.PipelineRun, 0, 5)
	for i := 0; i < 5; i++ {
		all = append(all, models.PipelineRun{
			ID:        fmt.Sprintf("run-%d", i),
			Status:    "Pending",
			CreatedAt: base.Add(-time.Duration(i) * time.Second),
		})
	}
	uc := &Usecase{runRepo: newMockRunRepoWith(all)}
	if _, err := uc.loadRunsForWatcherSync(context.Background()); err != nil {
		t.Fatal(err)
	}
	uc.watcherLoadCursorMu.Lock()
	defer uc.watcherLoadCursorMu.Unlock()
	if uc.watcherLoadCursorID != "" || !uc.watcherLoadCursorCreatedAt.IsZero() {
		t.Fatalf("cursor must stay empty on a short page; got id=%q createdAt=%v",
			uc.watcherLoadCursorID, uc.watcherLoadCursorCreatedAt)
	}
}
