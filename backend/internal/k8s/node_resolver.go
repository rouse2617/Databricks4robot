package k8s

import (
	"context"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GKE node labels used to resolve a node's cost profile.
const (
	labelInstanceType = "node.kubernetes.io/instance-type"
	labelAccelerator  = "cloud.google.com/gke-accelerator"
	labelProvisioning = "cloud.google.com/gke-provisioning"
)

// NodeProfile is the cost-relevant identity of a Kubernetes node.
type NodeProfile struct {
	InstanceType string // e.g. g2-standard-16, t2d-standard-8
	Accelerator  string // e.g. nvidia-l4; "" when none
	Provisioning string // standard | spot; "" when unset
}

// NodeInstanceResolver resolves a node name to its machine profile from node
// labels, cached in memory (nodes are stable). Used for per-pod cost pricing
// (CYB-3073). Best-effort: on error or missing labels it returns a zero profile
// so callers can fall back to defaults; it never errors out the cost path.
type NodeInstanceResolver struct {
	clientset kubernetes.Interface
	mu        sync.RWMutex
	cache     map[string]NodeProfile
}

// NewNodeInstanceResolver constructs a resolver. A nil clientset yields a
// resolver whose Resolve always returns a zero profile (feature disabled).
func NewNodeInstanceResolver(clientset kubernetes.Interface) *NodeInstanceResolver {
	return &NodeInstanceResolver{clientset: clientset, cache: map[string]NodeProfile{}}
}

// Resolve returns the node's cost profile, caching successful lookups. Returns a
// zero NodeProfile (and no error) when the node name is empty, the clientset is
// unavailable, or the lookup fails — the caller then keeps its default pricing.
func (r *NodeInstanceResolver) Resolve(ctx context.Context, nodeName string) NodeProfile {
	if r == nil || r.clientset == nil || nodeName == "" {
		return NodeProfile{}
	}
	r.mu.RLock()
	if p, ok := r.cache[nodeName]; ok {
		r.mu.RUnlock()
		return p
	}
	r.mu.RUnlock()

	node, err := r.clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil || node == nil {
		return NodeProfile{}
	}
	p := NodeProfile{
		InstanceType: node.Labels[labelInstanceType],
		Accelerator:  node.Labels[labelAccelerator],
		Provisioning: node.Labels[labelProvisioning],
	}
	// Only cache resolved profiles so a transient miss can be retried.
	if p.InstanceType != "" {
		r.mu.Lock()
		r.cache[nodeName] = p
		r.mu.Unlock()
	}
	return p
}

// ResolveNodeInstance is a string-tuple form of Resolve so callers (e.g. the
// pipeline usecase) can depend on a small interface without importing this
// package's NodeProfile type.
func (r *NodeInstanceResolver) ResolveNodeInstance(ctx context.Context, nodeName string) (instanceType, accelerator, provisioning string) {
	p := r.Resolve(ctx, nodeName)
	return p.InstanceType, p.Accelerator, p.Provisioning
}
