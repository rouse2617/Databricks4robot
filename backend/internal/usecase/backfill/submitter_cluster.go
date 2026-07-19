package backfill

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/CyberOrigin2077/cyber-databrew/internal/metrics"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
)

// CYB-3678 — per-cluster dispatch channels.
//
// The submitter shards each cycle by target CLUSTER: one goroutine + one
// governor + one advisory lock per cluster, so a slow or unreachable cluster
// only stalls its own channel (no head-of-line blocking across clusters) and
// dispatch economy scales per cluster, not globally.

// submitOutcome classifies one candidate's dispatch attempt for the governor
// and the channel breaker.
type submitOutcome int

const (
	outcomeSkip      submitOutcome = iota // not pending anymore / job paused
	outcomeOK                             // submitted (incl. AlreadyExists heal)
	outcomeTransient                      // infra trouble — retried next cycle
	outcomePermanent                      // deterministic failure — item failed
)

// maxSubmitAttempts caps transient retries per item (CYB-3678 DLQ): a poison
// item stops burning cycles after this many attempts and lands in the DLQ
// (status=failed, reason recorded) for explicit human retry.
var maxSubmitAttempts = func() int {
	if v := os.Getenv("BACKFILL_MAX_SUBMIT_ATTEMPTS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 5
}()

// clusterSubmitRate is the per-cluster token bucket (creates/s). Sized to the
// Argo controller's CONSUMPTION rate (CNOE baseline 4.5–13.5 wf/s), not the
// API server's acceptance rate — submitting faster only piles up the
// controller workqueue. Calibrate per cluster via load test, tune online in
// CYB-3679.
var clusterSubmitRate = func() float64 {
	if v := os.Getenv("BACKFILL_CLUSTER_RATE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			return f
		}
	}
	return 10
}()

// consecutiveTransientBreaker trips a channel after this many transient
// failures in a row within one cycle: the cluster is unhealthy — stop burning
// tokens on it and let the next tick retry (items stay pending by design).
const consecutiveTransientBreaker = 10

// clusterGovernor bounds one cluster's dispatch: a token bucket caps the
// sustained CRD write rate, and an AIMD loop adapts the in-flight concurrency
// to observed submit health (halve on distress with a floor of configured/4,
// grow ×1.5 when healthy — review round 2/3 hardening).
type clusterGovernor struct {
	limiter *rate.Limiter

	mu         sync.Mutex
	configured int
	effective  int
	floor      int
	// batchLimit overrides perJobSubmitBatch when >0 (CYB-3679 online tuning).
	batchLimit int

	// per-cycle window
	slow      int // submits slower than slowSubmitThreshold
	total     int
	throttled bool // saw a throttle/timeout-class error this cycle
}

const slowSubmitThreshold = time.Second

func newClusterGovernor(configured int) *clusterGovernor {
	floor := configured / 4
	if floor < 1 {
		floor = 1
	}
	return &clusterGovernor{
		limiter:    rate.NewLimiter(rate.Limit(clusterSubmitRate), int(clusterSubmitRate)*2),
		configured: configured,
		effective:  configured,
		floor:      floor,
	}
}

// applyConfig retunes the governor online (CYB-3679): new concurrency ceiling
// (effective clamps into [floor, configured]), token-bucket rate, and per-job
// submit batch. Idempotent — applying the same config is a no-op.
func (g *clusterGovernor) applyConfig(maxConcurrency int, ratePerSec float64, submitBatch int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if maxConcurrency > 0 && maxConcurrency != g.configured {
		g.configured = maxConcurrency
		g.floor = maxConcurrency / 4
		if g.floor < 1 {
			g.floor = 1
		}
		if g.effective > g.configured {
			g.effective = g.configured
		}
		if g.effective < g.floor {
			g.effective = g.floor
		}
	}
	if ratePerSec > 0 && rate.Limit(ratePerSec) != g.limiter.Limit() {
		g.limiter.SetLimit(rate.Limit(ratePerSec))
		g.limiter.SetBurst(int(ratePerSec) * 2)
	}
	g.batchLimit = submitBatch
}

// submitBatchLimit returns the per-job batch override (0 = compiled default).
func (g *clusterGovernor) submitBatchLimit() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.batchLimit
}

// slots returns the current in-flight concurrency bound.
func (g *clusterGovernor) slots() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.effective
}

// record feeds one submit observation into the current cycle window.
func (g *clusterGovernor) record(d time.Duration, outcome submitOutcome) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.total++
	if d > slowSubmitThreshold {
		g.slow++
	}
	if outcome == outcomeTransient {
		g.throttled = true
	}
}

// endCycle applies AIMD from the window: distress (any transient, or >half
// the submits slow) halves effective down to the floor; health grows it ×1.5
// up to the configured ceiling. Never touches the token bucket — the bucket
// is the hard rate roof regardless of concurrency.
func (g *clusterGovernor) endCycle() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.total == 0 {
		return
	}
	distress := g.throttled || g.slow*2 > g.total
	if distress {
		g.effective /= 2
		if g.effective < g.floor {
			g.effective = g.floor
		}
	} else {
		g.effective = g.effective * 3 / 2
		if g.effective > g.configured {
			g.effective = g.configured
		}
	}
	g.slow, g.total, g.throttled = 0, 0, false
}

// governorFor returns (lazily creating) the cluster's governor.
func (uc *Usecase) governorFor(cluster string) *clusterGovernor {
	uc.governorMu.Lock()
	defer uc.governorMu.Unlock()
	if uc.governors == nil {
		uc.governors = map[string]*clusterGovernor{}
	}
	g, ok := uc.governors[cluster]
	if !ok {
		g = newClusterGovernor(maxConcurrentBatchItems)
		uc.governors[cluster] = g
	}
	return g
}

// classifySubmitError decides retry semantics (CYB-3678, review P1-1 v2):
// explicitly-permanent errors fail the item immediately; EVERYTHING ELSE is
// transient and retried with the attempt cap — misclassifying a transient
// blip as permanent mass-fails innocent items, so unknown defaults to
// transient.
func classifySubmitError(err error) submitOutcome {
	if err == nil {
		return outcomeOK
	}
	if errors.Is(err, pipelineUC.ErrInvalidArgument) ||
		errors.Is(err, pipelineUC.ErrTemplateNotFound) ||
		errors.Is(err, pipelineUC.ErrAssetNotFound) {
		return outcomePermanent
	}
	msg := strings.ToLower(err.Error())
	for _, marker := range []string{"transpile:", "parse pipeline", "asset_ids is required", "not supported", "不支持"} {
		if strings.Contains(msg, marker) {
			return outcomePermanent
		}
	}
	return outcomeTransient
}

// clusterCycleLocker is the per-cluster cross-instance mutual exclusion
// surface (CYB-3678, supersedes the CYB-3677 global cycle lock). Queues
// without it run unguarded — correctness never depends on the lock.
type clusterCycleLocker interface {
	WithSubmitterClusterLock(ctx context.Context, cluster string, fn func(context.Context) error) (bool, error)
}

// runClusterChannel dispatches one cluster's jobs under its own advisory
// lock, governor, and breaker. Returns true when any job filled its whole
// batch (backlog remains → the caller self-kicks instead of waiting a tick).
func (uc *Usecase) runClusterChannel(ctx context.Context, cluster string, jobs []*models.BackfillJob) bool {
	refill := false
	// Online tuning (CYB-3679): paused wins over everything — an operator
	// pause must stop dispatch within one tick. Non-paused config retunes the
	// governor in place (AIMD keeps working around the new ceiling).
	if cfg, ok := uc.dispatcherConfigFor(cluster); ok {
		if cfg.Paused {
			metrics.DispatcherChannelPaused.WithLabelValues(cluster).Set(1)
			slog.Info("submitter: channel paused by dispatcher config, skipping", "cluster", cluster)
			return false
		}
		uc.governorFor(cluster).applyConfig(cfg.MaxConcurrency, cfg.RatePerSec, cfg.SubmitBatch)
	}
	metrics.DispatcherChannelPaused.WithLabelValues(cluster).Set(0)
	body := func(ctx context.Context) error {
		gov := uc.governorFor(cluster)
		consecutiveTransient := 0
		for _, job := range jobs {
			attempted, outcomes := uc.submitJobBatch(ctx, job, gov)
			for _, o := range outcomes {
				if o == outcomeTransient {
					consecutiveTransient++
				} else if o == outcomeOK || o == outcomePermanent {
					consecutiveTransient = 0
				}
			}
			if attempted > 0 {
				_ = uc.syncJobProgress(ctx, job.ID)
			}
			if attempted >= perJobSubmitBatch {
				refill = true
			}
			if consecutiveTransient >= consecutiveTransientBreaker {
				// The cluster is unhealthy: stop burning tokens on it this
				// cycle. Items stay pending by design (a down cluster comes
				// back; per-item errors are what the DLQ cap is for).
				metrics.DispatcherChannelBreakerTotal.WithLabelValues(cluster).Inc()
				slog.Warn("submitter: channel breaker tripped, skipping rest of cycle",
					"cluster", cluster, "consecutiveTransient", consecutiveTransient)
				refill = false
				break
			}
		}
		gov.endCycle()
		metrics.DispatcherEffectiveConcurrency.WithLabelValues(cluster).Set(float64(gov.slots()))
		return nil
	}

	locker, ok := uc.submitQueue.(clusterCycleLocker)
	if !ok {
		_ = body(ctx)
		return refill
	}
	acquired, err := locker.WithSubmitterClusterLock(ctx, cluster, body)
	if err != nil {
		slog.Warn("submitter: cluster lock unavailable, running unguarded", "cluster", cluster, "err", err)
		_ = body(ctx)
		return refill
	}
	if !acquired {
		slog.Debug("submitter: cluster channel held by another instance, skipping", "cluster", cluster)
	}
	return refill
}
