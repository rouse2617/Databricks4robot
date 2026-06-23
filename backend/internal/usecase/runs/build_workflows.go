package runs

import (
	"fmt"
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const buildRunnerImage = "busybox:1.36"

func buildComponentBuildWorkflow(
	workflowName, namespace, runID string,
	in CreateComponentBuildInput,
) *wfv1.Workflow {
	imageRef := strings.TrimSpace(in.ImageRepository)
	if tag := strings.TrimSpace(in.ImageTag); tag != "" && imageRef != "" {
		if !strings.Contains(imageRef, ":") {
			imageRef = fmt.Sprintf("%s:%s", imageRef, tag)
		}
	}
	script := fmt.Sprintf(`set -e
echo "component=%s"
echo "repo=%s"
echo "ref=%s"
echo "dockerfile=%s"
echo "context=%s"
echo "image=%s"
echo "run_id=%s"
`, in.ComponentName, in.RepoURL, in.GitRef, in.Dockerfile, in.BuildContext, imageRef, runID)

	return &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workflowName,
			Namespace: namespace,
			Labels: map[string]string{
				"databrew.run.type": "component_build",
				"databrew.run.id":   runID,
				"component-id":      in.ComponentID,
			},
		},
		Spec: wfv1.WorkflowSpec{
			Entrypoint: "component-build",
			Templates: []wfv1.Template{
				{
					Name: "component-build",
					Steps: []wfv1.ParallelSteps{
						{Steps: []wfv1.WorkflowStep{{Name: "checkout", Template: "checkout"}}},
						{Steps: []wfv1.WorkflowStep{{Name: "build", Template: "build"}}},
						{Steps: []wfv1.WorkflowStep{{Name: "register", Template: "register"}}},
					},
				},
				shellTemplate("checkout", fmt.Sprintf("echo checkout %s@%s", in.RepoURL, in.GitRef)),
				shellTemplate("build", script),
				shellTemplate("register", fmt.Sprintf("echo register release for %s", in.ComponentName)),
			},
		},
	}
}

func buildRAGBuildWorkflow(
	workflowName, namespace, runID string,
	in CreateRAGBuildInput,
) *wfv1.Workflow {
	script := fmt.Sprintf(`set -e
echo "knowledge_base=%s"
echo "embedding_model=%s"
echo "vector_index=%s"
echo "release_version=%s"
echo "run_id=%s"
`, in.KnowledgeBaseID, in.EmbeddingModel, in.VectorIndexName, in.ReleaseVersion, runID)

	return &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workflowName,
			Namespace: namespace,
			Labels: map[string]string{
				"databrew.run.type": "rag_build",
				"databrew.run.id":   runID,
				"knowledge-base-id": in.KnowledgeBaseID,
			},
		},
		Spec: wfv1.WorkflowSpec{
			Entrypoint: "rag-build",
			Templates: []wfv1.Template{
				{
					Name: "rag-build",
					Steps: []wfv1.ParallelSteps{
						{Steps: []wfv1.WorkflowStep{{Name: "load", Template: "load-datasource"}}},
						{Steps: []wfv1.WorkflowStep{{Name: "chunk", Template: "chunk"}}},
						{Steps: []wfv1.WorkflowStep{{Name: "embed", Template: "embedding"}}},
						{Steps: []wfv1.WorkflowStep{{Name: "publish", Template: "publish-index"}}},
					},
				},
				shellTemplate("load-datasource", fmt.Sprintf("echo load datasource for %s", in.KnowledgeBaseID)),
				shellTemplate("chunk", "echo chunk documents"),
				shellTemplate("embedding", script),
				shellTemplate("publish-index", fmt.Sprintf("echo publish index %s", in.VectorIndexName)),
			},
		},
	}
}

func shellTemplate(name, script string) wfv1.Template {
	return wfv1.Template{
		Name: name,
		Container: &corev1.Container{
			Image:   buildRunnerImage,
			Command: []string{"sh", "-c"},
			Args:    []string{script},
		},
	}
}
