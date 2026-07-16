package pipeline

import (
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// TestApplyElasticQuotaPodLabel_NilTarget guards against a nil-deref: legacy
// paths that call the helper without a target must not panic.
func TestApplyElasticQuotaPodLabel_NilTarget(t *testing.T) {
	got := applyElasticQuotaPodLabel(nil, nil)
	if got != nil {
		t.Errorf("nil target must leave labels unchanged, got %v", got)
	}
}

// TestApplyElasticQuotaPodLabel_EmptyName covers the default case: no EQ on
// the target → labels pass through unmodified. This is the pre-pool.1
// behavior and must stay exact so existing pipelines are byte-identical.
func TestApplyElasticQuotaPodLabel_EmptyName(t *testing.T) {
	preExisting := map[string]string{"cyber-databrew/owner": "team-vision"}
	got := applyElasticQuotaPodLabel(preExisting, &models.ExecutionTarget{})
	if _, has := got[elasticQuotaPodLabelKey]; has {
		t.Errorf("empty ElasticQuotaName must not inject the EQ label")
	}
	if got["cyber-databrew/owner"] != "team-vision" {
		t.Errorf("pre-existing labels must survive: %v", got)
	}
}

// TestApplyElasticQuotaPodLabel_TrimsWhitespace guards against blank-with-
// whitespace ClusterName values that shouldn't count as "set".
func TestApplyElasticQuotaPodLabel_TrimsWhitespace(t *testing.T) {
	got := applyElasticQuotaPodLabel(nil, &models.ExecutionTarget{ElasticQuotaName: "  \t "})
	if got != nil && len(got) > 0 {
		t.Errorf("whitespace-only ElasticQuotaName must be treated as empty; got %v", got)
	}
}

// TestApplyElasticQuotaPodLabel_InjectsIntoNilMap covers the fresh-map path:
// callers may pass nil labels; the helper allocates the map on demand.
func TestApplyElasticQuotaPodLabel_InjectsIntoNilMap(t *testing.T) {
	got := applyElasticQuotaPodLabel(nil, &models.ExecutionTarget{ElasticQuotaName: "team-vision-eq"})
	if got == nil {
		t.Fatal("expected non-nil labels map")
	}
	if got[elasticQuotaPodLabelKey] != "team-vision-eq" {
		t.Errorf("EQ label not injected; got %v", got)
	}
}

// TestApplyElasticQuotaPodLabel_MergesWithExisting verifies pre-existing
// cost-tracking labels are preserved.
func TestApplyElasticQuotaPodLabel_MergesWithExisting(t *testing.T) {
	preExisting := map[string]string{
		"cyber-databrew/owner":      "team-vision",
		"cyber-databrew/pipeline":   "p1",
	}
	got := applyElasticQuotaPodLabel(preExisting, &models.ExecutionTarget{ElasticQuotaName: "team-vision-eq"})
	if got[elasticQuotaPodLabelKey] != "team-vision-eq" {
		t.Errorf("EQ label not injected")
	}
	if got["cyber-databrew/owner"] != "team-vision" || got["cyber-databrew/pipeline"] != "p1" {
		t.Errorf("cost-tracking labels dropped: %v", got)
	}
}

// TestApplyElasticQuotaPodLabel_LabelKey pins the label key so a bump of
// koord upstream (e.g. quota.koordinator.sh/name vs
// quota.scheduling.koordinator.sh/name) is caught at test time rather than
// silently mis-routing pods to the ns default EQ in production.
func TestApplyElasticQuotaPodLabel_LabelKey(t *testing.T) {
	if elasticQuotaPodLabelKey != "quota.scheduling.koordinator.sh/name" {
		t.Errorf("label key changed to %q — verify koord version compatibility before landing", elasticQuotaPodLabelKey)
	}
}
