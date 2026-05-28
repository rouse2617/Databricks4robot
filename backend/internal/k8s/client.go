package k8s

import (
	"encoding/base64"
	"fmt"
	"os"

	argowfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"

	// Load default auth providers (gcp, oidc, etc.)
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// Client holds K8s, Argo Workflow, and Metrics clientsets.
type Client struct {
	Namespace        string
	KubeClientset    kubernetes.Interface
	ArgoClientset    argowfclientset.Interface
	MetricsClientset metricsclientset.Interface
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

	metricsClientset, err := metricsclientset.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create metrics clientset: %w", err)
	}

	return &Client{
		Namespace:        namespace,
		KubeClientset:    kubeClientset,
		ArgoClientset:    argoClientset,
		MetricsClientset: metricsClientset,
	}, nil
}

// buildConfig resolves a rest.Config using the provided kubeconfig path.
//   - If kubeconfigPath is non-empty, it is used explicitly.
//   - Otherwise, InClusterConfig is tried first, falling back to the default
//     kubeconfig search path (~/.kube/config).
//   - As a final fallback, K8S_BEARER_TOKEN + K8S_API_ENDPOINT env vars are
//     used (e.g. on Cloud Run where neither kubeconfig nor in-cluster works).
//     K8S_CA_CERT_BASE64 (base64-encoded CA cert) is optional; if omitted the
//     system CA pool is used.
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
	config, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		configOverrides,
	).ClientConfig()
	if err == nil {
		return config, nil
	}

	// Final fallback: bearer token from env vars (Cloud Run with no kubeconfig).
	token := os.Getenv("K8S_BEARER_TOKEN")
	endpoint := os.Getenv("K8S_API_ENDPOINT")
	if token != "" && endpoint != "" {
		tlsConfig := rest.TLSClientConfig{Insecure: false}
		if ca := os.Getenv("K8S_CA_CERT_BASE64"); ca != "" {
			if raw, err := base64.StdEncoding.DecodeString(ca); err == nil {
				tlsConfig.CAData = raw
			}
		}
		return &rest.Config{
			Host:            endpoint,
			BearerToken:     token,
			TLSClientConfig: tlsConfig,
		}, nil
	}

	return nil, err
}
