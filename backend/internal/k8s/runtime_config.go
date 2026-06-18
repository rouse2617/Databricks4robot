package k8s

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type RuntimeConfigStore struct {
	clientset kubernetes.Interface
}

func NewRuntimeConfigStore(clientset kubernetes.Interface) *RuntimeConfigStore {
	return &RuntimeConfigStore{clientset: clientset}
}

var invalidConfigMapNameChars = regexp.MustCompile(`[^a-z0-9-]+`)

func (s *RuntimeConfigStore) Create(ctx context.Context, namespace string, deploymentID string, input pipelineUC.RuntimeConfigProjection) (string, error) {
	if s == nil || s.clientset == nil {
		return "", fmt.Errorf("%w: runtime config store is not configured", ErrUnavailable)
	}
	namespace = strings.TrimSpace(namespace)
	deploymentID = strings.TrimSpace(deploymentID)
	fileName := strings.TrimSpace(input.FileName)
	if namespace == "" || deploymentID == "" || fileName == "" {
		return "", fmt.Errorf("namespace, deploymentID, and fileName are required")
	}
	name := sanitizeRuntimeConfigName("runtime-config-" + deploymentID)
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "cyber-databrew",
				"cyberorigin.ai/runtime-config": "true",
			},
		},
		Data: map[string]string{
			fileName: input.Content,
		},
	}
	_, err := s.clientset.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return name, nil
		}
		return "", err
	}
	return name, nil
}

func sanitizeRuntimeConfigName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = invalidConfigMapNameChars.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "runtime-config"
	}
	if len(value) > 63 {
		value = strings.Trim(value[:63], "-")
	}
	if value == "" {
		return "runtime-config"
	}
	return value
}
