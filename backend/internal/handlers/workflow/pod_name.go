package workflow

import (
	"strings"
	"sync"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

	"github.com/CyberOrigin2077/cyber-databrew/internal/argo"
)

const podNameCacheTTL = 2 * time.Minute

type podNameCacheEntry struct {
	podName   string
	expiresAt time.Time
}

type podNameCache struct {
	mu    sync.RWMutex
	items map[string]podNameCacheEntry
}

var globalPodNameCache = &podNameCache{
	items: make(map[string]podNameCacheEntry),
}

func (c *podNameCache) get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.items[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return "", false
	}
	return entry.podName, true
}

func (c *podNameCache) set(key, podName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = podNameCacheEntry{
		podName:   podName,
		expiresAt: time.Now().Add(podNameCacheTTL),
	}
}

// InvalidatePodNameCache removes all cached entries for a workflow.
func InvalidatePodNameCache(workflowName string) {
	globalPodNameCache.mu.Lock()
	defer globalPodNameCache.mu.Unlock()
	prefix := workflowName + "/"
	for key := range globalPodNameCache.items {
		if strings.HasPrefix(key, prefix) {
			delete(globalPodNameCache.items, key)
		}
	}
}

// resolveWorkflowPodName delegates to argo.PodNameForNode, the single source of
// truth for argo POD_NAMES=v2 pod naming shared with the argo CRD client.
func resolveWorkflowPodName(wf *wfv1.Workflow, nodeID string) (string, bool) {
	return argo.PodNameForNode(wf, nodeID)
}

func resolveCachedWorkflowPodName(wf *wfv1.Workflow, nodeID string) (string, bool) {
	if wf == nil || strings.TrimSpace(nodeID) == "" {
		return "", false
	}
	key := wf.Name + "/" + nodeID
	if podName, ok := globalPodNameCache.get(key); ok {
		return podName, true
	}
	podName, ok := resolveWorkflowPodName(wf, nodeID)
	if ok {
		globalPodNameCache.set(key, podName)
	}
	return podName, ok
}
