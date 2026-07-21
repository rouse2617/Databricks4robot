package schedtask

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/schedtask/source"
)

// ─── test doubles ─────────────────────────────────────────────

type fakeRepo struct {
	mu           sync.Mutex
	rules        []Rule
	claimReturns []Rule
	successes    []recordCall
	failures     []recordCall
	setEnabled   []struct {
		id string
		on bool
	}
}
type recordCall struct {
	id, status, cursor, batchID, err string
	at                               time.Time
}

func (f *fakeRepo) ClaimDueRules(_ context.Context, _ time.Time, _ int) ([]Rule, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := append([]Rule(nil), f.claimReturns...)
	f.claimReturns = nil
	return out, nil
}
func (f *fakeRepo) RecordSuccess(_ context.Context, id, status, cursor, batchID string, at time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.successes = append(f.successes, recordCall{id: id, status: status, cursor: cursor, batchID: batchID, at: at})
	return nil
}
func (f *fakeRepo) SetEnabled(_ context.Context, id string, on bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.setEnabled = append(f.setEnabled, struct {
		id string
		on bool
	}{id, on})
	return nil
}

func (f *fakeRepo) RecordFailure(_ context.Context, id, errMsg string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failures = append(f.failures, recordCall{id: id, err: errMsg})
	return nil
}

type fakeSource struct {
	ids    []string
	cursor string
	err    error
}

func (f fakeSource) Fetch(_ context.Context, _ string, _ source.Window) ([]string, string, error) {
	return f.ids, f.cursor, f.err
}

type fakeSourceFactory struct{ inner source.AssetSource }

func (f fakeSourceFactory) Build(_ string, _ json.RawMessage) (source.AssetSource, error) {
	return f.inner, nil
}

type fakeBatches struct {
	mu        sync.Mutex
	created   []batchCall
	returnID  string
	returnErr error
}
type batchCall struct {
	templateID, name, targetID, owner string
	ids                               []string
}

func (f *fakeBatches) CreateBatch(_ context.Context, tmpl, name string, ids []string, target string, _ int, owner string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.created = append(f.created, batchCall{templateID: tmpl, name: name, targetID: target, owner: owner, ids: append([]string(nil), ids...)})
	if f.returnErr != nil {
		return "", f.returnErr
	}
	if f.returnID == "" {
		return "batch-1", nil
	}
	return f.returnID, nil
}

type fakeNotifier struct {
	mu      sync.Mutex
	fails   []Rule
	stucks  []Rule
	empties []Rule
}

func (n *fakeNotifier) NotifyRuleFailure(_ context.Context, r Rule, _ error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.fails = append(n.fails, r)
}
func (n *fakeNotifier) NotifyRuleStuck(_ context.Context, r Rule, _ time.Duration) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.stucks = append(n.stucks, r)
}
func (n *fakeNotifier) NotifyRuleEmpty(_ context.Context, r Rule) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.empties = append(n.empties, r)
}

func newUC(repo *fakeRepo, src source.AssetSource, batches *fakeBatches, notify Notifier, now time.Time) *Usecase {
	uc := New(repo, fakeSourceFactory{inner: src}, batches, notify, Options{})
	uc.now = func() time.Time { return now }
	return uc
}

func triggerJSON(t *testing.T, tc TriggerConfig) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(tc)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// ─── tests ────────────────────────────────────────────────────

func TestExecuteRule_IncrementalCreatesBatchAndAdvancesCursor(t *testing.T) {
	now := time.Date(2026, 7, 21, 6, 0, 0, 0, time.UTC)
	rule := Rule{
		ID: "r1", Name: "grace-inc",
		TemplateID: "tpl-1", TargetID: "target-1",
		SourceType: "rest", SourceConfig: json.RawMessage(`{}`),
		TriggerMode:   TriggerIncremental,
		TriggerConfig: triggerJSON(t, TriggerConfig{IntervalSeconds: 3600, InitialLookbackSeconds: 7200}),
		Cursor:        "2026-07-21T05:00:00Z",
	}
	repo := &fakeRepo{}
	src := fakeSource{ids: []string{"a", "b", "c"}, cursor: "2026-07-21T05:59:00Z"}
	batches := &fakeBatches{returnID: "batch-xyz"}
	notify := &fakeNotifier{}
	uc := newUC(repo, src, batches, notify, now)

	if err := uc.executeRule(context.Background(), rule); err != nil {
		t.Fatalf("executeRule: %v", err)
	}
	if len(batches.created) != 1 || len(batches.created[0].ids) != 3 || batches.created[0].templateID != "tpl-1" {
		t.Fatalf("batch call wrong: %+v", batches.created)
	}
	if len(repo.successes) != 1 || repo.successes[0].cursor != "2026-07-21T05:59:00Z" || repo.successes[0].batchID != "batch-xyz" || repo.successes[0].status != RunStatusSucceeded {
		t.Fatalf("success record wrong: %+v", repo.successes)
	}
	if len(repo.failures) != 0 || len(notify.fails) != 0 {
		t.Fatalf("unexpected failure/notify: %+v %+v", repo.failures, notify.fails)
	}
}

func TestExecuteRule_EmptyFetchRecordsEmptyAndNotifiesOnlyIfOptedIn(t *testing.T) {
	now := time.Now().UTC()
	baseRule := Rule{
		ID: "r-empty", Name: "empty",
		TemplateID: "tpl", TargetID: "target",
		SourceType: "rest", SourceConfig: json.RawMessage(`{}`),
		TriggerMode: TriggerIncremental,
	}
	// Without opt-in: no empty notification.
	{
		rule := baseRule
		rule.TriggerConfig = triggerJSON(t, TriggerConfig{IntervalSeconds: 3600})
		repo := &fakeRepo{}
		notify := &fakeNotifier{}
		uc := newUC(repo, fakeSource{ids: nil}, &fakeBatches{}, notify, now)
		if err := uc.executeRule(context.Background(), rule); err != nil {
			t.Fatal(err)
		}
		if len(repo.successes) != 1 || repo.successes[0].status != RunStatusEmpty {
			t.Fatalf("expected one empty success, got %+v", repo.successes)
		}
		if len(notify.empties) != 0 {
			t.Fatalf("did not opt in to empty notify but got %d", len(notify.empties))
		}
	}
	// With opt-in: empty notification fires.
	{
		rule := baseRule
		rule.TriggerConfig = triggerJSON(t, TriggerConfig{IntervalSeconds: 3600, NotifyOnEmpty: true})
		repo := &fakeRepo{}
		notify := &fakeNotifier{}
		uc := newUC(repo, fakeSource{ids: nil}, &fakeBatches{}, notify, now)
		if err := uc.executeRule(context.Background(), rule); err != nil {
			t.Fatal(err)
		}
		if len(notify.empties) != 1 {
			t.Fatalf("expected empty notify, got %d", len(notify.empties))
		}
	}
}

func TestExecuteRule_FetchErrorRecordsFailureAndNotifies_NoCursorAdvance(t *testing.T) {
	now := time.Now().UTC()
	rule := Rule{
		ID: "r-fail", Name: "boom",
		TemplateID: "tpl", TargetID: "target",
		SourceType: "rest", SourceConfig: json.RawMessage(`{}`),
		TriggerMode:   TriggerIncremental,
		TriggerConfig: triggerJSON(t, TriggerConfig{IntervalSeconds: 3600, InitialLookbackSeconds: 60}),
		Cursor:        "2026-07-21T05:00:00Z",
	}
	repo := &fakeRepo{}
	notify := &fakeNotifier{}
	batches := &fakeBatches{}
	uc := newUC(repo, fakeSource{err: errors.New("boom")}, batches, notify, now)
	err := uc.executeRule(context.Background(), rule)
	if err == nil {
		t.Fatal("expected error")
	}
	if len(repo.failures) != 1 || len(repo.successes) != 0 {
		t.Fatalf("wrong records: fail=%+v success=%+v", repo.failures, repo.successes)
	}
	if len(batches.created) != 0 {
		t.Fatal("must not create batch on source error")
	}
	if len(notify.fails) != 1 {
		t.Fatalf("expected 1 failure notify, got %d", len(notify.fails))
	}
}

func TestExecuteRule_BatchCreateErrorRecordsFailure_NoCursorAdvance(t *testing.T) {
	now := time.Now().UTC()
	rule := Rule{
		ID: "r-bfail", Name: "batch-fail",
		TemplateID: "tpl", TargetID: "target",
		SourceType: "rest", SourceConfig: json.RawMessage(`{}`),
		TriggerMode:   TriggerIncremental,
		TriggerConfig: triggerJSON(t, TriggerConfig{IntervalSeconds: 3600}),
	}
	repo := &fakeRepo{}
	notify := &fakeNotifier{}
	batches := &fakeBatches{returnErr: fmt.Errorf("nope")}
	uc := newUC(repo, fakeSource{ids: []string{"a"}, cursor: "later"}, batches, notify, now)
	if err := uc.executeRule(context.Background(), rule); err == nil {
		t.Fatal("expected error")
	}
	if len(repo.successes) != 0 || len(repo.failures) != 1 {
		t.Fatalf("wrong records after batch error: %+v %+v", repo.successes, repo.failures)
	}
	if len(notify.fails) != 1 {
		t.Fatalf("expected failure notify, got %d", len(notify.fails))
	}
}

func TestResolveWindow_AllModes(t *testing.T) {
	now := time.Date(2026, 7, 21, 6, 0, 0, 0, time.UTC)

	// incremental with cursor
	{
		rule := Rule{TriggerMode: TriggerIncremental, Cursor: "2026-07-21T05:00:00Z"}
		w, err := resolveWindow(rule, TriggerConfig{IntervalSeconds: 3600}, now)
		if err != nil {
			t.Fatal(err)
		}
		if w.Start == nil || !w.Start.Equal(time.Date(2026, 7, 21, 5, 0, 0, 0, time.UTC)) {
			t.Fatalf("incremental start wrong: %+v", w.Start)
		}
		if w.End == nil || !w.End.Equal(now) {
			t.Fatalf("incremental end wrong")
		}
	}
	// incremental first run uses InitialLookbackSeconds
	{
		rule := Rule{TriggerMode: TriggerIncremental}
		w, err := resolveWindow(rule, TriggerConfig{InitialLookbackSeconds: 300}, now)
		if err != nil {
			t.Fatal(err)
		}
		want := now.Add(-300 * time.Second)
		if !w.Start.Equal(want) {
			t.Fatalf("first-run start = %v want %v", w.Start, want)
		}
	}
	// rolling
	{
		rule := Rule{TriggerMode: TriggerRolling}
		w, err := resolveWindow(rule, TriggerConfig{LookbackSeconds: 1800}, now)
		if err != nil {
			t.Fatal(err)
		}
		want := now.Add(-1800 * time.Second)
		if !w.Start.Equal(want) || !w.End.Equal(now) {
			t.Fatalf("rolling window wrong: %+v..%+v", w.Start, w.End)
		}
	}
	// range
	{
		from := now.Add(-24 * time.Hour)
		to := now
		rule := Rule{TriggerMode: TriggerRange}
		w, err := resolveWindow(rule, TriggerConfig{From: &from, To: &to}, now)
		if err != nil {
			t.Fatal(err)
		}
		if !w.Start.Equal(from) || !w.End.Equal(to) {
			t.Fatalf("range window wrong: %+v..%+v", w.Start, w.End)
		}
	}
	// range missing from/to
	{
		rule := Rule{TriggerMode: TriggerRange}
		if _, err := resolveWindow(rule, TriggerConfig{}, now); err == nil {
			t.Fatal("expected error on range without from/to")
		}
	}
	// ids
	{
		rule := Rule{TriggerMode: TriggerIDs}
		w, err := resolveWindow(rule, TriggerConfig{IDs: []string{"a", "b"}}, now)
		if err != nil {
			t.Fatal(err)
		}
		if len(w.IDs) != 2 || w.IDs[0] != "a" {
			t.Fatalf("ids window wrong: %+v", w.IDs)
		}
	}
	// unknown mode
	{
		rule := Rule{TriggerMode: "wat"}
		if _, err := resolveWindow(rule, TriggerConfig{}, now); err == nil {
			t.Fatal("expected error on unknown mode")
		}
	}
}

// TestExecuteRule_OneShotDisablesAfterSuccess covers the gemini-review point:
// range and ids rules must auto-disable after a successful run so a scheduled
// cycle doesn't keep re-running them with the same window / same ids.
func TestExecuteRule_OneShotDisablesAfterSuccess(t *testing.T) {
	now := time.Now().UTC()
	for _, mode := range []TriggerMode{TriggerRange, TriggerIDs} {
		t.Run(string(mode), func(t *testing.T) {
			from, to := now.Add(-time.Hour), now
			tc := TriggerConfig{}
			if mode == TriggerRange {
				tc.From, tc.To = &from, &to
			} else {
				tc.IDs = []string{"x", "y"}
			}
			rule := Rule{
				ID: "r-oneshot-" + string(mode), Name: string(mode),
				TemplateID: "tpl", TargetID: "target",
				SourceType: "rest", SourceConfig: json.RawMessage(`{}`),
				TriggerMode:   mode,
				TriggerConfig: triggerJSON(t, tc),
			}
			repo := &fakeRepo{}
			batches := &fakeBatches{returnID: "b1"}
			uc := newUC(repo, fakeSource{ids: []string{"x", "y"}}, batches, &fakeNotifier{}, now)
			if err := uc.executeRule(context.Background(), rule); err != nil {
				t.Fatal(err)
			}
			// One-shot rule should have been disabled.
			if len(repo.setEnabled) != 1 || repo.setEnabled[0].on {
				t.Fatalf("expected one setEnabled(false) for %s, got %+v", mode, repo.setEnabled)
			}
		})
	}
	// Incremental / rolling MUST NOT be disabled after a successful run.
	for _, mode := range []TriggerMode{TriggerIncremental, TriggerRolling} {
		t.Run(string(mode), func(t *testing.T) {
			rule := Rule{
				ID: "r-recurring-" + string(mode), Name: string(mode),
				TemplateID: "tpl", TargetID: "target",
				SourceType: "rest", SourceConfig: json.RawMessage(`{}`),
				TriggerMode:   mode,
				TriggerConfig: triggerJSON(t, TriggerConfig{IntervalSeconds: 3600, InitialLookbackSeconds: 60, LookbackSeconds: 60}),
			}
			repo := &fakeRepo{}
			batches := &fakeBatches{returnID: "b1"}
			uc := newUC(repo, fakeSource{ids: []string{"a"}}, batches, &fakeNotifier{}, now)
			if err := uc.executeRule(context.Background(), rule); err != nil {
				t.Fatal(err)
			}
			if len(repo.setEnabled) != 0 {
				t.Fatalf("recurring mode %s must not be disabled, got %+v", mode, repo.setEnabled)
			}
		})
	}
}

func TestExecuteRule_IDsModeShortCircuits(t *testing.T) {
	now := time.Now().UTC()
	rule := Rule{
		ID: "r-ids", Name: "ids-rule",
		TemplateID: "tpl", TargetID: "target",
		SourceType: "rest", SourceConfig: json.RawMessage(`{}`),
		TriggerMode:   TriggerIDs,
		TriggerConfig: triggerJSON(t, TriggerConfig{IDs: []string{"x", "y"}}),
	}
	repo := &fakeRepo{}
	notify := &fakeNotifier{}
	batches := &fakeBatches{returnID: "batch-ids"}
	// The source will still be called (with window.IDs) — our restSource short-circuits;
	// the fake source just returns the ids we pre-programmed to prove the wiring.
	uc := newUC(repo, fakeSource{ids: []string{"x", "y"}}, batches, notify, now)
	if err := uc.executeRule(context.Background(), rule); err != nil {
		t.Fatal(err)
	}
	if len(batches.created) != 1 || len(batches.created[0].ids) != 2 {
		t.Fatalf("expected 1 batch with 2 ids, got %+v", batches.created)
	}
}

func TestRunOnce_ProcessesClaimedRules(t *testing.T) {
	now := time.Date(2026, 7, 21, 6, 0, 0, 0, time.UTC)
	repo := &fakeRepo{claimReturns: []Rule{
		{
			ID: "r1", Name: "n1",
			TemplateID: "tpl", TargetID: "target",
			SourceType: "rest", SourceConfig: json.RawMessage(`{}`),
			TriggerMode:   TriggerIncremental,
			TriggerConfig: triggerJSON(t, TriggerConfig{IntervalSeconds: 3600, InitialLookbackSeconds: 60}),
		},
	}}
	uc := newUC(repo, fakeSource{ids: []string{"a"}, cursor: "c1"}, &fakeBatches{}, &fakeNotifier{}, now)
	uc.runOnce(context.Background())
	if len(repo.successes) != 1 {
		t.Fatalf("expected 1 success, got %+v", repo.successes)
	}
}
