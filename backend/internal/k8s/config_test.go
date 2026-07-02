package k8s

import (
	"encoding/base64"
	"strings"
	"testing"
)

const testCA = "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----"

func TestBuildConfigUsesCADataForExplicitBearerTarget(t *testing.T) {
	t.Setenv("K8S_API_ENDPOINT", "https://example.invalid")
	t.Setenv("K8S_BEARER_TOKEN", "token")
	t.Setenv("K8S_CA_DATA", base64.StdEncoding.EncodeToString([]byte(testCA)))
	t.Setenv("K8S_CA_FILE", "/should/not/be/used.crt")

	config, err := buildConfig("")
	if err != nil {
		t.Fatalf("buildConfig: %v", err)
	}

	if string(config.TLSClientConfig.CAData) != testCA {
		t.Fatalf("expected CAData from K8S_CA_DATA, got %q", string(config.TLSClientConfig.CAData))
	}
	if config.TLSClientConfig.CAFile != "" {
		t.Fatalf("expected K8S_CA_DATA to take precedence over K8S_CA_FILE, got %q", config.TLSClientConfig.CAFile)
	}
}

func TestBuildConfigAcceptsPEMCADataForExplicitBearerTarget(t *testing.T) {
	t.Setenv("K8S_API_ENDPOINT", "https://example.invalid")
	t.Setenv("K8S_BEARER_TOKEN", "token")
	t.Setenv("K8S_CA_DATA", testCA)

	config, err := buildConfig("")
	if err != nil {
		t.Fatalf("buildConfig: %v", err)
	}

	if string(config.TLSClientConfig.CAData) != testCA {
		t.Fatalf("expected raw PEM CAData, got %q", string(config.TLSClientConfig.CAData))
	}
}

func TestBuildConfigRejectsInvalidCAData(t *testing.T) {
	t.Setenv("K8S_API_ENDPOINT", "https://example.invalid")
	t.Setenv("K8S_BEARER_TOKEN", "token")
	t.Setenv("K8S_CA_DATA", "not-base64-or-pem")

	_, err := buildConfig("")
	if err == nil {
		t.Fatal("expected invalid K8S_CA_DATA to fail")
	}
	if !strings.Contains(err.Error(), "K8S_CA_DATA") {
		t.Fatalf("expected K8S_CA_DATA error, got %v", err)
	}
}

func TestBuildConfigFallsBackToCAFile(t *testing.T) {
	t.Setenv("K8S_API_ENDPOINT", "https://example.invalid")
	t.Setenv("K8S_BEARER_TOKEN", "token")
	t.Setenv("K8S_CA_FILE", "/var/secrets/k8s/ca.crt")

	config, err := buildConfig("")
	if err != nil {
		t.Fatalf("buildConfig: %v", err)
	}

	if config.TLSClientConfig.CAFile != "/var/secrets/k8s/ca.crt" {
		t.Fatalf("expected CAFile fallback, got %q", config.TLSClientConfig.CAFile)
	}
}
