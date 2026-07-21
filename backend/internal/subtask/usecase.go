package subtask

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type Repo interface {
	ClaimDueTasks(ctx context.Context, now time.Time) ([]Task, error)
	RecordSuccess(ctx context.Context, id string, status string, batchID string, at time.Time) error
	RecordFailure(ctx context.Context, id string, errMsg string) error
	SetEnabled(ctx context.Context, id string, enabled bool) error
}

type BatchCreator interface {
	CreateBatch(ctx context.Context, templateID, name string, assetIDs []string, targetID string, templateVersion int, owner string) (batchID string, err error)
}

type Notifier interface {
	NotifyTaskFailure(ctx context.Context, task Task, err error)
	NotifyTaskEmpty(ctx context.Context, task Task)
}

type Options struct {
	DefaultPullIntervalSec int
}

type Usecase struct {
	repo    Repo
	pubsub  *PubSubClient
	batches BatchCreator
	notify  Notifier
	opt     Options
	now     func() time.Time
}

func New(repo Repo, ps *PubSubClient, batches BatchCreator, notify Notifier, opt Options) *Usecase {
	if opt.DefaultPullIntervalSec <= 0 {
		opt.DefaultPullIntervalSec = 10
	}
	return &Usecase{repo: repo, pubsub: ps, batches: batches, notify: notify, opt: opt, now: time.Now}
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

	result, err := uc.pubsub.Pull(ctx, task.ProjectID, task.SubscriptionID, maxMsg)
	if err != nil {
		return uc.recordAndNotifyFailure(ctx, task, fmt.Errorf("pubsub pull: %w", err))
	}

	if len(result.IDs) == 0 {
		if err := uc.repo.RecordSuccess(ctx, task.ID, RunStatusEmpty, "", uc.now()); err != nil {
			slog.Warn("subtask: record empty run failed", "taskID", task.ID, "err", err)
		}
		return nil
	}

	tmplVersion := 0
	if task.TemplateVersion != nil {
		tmplVersion = *task.TemplateVersion
	}
	batchName := task.Name + "-" + uc.now().UTC().Format("20060102-150405")
	batchID, err := uc.batches.CreateBatch(ctx, task.TemplateID, batchName, result.IDs, task.TargetID, tmplVersion, "subscription-task:"+task.ID)
	if err != nil {
		result.Nack()
		return uc.recordAndNotifyFailure(ctx, task, fmt.Errorf("create batch: %w", err))
	}

	result.Ack()

	if err := uc.repo.RecordSuccess(ctx, task.ID, RunStatusSucceeded, batchID, uc.now()); err != nil {
		slog.Warn("subtask: record success failed", "taskID", task.ID, "err", err)
	}
	slog.Info("subtask: batch created", "taskID", task.ID, "batchID", batchID, "assetCount", len(result.IDs))
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
