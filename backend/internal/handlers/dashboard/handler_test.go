package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	dashboardUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/dashboard"
)

type stubRepo struct {
	gotAssetType string
	resp         *models.DurationDistribution
}

func (s *stubRepo) DurationDistribution(_ context.Context, assetType string) (*models.DurationDistribution, error) {
	s.gotAssetType = assetType
	return s.resp, nil
}

func newTestHandler(resp *models.DurationDistribution) (*Handler, *stubRepo) {
	repo := &stubRepo{resp: resp}
	uc := dashboardUC.New(repo)
	return New(uc), repo
}

func fixtureDistribution() *models.DurationDistribution {
	// A minimal payload that still exercises every serialized field, including
	// the top-bucket HiMs=nil path.
	buckets := make([]models.DurationBucket, len(models.DurationBucketOrder))
	for i, def := range models.DurationBucketOrder {
		buckets[i] = models.DurationBucket{
			Label: def.Label,
			LoMs:  def.LoMs,
			HiMs:  def.HiMs,
			Count: 0,
		}
	}
	buckets[0].Count = 12
	buckets[0].TotalMs = 250_000
	buckets[4].Count = 3
	buckets[4].TotalMs = 15_000_000
	return &models.DurationDistribution{
		Buckets:     buckets,
		TotalAssets: 15,
		TotalMs:     15_250_000,
		MeanMs:      1_016_666,
		MinMs:       500,
		MaxMs:       6_500_000,
		P50Ms:       850_000,
		P90Ms:       2_900_000,
	}
}

func doGet(t *testing.T, h *Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/dashboard/duration-distribution", h.DurationDistribution)
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestDurationDistributionReturnsJSONShape(t *testing.T) {
	h, repo := newTestHandler(fixtureDistribution())

	w := doGet(t, h, "/dashboard/duration-distribution?asset_type=raw_mcap")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if repo.gotAssetType != "raw_mcap" {
		t.Fatalf("repo got asset_type=%q, want raw_mcap", repo.gotAssetType)
	}

	// Decode into a raw map so we can assert on JSON keys (snake_case) and
	// null-vs-number for hi_ms without leaning on Go type recovery.
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v; raw=%s", err, w.Body.String())
	}

	if body["asset_type"] != "raw_mcap" {
		t.Fatalf("asset_type = %v, want raw_mcap", body["asset_type"])
	}
	buckets, ok := body["buckets"].([]any)
	if !ok {
		t.Fatalf("buckets missing or wrong type: %T", body["buckets"])
	}
	if len(buckets) != 5 {
		t.Fatalf("buckets len = %d, want 5", len(buckets))
	}

	labels := []string{"<1min", "1-10min", "10-30min", "30-60min", "60min+"}
	for i, want := range labels {
		row, ok := buckets[i].(map[string]any)
		if !ok {
			t.Fatalf("buckets[%d] not object: %T", i, buckets[i])
		}
		if row["label"] != want {
			t.Fatalf("buckets[%d].label = %v, want %s", i, row["label"], want)
		}
		// Every bucket must ship count + lo_ms + hi_ms + total_ms keys.
		for _, k := range []string{"count", "lo_ms", "hi_ms", "total_ms"} {
			if _, present := row[k]; !present {
				t.Fatalf("buckets[%d] missing key %q", i, k)
			}
		}
	}

	// Top bucket ships hi_ms=null, every other bucket ships a finite number.
	top, _ := buckets[4].(map[string]any)
	if top["hi_ms"] != nil {
		t.Fatalf("top bucket hi_ms = %v, want null", top["hi_ms"])
	}
	first, _ := buckets[0].(map[string]any)
	if first["hi_ms"] == nil {
		t.Fatalf("first bucket hi_ms must not be null")
	}

	// Overall stats — every key must be present and snake_case.
	for _, k := range []string{
		"total_assets", "total_ms", "mean_ms", "min_ms", "max_ms", "p50_ms", "p90_ms",
	} {
		if _, ok := body[k]; !ok {
			t.Fatalf("response missing key %q", k)
		}
	}
}

func TestDurationDistributionNoAssetTypeParam(t *testing.T) {
	h, repo := newTestHandler(fixtureDistribution())

	w := doGet(t, h, "/dashboard/duration-distribution")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if repo.gotAssetType != "" {
		t.Fatalf("repo got asset_type=%q, want empty", repo.gotAssetType)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body["asset_type"] != nil {
		t.Fatalf("asset_type = %v, want null when absent", body["asset_type"])
	}
}

func TestDurationDistributionWhitespaceOnlyParamCollapsesToNil(t *testing.T) {
	// A query like `?asset_type=%20%20` (all whitespace) should behave like
	// no filter — dashboard is permissive.
	h, repo := newTestHandler(fixtureDistribution())
	w := doGet(t, h, "/dashboard/duration-distribution?asset_type=%20%20")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if repo.gotAssetType != "" {
		t.Fatalf("repo got asset_type=%q, want empty after trim", repo.gotAssetType)
	}
}
