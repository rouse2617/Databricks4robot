package k8s

import (
	"errors"
	"fmt"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/transport"
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

	// Workload Identity / GCE metadata-server token path.
	// Preferred for Cloud Run + GKE production: backend's Cloud Run SA
	// impersonates the GKE KSA via Workload Identity; the access token is
	// minted per-request from the GCE metadata server bound to the KSA's
	// audience, so GKE accepts it as a ServiceAccountToken. The cached
	// transport wrapper refreshes the token on expiry.
	if os.Getenv("K8S_USE_METADATA_TOKEN") == "true" {
		audience := os.Getenv("K8S_AUDIENCE")
		endpoint := os.Getenv("K8S_API_ENDPOINT")
		if audience == "" || endpoint == "" {
			return nil, fmt.Errorf("K8S_USE_METADATA_TOKEN requires K8S_AUDIENCE and K8S_API_ENDPOINT")
		}
		cached := transport.NewCachedTokenSource(newMetadataTokenSource(audience))
		return &rest.Config{
			Host: endpoint,
			TLSClientConfig: rest.TLSClientConfig{
				CAFile: os.Getenv("K8S_CA_FILE"),
			},
			// WrapTransport runs after client-go's default bearer wrappers,
			// so our token wins. NewOAuth2RoundTripper injects Authorization
			// on every request and refreshes via the cached TokenSource.
			WrapTransport: newOauth2RoundTripper(cached),
		}, nil
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

	if config, err := rest.InClusterConfig(); err == nil {
		return config, nil
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
