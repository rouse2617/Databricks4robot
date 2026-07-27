package subtask

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type Repo interface {
	ClaimDueTasks(ctx context.Context, now time.Time) ([]Task, error)
	RecordSuccess(ctx context.Context, id string, status string, batchIDs []string, at time.Time) error
	RecordFailure(ctx context.Context, id string, errMsg string) error
	SetEnabled(ctx context.Context, id string, enabled bool) error
}

type BatchCreator interface {
	CreateBatch(ctx context.Context, templateID, name string, assetIDs []string, targetID string, templateVersion int, owner string) (batchID string, err error)
}

// RunCreator dispatches a single asset as a first-class pipeline run (as opposed
// to a batch). Used when a message carries exactly one asset.
type RunCreator interface {
	CreateRun(ctx context.Context, templateID, name, assetID, targetID string, templateVersion int, owner string) (runID string, err error)
}

type Notifier interface {
	NotifyTaskFailure(ctx context.Context, task Task, err error)
	NotifyTaskEmpty(ctx context.Context, task Task)
}

// Puller is the message-pull seam of PubSubClient, narrowed to what the
// dispatch loop needs. *PubSubClient satisfies it; extracting the interface
// lets executeTask be tested without a live Pub/Sub subscriber.
type Puller interface {
	Pull(ctx context.Context, projectID, subscriptionID string, maxMessages int) (*PullResult, error)
}

type Options struct {
	DefaultPullIntervalSec int
}

type Usecase struct {
	repo    Repo
	puller  Puller
	batches BatchCreator
	runs    RunCreator
	notify  Notifier
	opt     Options
	now     func() time.Time
}

func New(repo Repo, ps Puller, batches BatchCreator, runs RunCreator, notify Notifier, opt Options) *Usecase {
	if opt.DefaultPullIntervalSec <= 0 {
		opt.DefaultPullIntervalSec = 10
	}
	return &Usecase{repo: repo, puller: ps, batches: batches, runs: runs, notify: notify, opt: opt, now: time.Now}
}

func (uc *Usecase) StartLoop(ctx context.Context) {
	go func() {
		run := func() {
			c, cancel := context.WithTimeout(ctx, 5*time.Minute)
			defer cancel()
			uc.runOnce(c)
		}
		run()
		t := time.NewTicker(time.Duration(uc.opt.DefaultPullIntervalSec) * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				run()
			}
		}
	}()
}

func (uc *Usecase) runOnce(ctx context.Context) {
	tasks, err := uc.repo.ClaimDueTasks(ctx, uc.now())
	if err != nil {
		slog.Warn("subtask: claim due tasks failed", "err", err)
		return
	}
	for i := range tasks {
		task := tasks[i]
		if err := uc.executeTask(ctx, task); err != nil {
			_ = err
		}
	}
}

func (uc *Usecase) executeTask(ctx context.Context, task Task) error {
	slog.Info("subtask: executing", "taskID", task.ID, "name", task.Name, "sub", task.SubscriptionID)

	maxMsg := task.MaxMessagesPerPull
	if maxMsg <= 0 {
		maxMsg = 1000
	}

	result, err := uc.puller.Pull(ctx, task.ProjectID, task.SubscriptionID, maxMsg)
	if err != nil {
		return uc.recordAndNotifyFailure(ctx, task, fmt.Errorf("pubsub pull: %w", err))
	}

	if len(result.Messages) == 0 {
		if err := uc.repo.RecordSuccess(ctx, task.ID, RunStatusEmpty, nil, uc.now()); err != nil {
			slog.Warn("subtask: record empty run failed", "taskID", task.ID, "err", err)
		}
		return nil
	}

	if len(task.PipelineBindings) == 0 {
		result.Nack()
		return uc.recordAndNotifyFailure(ctx, task, fmt.Errorf("no pipeline bindings configured"))
	}

	// Dispatch each message independently: one asset → a single pipeline run,
	// more than one → a batch. Each binding is dispatched once per message
	// (fan-out). All dispatches must succeed before we Ack — any failure Nacks
	// the whole pull so it retries (at-least-once; a retry may re-dispatch units
	// that already succeeded).
	owner := "subscription-task:" + task.ID
	ts := uc.now().UTC().Format("20060102-150405")
	dispatchedIDs := make([]string, 0, len(result.Messages)*len(task.PipelineBindings))
	runCount, batchCount := 0, 0
	for mi, assetIDs := range result.Messages {
		for bi, b := range task.PipelineBindings {
			tmplVersion := 0
			if b.TemplateVersion != nil {
				tmplVersion = *b.TemplateVersion
			}
			name := fmt.Sprintf("%s-%s-m%d-p%d", task.Name, ts, mi+1, bi+1)
			var (
				id   string
				derr error
			)
			if len(assetIDs) == 1 {
				id, derr = uc.runs.CreateRun(ctx, b.TemplateID, name, assetIDs[0], b.TargetID, tmplVersion, owner)
				if derr == nil {
					runCount++
				}
			} else {
				id, derr = uc.batches.CreateBatch(ctx, b.TemplateID, name, assetIDs, b.TargetID, tmplVersion, owner)
				if derr == nil {
					batchCount++
				}
			}
			if derr != nil {
				result.Nack()
				return uc.recordAndNotifyFailure(ctx, task, fmt.Errorf("dispatch template %s: %w", b.TemplateID, derr))
			}
			dispatchedIDs = append(dispatchedIDs, id)
		}
	}

	result.Ack()

	if err := uc.repo.RecordSuccess(ctx, task.ID, RunStatusSucceeded, dispatchedIDs, uc.now()); err != nil {
		slog.Warn("subtask: record success failed", "taskID", task.ID, "err", err)
	}
	slog.Info("subtask: dispatched", "taskID", task.ID, "runs", runCount, "batches", batchCount, "messages", len(result.Messages), "bindings", len(task.PipelineBindings))
	return nil
}

func (uc *Usecase) recordAndNotifyFailure(ctx context.Context, task Task, err error) error {
	slog.Warn("subtask: task failed", "taskID", task.ID, "name", task.Name, "err", err)
	if rerr := uc.repo.RecordFailure(ctx, task.ID, err.Error()); rerr != nil {
		slog.Warn("subtask: record failure persist failed", "taskID", task.ID, "err", rerr)
	}
	if uc.notify != nil {
		uc.notify.NotifyTaskFailure(ctx, task, err)
	}
	return err
}
