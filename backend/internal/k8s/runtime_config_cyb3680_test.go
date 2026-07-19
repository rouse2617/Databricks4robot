package k8s

import (
	"context"
	"testing"
	"time"

	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

// ── CYB-3680: content-addressed create ───────────────────────────────────────

// Content-addressed CMs are shared across runs: they must carry the janitor
// label + sliding-reference annotation and MUST stay owner-less even when a
// caller passes an owner (cascading with one Workflow would strand the rest).
func TestRuntimeConfigStoreCreateContentAddressed(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	store := NewRuntimeConfigStore(client)

	name, err := store.Create(ctx, "ns", "dep-1", pipelineUC.RuntimeConfigProjection{
		VolumeName:  "runtime-config-abc123",
		ContentHash: "abc123",
		Files:       map[string]string{"cfg.yaml": "a: 1\n"},
	}, &pipelineUC.RuntimeConfigOwnerReference{
		APIVersion: "argoproj.io/v1alpha1", Kind: "Workflow", Name: "wf-1", UID: "u1",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := client.CoreV1().ConfigMaps("ns").Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(got.OwnerReferences) != 0 {
		t.Fatalf("content-addressed CM must be owner-less, got %#v", got.OwnerReferences)
	}
	if got.Labels[LabelContentAddressed] != "true" {
		t.Fatalf("missing %s label: %#v", LabelContentAddressed, got.Labels)
	}
	raw, ok := got.Annotations[AnnotationLastReferenced]
	if !ok {
		t.Fatalf("missing %s annotation: %#v", AnnotationLastReferenced, got.Annotations)
	}
	if _, err := time.Parse(time.RFC3339, raw); err != nil {
		t.Fatalf("last-referenced not RFC3339: %q", raw)
	}
}

// Legacy per-run projections (no ContentHash) keep the owner-cascade
// lifecycle byte-for-byte.
func TestRuntimeConfigStoreCreateLegacyKeepsOwner(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	store := NewRuntimeConfigStore(client)

	name, err := store.Create(ctx, "ns", "dep-1", pipelineUC.RuntimeConfigProjection{
		VolumeName: "runtime-config-dep-1",
		Files:      map[string]string{"cfg.yaml": "a: 1\n"},
	}, &pipelineUC.RuntimeConfigOwnerReference{
		APIVersion: "argoproj.io/v1alpha1", Kind: "Workflow", Name: "wf-1", UID: "u1",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, _ := client.CoreV1().ConfigMaps("ns").Get(ctx, name, metav1.GetOptions{})
	if len(got.OwnerReferences) != 1 || got.OwnerReferences[0].Name != "wf-1" {
		t.Fatalf("legacy CM must keep owner, got %#v", got.OwnerReferences)
	}
	if got.Labels[LabelContentAddressed] == "true" {
		t.Fatal("legacy CM must not carry the content-addressed label")
	}
}

// AlreadyExists is reuse: the shared CM's timestamp is touched, rate-limited
// to one PATCH per hash per interval.
func TestRuntimeConfigStoreEnsureRefreshesRateLimited(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	store := NewRuntimeConfigStore(client)

	patches := 0
	client.PrependReactor("patch", "configmaps", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
		patches++
		return false, nil, nil // fall through to default reactor
	})

	proj := pipelineUC.RuntimeConfigProjection{
		VolumeName:  "runtime-config-abc123",
		ContentHash: "abc123",
		Files:       map[string]string{"cfg.yaml": "a: 1\n"},
	}
	for i := 0; i < 3; i++ { // 1 create + 2 ensures within the interval
		if _, err := store.Create(ctx, "ns", "dep", proj, nil); err != nil {
			t.Fatalf("ensure #%d: %v", i, err)
		}
	}
	if patches != 1 {
		t.Fatalf("patches = %d, want exactly 1 (first AlreadyExists touches, second is rate-limited)", patches)
	}

	// After the interval elapses the next ensure touches again.
	store.refresh.now = func() time.Time { return time.Now().Add(2 * lastReferencedRefreshEvery) }
	if _, err := store.Create(ctx, "ns", "dep", proj, nil); err != nil {
		t.Fatalf("ensure post-interval: %v", err)
	}
	if patches != 2 {
		t.Fatalf("patches = %d, want 2 after interval elapsed", patches)
	}
}

func TestRefreshCacheShouldRefresh(t *testing.T) {
	c := newRefreshCache()
	base := time.Now()
	c.now = func() time.Time { return base }
	if !c.shouldRefresh("k") {
		t.Fatal("first call must refresh")
	}
	if c.shouldRefresh("k") {
		t.Fatal("second call within interval must not refresh")
	}
	c.now = func() time.Time { return base.Add(lastReferencedRefreshEvery + time.Minute) }
	if !c.shouldRefresh("k") {
		t.Fatal("call after interval must refresh")
	}
}

// A failed PATCH is tolerated (warn only): ensure still succeeds.
func TestTouchLastReferencedFailureIsTolerated(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	client.PrependReactor("patch", "configmaps", func(k8stesting.Action) (bool, k8sruntime.Object, error) {
		return true, nil, context.DeadlineExceeded
	})
	store := NewRuntimeConfigStore(client)
	proj := pipelineUC.RuntimeConfigProjection{
		VolumeName: "runtime-config-abc", ContentHash: "abc",
		Files: map[string]string{"f": "x"},
	}
	if _, err := store.Create(ctx, "ns", "d", proj, nil); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := store.Create(ctx, "ns", "d", proj, nil); err != nil {
		t.Fatalf("ensure with failing patch must still succeed: %v", err)
	}
}

// ── CYB-3680: janitor ────────────────────────────────────────────────────────

func janitorCM(name, ns string, contentAddressed bool, lastRef string) *corev1.ConfigMap {
	labels := map[string]string{"cyberorigin.ai/runtime-config": "true"}
	if contentAddressed {
		labels[LabelContentAddressed] = "true"
	}
	var ann map[string]string
	if lastRef != "" {
		ann = map[string]string{AnnotationLastReferenced: lastRef}
	}
	return &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name: name, Namespace: ns, Labels: labels, Annotations: ann,
	}}
}

func TestSweepRuntimeConfigs(t *testing.T) {
	ctx := context.Background()
	expired := time.Now().UTC().Add(-40 * 24 * time.Hour).Format(time.RFC3339)
	fresh := time.Now().UTC().Format(time.RFC3339)

	client := fake.NewSimpleClientset(
		janitorCM("rc-expired", "ns-a", true, expired),
		janitorCM("rc-fresh", "ns-a", true, fresh),
		janitorCM("rc-no-annotation", "ns-b", true, ""),
		janitorCM("rc-bad-timestamp", "ns-b", true, "not-a-time"),
		janitorCM("rc-legacy", "ns-b", false, ""),
	)
	// Assert the janitor lists with the content-addressed selector (the fake
	// tracker also filters by it, but pin the contract explicitly).
	var selector string
	client.PrependReactor("list", "configmaps", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
		selector = action.(k8stesting.ListAction).GetListRestrictions().Labels.String()
		return false, nil, nil
	})

	deleted, err := SweepRuntimeConfigs(ctx, client, 35*24*time.Hour)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want exactly 1 (only the expired one)", deleted)
	}
	if selector != LabelContentAddressed+"=true" {
		t.Fatalf("list selector = %q, want %s=true", selector, LabelContentAddressed)
	}
	if _, err := client.CoreV1().ConfigMaps("ns-a").Get(ctx, "rc-expired", metav1.GetOptions{}); err == nil {
		t.Fatal("expired CM must be deleted")
	}
	for _, keep := range []struct{ ns, name string }{
		{"ns-a", "rc-fresh"}, {"ns-b", "rc-no-annotation"}, {"ns-b", "rc-bad-timestamp"}, {"ns-b", "rc-legacy"},
	} {
		if _, err := client.CoreV1().ConfigMaps(keep.ns).Get(ctx, keep.name, metav1.GetOptions{}); err != nil {
			t.Fatalf("%s/%s must be kept: %v", keep.ns, keep.name, err)
		}
	}
}

// Delete failures are tolerated per-item: the sweep continues and reports
// only successful reclaims.
func TestSweepRuntimeConfigsDeleteFailureTolerated(t *testing.T) {
	ctx := context.Background()
	expired := time.Now().UTC().Add(-40 * 24 * time.Hour).Format(time.RFC3339)
	client := fake.NewSimpleClientset(
		janitorCM("rc-1", "ns", true, expired),
		janitorCM("rc-2", "ns", true, expired),
	)
	client.PrependReactor("delete", "configmaps", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
		if action.(k8stesting.DeleteAction).GetName() == "rc-1" {
			return true, nil, context.DeadlineExceeded
		}
		return false, nil, nil
	})
	deleted, err := SweepRuntimeConfigs(ctx, client, 35*24*time.Hour)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1 (rc-2 only; rc-1 delete failed)", deleted)
	}
}

func TestSweepRuntimeConfigsListError(t *testing.T) {
	client := fake.NewSimpleClientset()
	client.PrependReactor("list", "configmaps", func(k8stesting.Action) (bool, k8sruntime.Object, error) {
		return true, nil, context.DeadlineExceeded
	})
	if _, err := SweepRuntimeConfigs(context.Background(), client, time.Hour); err == nil {
		t.Fatal("want list error surfaced")
	}
}

// StartRuntimeConfigJanitor runs an eager sweep on boot and stops with ctx.
func TestStartRuntimeConfigJanitorEagerSweepAndStop(t *testing.T) {
	client := fake.NewSimpleClientset()
	lists := make(chan struct{}, 8)
	client.PrependReactor("list", "configmaps", func(k8stesting.Action) (bool, k8sruntime.Object, error) {
		select {
		case lists <- struct{}{}:
		default:
		}
		return false, nil, nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	StartRuntimeConfigJanitor(ctx, client, time.Hour, 10*time.Millisecond)
	select {
	case <-lists: // eager sweep observed
	case <-time.After(3 * time.Second):
		t.Fatal("eager sweep never ran")
	}
	cancel() // loop must exit without panic/leak (verified by -race on suite)
}
