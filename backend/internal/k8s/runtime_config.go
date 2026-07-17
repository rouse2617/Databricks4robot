package k8s

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

type RuntimeConfigStore struct {
	clientset kubernetes.Interface
}

func NewRuntimeConfigStore(clientset kubernetes.Interface) *RuntimeConfigStore {
	return &RuntimeConfigStore{clientset: clientset}
}

// RuntimeConfigStoreFactory resolves a per-cluster RuntimeConfigStore from a
// ClientFactory, so the runtime-config ConfigMap is created on the target's own
// cluster rather than the default-cluster singleton (CYB-3486). Implements
// pipelineUC.RuntimeConfigStoreFactory.
type RuntimeConfigStoreFactory struct {
	clients ClientFactory
}

// NewRuntimeConfigStoreFactory wires a per-cluster runtime-config store factory
// over the shared K8s ClientFactory.
func NewRuntimeConfigStoreFactory(clients ClientFactory) *RuntimeConfigStoreFactory {
	return &RuntimeConfigStoreFactory{clients: clients}
}

// ForTarget returns a RuntimeConfigStore bound to the target's cluster clientset.
func (f *RuntimeConfigStoreFactory) ForTarget(ctx context.Context, target *models.ExecutionTarget) (pipelineUC.RuntimeConfigStore, error) {
	if f == nil || f.clients == nil {
		return nil, fmt.Errorf("%w: runtime config store factory is not configured", ErrUnavailable)
	}
	clientset, err := f.clients.ForTarget(ctx, target)
	if err != nil {
		return nil, err
	}
	return NewRuntimeConfigStore(clientset), nil
}

var invalidConfigMapNameChars = regexp.MustCompile(`[^a-z0-9-]+`)

func (s *RuntimeConfigStore) Create(ctx context.Context, namespace string, deploymentID string, input pipelineUC.RuntimeConfigProjection, owner *pipelineUC.RuntimeConfigOwnerReference) (string, error) {
	if s == nil || s.clientset == nil {
		return "", fmt.Errorf("%w: runtime config store is not configured", ErrUnavailable)
	}
	namespace = strings.TrimSpace(namespace)
	deploymentID = strings.TrimSpace(deploymentID)
	volumeName := strings.TrimSpace(input.VolumeName)
	if volumeName == "" {
		volumeName = "runtime-config-" + deploymentID
	}
	fileName := strings.TrimSpace(input.FileName)
	files := make(map[string]string)
	for name, content := range input.Files {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		files[name] = content
	}
	if fileName != "" {
		files[fileName] = input.Content
	}
	if namespace == "" || deploymentID == "" || len(files) == 0 {
		return "", fmt.Errorf("namespace, deploymentID, and at least one config file are required")
	}
	name := sanitizeRuntimeConfigName(volumeName)
	ownerRefs := []metav1.OwnerReference(nil)
	if owner != nil && owner.Name != "" && owner.UID != "" {
		ownerRefs = append(ownerRefs, metav1.OwnerReference{
			APIVersion: owner.APIVersion,
			Kind:       owner.Kind,
			Name:       owner.Name,
			UID:        types.UID(owner.UID),
		})
	}
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			OwnerReferences: ownerRefs,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by":  "cyber-databrew",
				"cyberorigin.ai/runtime-config": "true",
			},
		},
		Data: files,
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
