package workflow

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/k8s"
)

// ResourceQuotaInfo carries live usage/hard data for one quota.
type ResourceQuotaInfo struct {
	CPU    ResourceMetric `json:"cpu"`
	Memory ResourceMetric `json:"memory"`
	Limits ResourceMetric `json:"limits"`
}

type ResourceMetric struct {
	Used string `json:"used"`
	Hard string `json:"hard"`
}

// ListResourceQuotas handles GET /api/v1/resource-quotas.
// Returns live ResourceQuota data for all namespaces.
// Accepts `?clusterId=` (default `cluster-default`) — CYB-3486 PR 4b.
func (h *Handler) ListResourceQuotas(c *gin.Context) {
	clusterID := c.Query("clusterId")
	client, err := newK8sClientset(c.Request.Context(), h.k8sFactory, clusterID)
	if err != nil {
		httpresp.Internal(c, "k8s client: "+err.Error())
		return
	}

	quotas, err := client.CoreV1().ResourceQuotas("").List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		httpresp.Internal(c, "list quotas: "+err.Error())
		return
	}

	result := make(map[string]ResourceQuotaInfo, len(quotas.Items))
	for _, q := range quotas.Items {
		info := ResourceQuotaInfo{
			CPU:    extractResource(q, corev1.ResourceRequestsCPU),
			Memory: extractResource(q, corev1.ResourceRequestsMemory),
			Limits: ResourceMetric{
				Used: extractResource(q, corev1.ResourceLimitsCPU).Used,
				Hard: extractResource(q, corev1.ResourceLimitsCPU).Hard,
			},
		}
		// Skip system quotas (kube-system, koordinator-system, etc.)
		if info.CPU.Hard == "0" && info.Memory.Hard == "0" {
			continue
		}
		result[q.Namespace] = info
	}

	c.JSON(http.StatusOK, gin.H{"items": result})
}

func extractResource(q corev1.ResourceQuota, name corev1.ResourceName) ResourceMetric {
	m := ResourceMetric{Used: "0", Hard: "0"}
	if v, ok := q.Status.Used[name]; ok {
		m.Used = formatQuantity(v)
	}
	if v, ok := q.Spec.Hard[name]; ok {
		m.Hard = formatQuantity(v)
	}
	return m
}

func formatQuantity(q resource.Quantity) string {
	v := q.MilliValue()
	if v >= 1000 {
		return q.String()
	}
	return q.String()
}

// newK8sClientset picks the clientset for the target cluster: the factory
// path when configured, otherwise the env-based singleton (pre-3486 compat).
func newK8sClientset(ctx context.Context, factory k8s.ClientFactory, clusterID string) (kubernetes.Interface, error) {
	if factory == nil {
		return k8s.NewClientset("")
	}
	if clusterID == "" {
		clusterID = "cluster-default"
	}
	return factory.ForCluster(ctx, clusterID)
}
