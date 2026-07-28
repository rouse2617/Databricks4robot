package subtask

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

// These tests pin executeTask's dispatch contract (CYB-3801): each pulled
// message is dispatched independently — one asset → a single pipeline run,
// more than one → a batch — fanned out once per pipeline binding. The whole
// pull is Acked only if every dispatch succeeds; any failure Nacks it (so it
// redelivers) and records + notifies the failure. See CYB-4262.

// fixedNow pins uc.now so dispatched names are deterministic.
var fixedNow = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

// fixedTS is fixedNow rendered by executeTask's name template.
const fixedTS = "20260102-030405"

// ---- fakes ----

type recordedRun struct {
	templateID, name, assetID, targetID, owner string
	version                                    int
}

type recordedBatch struct {
	templateID, name, targetID, owner string
	assetIDs                          []string
	version                           int
}

// fakePuller stands in for *PubSubClient via the Puller seam, returning a
// canned message set and counting the pull-level Ack/Nack the caller invokes.
type fakePuller struct {
	msgs               [][]string
	err                error
	acks, nacks        int
	gotProject, gotSub string
	gotMax             int
}

func (f *fakePuller) Pull(_ context.Context, projectID, subscriptionID string, maxMessages int) (*PullResult, error) {
	f.gotProject, f.gotSub, f.gotMax = projectID, subscriptionID, maxMessages
	if f.err != nil {
		return nil, f.err
	}
	return &PullResult{
		Messages: f.msgs,
		Ack:      func() { f.acks++ },
		Nack:     func() { f.nacks++ },
	}, nil
}

type fakeRunCreator struct {
	calls []recordedRun
	err   error
}

func (f *fakeRunCreator) CreateRun(_ context.Context, templateID, name, assetID, targetID string, version int, owner string) (string, error) {
	f.calls = append(f.calls, recordedRun{templateID, name, assetID, targetID, owner, version})
	if f.err != nil {
		return "", f.err
	}
	return "run-" + assetID, nil
}

type fakeBatchCreator struct {
	calls []recordedBatch
	err   error
}

func (f *fakeBatchCreator) CreateBatch(_ context.Context, templateID, name string, assetIDs []string, targetID string, version int, owner string) (string, error) {
	f.calls = append(f.calls, recordedBatch{templateID, name, targetID, owner, assetIDs, version})
	if f.err != nil {
		return "", f.err
	}
	return "batch-" + name, nil
}

type recordedSuccess struct {
	id       string
	status   string
	batchIDs []string
}

type recordedFailure struct{ id, msg string }

type fakeRepo struct {
	successes []recordedSuccess
	failures  []recordedFailure
}

func (f *fakeRepo) ClaimDueTasks(context.Context, time.Time) ([]Task, error) { return nil, nil }
func (f *fakeRepo) RecordSuccess(_ context.Context, id, status string, batchIDs []string, _ time.Time) error {
	f.successes = append(f.successes, recordedSuccess{id, status, batchIDs})
	return nil
}
func (f *fakeRepo) RecordFailure(_ context.Context, id, errMsg string) error {
	f.failures = append(f.failures, recordedFailure{id, errMsg})
	return nil
}
func (f *fakeRepo) SetEnabled(context.Context, string, bool) error { return nil }

type fakeNotifier struct{ failures, empties int }

func (f *fakeNotifier) NotifyTaskFailure(context.Context, Task, error) { f.failures++ }
func (f *fakeNotifier) NotifyTaskEmpty(context.Context, Task)          { f.empties++ }

// ---- harness ----

type fakes struct {
	puller  *fakePuller
	runs    *fakeRunCreator
	batches *fakeBatchCreator
	repo    *fakeRepo
	notify  *fakeNotifier
}

func newFakes(msgs [][]string) *fakes {
	return &fakes{
		puller:  &fakePuller{msgs: msgs},
		runs:    &fakeRunCreator{},
		batches: &fakeBatchCreator{},
		repo:    &fakeRepo{},
		notify:  &fakeNotifier{},
	}
}

func (f *fakes) usecase() *Usecase {
	uc := New(f.repo, f.puller, f.batches, f.runs, f.notify, Options{})
	uc.now = func() time.Time { return fixedNow }
	return uc
}

func ptr(i int) *int { return &i }

func binding(tpl, target string, version *int) PipelineBinding {
	return PipelineBinding{TemplateID: tpl, TargetID: target, TemplateVersion: version}
}

func task(name string, bindings ...PipelineBinding) Task {
	return Task{
		ID:               "task1",
		Name:             name,
		ProjectID:        "proj",
		SubscriptionID:   "sub",
		PipelineBindings: bindings,
	}
}

// ---- tests ----

func TestExecuteTask_SingleAssetDispatchesRun(t *testing.T) {
	f := newFakes([][]string{{"a"}})
	if err := f.usecase().executeTask(context.Background(), task("t", binding("tpl-x", "g", nil))); err != nil {
		t.Fatalf("executeTask: %v", err)
	}
	if len(f.batches.calls) != 0 {
		t.Errorf("expected no batch for a single asset, got %d", len(f.batches.calls))
	}
	if len(f.runs.calls) != 1 {
		t.Fatalf("expected 1 run, got %d", len(f.runs.calls))
	}
	want := recordedRun{
		templateID: "tpl-x", name: "t-" + fixedTS + "-m1-p1", assetID: "a",
		targetID: "g", owner: "subscription-task:task1", version: 0,
	}
	if f.runs.calls[0] != want {
		t.Errorf("run call = %+v, want %+v", f.runs.calls[0], want)
	}
	if f.puller.acks != 1 || f.puller.nacks != 0 {
		t.Errorf("acks=%d nacks=%d, want 1/0", f.puller.acks, f.puller.nacks)
	}
	if len(f.repo.successes) != 1 || f.repo.successes[0].status != RunStatusSucceeded {
		t.Fatalf("successes=%+v, want one %q", f.repo.successes, RunStatusSucceeded)
	}
	if want := []string{"run-a"}; !reflect.DeepEqual(f.repo.successes[0].batchIDs, want) {
		t.Errorf("dispatchedIDs=%v, want %v", f.repo.successes[0].batchIDs, want)
	}
	if f.notify.failures != 0 {
		t.Errorf("unexpected failure notification")
	}
}

func TestExecuteTask_MultiAssetDispatchesBatch(t *testing.T) {
	f := newFakes([][]string{{"a", "b"}})
	if err := f.usecase().executeTask(context.Background(), task("t", binding("tpl-x", "g", nil))); err != nil {
		t.Fatalf("executeTask: %v", err)
	}
	if len(f.runs.calls) != 0 {
		t.Errorf("expected no run for a multi asset, got %d", len(f.runs.calls))
	}
	if len(f.batches.calls) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(f.batches.calls))
	}
	want := recordedBatch{
		templateID: "tpl-x", name: "t-" + fixedTS + "-m1-p1", targetID: "g",
		owner: "subscription-task:task1", assetIDs: []string{"a", "b"}, version: 0,
	}
	if !reflect.DeepEqual(f.batches.calls[0], want) {
		t.Errorf("batch call = %+v, want %+v", f.batches.calls[0], want)
	}
	if f.puller.acks != 1 {
		t.Errorf("acks=%d, want 1", f.puller.acks)
	}
	if f.repo.successes[0].status != RunStatusSucceeded {
		t.Errorf("status=%s, want %s", f.repo.successes[0].status, RunStatusSucceeded)
	}
}

func TestExecuteTask_FanOutPerBindingWithVersion(t *testing.T) {
	f := newFakes([][]string{{"a"}})
	tk := task("t", binding("tpl-x", "g", nil), binding("tpl-y", "h", ptr(2)))
	if err := f.usecase().executeTask(context.Background(), tk); err != nil {
		t.Fatalf("executeTask: %v", err)
	}
	if len(f.runs.calls) != 2 {
		t.Fatalf("expected 2 runs (one per binding), got %d", len(f.runs.calls))
	}
	if f.runs.calls[0].name != "t-"+fixedTS+"-m1-p1" || f.runs.calls[1].name != "t-"+fixedTS+"-m1-p2" {
		t.Errorf("names = %q, %q; want ...-m1-p1, ...-m1-p2", f.runs.calls[0].name, f.runs.calls[1].name)
	}
	if f.runs.calls[0].version != 0 || f.runs.calls[1].version != 2 {
		t.Errorf("versions = %d, %d; want 0, 2 (nil binding → 0, *2 → 2)", f.runs.calls[0].version, f.runs.calls[1].version)
	}
	if f.runs.calls[1].templateID != "tpl-y" || f.runs.calls[1].targetID != "h" {
		t.Errorf("binding-2 dispatch mismatch: %+v", f.runs.calls[1])
	}
	if f.puller.acks != 1 {
		t.Errorf("acks=%d, want 1", f.puller.acks)
	}
}

func TestExecuteTask_MixedMessages(t *testing.T) {
	// Two messages, one single-asset and one multi-asset, one binding:
	// message boundaries are preserved, so it is 1 run + 1 batch.
	f := newFakes([][]string{{"a"}, {"b", "c"}})
	if err := f.usecase().executeTask(context.Background(), task("t", binding("tpl-x", "g", nil))); err != nil {
		t.Fatalf("executeTask: %v", err)
	}
	if len(f.runs.calls) != 1 || f.runs.calls[0].name != "t-"+fixedTS+"-m1-p1" {
		t.Errorf("runs = %+v, want one named ...-m1-p1", f.runs.calls)
	}
	if len(f.batches.calls) != 1 || f.batches.calls[0].name != "t-"+fixedTS+"-m2-p1" {
		t.Errorf("batches = %+v, want one named ...-m2-p1", f.batches.calls)
	}
	want := []string{"run-a", "batch-t-" + fixedTS + "-m2-p1"}
	if !reflect.DeepEqual(f.repo.successes[0].batchIDs, want) {
		t.Errorf("dispatchedIDs=%v, want %v", f.repo.successes[0].batchIDs, want)
	}
}

func TestExecuteTask_EmptyPullRecordsEmpty(t *testing.T) {
	f := newFakes(nil) // no messages pulled
	if err := f.usecase().executeTask(context.Background(), task("t", binding("tpl-x", "g", nil))); err != nil {
		t.Fatalf("executeTask: %v", err)
	}
	if len(f.runs.calls) != 0 || len(f.batches.calls) != 0 {
		t.Errorf("expected no dispatch on an empty pull")
	}
	if f.puller.acks != 0 || f.puller.nacks != 0 {
		t.Errorf("empty pull must neither ack nor nack; acks=%d nacks=%d", f.puller.acks, f.puller.nacks)
	}
	if len(f.repo.successes) != 1 || f.repo.successes[0].status != RunStatusEmpty {
		t.Errorf("successes=%+v, want one %q", f.repo.successes, RunStatusEmpty)
	}
}

func TestExecuteTask_NoBindingsNacksAndFails(t *testing.T) {
	f := newFakes([][]string{{"a"}})
	err := f.usecase().executeTask(context.Background(), task("t")) // no bindings
	if err == nil {
		t.Fatal("expected an error when no pipeline bindings are configured")
	}
	if f.puller.nacks != 1 {
		t.Errorf("nacks=%d, want 1 (redeliver)", f.puller.nacks)
	}
	if len(f.runs.calls) != 0 || len(f.batches.calls) != 0 {
		t.Errorf("expected no dispatch")
	}
	if len(f.repo.failures) != 1 || !strings.Contains(f.repo.failures[0].msg, "no pipeline bindings") {
		t.Fatalf("failures=%+v, want one mentioning bindings", f.repo.failures)
	}
	if f.notify.failures != 1 {
		t.Errorf("notify failures=%d, want 1", f.notify.failures)
	}
	if len(f.repo.successes) != 0 {
		t.Errorf("unexpected success recorded")
	}
}

func TestExecuteTask_DispatchErrorNacksAndNotifies(t *testing.T) {
	sentinel := errors.New("boom")
	t.Run("run creator error", func(t *testing.T) {
		f := newFakes([][]string{{"a"}})
		f.runs.err = sentinel
		err := f.usecase().executeTask(context.Background(), task("t", binding("tpl-x", "g", nil)))
		if err == nil {
			t.Fatal("expected error")
		}
		if f.puller.nacks != 1 || f.puller.acks != 0 {
			t.Errorf("acks=%d nacks=%d, want 0/1", f.puller.acks, f.puller.nacks)
		}
		if f.notify.failures != 1 {
			t.Errorf("notify failures=%d, want 1", f.notify.failures)
		}
		if len(f.repo.failures) != 1 || !strings.Contains(f.repo.failures[0].msg, "dispatch template tpl-x") {
			t.Errorf("failures=%+v, want one mentioning the template", f.repo.failures)
		}
		if len(f.repo.successes) != 0 {
			t.Errorf("must not record success when a dispatch failed")
		}
	})
	t.Run("batch creator error", func(t *testing.T) {
		f := newFakes([][]string{{"a", "b"}})
		f.batches.err = sentinel
		err := f.usecase().executeTask(context.Background(), task("t", binding("tpl-x", "g", nil)))
		if err == nil {
			t.Fatal("expected error")
		}
		if f.puller.nacks != 1 {
			t.Errorf("nacks=%d, want 1", f.puller.nacks)
		}
		if f.notify.failures != 1 {
			t.Errorf("notify failures=%d, want 1", f.notify.failures)
		}
	})
}

func TestExecuteTask_PullErrorRecordsFailure(t *testing.T) {
	f := newFakes(nil)
	f.puller.err = errors.New("pull failed")
	err := f.usecase().executeTask(context.Background(), task("t", binding("tpl-x", "g", nil)))
	if err == nil {
		t.Fatal("expected error")
	}
	if len(f.repo.failures) != 1 || !strings.Contains(f.repo.failures[0].msg, "pubsub pull") {
		t.Errorf("failures=%+v, want one mentioning the pull", f.repo.failures)
	}
	if f.notify.failures != 1 {
		t.Errorf("notify failures=%d, want 1", f.notify.failures)
	}
	if len(f.runs.calls) != 0 || len(f.batches.calls) != 0 {
		t.Errorf("expected no dispatch when the pull itself failed")
	}
}

func TestExecuteTask_PullArgsAndMaxMessages(t *testing.T) {
	t.Run("explicit max is passed through", func(t *testing.T) {
		f := newFakes([][]string{{"a"}})
		tk := task("t", binding("tpl-x", "g", nil))
		tk.MaxMessagesPerPull = 5
		_ = f.usecase().executeTask(context.Background(), tk)
		if f.puller.gotProject != "proj" || f.puller.gotSub != "sub" {
			t.Errorf("pull addressed %q/%q, want proj/sub", f.puller.gotProject, f.puller.gotSub)
		}
		if f.puller.gotMax != 5 {
			t.Errorf("gotMax=%d, want 5", f.puller.gotMax)
		}
	})
	t.Run("unset max defaults to 1000", func(t *testing.T) {
		f := newFakes([][]string{{"a"}})
		tk := task("t", binding("tpl-x", "g", nil))
		tk.MaxMessagesPerPull = 0
		_ = f.usecase().executeTask(context.Background(), tk)
		if f.puller.gotMax != 1000 {
			t.Errorf("gotMax=%d, want 1000 (default)", f.puller.gotMax)
		}
	})
}
