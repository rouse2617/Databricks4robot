package k8s

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// ErrClusterNotFound is returned by ClientFactory when the given clusterID
// does not exist in the clusters table (or has been soft-deleted).
var ErrClusterNotFound = errors.New("cluster not found")

// ErrClusterMisconfigured is returned when the cluster row lacks the fields
// needed to build a K8s config (e.g. non-default cluster with empty audience).
var ErrClusterMisconfigured = errors.New("cluster misconfigured")

// ClientFactory abstracts the "which K8s client for this cluster" decision so
// consumers can call ForCluster(ctx, target.ClusterID) instead of hard-coding
// a global singleton. This is the CYB-3486 Karmada-ready seam: the Stage 1
// implementation (dbClientFactory) reads config from the clusters table; a
// future Stage 2 implementation (karmadaClientFactory) can swap in without
// touching any consumer.
type ClientFactory interface {
	// ForCluster returns a typed clientset for the given clusterID.
	ForCluster(ctx context.Context, clusterID string) (kubernetes.Interface, error)
	// DynamicForCluster returns a dynamic client for CRD access (used by
	// ElasticQuota list, Koordinator scheduler.sigs.k8s.io resources, etc).
	DynamicForCluster(ctx context.Context, clusterID string) (dynamic.Interface, error)
	// ForTarget resolves the target's cluster_id and returns a clientset.
	// Empty cluster_id is treated as the default cluster (for pre-3425 rows
	// that predate the FK column).
	ForTarget(ctx context.Context, t *models.ExecutionTarget) (kubernetes.Interface, error)
	// Invalidate drops any cached client / config for the cluster so the next
	// ForCluster call rebuilds from fresh DB row. Called after admin CRUD.
	Invalidate(clusterID string)
}

// cachedEntry is a single cluster's cached client bundle. Kept together so
// the clientset, dynamic client, and their underlying rest.Config share the
// same lifetime (invalidation drops all three at once).
type cachedEntry struct {
	clientset kubernetes.Interface
	dynamic   dynamic.Interface
	fetchedAt time.Time
}

// dbClientFactory reads cluster config from a ClusterRepository, builds a
// rest.Config per cluster (WIF metadata token per cluster audience), and
// caches the resulting clientsets with a TTL. Cluster rows with empty
// K8sAPIEndpoint fall back to the process env vars (K8S_API_ENDPOINT etc)
// so the pre-3486 "default cluster" behavior stays intact.
type dbClientFactory struct {
	repo    repository.ClusterRepository
	ttl     time.Duration
	mu      sync.RWMutex
	entries map[string]*cachedEntry
}

// FactoryOption customizes dbClientFactory behavior.
type FactoryOption func(*dbClientFactory)

// WithTTL overrides the default 60s cache TTL. Shorter values increase DB
// load but pick up cluster config changes faster; TTL=0 disables the cache
// entirely (each call re-reads and rebuilds — useful for tests).
func WithTTL(ttl time.Duration) FactoryOption {
	return func(f *dbClientFactory) { f.ttl = ttl }
}

// NewClientFactory wires a ClientFactory backed by the given cluster repo.
func NewClientFactory(repo repository.ClusterRepository, opts ...FactoryOption) ClientFactory {
	f := &dbClientFactory{
		repo:    repo,
		ttl:     60 * time.Second,
		entries: map[string]*cachedEntry{},
	}
	for _, o := range opts {
		o(f)
	}
	return f
}

func (f *dbClientFactory) ForCluster(ctx context.Context, clusterID string) (kubernetes.Interface, error) {
	e, err := f.load(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	return e.clientset, nil
}

func (f *dbClientFactory) DynamicForCluster(ctx context.Context, clusterID string) (dynamic.Interface, error) {
	e, err := f.load(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	return e.dynamic, nil
}

func (f *dbClientFactory) ForTarget(ctx context.Context, t *models.ExecutionTarget) (kubernetes.Interface, error) {
	if t == nil {
		return nil, fmt.Errorf("%w: nil target", ErrClusterMisconfigured)
	}
	id := t.ClusterID
	if id == "" {
		// Pre-CYB-3486 targets without an explicit FK land on the default
		// cluster row (seed id "cluster-default"). Callers using new rows
		// created after PR #397 always carry an explicit clusterID.
		id = "cluster-default"
	}
	return f.ForCluster(ctx, id)
}

func (f *dbClientFactory) Invalidate(clusterID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.entries, clusterID)
}

// load returns a cached entry for the cluster, rebuilding from DB when the
// cache is missing or expired. Cache reads and writes are serialized on f.mu;
// the underlying clientset/dynamic types are goroutine-safe once built.
func (f *dbClientFactory) load(ctx context.Context, clusterID string) (*cachedEntry, error) {
	// Fast path: cache hit within TTL.
	f.mu.RLock()
	if e, ok := f.entries[clusterID]; ok && time.Since(e.fetchedAt) < f.ttl {
		f.mu.RUnlock()
		return e, nil
	}
	f.mu.RUnlock()

	cluster, err := f.repo.Get(ctx, clusterID)
	if err != nil {
		return nil, fmt.Errorf("k8s.ClientFactory: read cluster %q: %w", clusterID, err)
	}
	if cluster == nil {
		return nil, fmt.Errorf("%w: %s", ErrClusterNotFound, clusterID)
	}

	cfg, err := buildConfigFromCluster(cluster)
	if err != nil {
		return nil, err
	}

	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: cluster %q clientset: %v", ErrUnavailable, clusterID, err)
	}
	dc, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: cluster %q dynamic: %v", ErrUnavailable, clusterID, err)
	}

	e := &cachedEntry{clientset: cs, dynamic: dc, fetchedAt: time.Now()}

	// Write to cache. Another goroutine may have raced and inserted its own
	// entry; overwriting is safe because both were built from the same DB
	// row (or from env) and share the same auth path.
	f.mu.Lock()
	f.entries[clusterID] = e
	f.mu.Unlock()
	return e, nil
}

// buildConfigFromCluster resolves a rest.Config from a cluster DB row. Empty
// K8sAPIEndpoint means "use the process env vars" — this is how the default
// cluster keeps working without requiring admins to backfill env-derived
// fields into the DB row.
func buildConfigFromCluster(c *models.Cluster) (*rest.Config, error) {
	// Default-cluster compatibility path: empty API endpoint → env-derived.
	if c.K8sAPIEndpoint == "" {
		return buildConfig("")
	}
	// Non-default clusters must carry an audience for WIF metadata-token
	// auth. Bearer-token / kubeconfig-file paths are intentionally not
	// supported per-cluster right now (defense against key sprawl); WIF only.
	if c.K8sAudience == "" {
		return nil, fmt.Errorf("%w: cluster %q missing k8s_audience", ErrClusterMisconfigured, c.Name)
	}
	tls, err := buildTLSFromCluster(c)
	if err != nil {
		return nil, err
	}
	cached := transport.NewCachedTokenSource(newMetadataTokenSource(c.K8sAudience))
	return &rest.Config{
		Host:            c.K8sAPIEndpoint,
		TLSClientConfig: tls,
		WrapTransport:   newOauth2RoundTripper(cached),
	}, nil
}

// buildTLSFromCluster resolves the TLS config from the cluster row.
// K8sCaData is either PEM text or base64-encoded PEM; both are accepted.
// Empty CA data means "use system trust store" (rare for GKE but valid).
func buildTLSFromCluster(c *models.Cluster) (rest.TLSClientConfig, error) {
	if c.K8sCAData == "" {
		return rest.TLSClientConfig{}, nil
	}
	// Try base64 first (GKE describe emits base64); fall back to raw PEM.
	if decoded, err := base64DecodeIfBase64(c.K8sCAData); err == nil && decoded != nil {
		return rest.TLSClientConfig{CAData: decoded}, nil
	}
	return rest.TLSClientConfig{CAData: []byte(c.K8sCAData)}, nil
}
