package grace

import (
	"context"
	"log/slog"
	"time"
)

// durationRepo persists video durations (satisfied by postgres.VideoDurationRepo).
type durationRepo interface {
	Upsert(ctx context.Context, durations map[string]float64) error
}

// Syncer pulls durations from Grace and upserts them into the store. It is
// best-effort: Grace being slow or down never blocks callers (CYB-3072).
type Syncer struct {
	client *Client
	repo   durationRepo
}

// NewSyncer constructs a Syncer.
func NewSyncer(client *Client, repo durationRepo) *Syncer {
	return &Syncer{client: client, repo: repo}
}

// SyncAll fetches all Grace video durations and upserts them. Idempotent.
func (s *Syncer) SyncAll(ctx context.Context) error {
	durs, err := s.client.FetchVideoDurations(ctx)
	if err != nil {
		return err
	}
	if err := s.repo.Upsert(ctx, durs); err != nil {
		return err
	}
	slog.Info("video duration sync complete", "count", len(durs))
	return nil
}

// SyncAsync runs SyncAll in the background, swallowing errors (best-effort). Used
// as the post-batch-creation kick so it never blocks batch creation.
func (s *Syncer) SyncAsync() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if err := s.SyncAll(ctx); err != nil {
			slog.Warn("video duration async sync failed", "err", err)
		}
	}()
}

// StartSyncLoop runs SyncAll immediately and then every interval until ctx is
// done. Errors are logged and swallowed so a Grace outage cannot crash or stall
// the server. interval <= 0 falls back to 10 minutes.
func (s *Syncer) StartSyncLoop(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 10 * time.Minute
	}
	go func() {
		run := func() {
			c, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()
			if err := s.SyncAll(c); err != nil {
				slog.Warn("video duration sync failed", "err", err)
			}
		}
		run()
		t := time.NewTicker(interval)
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
