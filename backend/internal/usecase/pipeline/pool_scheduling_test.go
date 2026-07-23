package pipeline

import (
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// targetWithScheduling builds a target whose pool scheduling config lives in
// resource_defaults.scheduling (JSONB) — the data-driven pool model (CYB-3486
// pool). No dedicated columns; the backend reads whatever the config declares.
func targetWithScheduling(scheduling map[string]interface{}) *models.ExecutionTarget {
	return &models.ExecutionTarget{
		ResourceDefaults: map[string]interface{}{"scheduling": scheduling},
	}
}

func TestExecutionTargetSchedulerName(t *testing.T) {
	// Koord pool: config declares the literal scheduler — backend hardcodes none.
	tg := targetWithScheduling(map[string]interface{}{"schedulerName": "koord-scheduler"})
	if got := executionTargetSchedulerName(tg); got != "koord-scheduler" {
		t.Errorf("want koord-scheduler, got %q", got)
	}
	// Scheduler-agnostic pool: no scheduler declared → empty (cluster default).
	if got := executionTargetSchedulerName(targetWithScheduling(nil)); got != "" {
		t.Errorf("empty scheduling should yield no scheduler, got %q", got)
	}
	if got := executionTargetSchedulerName(nil); got != "" {
		t.Errorf("nil target should yield no scheduler, got %q", got)
	}
}

func TestExecutionTargetPriorityClassName(t *testing.T) {
	tg := targetWithScheduling(map[string]interface{}{"priorityClassName": "cyber-databrew-prod"})
	if got := executionTargetPriorityClassName(tg); got != "cyber-databrew-prod" {
		t.Errorf("want cyber-databrew-prod, got %q", got)
	}
	if got := executionTargetPriorityClassName(targetWithScheduling(nil)); got != "" {
		t.Errorf("no priorityClass should yield empty, got %q", got)
	}
}

func TestExecutionTargetWorkflowPriority(t *testing.T) {
	// Pool default priority lives top-level in resource_defaults (like
	// maxActiveWorkflows), not under scheduling. JSON numbers decode to float64.
	tg := &models.ExecutionTarget{ResourceDefaults: map[string]interface{}{"priority": float64(-100)}}
	if got := executionTargetWorkflowPriority(tg); got == nil || *got != -100 {
		t.Errorf("want -100, got %v", got)
	}
	// Numeric string is also accepted.
	tg2 := &models.ExecutionTarget{ResourceDefaults: map[string]interface{}{"priority": "100"}}
	if got := executionTargetWorkflowPriority(tg2); got == nil || *got != 100 {
		t.Errorf("want 100 from string, got %v", got)
	}
	// Absent → nil so Transpile leaves the field unset (Argo default 0 = normal).
	if got := executionTargetWorkflowPriority(&models.ExecutionTarget{ResourceDefaults: map[string]interface{}{}}); got != nil {
		t.Errorf("absent priority should yield nil, got %v", *got)
	}
	if got := executionTargetWorkflowPriority(nil); got != nil {
		t.Errorf("nil target should yield nil, got %v", *got)
	}
}

func TestExecutionTargetPodLabels(t *testing.T) {
	// The pool config supplies BOTH the label key and value (here a Koordinator
	// EQ label) — the backend does not hardcode the koord label key.
	tg := targetWithScheduling(map[string]interface{}{
		"podLabels": map[string]interface{}{
			"quota.scheduling.koordinator.sh/name": "cyberorigin-delivery-low",
			"team":                                 "vision",
		},
	})
	got := executionTargetPodLabels(tg)
	if got["quota.scheduling.koordinator.sh/name"] != "cyberorigin-delivery-low" {
		t.Errorf("EQ label not read: %v", got)
	}
	if got["team"] != "vision" {
		t.Errorf("arbitrary label not read: %v", got)
	}
	if executionTargetPodLabels(targetWithScheduling(nil)) != nil {
		t.Errorf("no podLabels should yield nil")
	}
}

func TestExecutionTargetPodAnnotations(t *testing.T) {
	tg := targetWithScheduling(map[string]interface{}{
		"podAnnotations": map[string]interface{}{"scheduling.koordinator.sh/tier": "batch"},
	})
	got := executionTargetPodAnnotations(tg)
	if got["scheduling.koordinator.sh/tier"] != "batch" {
		t.Errorf("annotation not read: %v", got)
	}
	if executionTargetPodAnnotations(targetWithScheduling(nil)) != nil {
		t.Errorf("no podAnnotations should yield nil")
	}
}

func TestMergePoolPodLabels(t *testing.T) {
	base := map[string]string{"cyber-databrew/owner": "team-a", "shared": "base"}
	tg := targetWithScheduling(map[string]interface{}{
		"podLabels": map[string]interface{}{
			"quota.scheduling.koordinator.sh/name": "eq-low",
			"shared":                               "pool", // pool wins on conflict
		},
	})
	got := mergePoolPodLabels(base, tg)
	if got["cyber-databrew/owner"] != "team-a" {
		t.Errorf("base label dropped: %v", got)
	}
	if got["quota.scheduling.koordinator.sh/name"] != "eq-low" {
		t.Errorf("pool label not merged: %v", got)
	}
	if got["shared"] != "pool" {
		t.Errorf("pool label should win on conflict, got %q", got["shared"])
	}
	// No pool labels → base returned unchanged (may be nil).
	if mergePoolPodLabels(nil, targetWithScheduling(nil)) != nil {
		t.Errorf("nil base + no pool labels should stay nil")
	}
}

// TestPoolConfig_TopLevelAlsoWorks verifies scheduling keys are read whether
// nested under "scheduling" or placed directly in resource_defaults (both are
// scanned by executionTargetSchedulingMaps), so admins have flexibility.
func TestPoolConfig_TopLevelAlsoWorks(t *testing.T) {
	tg := &models.ExecutionTarget{
		ResourceDefaults: map[string]interface{}{"schedulerName": "custom-scheduler"},
	}
	if got := executionTargetSchedulerName(tg); got != "custom-scheduler" {
		t.Errorf("top-level schedulerName should be read, got %q", got)
	}
}
