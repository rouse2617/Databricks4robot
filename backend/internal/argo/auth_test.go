package argo

import "testing"

func TestConfigFromEnvServerURLFallback(t *testing.T) {
	t.Setenv("ARGO_BASE_URL", "https://argo-base.example.test")
	t.Setenv("ARGO_SERVER_URL", "")

	cfg := ConfigFromEnv()
	if cfg.ServerURL != "https://argo-base.example.test" {
		t.Fatalf("ServerURL = %q, want ARGO_BASE_URL fallback", cfg.ServerURL)
	}
}

func TestConfigFromEnvServerURLPrefersExplicitServerURL(t *testing.T) {
	t.Setenv("ARGO_BASE_URL", "https://argo-base.example.test")
	t.Setenv("ARGO_SERVER_URL", "https://argo-server.example.test")

	cfg := ConfigFromEnv()
	if cfg.ServerURL != "https://argo-server.example.test" {
		t.Fatalf("ServerURL = %q, want ARGO_SERVER_URL", cfg.ServerURL)
	}
}
