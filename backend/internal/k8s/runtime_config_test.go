package k8s

import (
	"context"
	"testing"

	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRuntimeConfigStoreCreateAddsOwnerReferenceAndFiles(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	store := NewRuntimeConfigStore(client)

	name, err := store.Create(ctx, "argo-ns", "dep-1", pipelineUC.RuntimeConfigProjection{
		VolumeName: "runtime-config-dep-1",
		Files: map[string]string{
			"01-node-a.yaml": "a: true\n",
			"02-node-b.yaml": "b: true\n",
		},
	}, &pipelineUC.RuntimeConfigOwnerReference{
		APIVersion: "argoproj.io/v1alpha1",
		Kind:       "Workflow",
		Name:       "wf-1",
		UID:        "wf-uid",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if name != "runtime-config-dep-1" {
		t.Fatalf("name = %q, want runtime-config-dep-1", name)
	}

	got, err := client.CoreV1().ConfigMaps("argo-ns").Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get configmap: %v", err)
	}
	if got.Data["01-node-a.yaml"] != "a: true\n" || got.Data["02-node-b.yaml"] != "b: true\n" {
		t.Fatalf("unexpected data: %#v", got.Data)
	}
	if len(got.OwnerReferences) != 1 {
		t.Fatalf("owner refs = %#v", got.OwnerReferences)
	}
	owner := got.OwnerReferences[0]
	if owner.Kind != "Workflow" || owner.Name != "wf-1" || owner.UID != types.UID("wf-uid") {
		t.Fatalf("unexpected owner ref: %#v", owner)
	}
}
