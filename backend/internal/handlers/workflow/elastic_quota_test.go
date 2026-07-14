package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestListElasticQuotas_ReturnsQuotas(t *testing.T) {
	gin.SetMode(gin.TestMode)

	restore := stubElasticQuotaLister(func(context.Context) (*unstructured.UnstructuredList, error) {
		return &unstructured.UnstructuredList{
			Items: []unstructured.Unstructured{
				makeQuota("cyberorigin-delivery-high", "cyber-databrew-dev",
					map[string]string{"cpu": "4", "memory": "8Gi"},
					map[string]string{"cpu": "24", "memory": "48Gi"},
					map[string]string{"cpu": "6", "memory": "12Gi"},
				),
				makeQuota("cyberorigin-delivery-mid", "cyber-databrew-dev",
					map[string]string{"cpu": "2", "memory": "4Gi"},
					map[string]string{"cpu": "14", "memory": "28Gi"},
					map[string]string{"cpu": "0", "memory": "0"},
				),
				makeQuota("cyberorigin-delivery-low", "cyber-databrew-dev",
					map[string]string{"cpu": "1", "memory": "2Gi"},
					map[string]string{"cpu": "10", "memory": "20Gi"},
					map[string]string{"cpu": "0", "memory": "0"},
				),
			},
		}, nil
	})
	defer restore()

	h := &Handler{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/elastic-quotas", nil)

	h.ListElasticQuotas(c)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Items []ElasticQuotaEntry `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) != 3 {
		t.Fatalf("want 3 items, got %d", len(resp.Items))
	}

	high := resp.Items[0]
	if high.Name != "cyberorigin-delivery-high" {
		t.Errorf("high name = %q", high.Name)
	}
	if high.Namespace != "cyber-databrew-dev" {
		t.Errorf("high namespace = %q", high.Namespace)
	}
	if high.Min.CPU != "4" || high.Max.CPU != "24" || high.Used.CPU != "6" {
		t.Errorf("high cpu unexpected: min=%q max=%q used=%q", high.Min.CPU, high.Max.CPU, high.Used.CPU)
	}
	// utilization: used=6c / max=24c = 25%
	if high.UtilizationPercent.CPU != 25.0 {
		t.Errorf("high cpu utilization = %v (want 25.0)", high.UtilizationPercent.CPU)
	}
	// memory: 12Gi / 48Gi = 25%
	if high.UtilizationPercent.Memory != 25.0 {
		t.Errorf("high memory utilization = %v (want 25.0)", high.UtilizationPercent.Memory)
	}

	// Empty used → 0%
	mid := resp.Items[1]
	if mid.UtilizationPercent.CPU != 0 || mid.UtilizationPercent.Memory != 0 {
		t.Errorf("mid utilization should be 0, got cpu=%v mem=%v", mid.UtilizationPercent.CPU, mid.UtilizationPercent.Memory)
	}
	if mid.Used.CPU != "0" || mid.Used.Memory != "0" {
		t.Errorf("mid used should be '0', got cpu=%q mem=%q", mid.Used.CPU, mid.Used.Memory)
	}
}

func TestListElasticQuotas_CRDNotInstalled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// meta.IsNoMatchError requires a *meta.NoKindMatchError; simulate by
	// returning a 404 NotFound (both branches are handled by the handler).
	notFound := apierrors.NewNotFound(schema.GroupResource{Group: "scheduling.sigs.k8s.io", Resource: "elasticquotas"}, "")
	restore := stubElasticQuotaLister(func(context.Context) (*unstructured.UnstructuredList, error) {
		return nil, notFound
	})
	defer restore()

	h := &Handler{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/elastic-quotas", nil)

	h.ListElasticQuotas(c)

	if w.Code != http.StatusOK {
		t.Fatalf("CRD-missing should return 200, got %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Items []ElasticQuotaEntry `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) != 0 {
		t.Fatalf("want empty items, got %d", len(resp.Items))
	}
}

func TestListElasticQuotas_UpstreamError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	restore := stubElasticQuotaLister(func(context.Context) (*unstructured.UnstructuredList, error) {
		return nil, errors.New("connection refused")
	})
	defer restore()

	h := &Handler{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/elastic-quotas", nil)

	h.ListElasticQuotas(c)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("upstream failure should return 502, got %d", w.Code)
	}
}

// stubElasticQuotaLister replaces the package-level listElasticQuotas closure
// for a test and returns a restore func for defer.
func stubElasticQuotaLister(fn func(context.Context) (*unstructured.UnstructuredList, error)) func() {
	prev := listElasticQuotas
	listElasticQuotas = fn
	return func() { listElasticQuotas = prev }
}

func makeQuota(name, ns string, min, max, used map[string]string) unstructured.Unstructured {
	stringMap := func(m map[string]string) map[string]any {
		out := make(map[string]any, len(m))
		for k, v := range m {
			out[k] = v
		}
		return out
	}
	return unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "scheduling.sigs.k8s.io/v1alpha1",
		"kind":       "ElasticQuota",
		"metadata": map[string]any{
			"name":      name,
			"namespace": ns,
		},
		"spec": map[string]any{
			"min": stringMap(min),
			"max": stringMap(max),
		},
		"status": map[string]any{
			"used": stringMap(used),
		},
	}}
}
