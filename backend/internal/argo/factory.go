package argo

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// K8sFactory is the subset of k8s.ClientFactory the argo package uses to reach
// the CRD path. Declared here (rather than importing internal/k8s directly) to
// keep the argo → k8s edge out of the DAG — k8s already imports usecase/pipeline
// which imports back into argo. Callers pass their k8s.ClientFactory unchanged;
// it structurally satisfies this interface.
type K8sFactory interface {
	ForCluster(ctx context.Context, clusterID string) (kubernetes.Interface, error)
	DynamicForCluster(ctx context.Context, clusterID string) (dynamic.Interface, error)
}

// ErrClusterNotFound is returned by ClientFactory when the given clusterID
// does not exist in the clusters table (or has been soft-deleted).
var ErrClusterNotFound = errors.New("cluster not found")

// ErrClusterMisconfigured is returned when a cluster row can't be resolved to
// any concrete WorkflowClient — e.g. no argo-server URL and no k8s.ClientFactory
// wired for the CRD path. Distinct from ErrClusterNotFound (the row exists).
var ErrClusterMisconfigured = errors.New("cluster misconfigured for argo client")

// defaultArgoNamespace is the ns crdWorkflowClient falls back to when the
// cluster row leaves argo_namespace blank. Argo's own charts default here.
const defaultArgoNamespace = "argo"

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
	repo       repository.ClusterRepository
	envCfg     *Config    // fallback URL/token for the default cluster (empty ArgoServerURL)
	k8sFactory K8sFactory // required for CRD mode (empty resolved URL); nil disables CRD mode
	ttl        time.Duration
	mu         sync.RWMutex
	entries    map[string]*cachedArgoEntry
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

// WithK8sFactory injects the per-cluster K8s client factory used by the CRD
// mode (crdWorkflowClient). Pass the same instance that Deploy usecase and
// handlers already share; without it, a cluster row that resolves to an empty
// argo-server URL will fail with ErrClusterMisconfigured instead of returning
// an unusable HTTP client with no server URL. See design doc D1/D7.
func WithK8sFactory(k8sFactory K8sFactory) FactoryOption {
	return func(f *dbClientFactory) { f.k8sFactory = k8sFactory }
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

	client, err := f.buildClient(ctx, cluster)
	if err != nil {
		return nil, err
	}

	e := &cachedArgoEntry{client: client, fetchedAt: time.Now()}
	f.mu.Lock()
	f.entries[clusterID] = e
	f.mu.Unlock()
	return client, nil
}

// buildClient picks between HTTP (argo-server) and CRD (dynamic client) mode.
//
// Mode is decided by the cluster's OWN argo_server_url plus the env fallback,
// where the env fallback applies ONLY to the default cluster. This is the
// crux of CYB-3486d1d: a non-default cluster (delivery-clust) with an empty
// argo_server_url deliberately means "use CRD mode", and must NOT inherit the
// env ARGO_SERVER_URL (which points at cyber-clust's argo-server). Before the
// fix, configForCluster's env fallback fired for every empty-URL cluster, so
// clearing delivery-clust's URL silently routed its workflows to cyber-clust's
// argo-server — producing "namespaces cyber-delivery-dev not found".
func (f *dbClientFactory) buildClient(ctx context.Context, cluster *models.Cluster) (WorkflowClient, error) {
	cfg := f.configForCluster(cluster)
	if cfg.ServerURL != "" {
		return NewClientFromConfig(cfg), nil
	}
	if f.k8sFactory == nil {
		return nil, fmt.Errorf("%w: cluster %q has no argo-server URL and no k8s factory wired",
			ErrClusterMisconfigured, cluster.Name)
	}
	dyn, err := f.k8sFactory.DynamicForCluster(ctx, cluster.ID)
	if err != nil {
		return nil, fmt.Errorf("argo.ClientFactory: dynamic client for %q: %w", cluster.Name, err)
	}
	typed, err := f.k8sFactory.ForCluster(ctx, cluster.ID)
	if err != nil {
		return nil, fmt.Errorf("argo.ClientFactory: typed client for %q: %w", cluster.Name, err)
	}
	ns := cluster.ArgoNamespace
	if ns == "" {
		ns = defaultArgoNamespace
	}
	return newCRDWorkflowClient(dyn, typed, ns), nil
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

// configForCluster resolves the Argo Config to use for the cluster.
//
// Empty ArgoServerURL is resolved differently by cluster kind:
//   - default cluster → fall back to the env-derived Config (byte-identical
//     to the pre-3486 singleton startup). This keeps cyber-clust on HTTP.
//   - non-default cluster → return an empty Config so buildClient routes to
//     CRD mode. Critically we do NOT leak the env's argo-server URL here, or
//     the cluster would wrongly talk HTTP to cyber-clust's argo-server
//     (CYB-3486d1d regression).
//
// Non-empty ArgoServerURL always means HTTP with that URL (token / TLS still
// come from env because per-cluster secret storage is out of scope for now).
func (f *dbClientFactory) configForCluster(c *models.Cluster) *Config {
	if c.ArgoServerURL == "" {
		if isDefaultCluster(c) && f.envCfg != nil {
			return f.envCfg
		}
		return &Config{}
	}
	cfg := &Config{ServerURL: c.ArgoServerURL}
	if f.envCfg != nil {
		cfg.Token = f.envCfg.Token
		cfg.InsecureSkipVerify = f.envCfg.InsecureSkipVerify
		cfg.CACertBase64 = f.envCfg.CACertBase64
	}
	return cfg
}

// isDefaultCluster reports whether the cluster is the legacy single-cluster
// row that inherits env-derived Argo/K8s config. Matches on the explicit
// IsDefault flag or the seed id, so either a correctly-flagged row or the
// pre-flag "cluster-default" seed is recognized.
func isDefaultCluster(c *models.Cluster) bool {
	return c.IsDefault || c.ID == "cluster-default"
}
