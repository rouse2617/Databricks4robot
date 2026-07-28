package backfill

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// CYB-4350: batch total-duration is populated onto every job returned by
// ListJobs/GetJob, and appears in the Feishu completion notification when
// non-zero (and is omitted when zero).

func TestFormatBatchDurationMs(t *testing.T) {
	tests := []struct {
		name string
		ms   int64
		want string
	}{
		{"zero", 0, ""},
		{"negative", -1, ""},
		{"sub-second", 800, "<1s"},
		{"one second", 1000, "1s"},
		{"seconds only", 45_000, "45s"},
		{"exact minute", 60_000, "1m 0s"},
		{"minutes and seconds", 90_500, "1m 30s"},
		{"exact hour", 3_600_000, "1h 0m"},
		{"hours and minutes drops seconds", 3_665_500, "1h 1m"},
		{"many hours", 7_200_000, "2h 0m"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatBatchDurationMs(tc.ms); got != tc.want {
				t.Errorf("formatBatchDurationMs(%d) = %q, want %q", tc.ms, got, tc.want)
			}
		})
	}
}

func TestFormatBatchJobNotificationText_TotalDurationLine(t *testing.T) {
	job := &models.BackfillJob{
		ID:         "job-x",
		Name:       "my-batch",
		TotalCount: 3,
		CreatedBy:  "alice@cyberorigin.ai",
		CreatedAt:  time.Now().Add(-30 * time.Second),
	}
	summary := repository.BackfillItemStatusSummary{Completed: 3}

	// >0 → renders line
	text := formatBatchJobNotificationText(job, "completed", summary, 7_200_000, "")
	if !strings.Contains(text, "总时长：2h 0m") {
		t.Errorf("expected 总时长 line in text, got: %s", text)
	}

	// 0 → line omitted
	textZero := formatBatchJobNotificationText(job, "completed", summary, 0, "")
	if strings.Contains(textZero, "总时长") {
		t.Errorf("expected 总时长 line to be omitted when duration is 0, got: %s", textZero)
	}

	// Ordering: 总时长 appears between counters and 创建人
	idxCount := strings.Index(text, "失败：0")
	idxTotal := strings.Index(text, "总时长")
	idxCreator := strings.Index(text, "创建人")
	if !(idxCount < idxTotal && idxTotal < idxCreator) {
		t.Errorf("expected order 失败…<总时长<创建人, got positions %d < %d < %d in: %s", idxCount, idxTotal, idxCreator, text)
	}
}

func TestListJobs_PopulatesTotalDurationMs(t *testing.T) {
	ctx := context.Background()
	repo := &listJobsDurationRepo{
		jobs: []models.BackfillJob{
			{ID: "j1"},
			{ID: "j2"},
			{ID: "j3"}, // absent from durations → implicit 0
		},
		durations: map[string]int64{"j1": 5000, "j2": 90_500},
	}
	uc := New(repo, nil)

	got, err := uc.ListJobs(ctx, "")
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 jobs, got %d", len(got))
	}
	byID := map[string]int64{}
	for _, j := range got {
		byID[j.ID] = j.TotalDurationMs
	}
	if byID["j1"] != 5000 || byID["j2"] != 90_500 || byID["j3"] != 0 {
		t.Errorf("wrong totals: %+v", byID)
	}
}

func TestGetJob_PopulatesTotalDurationMs(t *testing.T) {
	ctx := context.Background()
	repo := &listJobsDurationRepo{
		jobs:      []models.BackfillJob{{ID: "j1"}},
		durations: map[string]int64{"j1": 42},
	}
	uc := New(repo, nil)

	got, err := uc.GetJob(ctx, "j1")
	if err != nil {
		t.Fatalf("GetJob: %v", err)
	}
	if got == nil {
		t.Fatalf("expected job, got nil")
	}
	if got.TotalDurationMs != 42 {
		t.Errorf("expected TotalDurationMs=42, got %d", got.TotalDurationMs)
	}
}

func TestSyncJobProgress_NotifiesWithTotalDurationLine(t *testing.T) {
	ctx := context.Background()
	jobID := "job-td-1"
	repo := &pausedSyncRepo{
		job: &models.BackfillJob{
			ID:         jobID,
			Name:       "td-batch",
			Status:     "running",
			TotalCount: 1,
			CreatedAt:  time.Now().Add(-30 * time.Second),
		},
		items: []models.BackfillItem{
			{ID: "it-1", JobID: jobID, Status: "completed"},
		},
		totalDurations: map[string]int64{jobID: 3_600_000 + 5*60_000}, // 1h5m
	}
	notifier := &fakeNotifier{}
	uc := New(repo, nil)
	uc.SetNotifier(notifier, "")

	if err := uc.syncJobProgressForce(ctx, jobID); err != nil {
		t.Fatalf("syncJobProgressForce: %v", err)
	}
	if notifier.callCount() != 1 {
		t.Fatalf("expected 1 notification, got %d", notifier.callCount())
	}
	text := notifier.lastCall()
	if !strings.Contains(text, "总时长：1h 5m") {
		t.Errorf("expected 总时长 line in notification, got: %s", text)
	}
}

// listJobsDurationRepo is a targeted fake for CYB-4350 tests. It embeds
// pausedSyncRepo to reuse the interface implementations we don't care about,
// and overrides only FindAllJobs / FindJobByID / TotalDurationByBatchIDs.
type listJobsDurationRepo struct {
	pausedSyncRepo
	jobs      []models.BackfillJob
	durations map[string]int64
}

func (r *listJobsDurationRepo) FindAllJobs(context.Context, string) ([]models.BackfillJob, error) {
	out := make([]models.BackfillJob, len(r.jobs))
	copy(out, r.jobs)
	return out, nil
}

func (r *listJobsDurationRepo) FindJobByID(_ context.Context, id string) (*models.BackfillJob, error) {
	for i := range r.jobs {
		if r.jobs[i].ID == id {
			cp := r.jobs[i]
			return &cp, nil
		}
	}
	return nil, nil
}

func (r *listJobsDurationRepo) TotalDurationByBatchIDs(_ context.Context, ids []string) (map[string]int64, error) {
	out := make(map[string]int64, len(ids))
	for _, id := range ids {
		if v, ok := r.durations[id]; ok {
			out[id] = v
		}
	}
	return out, nil
}
