package pipeline

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

// These pin the default/edge coercion scheduling.go applies when the
// user-editable resourceDefaults.scheduling JSONB omits or mistypes a
// toleration field. scheduling_test.go covers the operator="Equal" /
// effect="NoSchedule" happy path; this covers the fallbacks so a refactor
// can't silently change how an omitted field maps to k8s scheduling. CYB-4268.

func TestTolerationOperator(t *testing.T) {
	tests := []struct {
		name  string
		op    string
		value string
		want  corev1.TolerationOperator
	}{
		{"explicit exists", "Exists", "", corev1.TolerationOpExists},
		{"exists is case-insensitive and trimmed", "  eXiStS ", "v", corev1.TolerationOpExists},
		{"explicit equal with value", "Equal", "dev", corev1.TolerationOpEqual},
		{"omitted operator + value -> equal", "", "dev", corev1.TolerationOpEqual},
		{"omitted operator + no value -> exists", "", "", corev1.TolerationOpExists},
		{"whitespace operator + whitespace value -> exists", "  ", "   ", corev1.TolerationOpExists},
		{"unknown operator + value -> equal", "garbage", "dev", corev1.TolerationOpEqual},
		{"unknown operator + no value -> exists", "garbage", "", corev1.TolerationOpExists},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tolerationOperator(tt.op, tt.value); got != tt.want {
				t.Errorf("tolerationOperator(%q, %q) = %q, want %q", tt.op, tt.value, got, tt.want)
			}
		})
	}
}

func TestTolerationEffect(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  corev1.TaintEffect
	}{
		{"noexecute", "NoExecute", corev1.TaintEffectNoExecute},
		{"prefernoschedule is case-insensitive", "preferNoSchedule", corev1.TaintEffectPreferNoSchedule},
		{"noschedule explicit", "NoSchedule", corev1.TaintEffectNoSchedule},
		{"omitted -> noschedule", "", corev1.TaintEffectNoSchedule},
		{"unknown -> noschedule", "banana", corev1.TaintEffectNoSchedule},
		{"whitespace + case still resolves", "  noexecute  ", corev1.TaintEffectNoExecute},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tolerationEffect(tt.value); got != tt.want {
				t.Errorf("tolerationEffect(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestTolerationFromMap(t *testing.T) {
	t.Run("full map builds the toleration", func(t *testing.T) {
		got, ok := tolerationFromMap(map[string]interface{}{
			"key": "nvidia.com/gpu", "operator": "Equal", "value": "present", "effect": "NoSchedule",
		})
		if !ok {
			t.Fatal("expected ok=true")
		}
		want := corev1.Toleration{
			Key: "nvidia.com/gpu", Operator: corev1.TolerationOpEqual,
			Value: "present", Effect: corev1.TaintEffectNoSchedule,
		}
		if got != want {
			t.Errorf("got %#v, want %#v", got, want)
		}
	})
	t.Run("keyless + non-exists operator is dropped", func(t *testing.T) {
		// no key, value present -> operator resolves to Equal -> invalid, skipped.
		if _, ok := tolerationFromMap(map[string]interface{}{"value": "dev", "effect": "NoSchedule"}); ok {
			t.Error("expected ok=false for a keyless Equal toleration")
		}
	})
	t.Run("keyless + exists operator is kept (tolerate all)", func(t *testing.T) {
		got, ok := tolerationFromMap(map[string]interface{}{"operator": "Exists"})
		if !ok {
			t.Fatal("expected ok=true for a keyless Exists toleration")
		}
		if got.Key != "" || got.Operator != corev1.TolerationOpExists {
			t.Errorf("got %#v, want keyless Exists", got)
		}
	})
	t.Run("defaults applied when operator/effect omitted", func(t *testing.T) {
		got, ok := tolerationFromMap(map[string]interface{}{"key": "environment", "value": "dev"})
		if !ok {
			t.Fatal("expected ok=true")
		}
		if got.Operator != corev1.TolerationOpEqual {
			t.Errorf("operator = %q, want Equal (op omitted, value present)", got.Operator)
		}
		if got.Effect != corev1.TaintEffectNoSchedule {
			t.Errorf("effect = %q, want NoSchedule (effect omitted)", got.Effect)
		}
	})
	t.Run("string values are trimmed", func(t *testing.T) {
		got, ok := tolerationFromMap(map[string]interface{}{"key": "  environment  ", "value": "  dev  "})
		if !ok || got.Key != "environment" || got.Value != "dev" {
			t.Errorf("expected trimmed key/value, got %#v (ok=%v)", got, ok)
		}
	})
}
