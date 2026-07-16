package argo

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// stubClusterRepo mirrors the k8s package's test stub — kept local to avoid
// cross-package test dependencies.
type stubClusterRepo struct {
	mu       sync.Mutex
	byID     map[string]*models.Cluster
	getCalls int
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
	return nil, nil
}
func (r *stubClusterRepo) Get(_ context.Context, id string) (*models.Cluster, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.getCalls++
	c, ok := r.byID[id]
	if !ok {
		return nil, nil
	}
	cp := *c
	return &cp, nil
}
func (r *stubClusterRepo) GetByName(context.Context, string) (*models.Cluster, error) {
	return nil, nil
}
func (r *stubClusterRepo) Create(context.Context, *models.Cluster) (*models.Cluster, error) {
	return nil, nil
}
func (r *stubClusterRepo) Update(context.Context, *models.Cluster) (*models.Cluster, error) {
	return nil, nil
}
func (r *stubClusterRepo) SoftDelete(context.Context, string) error { return nil }
func (r *stubClusterRepo) CountReferencingTargets(context.Context, string) (int, error) {
	return 0, nil
}

var _ repository.ClusterRepository = (*stubClusterRepo)(nil)

func TestArgoFactory_ForCluster_NotFound(t *testing.T) {
	f := NewClientFactory(newStubRepo())
	_, err := f.ForCluster(context.Background(), "no-such-cluster")
	if !errors.Is(err, ErrClusterNotFound) {
		t.Fatalf("want ErrClusterNotFound, got %v", err)
	}
}

func TestArgoFactory_DefaultClusterUsesEnvFallback(t *testing.T) {
	envCfg := &Config{ServerURL: "http://argo.default.svc:2746", Token: "envtoken"}
	repo := newStubRepo(&models.Cluster{
		ID:            "cluster-default",
		Name:          "cyber-clust",
		ArgoServerURL: "", // empty → use env fallback
	})
	f := NewClientFactory(repo, WithEnvFallback(envCfg))
	c, err := f.ForCluster(context.Background(), "cluster-default")
	if err != nil {
		t.Fatal(err)
	}
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestArgoFactory_ExplicitClusterOverridesURL(t *testing.T) {
	envCfg := &Config{ServerURL: "http://argo.default.svc:2746", Token: "envtoken"}
	repo := newStubRepo(&models.Cluster{
		ID:            "cluster-delivery",
		Name:          "delivery-clust",
		ArgoServerURL: "http://argo.delivery.svc:2746",
	})
	f := NewClientFactory(repo, WithEnvFallback(envCfg))
	if _, err := f.ForCluster(context.Background(), "cluster-delivery"); err != nil {
		t.Fatal(err)
	}
}

func TestArgoFactory_CacheHit(t *testing.T) {
	repo := newStubRepo(&models.Cluster{ID: "c1", Name: "c1"})
	f := NewClientFactory(repo, WithTTL(time.Hour))
	ctx := context.Background()
	if _, err := f.ForCluster(ctx, "c1"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.ForCluster(ctx, "c1"); err != nil {
		t.Fatal(err)
	}
	if repo.getCalls != 1 {
		t.Errorf("cache miss on 2nd call, got %d", repo.getCalls)
	}
}

func TestArgoFactory_Invalidate(t *testing.T) {
	repo := newStubRepo(&models.Cluster{ID: "c1", Name: "c1"})
	f := NewClientFactory(repo, WithTTL(time.Hour))
	ctx := context.Background()
	_, _ = f.ForCluster(ctx, "c1")
	f.Invalidate("c1")
	_, _ = f.ForCluster(ctx, "c1")
	if repo.getCalls != 2 {
		t.Errorf("Invalidate should force fresh Get, got %d", repo.getCalls)
	}
}

func TestArgoFactory_ForTarget_EmptyClusterIDFallsToDefault(t *testing.T) {
	repo := newStubRepo(&models.Cluster{ID: "cluster-default", Name: "d"})
	f := NewClientFactory(repo, WithEnvFallback(&Config{}))
	target := &models.ExecutionTarget{ID: "target-1", ClusterID: ""}
	if _, err := f.ForTarget(context.Background(), target); err != nil {
		t.Fatalf("empty clusterID should resolve to cluster-default: %v", err)
	}
}

func TestArgoFactory_ForTarget_NilTarget(t *testing.T) {
	f := NewClientFactory(newStubRepo())
	if _, err := f.ForTarget(context.Background(), nil); err == nil {
		t.Fatal("nil target should error")
	}
}
