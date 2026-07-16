package argo

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// ErrClusterNotFound is returned by ClientFactory when the given clusterID
// does not exist in the clusters table (or has been soft-deleted).
var ErrClusterNotFound = errors.New("cluster not found")

// ClientFactory abstracts "which Argo Workflow client for this cluster" so
// consumers (Deploy usecase, run_watcher, log streamer) can call ForTarget
// instead of hard-coding a global argo.Client singleton. Same Karmada-ready
// seam as k8s.ClientFactory (CYB-3486).
type ClientFactory interface {
	// ForCluster returns a WorkflowClient for the given clusterID.
	ForCluster(ctx context.Context, clusterID string) (WorkflowClient, error)
	// ForTarget resolves the target's cluster_id to a WorkflowClient. Empty
	// cluster_id is treated as the default cluster for pre-CYB-3425 rows.
	ForTarget(ctx context.Context, t *models.ExecutionTarget) (WorkflowClient, error)
	// Invalidate drops the cached client so the next lookup rebuilds from
	// fresh DB row (called after admin cluster CRUD).
	Invalidate(clusterID string)
}

// cachedArgoEntry is one cluster's cached Argo client. Argo clients are
// currently plain HTTP clients (no long-lived token source), so caching is a
// small optimization; TTL exists mainly to pick up admin edits to
// ArgoServerURL / CA cert without a restart.
type cachedArgoEntry struct {
	client    WorkflowClient
	fetchedAt time.Time
}

// dbClientFactory builds argo.Client per cluster row, with TTL cache.
type dbClientFactory struct {
	repo    repository.ClusterRepository
	envCfg  *Config // fallback for the default cluster (empty ArgoServerURL)
	ttl     time.Duration
	mu      sync.RWMutex
	entries map[string]*cachedArgoEntry
}

// FactoryOption customizes dbClientFactory behavior.
type FactoryOption func(*dbClientFactory)

// WithTTL overrides the default 60s cache TTL.
func WithTTL(ttl time.Duration) FactoryOption {
	return func(f *dbClientFactory) { f.ttl = ttl }
}

// WithEnvFallback provides the Config that gets used when a cluster row has
// empty ArgoServerURL (default-cluster compat path). Pass ConfigFromEnv() to
// wire the existing env-based startup config as fallback.
func WithEnvFallback(cfg *Config) FactoryOption {
	return func(f *dbClientFactory) { f.envCfg = cfg }
}

// NewClientFactory wires a ClientFactory backed by the given cluster repo.
func NewClientFactory(repo repository.ClusterRepository, opts ...FactoryOption) ClientFactory {
	f := &dbClientFactory{
		repo:    repo,
		ttl:     60 * time.Second,
		entries: map[string]*cachedArgoEntry{},
	}
	for _, o := range opts {
		o(f)
	}
	return f
}

func (f *dbClientFactory) ForCluster(ctx context.Context, clusterID string) (WorkflowClient, error) {
	// Fast path: cache hit within TTL.
	f.mu.RLock()
	if e, ok := f.entries[clusterID]; ok && time.Since(e.fetchedAt) < f.ttl {
		f.mu.RUnlock()
		return e.client, nil
	}
	f.mu.RUnlock()

	cluster, err := f.repo.Get(ctx, clusterID)
	if err != nil {
		return nil, fmt.Errorf("argo.ClientFactory: read cluster %q: %w", clusterID, err)
	}
	if cluster == nil {
		return nil, fmt.Errorf("%w: %s", ErrClusterNotFound, clusterID)
	}

	cfg := f.configForCluster(cluster)
	client := NewClientFromConfig(cfg)

	e := &cachedArgoEntry{client: client, fetchedAt: time.Now()}
	f.mu.Lock()
	f.entries[clusterID] = e
	f.mu.Unlock()
	return client, nil
}

func (f *dbClientFactory) ForTarget(ctx context.Context, t *models.ExecutionTarget) (WorkflowClient, error) {
	if t == nil {
		return nil, errors.New("argo.ClientFactory: nil target")
	}
	id := t.ClusterID
	if id == "" {
		id = "cluster-default"
	}
	return f.ForCluster(ctx, id)
}

func (f *dbClientFactory) Invalidate(clusterID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.entries, clusterID)
}

// configForCluster resolves the Argo Config to use for the cluster. When
// ArgoServerURL is empty the factory falls back to the env-derived Config
// captured at startup (default-cluster compat). Non-empty URL overrides;
// token / TLS come from the cluster row when populated, otherwise env.
func (f *dbClientFactory) configForCluster(c *models.Cluster) *Config {
	if c.ArgoServerURL == "" {
		// Default cluster path: reuse the env config passed via
		// WithEnvFallback so backend behavior is byte-identical to the
		// pre-3486 singleton startup.
		if f.envCfg != nil {
			return f.envCfg
		}
		return &Config{}
	}
	// Non-default cluster: use its ArgoServerURL; token/TLS still come from
	// env because per-cluster secret storage is intentionally out of scope
	// for PR 4a (defense against key sprawl; add per-cluster fields when
	// there's a real need).
	cfg := &Config{ServerURL: c.ArgoServerURL}
	if f.envCfg != nil {
		cfg.Token = f.envCfg.Token
		cfg.InsecureSkipVerify = f.envCfg.InsecureSkipVerify
		cfg.CACertBase64 = f.envCfg.CACertBase64
	}
	return cfg
}
