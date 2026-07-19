package backfill

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// CYB-3679 — online dispatcher tuning.
//
// One dispatcher_configs row per cluster; the submitter re-reads all rows at
// the start of every cycle, so a PUT takes effect within one tick (≤15s)
// without a deploy. Absent row = compiled defaults. Config falls back to the
// last-known snapshot when the DB read fails — a transient DB blip must not
// silently reset a paused cluster to running.

// dispatcherConfigStore is the persistence surface (postgres.DispatcherConfigRepo).
type dispatcherConfigStore interface {
	List(ctx context.Context) ([]models.DispatcherConfig, error)
	Upsert(ctx context.Context, cfg *models.DispatcherConfig) error
	Delete(ctx context.Context, clusterID string) error
}

// Validation ranges — a fat-fingered PUT must not be able to stampede a
// cluster (rate 10000/s) or brick it (concurrency 0).
const (
	dispatcherMaxConcurrencyMin = 1
	dispatcherMaxConcurrencyMax = 256
	dispatcherSubmitBatchMin    = 1
	dispatcherSubmitBatchMax    = 200
	dispatcherRateMin           = 0.1
	dispatcherRateMax           = 100.0
)

// dispatcherConfigState is the cycle-refreshed config snapshot.
type dispatcherConfigState struct {
	mu   sync.RWMutex
	byID map[string]models.DispatcherConfig
}

// SetDispatcherConfigRepo wires optional online tuning (nil = defaults only).
func (uc *Usecase) SetDispatcherConfigRepo(r dispatcherConfigStore) {
	uc.dispatcherCfgRepo = r
}

// refreshDispatcherConfigs reloads all rows into the snapshot. On error the
// previous snapshot stays — see the fail-safe note above.
func (uc *Usecase) refreshDispatcherConfigs(ctx context.Context) {
	if uc.dispatcherCfgRepo == nil {
		return
	}
	rows, err := uc.dispatcherCfgRepo.List(ctx)
	if err != nil {
		slog.Warn("dispatcher config: refresh failed, keeping last snapshot", "err", err)
		return
	}
	byID := make(map[string]models.DispatcherConfig, len(rows))
	for _, r := range rows {
		byID[r.ClusterID] = r
	}
	uc.dispatcherCfgState.mu.Lock()
	uc.dispatcherCfgState.byID = byID
	uc.dispatcherCfgState.mu.Unlock()
}

// dispatcherConfigFor returns the cluster's explicit config, if any.
func (uc *Usecase) dispatcherConfigFor(cluster string) (models.DispatcherConfig, bool) {
	uc.dispatcherCfgState.mu.RLock()
	defer uc.dispatcherCfgState.mu.RUnlock()
	cfg, ok := uc.dispatcherCfgState.byID[cluster]
	return cfg, ok
}

// defaultDispatcherConfig materializes the compiled defaults for a cluster.
func defaultDispatcherConfig(cluster string) models.DispatcherConfig {
	return models.DispatcherConfig{
		ClusterID:      cluster,
		MaxConcurrency: maxConcurrentBatchItems,
		SubmitBatch:    perJobSubmitBatch,
		RatePerSec:     clusterSubmitRate,
	}
}

// validateDispatcherConfig enforces the tuning ranges.
func validateDispatcherConfig(cfg *models.DispatcherConfig) error {
	if cfg == nil || cfg.ClusterID == "" {
		return fmt.Errorf("%w: cluster_id is required", ErrInvalidDispatcherConfig)
	}
	if cfg.MaxConcurrency < dispatcherMaxConcurrencyMin || cfg.MaxConcurrency > dispatcherMaxConcurrencyMax {
		return fmt.Errorf("%w: max_concurrency must be in [%d,%d]", ErrInvalidDispatcherConfig, dispatcherMaxConcurrencyMin, dispatcherMaxConcurrencyMax)
	}
	if cfg.SubmitBatch < dispatcherSubmitBatchMin || cfg.SubmitBatch > dispatcherSubmitBatchMax {
		return fmt.Errorf("%w: submit_batch must be in [%d,%d]", ErrInvalidDispatcherConfig, dispatcherSubmitBatchMin, dispatcherSubmitBatchMax)
	}
	if cfg.RatePerSec < dispatcherRateMin || cfg.RatePerSec > dispatcherRateMax {
		return fmt.Errorf("%w: rate_per_sec must be in [%g,%g]", ErrInvalidDispatcherConfig, dispatcherRateMin, dispatcherRateMax)
	}
	return nil
}

// ErrInvalidDispatcherConfig marks range-validation failures (handler → 400).
var ErrInvalidDispatcherConfig = fmt.Errorf("invalid dispatcher config")

// SaveDispatcherConfig validates and persists one cluster's tuning, then
// refreshes the snapshot and kicks a cycle so the change bites immediately.
func (uc *Usecase) SaveDispatcherConfig(ctx context.Context, cfg *models.DispatcherConfig) error {
	if uc.dispatcherCfgRepo == nil {
		return fmt.Errorf("dispatcher config repository is not configured")
	}
	if err := validateDispatcherConfig(cfg); err != nil {
		return err
	}
	if err := uc.dispatcherCfgRepo.Upsert(ctx, cfg); err != nil {
		return err
	}
	uc.refreshDispatcherConfigs(ctx)
	uc.KickSubmitter()
	return nil
}

// DeleteDispatcherConfig removes a cluster's row (back to defaults).
func (uc *Usecase) DeleteDispatcherConfig(ctx context.Context, cluster string) error {
	if uc.dispatcherCfgRepo == nil {
		return fmt.Errorf("dispatcher config repository is not configured")
	}
	if cluster == "" {
		return fmt.Errorf("%w: cluster_id is required", ErrInvalidDispatcherConfig)
	}
	if err := uc.dispatcherCfgRepo.Delete(ctx, cluster); err != nil {
		return err
	}
	uc.refreshDispatcherConfigs(ctx)
	uc.KickSubmitter()
	return nil
}

// DispatcherStatus merges configured rows with live governor state for the
// admin UI ("configured vs effective" + suppression reason).
func (uc *Usecase) DispatcherStatus(ctx context.Context) ([]models.DispatcherClusterStatus, error) {
	uc.refreshDispatcherConfigs(ctx)

	clusters := map[string]bool{}
	uc.dispatcherCfgState.mu.RLock()
	for id := range uc.dispatcherCfgState.byID {
		clusters[id] = true
	}
	uc.dispatcherCfgState.mu.RUnlock()
	uc.governorMu.Lock()
	for id := range uc.governors {
		clusters[id] = true
	}
	uc.governorMu.Unlock()
	if len(clusters) == 0 {
		clusters["default"] = true
	}

	ids := make([]string, 0, len(clusters))
	for id := range clusters {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	out := make([]models.DispatcherClusterStatus, 0, len(ids))
	for _, id := range ids {
		cfg, hasRow := uc.dispatcherConfigFor(id)
		if !hasRow {
			cfg = defaultDispatcherConfig(id)
		}
		st := models.DispatcherClusterStatus{
			ClusterID:            id,
			Config:               cfg,
			HasRow:               hasRow,
			EffectiveConcurrency: cfg.MaxConcurrency,
		}
		uc.governorMu.Lock()
		g := uc.governors[id]
		uc.governorMu.Unlock()
		if g != nil {
			st.EffectiveConcurrency = g.slots()
		}
		switch {
		case cfg.Paused:
			st.SuppressionReason = "paused"
		case st.EffectiveConcurrency < cfg.MaxConcurrency:
			st.SuppressionReason = "aimd_backoff"
		}
		out = append(out, st)
	}
	return out, nil
}
