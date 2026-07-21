package schedtask

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/schedtask/source"
)

// ─── injected dependencies ────────────────────────────────────

// Repo is the persistence surface the usecase needs. The concrete implementation
// is postgres.ScheduledTaskRepo.
type Repo interface {
	ClaimDueRules(ctx context.Context, now time.Time, defaultIntervalSeconds int) ([]Rule, error)
	RecordSuccess(ctx context.Context, id string, status string, cursor string, batchID string, at time.Time) error
	RecordFailure(ctx context.Context, id string, errMsg string) error
}

// BatchCreator abstracts pipeline.CreateBatchJob so the scheduler package
// doesn't depend on the pipeline package (avoids import cycles with future
// growth). The concrete implementation just calls
// pipelineUC.CreateBatchJob(ctx, ...).
type BatchCreator interface {
	CreateBatch(ctx context.Context, templateID, name string, assetIDs []string, targetID string, templateVersion int, owner string) (batchID string, err error)
}

// SourceFactory constructs an AssetSource from a rule's source_type +
// source_config. Injected so we can add more types later without editing this
// package's core loop.
type SourceFactory interface {
	Build(sourceType string, sourceConfig json.RawMessage) (source.AssetSource, error)
}

// Notifier is the 飞书 sender. We keep it a tiny interface so v1 can wire a
// copy-of-grace-sync sender and later swap for a shared package with zero
// churn here.
type Notifier interface {
	NotifyRuleFailure(ctx context.Context, rule Rule, err error)
	NotifyRuleStuck(ctx context.Context, rule Rule, since time.Duration)
	NotifyRuleEmpty(ctx context.Context, rule Rule) // called only when rule opts in
}

// ─── usecase ──────────────────────────────────────────────────

// Options configures the scheduler loop. Zero values are meaningful defaults.
type Options struct {
	// TickInterval is how often the loop wakes to look for due rules. This is
	// NOT the per-rule frequency (which comes from trigger_config); it's just
	// the granularity at which "is anything due?" is checked. 30s is plenty
	// for hour-scale rule intervals.
	TickInterval time.Duration
	// DefaultIntervalSeconds fills in for rules that don't set their own
	// intervalSeconds (mostly a safety net; the UI should always set one).
	DefaultIntervalSeconds int
	// StuckThresholdSeconds triggers a "rule hasn't succeeded in a while"
	// alert. 0 disables. Per-rule override in TriggerConfig.StuckThresholdSeconds.
	StuckThresholdSeconds int
}

// Usecase runs the scheduler loop and executes rules.
type Usecase struct {
	repo    Repo
	sources SourceFactory
	batches BatchCreator
	notify  Notifier
	opt     Options
	now     func() time.Time
}

func New(repo Repo, sources SourceFactory, batches BatchCreator, notify Notifier, opt Options) *Usecase {
	if opt.TickInterval <= 0 {
		opt.TickInterval = 30 * time.Second
	}
	if opt.DefaultIntervalSeconds <= 0 {
		opt.DefaultIntervalSeconds = 3600
	}
	return &Usecase{repo: repo, sources: sources, batches: batches, notify: notify, opt: opt, now: time.Now}
}

// StartLoop mirrors grace.Syncer.StartSyncLoop: run once immediately then on
// each tick, until ctx is done. Errors are logged and swallowed so a bad rule
// can't crash the server.
func (uc *Usecase) StartLoop(ctx context.Context) {
	go func() {
		run := func() {
			c, cancel := context.WithTimeout(ctx, 5*time.Minute)
			defer cancel()
			uc.runOnce(c)
		}
		run()
		t := time.NewTicker(uc.opt.TickInterval)
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

// runOnce claims all due rules and executes each. Called by the ticker AND
// available for tests / manual invocation.
func (uc *Usecase) runOnce(ctx context.Context) {
	rules, err := uc.repo.ClaimDueRules(ctx, uc.now(), uc.opt.DefaultIntervalSeconds)
	if err != nil {
		slog.Warn("schedtask: claim due rules failed", "err", err)
		return
	}
	for i := range rules {
		rule := rules[i]
		// Executed sequentially — the LIMIT 64 on claim + rule cadences make
		// this cheap in practice, and it means "one rule blowing up" doesn't
		// take a burst of a batch-create with it.
		if err := uc.executeRule(ctx, rule); err != nil {
			// Errors are logged and reported to Feishu inside executeRule;
			// don't re-log here.
			_ = err
		}
	}
}

// executeRule resolves the window, fetches ids from the rule's source, and
// creates a batch. Returns nil on success (including "empty fetch"); returns
// the error on failure (already persisted + notified).
func (uc *Usecase) executeRule(ctx context.Context, rule Rule) error {
	slog.Info("schedtask: executing rule", "ruleID", rule.ID, "name", rule.Name, "mode", rule.TriggerMode)

	tc, err := decodeTriggerConfig(rule.TriggerConfig)
	if err != nil {
		return uc.recordAndNotifyFailure(ctx, rule, fmt.Errorf("decode trigger_config: %w", err))
	}

	win, err := resolveWindow(rule, tc, uc.now())
	if err != nil {
		return uc.recordAndNotifyFailure(ctx, rule, fmt.Errorf("resolve window: %w", err))
	}

	src, err := uc.sources.Build(rule.SourceType, rule.SourceConfig)
	if err != nil {
		return uc.recordAndNotifyFailure(ctx, rule, fmt.Errorf("build source: %w", err))
	}

	ids, nextCursor, err := src.Fetch(ctx, rule.Cursor, win)
	if err != nil {
		return uc.recordAndNotifyFailure(ctx, rule, fmt.Errorf("source fetch: %w", err))
	}

	if len(ids) == 0 {
		if err := uc.repo.RecordSuccess(ctx, rule.ID, RunStatusEmpty, nextCursor, "", uc.now()); err != nil {
			slog.Warn("schedtask: record empty run failed", "ruleID", rule.ID, "err", err)
		}
		if tc.NotifyOnEmpty && uc.notify != nil {
			uc.notify.NotifyRuleEmpty(ctx, rule)
		}
		// Range/ids modes are one-shot; disable after a successful (even
		// empty) run so they don't linger with run_now_requested_at cleared
		// and no interval to gate them. incremental/rolling stay enabled.
		return nil
	}

	tmplVersion := 0
	if rule.TemplateVersion != nil {
		tmplVersion = *rule.TemplateVersion
	}
	batchName := rule.Name + "-" + uc.now().UTC().Format("20060102-150405")
	batchID, err := uc.batches.CreateBatch(ctx, rule.TemplateID, batchName, ids, rule.TargetID, tmplVersion, "scheduled-task:"+rule.ID)
	if err != nil {
		return uc.recordAndNotifyFailure(ctx, rule, fmt.Errorf("create batch: %w", err))
	}

	if err := uc.repo.RecordSuccess(ctx, rule.ID, RunStatusSucceeded, nextCursor, batchID, uc.now()); err != nil {
		slog.Warn("schedtask: record success failed", "ruleID", rule.ID, "err", err)
	}
	slog.Info("schedtask: batch created", "ruleID", rule.ID, "batchID", batchID, "assetCount", len(ids))
	return nil
}

func (uc *Usecase) recordAndNotifyFailure(ctx context.Context, rule Rule, err error) error {
	slog.Warn("schedtask: rule failed", "ruleID", rule.ID, "name", rule.Name, "err", err)
	if rerr := uc.repo.RecordFailure(ctx, rule.ID, err.Error()); rerr != nil {
		slog.Warn("schedtask: record failure persist failed", "ruleID", rule.ID, "err", rerr)
	}
	if uc.notify != nil {
		uc.notify.NotifyRuleFailure(ctx, rule, err)
	}
	return err
}

// ─── window resolution ────────────────────────────────────────

func resolveWindow(rule Rule, tc TriggerConfig, now time.Time) (source.Window, error) {
	switch rule.TriggerMode {
	case TriggerIncremental:
		end := now
		var start time.Time
		if rule.Cursor != "" {
			t, err := parseCursor(rule.Cursor)
			if err != nil {
				// A bad cursor is recoverable: treat as first-run.
				slog.Warn("schedtask: bad cursor, treating as first run", "ruleID", rule.ID, "cursor", rule.Cursor, "err", err)
				start = firstRunStart(tc, now)
			} else {
				start = t
			}
		} else {
			start = firstRunStart(tc, now)
		}
		return source.Window{Start: &start, End: &end}, nil
	case TriggerRolling:
		lb := tc.LookbackSeconds
		if lb <= 0 {
			lb = 3600
		}
		start := now.Add(-time.Duration(lb) * time.Second)
		end := now
		return source.Window{Start: &start, End: &end}, nil
	case TriggerRange:
		if tc.From == nil || tc.To == nil {
			return source.Window{}, errors.New("range mode requires from and to")
		}
		return source.Window{Start: tc.From, End: tc.To}, nil
	case TriggerIDs:
		if len(tc.IDs) == 0 {
			return source.Window{}, errors.New("ids mode requires ids")
		}
		return source.Window{IDs: tc.IDs}, nil
	default:
		return source.Window{}, fmt.Errorf("unknown trigger_mode %q", rule.TriggerMode)
	}
}

func firstRunStart(tc TriggerConfig, now time.Time) time.Time {
	lb := tc.InitialLookbackSeconds
	if lb <= 0 {
		lb = tc.LookbackSeconds
	}
	if lb <= 0 {
		lb = 3600
	}
	return now.Add(-time.Duration(lb) * time.Second)
}

// parseCursor accepts RFC3339 for now (the only cursor shape restSource
// emits in v1); numeric-string cursors would need a rule-level format hint.
func parseCursor(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("empty cursor")
	}
	return time.Parse(time.RFC3339, s)
}

func decodeTriggerConfig(raw json.RawMessage) (TriggerConfig, error) {
	if len(raw) == 0 {
		return TriggerConfig{}, nil
	}
	var tc TriggerConfig
	if err := json.Unmarshal(raw, &tc); err != nil {
		return TriggerConfig{}, err
	}
	return tc, nil
}
