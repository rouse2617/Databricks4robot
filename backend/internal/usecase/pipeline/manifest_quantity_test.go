package pipeline

import (
	"testing"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	sigsyaml "sigs.k8s.io/yaml"
)

// Covers spec scenario: "终态 run 的 pipeline 声明了资源 requests/limits".
// Regression test for CYB-3065: yaml.v3's reflection-based Marshal cannot see
// resource.Quantity's private fields and silently drops the numeric value,
// leaving only the exported Format field — sigsyaml.Marshal (which round-trips
// through encoding/json, calling Quantity's MarshalJSON) must not have this bug.
func TestManifestSerialization_PreservesResourceQuantities(t *testing.T) {
	wf := &wfv1.Workflow{}
	wf.Name = "manifest-quantity-test"
	wf.Spec.Entrypoint = "step-a"
	wf.Spec.Templates = []wfv1.Template{{
		Name: "step-a",
		Container: &corev1.Container{
			Image: "busybox",
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("256Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("512Mi"),
				},
			},
		},
	}}

	manifestBytes, err := sigsyaml.Marshal(wf)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got wfv1.Workflow
	if err := sigsyaml.Unmarshal(manifestBytes, &got); err != nil {
		t.Fatalf("Unmarshal round-trip failed (this is the CYB-3065 bug if using yaml.v3): %v", err)
	}

	res := got.Spec.Templates[0].Container.Resources
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"cpu request", res.Requests.Cpu().String(), "500m"},
		{"memory request", res.Requests.Memory().String(), "256Mi"},
		{"cpu limit", res.Limits.Cpu().String(), "1"},
		{"memory limit", res.Limits.Memory().String(), "512Mi"},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("%s = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}
