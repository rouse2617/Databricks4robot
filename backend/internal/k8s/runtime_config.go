package k8s

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// Labels/annotations for content-addressed runtime configs (CYB-3680).
const (
	// LabelContentAddressed marks ConfigMaps whose name derives from content
	// and whose lifecycle belongs to the TTL janitor (owner-less, shared).
	LabelContentAddressed = "cyberorigin.ai/content-addressed"
	// AnnotationLastReferenced is the sliding-reference timestamp the janitor
	// ages against. Refreshed (rate-limited) every time a run ensures the CM.
	AnnotationLastReferenced = "cyberorigin.ai/last-referenced"
	// lastReferencedRefreshEvery bounds annotation write churn: a 10k-run
	// batch sharing one hash patches the CM at most once per interval.
	lastReferencedRefreshEvery = time.Hour
)

// refreshCache rate-limits last-referenced annotation writes per content
// hash, shared across per-target stores via the factory.
type refreshCache struct {
	mu   sync.Mutex
	seen map[string]time.Time
	now  func() time.Time
}

func newRefreshCache() *refreshCache {
	return &refreshCache{seen: map[string]time.Time{}, now: time.Now}
}

// shouldRefresh reports whether the hash's annotation is due and records the
// attempt (optimistic: a failed PATCH just waits for the next interval).
func (c *refreshCache) shouldRefresh(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	if last, ok := c.seen[key]; ok && now.Sub(last) < lastReferencedRefreshEvery {
		return false
	}
	c.seen[key] = now
	return true
}

type RuntimeConfigStore struct {
	clientset kubernetes.Interface
	refresh   *refreshCache
}

func NewRuntimeConfigStore(clientset kubernetes.Interface) *RuntimeConfigStore {
	return &RuntimeConfigStore{clientset: clientset, refresh: newRefreshCache()}
}

// RuntimeConfigStoreFactory resolves a per-cluster RuntimeConfigStore from a
// ClientFactory, so the runtime-config ConfigMap is created on the target's own
// cluster rather than the default-cluster singleton (CYB-3486). Implements
// pipelineUC.RuntimeConfigStoreFactory.
type RuntimeConfigStoreFactory struct {
	clients ClientFactory
	// refresh is shared across per-target stores so the last-referenced
	// rate limit survives the per-Deploy store instantiation (CYB-3680).
	refresh *refreshCache
}

// NewRuntimeConfigStoreFactory wires a per-cluster runtime-config store factory
// over the shared K8s ClientFactory.
func NewRuntimeConfigStoreFactory(clients ClientFactory) *RuntimeConfigStoreFactory {
	return &RuntimeConfigStoreFactory{clients: clients, refresh: newRefreshCache()}
}

// ForTarget returns a RuntimeConfigStore bound to the target's cluster clientset.
func (f *RuntimeConfigStoreFactory) ForTarget(ctx context.Context, target *models.ExecutionTarget) (pipelineUC.RuntimeConfigStore, error) {
	if f == nil || f.clients == nil {
		return nil, fmt.Errorf("%w: runtime config store factory is not configured", ErrUnavailable)
	}
	clientset, err := f.clients.ForTarget(ctx, target)
	if err != nil {
		return nil, err
	}
	return &RuntimeConfigStore{clientset: clientset, refresh: f.refresh}, nil
}

var invalidConfigMapNameChars = regexp.MustCompile(`[^a-z0-9-]+`)

func (s *RuntimeConfigStore) Create(ctx context.Context, namespace string, deploymentID string, input pipelineUC.RuntimeConfigProjection, owner *pipelineUC.RuntimeConfigOwnerReference) (string, error) {
	if s == nil || s.clientset == nil {
		return "", fmt.Errorf("%w: runtime config store is not configured", ErrUnavailable)
	}
	namespace = strings.TrimSpace(namespace)
	deploymentID = strings.TrimSpace(deploymentID)
	volumeName := strings.TrimSpace(input.VolumeName)
	if volumeName == "" {
		volumeName = "runtime-config-" + deploymentID
	}
	fileName := strings.TrimSpace(input.FileName)
	files := make(map[string]string)
	for name, content := range input.Files {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		files[name] = content
	}
	if fileName != "" {
		files[fileName] = input.Content
	}
	if namespace == "" || deploymentID == "" || len(files) == 0 {
		return "", fmt.Errorf("namespace, deploymentID, and at least one config file are required")
	}
	name := sanitizeRuntimeConfigName(volumeName)
	labels := map[string]string{
		"app.kubernetes.io/managed-by":  "cyber-databrew",
		"cyberorigin.ai/runtime-config": "true",
	}
	var annotations map[string]string
	ownerRefs := []metav1.OwnerReference(nil)
	if input.ContentHash != "" {
		// Content-addressed (CYB-3680): shared across runs, so it MUST be
		// owner-less — cascading with any single Workflow would strand every
		// other run mounting it. Lifecycle = sliding-reference TTL janitor.
		labels[LabelContentAddressed] = "true"
		annotations = map[string]string{
			AnnotationLastReferenced: time.Now().UTC().Format(time.RFC3339),
		}
	} else if owner != nil && owner.Name != "" && owner.UID != "" {
		// Legacy per-run CM: cascade-deleted with its owning Workflow.
		ownerRefs = append(ownerRefs, metav1.OwnerReference{
			APIVersion: owner.APIVersion,
			Kind:       owner.Kind,
			Name:       owner.Name,
			UID:        types.UID(owner.UID),
		})
	}
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			OwnerReferences: ownerRefs,
			Labels:          labels,
			Annotations:     annotations,
		},
		Data: files,
	}
	_, err := s.clientset.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			// Reuse IS the point of content addressing. Keep the shared CM's
			// sliding-reference timestamp fresh so the janitor never reclaims
			// a config that active batches still mount (rate-limited: one
			// PATCH per hash per interval, not per run).
			if input.ContentHash != "" && s.refresh != nil && s.refresh.shouldRefresh(namespace+"/"+name) {
				s.touchLastReferenced(ctx, namespace, name)
			}
			return name, nil
		}
		return "", err
	}
	return name, nil
}

// touchLastReferenced best-effort bumps the janitor's aging timestamp. A
// failed PATCH is tolerated: the next rate-limit interval retries, and the
// 35-day TTL dwarfs any transient gap.
func (s *RuntimeConfigStore) touchLastReferenced(ctx context.Context, namespace, name string) {
	patch := fmt.Sprintf(
		`{"metadata":{"annotations":{%q:%q},"labels":{%q:"true"}}}`,
		AnnotationLastReferenced, time.Now().UTC().Format(time.RFC3339),
		LabelContentAddressed,
	)
	if _, err := s.clientset.CoreV1().ConfigMaps(namespace).Patch(
		ctx, name, types.MergePatchType, []byte(patch), metav1.PatchOptions{},
	); err != nil {
		slog.Warn("runtime config last-referenced touch failed", "namespace", namespace, "name", name, "err", err)
	}
}

func sanitizeRuntimeConfigName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = invalidConfigMapNameChars.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "runtime-config"
	}
	if len(value) > 63 {
		value = strings.Trim(value[:63], "-")
	}
	if value == "" {
		return "runtime-config"
	}
	return value
}
