package pipeline

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// backpressureDefaultMaxActive is the fallback active-workflow ceiling used when
// a target's resource_defaults does not set maxActiveWorkflows (CYB-3681). Sized
// well under an Argo controller's memory ceiling (~66KB/workflow → a 4Gi
// controller holds ~36k) to leave etcd LIST headroom. Env overrides the global
// default; per-target resource_defaults is the primary, online-tunable knob.
var backpressureDefaultMaxActive = func() int {
	if v := os.Getenv("BACKFILL_BACKPRESSURE_DEFAULT_MAX_ACTIVE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	return 20000
}()

// executionTargetMaxActiveWorkflows reads the per-namespace active-workflow
// ceiling from a target's resource_defaults (online-tunable via the pool
// manager). Absent → compiled default. An explicit 0 disables backpressure for
// the target.
func executionTargetMaxActiveWorkflows(target *models.ExecutionTarget) int {
	if raw, ok := mapValue(target.ResourceDefaults, "maxActiveWorkflows", "max_active_workflows"); ok {
		if n, ok := intValue(raw); ok && n >= 0 {
			return n
		}
	}
	return backpressureDefaultMaxActive
}

// intValue coerces a JSON-decoded value (number as float64, or a numeric
// string) to an int.
func intValue(raw interface{}) (int, bool) {
	switch v := raw.(type) {
	case float64:
		return int(v), true
	case float32:
		return int(v), true
	case int:
		return v, true
	case int64:
		return int(v), true
	case int32:
		return int(v), true
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return int(n), true
		}
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n, true
		}
	}
	return 0, false
}

func executionTargetTemplateNodeSelector(target *models.ExecutionTarget) map[string]string {
	out := map[string]string{}
	for key, value := range stringMapValue(jsonMapEnv("PIPELINE_TEMPLATE_NODE_SELECTOR_JSON")) {
		out[key] = value
	}
	for _, source := range executionTargetSchedulingMaps(target) {
		raw, ok := mapValue(source, "templateNodeSelector", "nodeSelector", "nodeSelectors")
		if !ok {
			continue
		}
		for key, value := range stringMapValue(raw) {
			out[key] = value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// executionTargetGpuStepNodeSelector reads a GPU-step-only nodeSelector from
// the target's scheduling config (resource_defaults key "gpuStepNodeSelector"
// or "gpuNodeSelector"). The transpiler merges these labels onto GPU steps
// only (guarded by requiresGPU), so a pool can pin its GPU workload to a
// dedicated GPU node pool without disturbing sibling CPU steps in the same
// pipeline. Empty = no-op = old behavior.
func executionTargetGpuStepNodeSelector(target *models.ExecutionTarget) map[string]string {
	out := map[string]string{}
	for key, value := range stringMapValue(jsonMapEnv("PIPELINE_GPU_STEP_NODE_SELECTOR_JSON")) {
		out[key] = value
	}
	for _, source := range executionTargetSchedulingMaps(target) {
		raw, ok := mapValue(source, "gpuStepNodeSelector", "gpuNodeSelector")
		if !ok {
			continue
		}
		for key, value := range stringMapValue(raw) {
			out[key] = value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func executionTargetTemplateTolerations(target *models.ExecutionTarget) []corev1.Toleration {
	var out []corev1.Toleration
	for _, item := range mapSliceValue(jsonMapSliceEnv("PIPELINE_TEMPLATE_TOLERATIONS_JSON")) {
		toleration, ok := tolerationFromMap(item)
		if !ok {
			continue
		}
		if !containsToleration(out, toleration) {
			out = append(out, toleration)
		}
	}
	for _, source := range executionTargetSchedulingMaps(target) {
		raw, ok := mapValue(source, "templateTolerations", "tolerations")
		if !ok {
			continue
		}
		for _, item := range mapSliceValue(raw) {
			toleration, ok := tolerationFromMap(item)
			if !ok {
				continue
			}
			if !containsToleration(out, toleration) {
				out = append(out, toleration)
			}
		}
	}
	return out
}

// executionTargetSchedulerName reads the pod scheduler for the pool from the
// target's scheduling config (resource_defaults / quota_policy JSONB, or their
// "scheduling" sub-object). Empty → transpiler leaves it unset → the cluster's
// default scheduler runs the pods (scheduler-agnostic pool).
//
// CYB-3486 pool.3: fully data-driven — the backend hardcodes NO scheduler name.
// A Koordinator pool works because its config carries the literal
// "koord-scheduler"; the backend doesn't know or care what koord is.
func executionTargetSchedulerName(target *models.ExecutionTarget) string {
	for _, source := range executionTargetSchedulingMaps(target) {
		if raw, ok := mapValue(source, "schedulerName", "scheduler"); ok {
			if s := stringValue(raw); s != "" {
				return s
			}
		}
	}
	return ""
}

// executionTargetPriorityClassName reads the pool's PriorityClass from the
// scheduling config. Empty → K8s global default. Data-driven (no hardcoded
// class name).
func executionTargetPriorityClassName(target *models.ExecutionTarget) string {
	for _, source := range executionTargetSchedulingMaps(target) {
		if raw, ok := mapValue(source, "priorityClassName", "priorityClass"); ok {
			if s := stringValue(raw); s != "" {
				return s
			}
		}
	}
	return ""
}

// executionTargetPodLabels reads pool pod labels from the scheduling config
// (e.g. a Koordinator ElasticQuota label). Data-driven: the pool config
// supplies BOTH the label key and value — the backend hardcodes neither koord
// nor any quota-label key. Later sources win on key conflicts.
func executionTargetPodLabels(target *models.ExecutionTarget) map[string]string {
	out := map[string]string{}
	for _, source := range executionTargetSchedulingMaps(target) {
		if raw, ok := mapValue(source, "podLabels"); ok {
			for k, v := range stringMapValue(raw) {
				out[k] = v
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// executionTargetPodAnnotations reads pool pod annotations from the scheduling
// config. Same data-driven contract as executionTargetPodLabels.
func executionTargetPodAnnotations(target *models.ExecutionTarget) map[string]string {
	out := map[string]string{}
	for _, source := range executionTargetSchedulingMaps(target) {
		if raw, ok := mapValue(source, "podAnnotations"); ok {
			for k, v := range stringMapValue(raw) {
				out[k] = v
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func jsonMapEnv(envName string) map[string]interface{} {
	raw := strings.TrimSpace(os.Getenv(envName))
	if raw == "" {
		return nil
	}
	var values map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil
	}
	return values
}

func jsonMapSliceEnv(envName string) []map[string]interface{} {
	raw := strings.TrimSpace(os.Getenv(envName))
	if raw == "" {
		return nil
	}
	var values []map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil
	}
	return values
}

func executionTargetSchedulingMaps(target *models.ExecutionTarget) []map[string]interface{} {
	if target == nil {
		return nil
	}
	var out []map[string]interface{}
	for _, source := range []map[string]interface{}{target.ResourceDefaults, target.QuotaPolicy} {
		if len(source) == 0 {
			continue
		}
		out = append(out, source)
		if nested, ok := mapValue(source, "scheduling", "scheduler", "runtimeScheduling"); ok {
			if nestedMap, ok := nested.(map[string]interface{}); ok {
				out = append(out, nestedMap)
			}
		}
	}
	return out
}

func stringMapValue(raw interface{}) map[string]string {
	values, ok := raw.(map[string]interface{})
	if !ok {
		return nil
	}
	out := map[string]string{}
	for key, rawValue := range values {
		key = strings.TrimSpace(key)
		value := stringValue(rawValue)
		if key == "" || value == "" {
			continue
		}
		out[key] = value
	}
	return out
}

func mapSliceValue(raw interface{}) []map[string]interface{} {
	switch values := raw.(type) {
	case []interface{}:
		out := make([]map[string]interface{}, 0, len(values))
		for _, item := range values {
			if itemMap, ok := item.(map[string]interface{}); ok {
				out = append(out, itemMap)
			}
		}
		return out
	case []map[string]interface{}:
		return values
	default:
		return nil
	}
}

func tolerationFromMap(values map[string]interface{}) (corev1.Toleration, bool) {
	key := stringValue(values["key"])
	operator := tolerationOperator(stringValue(values["operator"]), stringValue(values["value"]))
	if key == "" && operator != corev1.TolerationOpExists {
		return corev1.Toleration{}, false
	}
	return corev1.Toleration{
		Key:      key,
		Operator: operator,
		Value:    stringValue(values["value"]),
		Effect:   tolerationEffect(stringValue(values["effect"])),
	}, true
}

func stringValue(raw interface{}) string {
	switch value := raw.(type) {
	case string:
		return strings.TrimSpace(value)
	case float64, float32, int, int64, int32, uint, uint64, uint32:
		if formatted, ok := firstPolicyString(map[string]interface{}{"value": value}, "value"); ok {
			return formatted
		}
	}
	return ""
}

func tolerationOperator(value, tolerationValue string) corev1.TolerationOperator {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "exists":
		return corev1.TolerationOpExists
	default:
		if strings.TrimSpace(tolerationValue) == "" {
			return corev1.TolerationOpExists
		}
		return corev1.TolerationOpEqual
	}
}

func tolerationEffect(value string) corev1.TaintEffect {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "noexecute":
		return corev1.TaintEffectNoExecute
	case "prefernoschedule":
		return corev1.TaintEffectPreferNoSchedule
	default:
		return corev1.TaintEffectNoSchedule
	}
}

func containsToleration(items []corev1.Toleration, candidate corev1.Toleration) bool {
	for _, item := range items {
		if item.Key == candidate.Key &&
			item.Operator == candidate.Operator &&
			item.Value == candidate.Value &&
			item.Effect == candidate.Effect {
			return true
		}
	}
	return false
}
