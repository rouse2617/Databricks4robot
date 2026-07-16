package argo

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// stubK8sFactory satisfies argo.K8sFactory for tests without pulling the real
// internal/k8s package (which would re-introduce the import cycle the interface
// was created to break).
type stubK8sFactory struct {
	dynCalls, typedCalls int
	dynClient            dynamic.Interface
	typedClient          kubernetes.Interface
	dynErr, typedErr     error
}

func newStubK8sFactory() *stubK8sFactory {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: "argoproj.io", Version: "v1alpha1", Kind: "Workflow"}, &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: "argoproj.io", Version: "v1alpha1", Kind: "WorkflowList"}, &unstructured.UnstructuredList{})
	gvrToListKind := map[schema.GroupVersionResource]string{workflowGVR: "WorkflowList"}
	return &stubK8sFactory{
		dynClient:   dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind),
		typedClient: k8sfake.NewSimpleClientset(),
	}
}

func (s *stubK8sFactory) ForCluster(context.Context, string) (kubernetes.Interface, error) {
	s.typedCalls++
	if s.typedErr != nil {
		return nil, s.typedErr
	}
	return s.typedClient, nil
}
func (s *stubK8sFactory) DynamicForCluster(context.Context, string) (dynamic.Interface, error) {
	s.dynCalls++
	if s.dynErr != nil {
		return nil, s.dynErr
	}
	return s.dynClient, nil
}

var _ K8sFactory = (*stubK8sFactory)(nil)

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
	f := NewClientFactory(repo, WithTTL(time.Hour),
		WithEnvFallback(&Config{ServerURL: "http://argo.default.svc:2746"}))
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
	f := NewClientFactory(repo, WithTTL(time.Hour),
		WithEnvFallback(&Config{ServerURL: "http://argo.default.svc:2746"}))
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
	f := NewClientFactory(repo,
		WithEnvFallback(&Config{ServerURL: "http://argo.default.svc:2746"}))
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

// TestArgoFactory_HTTPMode_ClusterHasURL verifies the HTTP path is chosen when
// the cluster row itself carries an argo-server URL, regardless of whether a
// k8s factory is wired.
func TestArgoFactory_HTTPMode_ClusterHasURL(t *testing.T) {
	repo := newStubRepo(&models.Cluster{
		ID:            "cluster-delivery",
		Name:          "delivery-clust",
		ArgoServerURL: "http://argo.delivery.svc:2746",
	})
	stub := newStubK8sFactory()
	f := NewClientFactory(repo, WithK8sFactory(stub))
	client, err := f.ForCluster(context.Background(), "cluster-delivery")
	if err != nil {
		t.Fatalf("http mode should build: %v", err)
	}
	if _, isHTTP := client.(*Client); !isHTTP {
		t.Errorf("expected *Client (HTTP mode), got %T", client)
	}
	if stub.dynCalls != 0 || stub.typedCalls != 0 {
		t.Errorf("k8s factory must NOT be called in HTTP mode (dyn=%d typed=%d)", stub.dynCalls, stub.typedCalls)
	}
}

// TestArgoFactory_HTTPMode_EnvFallback verifies the default-cluster env-fallback
// path stays HTTP: cluster row has empty URL, envCfg has one → HTTP path.
func TestArgoFactory_HTTPMode_EnvFallback(t *testing.T) {
	repo := newStubRepo(&models.Cluster{ID: "cluster-default", Name: "cyber-clust", ArgoServerURL: ""})
	stub := newStubK8sFactory()
	f := NewClientFactory(repo,
		WithEnvFallback(&Config{ServerURL: "http://argo.env.svc:2746"}),
		WithK8sFactory(stub))
	client, err := f.ForCluster(context.Background(), "cluster-default")
	if err != nil {
		t.Fatalf("http mode via env fallback should build: %v", err)
	}
	if _, isHTTP := client.(*Client); !isHTTP {
		t.Errorf("expected *Client (HTTP mode via env), got %T", client)
	}
	if stub.dynCalls != 0 || stub.typedCalls != 0 {
		t.Errorf("k8s factory must NOT be called when env has URL (dyn=%d typed=%d)", stub.dynCalls, stub.typedCalls)
	}
}

// TestArgoFactory_CRDMode_URLEmpty verifies the CRD path is chosen when both
// the cluster row and env fallback produce an empty URL, and a k8sFactory is
// wired.
func TestArgoFactory_CRDMode_URLEmpty(t *testing.T) {
	repo := newStubRepo(&models.Cluster{
		ID:            "cluster-delivery",
		Name:          "delivery-clust",
		ArgoServerURL: "",
		ArgoNamespace: "cyber-delivery-dev",
	})
	stub := newStubK8sFactory()
	f := NewClientFactory(repo, WithK8sFactory(stub))
	client, err := f.ForCluster(context.Background(), "cluster-delivery")
	if err != nil {
		t.Fatalf("crd mode should build: %v", err)
	}
	crd, ok := client.(*crdWorkflowClient)
	if !ok {
		t.Fatalf("expected *crdWorkflowClient, got %T", client)
	}
	if crd.defaultNS != "cyber-delivery-dev" {
		t.Errorf("defaultNS must come from cluster row, got %q", crd.defaultNS)
	}
	if stub.dynCalls != 1 || stub.typedCalls != 1 {
		t.Errorf("k8s factory should be called once each (dyn=%d typed=%d)", stub.dynCalls, stub.typedCalls)
	}
}

// TestArgoFactory_CRDMode_DefaultNamespaceFallback verifies that when the
// cluster row leaves argo_namespace blank, the client falls back to "argo".
func TestArgoFactory_CRDMode_DefaultNamespaceFallback(t *testing.T) {
	repo := newStubRepo(&models.Cluster{ID: "c", Name: "c", ArgoServerURL: ""})
	f := NewClientFactory(repo, WithK8sFactory(newStubK8sFactory()))
	client, err := f.ForCluster(context.Background(), "c")
	if err != nil {
		t.Fatalf("crd mode build: %v", err)
	}
	crd := client.(*crdWorkflowClient)
	if crd.defaultNS != defaultArgoNamespace {
		t.Errorf("blank argo_namespace should fall back to %q, got %q", defaultArgoNamespace, crd.defaultNS)
	}
}

// TestArgoFactory_CRDMode_MisconfiguredWhenNoK8sFactory guards D7: an empty
// resolved URL with no k8sFactory wired returns a distinct error so callers
// can distinguish "wrong URL" from "no CRD support in this deployment".
func TestArgoFactory_CRDMode_MisconfiguredWhenNoK8sFactory(t *testing.T) {
	repo := newStubRepo(&models.Cluster{ID: "c", Name: "c", ArgoServerURL: ""})
	f := NewClientFactory(repo) // no WithEnvFallback URL, no WithK8sFactory
	_, err := f.ForCluster(context.Background(), "c")
	if !errors.Is(err, ErrClusterMisconfigured) {
		t.Fatalf("want ErrClusterMisconfigured, got %v", err)
	}
}
