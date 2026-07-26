package pipeline

import "testing"

func TestActiveWorkflowObservationKeyIsolatesClusters(t *testing.T) {
	uc := &Usecase{}
	uc.recordActiveWorkflowCount("cluster-a", "argo", 150)
	uc.recordActiveWorkflowCount("cluster-b", "argo", 10)

	for _, tc := range []struct {
		cluster string
		want    int
	}{
		{cluster: "cluster-a", want: 150},
		{cluster: "cluster-b", want: 10},
	} {
		got, known := uc.ActiveWorkflowCount(tc.cluster, "argo")
		if !known || got != tc.want {
			t.Errorf("ActiveWorkflowCount(%q, argo) = (%d, %v), want (%d, true)", tc.cluster, got, known, tc.want)
		}
	}

	if got, known := uc.ActiveWorkflowCount("cluster-c", "argo"); known {
		t.Fatalf("unknown cluster observation = (%d, true), want (_, false)", got)
	}
}

func TestActiveWorkflowObservationKeyNormalizesDefaultAliases(t *testing.T) {
	uc := &Usecase{}
	uc.recordActiveWorkflowCount("cluster-default", "argo", 42)

	for _, cluster := range []string{"cluster-default", "default", ""} {
		got, known := uc.ActiveWorkflowCount(cluster, "argo")
		if !known || got != 42 {
			t.Errorf("ActiveWorkflowCount(%q, argo) = (%d, %v), want (42, true)", cluster, got, known)
		}
	}

	uc.recordActiveWorkflowCount("", "argo", 84)
	if got, known := uc.ActiveWorkflowCount("default", "argo"); !known || got != 84 {
		t.Fatalf("empty alias update read through default = (%d, %v), want (84, true)", got, known)
	}
}
