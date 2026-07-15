package k8s

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// stubClusterRepo is a minimal in-memory ClusterRepository for factory tests.
// We don't need CRUD here — just deterministic Get() responses so we can
// exercise cache hits/misses, fallback paths, and invalidation.
type stubClusterRepo struct {
	mu       sync.Mutex
	byID     map[string]*models.Cluster
	getCalls int
	getErr   error
}

func newStubRepo(clusters ...*models.Cluster) *stubClusterRepo {
	r := &stubClusterRepo{byID: map[string]*models.Cluster{}}
	for _, c := range clusters {
		cp := *c
		r.byID[c.ID] = &cp
	}
	return r
}

func (r *stubClusterRepo) List(context.Context, bool) ([]*models.Cluster, error) {
	return nil, errors.New("List not implemented in stub")
}
func (r *stubClusterRepo) Get(_ context.Context, id string) (*models.Cluster, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.getCalls++
	if r.getErr != nil {
		return nil, r.getErr
	}
	c, ok := r.byID[id]
	if !ok {
		return nil, nil
	}
	cp := *c
	return &cp, nil
}
func (r *stubClusterRepo) GetByName(context.Context, string) (*models.Cluster, error) {
	return nil, errors.New("GetByName not implemented in stub")
}
func (r *stubClusterRepo) Create(context.Context, *models.Cluster) (*models.Cluster, error) {
	return nil, errors.New("Create not implemented in stub")
}
func (r *stubClusterRepo) Update(context.Context, *models.Cluster) (*models.Cluster, error) {
	return nil, errors.New("Update not implemented in stub")
}
func (r *stubClusterRepo) SoftDelete(context.Context, string) error {
	return errors.New("SoftDelete not implemented in stub")
}
func (r *stubClusterRepo) CountReferencingTargets(context.Context, string) (int, error) {
	return 0, nil
}

var _ repository.ClusterRepository = (*stubClusterRepo)(nil)

// Compile-time reminder: stubClusterRepo satisfies the repo interface.

// defaultTestEnv sets env vars so the default-cluster fallback path can build
// a rest.Config successfully in tests. Callers must invoke unsetTestEnv when
// done to avoid leaking env between tests.
func defaultTestEnv(t *testing.T) {
	t.Helper()
	os.Setenv("K8S_USE_METADATA_TOKEN", "true")
	os.Setenv("K8S_API_ENDPOINT", "https://fake-endpoint.example.com")
	os.Setenv("K8S_AUDIENCE", "test-audience")
	os.Setenv("K8S_INSECURE_SKIP_VERIFY", "true")
	t.Cleanup(func() {
		os.Unsetenv("K8S_USE_METADATA_TOKEN")
		os.Unsetenv("K8S_API_ENDPOINT")
		os.Unsetenv("K8S_AUDIENCE")
		os.Unsetenv("K8S_INSECURE_SKIP_VERIFY")
	})
}

func TestFactory_ForCluster_NotFound(t *testing.T) {
	f := NewClientFactory(newStubRepo())
	_, err := f.ForCluster(context.Background(), "no-such-cluster")
	if !errors.Is(err, ErrClusterNotFound) {
		t.Fatalf("want ErrClusterNotFound, got %v", err)
	}
}

func TestFactory_ForCluster_DefaultUsesEnv(t *testing.T) {
	defaultTestEnv(t)
	repo := newStubRepo(&models.Cluster{
		ID:             "cluster-default",
		Name:           "cyber-clust",
		IsDefault:      true,
		K8sAPIEndpoint: "", // empty = fall back to env
	})
	f := NewClientFactory(repo)
	if _, err := f.ForCluster(context.Background(), "cluster-default"); err != nil {
		t.Fatalf("default cluster with empty endpoint should fall back to env: %v", err)
	}
}

func TestFactory_ForCluster_ExplicitEndpointRequiresAudience(t *testing.T) {
	repo := newStubRepo(&models.Cluster{
		ID:             "cluster-delivery",
		Name:           "delivery-clust",
		K8sAPIEndpoint: "https://34.44.27.160",
		K8sAudience:    "", // missing
	})
	f := NewClientFactory(repo)
	_, err := f.ForCluster(context.Background(), "cluster-delivery")
	if !errors.Is(err, ErrClusterMisconfigured) {
		t.Fatalf("missing audience should return ErrClusterMisconfigured, got %v", err)
	}
}

func TestFactory_ForCluster_ExplicitEndpointWithAudience_Success(t *testing.T) {
	repo := newStubRepo(&models.Cluster{
		ID:             "cluster-delivery",
		Name:           "delivery-clust",
		K8sAPIEndpoint: "https://34.44.27.160",
		K8sAudience:    "test-audience",
		K8sCAData:      "", // empty = system trust store
	})
	f := NewClientFactory(repo)
	// Note: this builds the rest.Config + clientset but does NOT actually
	// hit the K8s API — kubernetes.NewForConfig is lazy. So the test passes
	// even against a fake endpoint.
	if _, err := f.ForCluster(context.Background(), "cluster-delivery"); err != nil {
		t.Fatalf("valid cluster should build client, got %v", err)
	}
}

func TestFactory_CacheHit(t *testing.T) {
	defaultTestEnv(t)
	repo := newStubRepo(&models.Cluster{ID: "cluster-default", Name: "d"})
	f := NewClientFactory(repo, WithTTL(time.Hour)) // long TTL so 2nd call is cache hit
	ctx := context.Background()
	if _, err := f.ForCluster(ctx, "cluster-default"); err != nil {
		t.Fatalf("first ForCluster failed: %v", err)
	}
	if _, err := f.ForCluster(ctx, "cluster-default"); err != nil {
		t.Fatalf("second ForCluster failed: %v", err)
	}
	if repo.getCalls != 1 {
		t.Errorf("expected 1 repo.Get call (cache hit on 2nd), got %d", repo.getCalls)
	}
}

func TestFactory_CacheExpiryTriggersReload(t *testing.T) {
	defaultTestEnv(t)
	repo := newStubRepo(&models.Cluster{ID: "cluster-default", Name: "d"})
	f := NewClientFactory(repo, WithTTL(10*time.Millisecond))
	ctx := context.Background()
	if _, err := f.ForCluster(ctx, "cluster-default"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(30 * time.Millisecond)
	if _, err := f.ForCluster(ctx, "cluster-default"); err != nil {
		t.Fatal(err)
	}
	if repo.getCalls != 2 {
		t.Errorf("expected 2 repo.Get calls after TTL expiry, got %d", repo.getCalls)
	}
}

func TestFactory_Invalidate(t *testing.T) {
	defaultTestEnv(t)
	repo := newStubRepo(&models.Cluster{ID: "cluster-default", Name: "d"})
	f := NewClientFactory(repo, WithTTL(time.Hour))
	ctx := context.Background()
	if _, err := f.ForCluster(ctx, "cluster-default"); err != nil {
		t.Fatal(err)
	}
	f.Invalidate("cluster-default")
	if _, err := f.ForCluster(ctx, "cluster-default"); err != nil {
		t.Fatal(err)
	}
	if repo.getCalls != 2 {
		t.Errorf("Invalidate should force fresh Get, got %d calls (want 2)", repo.getCalls)
	}
}

func TestFactory_ForTarget_EmptyClusterIDFallsToDefault(t *testing.T) {
	defaultTestEnv(t)
	repo := newStubRepo(&models.Cluster{ID: "cluster-default", Name: "d"})
	f := NewClientFactory(repo)
	target := &models.ExecutionTarget{ID: "some-target", ClusterID: ""}
	if _, err := f.ForTarget(context.Background(), target); err != nil {
		t.Fatalf("target with empty clusterID should resolve to cluster-default: %v", err)
	}
}

func TestFactory_ForTarget_NilTarget(t *testing.T) {
	f := NewClientFactory(newStubRepo())
	_, err := f.ForTarget(context.Background(), nil)
	if !errors.Is(err, ErrClusterMisconfigured) {
		t.Fatalf("nil target should return ErrClusterMisconfigured, got %v", err)
	}
}

func TestFactory_FailureIsolation(t *testing.T) {
	defaultTestEnv(t)
	// Two clusters; one has bad config (missing audience), the other is fine.
	repo := newStubRepo(
		&models.Cluster{ID: "cluster-default", Name: "d"},
		&models.Cluster{ID: "cluster-broken", Name: "broken", K8sAPIEndpoint: "https://x", K8sAudience: ""},
	)
	f := NewClientFactory(repo)
	// Broken cluster fails.
	if _, err := f.ForCluster(context.Background(), "cluster-broken"); !errors.Is(err, ErrClusterMisconfigured) {
		t.Fatalf("broken cluster expected misconfigured err, got %v", err)
	}
	// Default cluster still works — one failure doesn't poison the cache.
	if _, err := f.ForCluster(context.Background(), "cluster-default"); err != nil {
		t.Fatalf("healthy cluster should still work after neighbor failed: %v", err)
	}
}
