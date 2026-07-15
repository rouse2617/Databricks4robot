package k8s

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNodeInstanceResolver_ReadsLabelsAndCaches(t *testing.T) {
	cs := fake.NewSimpleClientset(&corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "gke-cpu-node-1",
			Labels: map[string]string{
				labelInstanceType: "t2d-standard-8",
				labelProvisioning: "spot",
				// no accelerator label → CPU node
			},
		},
	})
	r := NewNodeInstanceResolver(cs)
	it, acc, prov := r.ResolveNodeInstance(context.Background(), "gke-cpu-node-1")
	if it != "t2d-standard-8" || acc != "" || prov != "spot" {
		t.Fatalf("resolved = %q/%q/%q, want t2d-standard-8//spot", it, acc, prov)
	}
	// cached (second call still works even without hitting API again)
	if p := r.Resolve(context.Background(), "gke-cpu-node-1"); p.InstanceType != "t2d-standard-8" {
		t.Fatalf("cached resolve = %+v", p)
	}
}

func TestNodeInstanceResolver_MissAndNilAreZero(t *testing.T) {
	r := NewNodeInstanceResolver(fake.NewSimpleClientset())
	if p := r.Resolve(context.Background(), "unknown"); p.InstanceType != "" {
		t.Fatalf("unknown node should be zero profile, got %+v", p)
	}
	var nilR *NodeInstanceResolver
	if p := nilR.Resolve(context.Background(), "x"); p.InstanceType != "" {
		t.Fatalf("nil resolver should be zero profile")
	}
	if it, _, _ := NewNodeInstanceResolver(nil).ResolveNodeInstance(context.Background(), "x"); it != "" {
		t.Fatalf("nil clientset should be zero profile")
	}
}
