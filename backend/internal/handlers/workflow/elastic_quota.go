package workflow

import (
	"context"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/k8s"
)

// elasticQuotaGVR is the Koordinator ElasticQuota CRD GroupVersionResource
// (scheduler-plugins upstream, adopted by koord-scheduler).
var elasticQuotaGVR = schema.GroupVersionResource{
	Group:    "scheduling.sigs.k8s.io",
	Version:  "v1alpha1",
	Resource: "elasticquotas",
}

// listElasticQuotas is overridable in tests. Production wires it to a live
// dynamic client via the per-cluster factory (CYB-3486 PR 4b); when factory
// is nil (tests without wired infra) it falls back to the pre-3486 env
// singleton so existing behavior is byte-identical.
var listElasticQuotas = func(ctx context.Context, factory k8s.ClientFactory, clusterID string) (*unstructured.UnstructuredList, error) {
	dc, err := resolveDynamicClient(ctx, factory, clusterID)
	if err != nil {
		return nil, err
	}
	return dc.Resource(elasticQuotaGVR).List(ctx, metav1.ListOptions{})
}

// resolveDynamicClient picks the dynamic client for the target cluster: the
// factory path when configured, otherwise the env-based singleton.
func resolveDynamicClient(ctx context.Context, factory k8s.ClientFactory, clusterID string) (dynamic.Interface, error) {
	if factory == nil {
		return k8s.NewDynamicClient("")
	}
	if clusterID == "" {
		clusterID = "cluster-default"
	}
	return factory.DynamicForCluster(ctx, clusterID)
}

type elasticQuotaResources struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

type elasticQuotaUtilization struct {
	CPU    float64 `json:"cpu"`
	Memory float64 `json:"memory"`
}

// ElasticQuotaEntry is the JSON shape the frontend consumes.
type ElasticQuotaEntry struct {
	Name               string                  `json:"name"`
	Namespace          string                  `json:"namespace"`
	Min                elasticQuotaResources   `json:"min"`
	Max                elasticQuotaResources   `json:"max"`
	Used               elasticQuotaResources   `json:"used"`
	UtilizationPercent elasticQuotaUtilization `json:"utilizationPercent"`
}

// ListElasticQuotas handles GET /api/v1/elastic-quotas.
// Returns live Koordinator ElasticQuota data across all namespaces.
// Accepts `?clusterId=` (default `cluster-default`) — CYB-3486 PR 4b.
// If the CRD is not installed (未装 Koordinator), returns 200 + empty list.
func (h *Handler) ListElasticQuotas(c *gin.Context) {
	clusterID := c.Query("clusterId")
	list, err := listElasticQuotas(c.Request.Context(), h.k8sFactory, clusterID)
	if err != nil {
		if meta.IsNoMatchError(err) || apierrors.IsNotFound(err) {
			c.JSON(http.StatusOK, gin.H{"items": []ElasticQuotaEntry{}})
			return
		}
		httpresp.Error(c, http.StatusBadGateway, "k8s_unavailable", "list elastic quotas: "+err.Error(), nil)
		return
	}

	items := make([]ElasticQuotaEntry, 0, len(list.Items))
	for i := range list.Items {
		u := &list.Items[i]
		if isKoordinatorSystemQuota(u.GetNamespace()) {
			continue
		}
		items = append(items, buildElasticQuotaEntry(u))
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// isKoordinatorSystemQuota filters out the internal ElasticQuotas Koordinator
// itself creates in its control-plane namespace (root / default / system
// quotas with int64-max sentinel caps). Users only care about their own
// pools, and the sentinel caps otherwise blow up utilizationPercent math.
func isKoordinatorSystemQuota(namespace string) bool {
	return namespace == "koordinator-system"
}

// buildElasticQuotaEntry extracts a JSON-friendly entry from a single
// ElasticQuota unstructured object.
func buildElasticQuotaEntry(u *unstructured.Unstructured) ElasticQuotaEntry {
	minCPU, minMem := readCPUMem(u.Object, "spec", "min")
	maxCPU, maxMem := readCPUMem(u.Object, "spec", "max")
	usedCPU, usedMem := readCPUMem(u.Object, "status", "used")
	return ElasticQuotaEntry{
		Name:      u.GetName(),
		Namespace: u.GetNamespace(),
		Min:       elasticQuotaResources{CPU: quantityString(minCPU), Memory: quantityString(minMem)},
		Max:       elasticQuotaResources{CPU: quantityString(maxCPU), Memory: quantityString(maxMem)},
		Used:      elasticQuotaResources{CPU: quantityString(usedCPU), Memory: quantityString(usedMem)},
		UtilizationPercent: elasticQuotaUtilization{
			CPU:    utilization(usedCPU, maxCPU, func(q resource.Quantity) int64 { return q.MilliValue() }),
			Memory: utilization(usedMem, maxMem, func(q resource.Quantity) int64 { return q.Value() }),
		},
	}
}

// readCPUMem reads {cpu, memory} out of a nested ResourceList map inside the
// unstructured object. Missing keys yield zero-value Quantity.
func readCPUMem(obj map[string]any, path ...string) (cpu, mem resource.Quantity) {
	m, found, err := unstructured.NestedMap(obj, path...)
	if err != nil || !found {
		return resource.Quantity{}, resource.Quantity{}
	}
	if v, ok := m["cpu"].(string); ok {
		cpu, _ = resource.ParseQuantity(v)
	}
	if v, ok := m["memory"].(string); ok {
		mem, _ = resource.ParseQuantity(v)
	}
	return cpu, mem
}

// quantityString formats a Quantity for JSON output. Zero-value returns "0"
// (rather than an empty string) so the frontend renders it consistently.
func quantityString(q resource.Quantity) string {
	if q.IsZero() {
		return "0"
	}
	return q.String()
}

// utilization computes 100 * used / max, rounded to 1 decimal. Extractor picks
// the unit (MilliValue for CPU, Value for memory bytes) so both dimensions can
// share the same math without loss of precision on sub-core CPU usage. A
// non-positive extractor result (zero, or a sentinel that overflowed int64 —
// Koordinator uses 1844674407370955161 for "unlimited") short-circuits to 0
// so we never render a negative or NaN percentage.
func utilization(used, max resource.Quantity, extractor func(resource.Quantity) int64) float64 {
	m := extractor(max)
	if m <= 0 {
		return 0
	}
	u := extractor(used)
	if u < 0 {
		return 0
	}
	pct := 100.0 * float64(u) / float64(m)
	if math.IsNaN(pct) || math.IsInf(pct, 0) {
		return 0
	}
	return math.Round(pct*10) / 10
}
