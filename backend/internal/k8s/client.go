package k8s

import (
	"fmt"

	argowfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	// Load default auth providers (gcp, oidc, etc.)
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// Client holds K8s and Argo Workflow clientsets along with the target namespace.
type Client struct {
	Namespace      string
	KubeClientset  kubernetes.Interface
	ArgoClientset  argowfclientset.Interface
}

// NewClient creates a Client by loading a rest.Config from the given kubeconfig
// path or, if empty, by trying in-cluster config and then the default kubeconfig
// search path.
func NewClient(kubeconfigPath, namespace string) (*Client, error) {
	config, err := buildConfig(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("build rest config: %w", err)
	}

	kubeClientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create kubernetes clientset: %w", err)
	}

	argoClientset, err := argowfclientset.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create argo workflow clientset: %w", err)
	}

	return &Client{
		Namespace:     namespace,
		KubeClientset: kubeClientset,
		ArgoClientset: argoClientset,
	}, nil
}

// buildConfig resolves a rest.Config using the provided kubeconfig path.
//   - If kubeconfigPath is non-empty, it is used explicitly.
//   - Otherwise, InClusterConfig is tried first, falling back to the default
//     kubeconfig search path (~/.kube/config).
func buildConfig(kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}

	// Try in-cluster first (for when this runs inside a pod).
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// Fall back to the default kubeconfig loading rules.
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		configOverrides,
	).ClientConfig()
}
