package k8s

import (
	"errors"
	"fmt"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var ErrUnavailable = errors.New("kubernetes client unavailable")

// NewClientset builds a Kubernetes client for DataBrew Pod diagnostics.
// Preferred production configuration is a backend-held GCP/GKE identity with
// namespace-scoped RBAC. K8S_BEARER_TOKEN is only a temporary dev bridge.
func NewClientset(kubeconfigPath string) (kubernetes.Interface, error) {
	config, err := buildConfig(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("%w: create clientset: %v", ErrUnavailable, err)
	}
	return clientset, nil
}

func buildConfig(kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}

	if config, err := rest.InClusterConfig(); err == nil {
		return config, nil
	}

	token := os.Getenv("K8S_BEARER_TOKEN")
	endpoint := os.Getenv("K8S_API_ENDPOINT")
	if token != "" && endpoint != "" {
		return &rest.Config{
			Host:        endpoint,
			BearerToken: token,
			TLSClientConfig: rest.TLSClientConfig{
				CAFile: os.Getenv("K8S_CA_FILE"),
			},
		}, nil
	}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err == nil {
		return config, nil
	}

	return nil, fmt.Errorf("no in-cluster, explicit token, or kubeconfig configuration: %w", err)
}
