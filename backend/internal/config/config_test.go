package config

import "testing"

func TestLoadOpenLineageConfig(t *testing.T) {
	t.Setenv("OPENLINEAGE_EMITTER_ENABLED", "true")
	t.Setenv("OPENLINEAGE_ENDPOINT", "http://marquez.example/api/v1/lineage")
	t.Setenv("OPENLINEAGE_SUBSCRIPTION", "asset-events-openlineage")
	t.Setenv("OPENLINEAGE_NAMESPACE", "cyber-databrew-dev")
	t.Setenv("OPENLINEAGE_PRODUCER", "test-producer")
	t.Setenv("OPENLINEAGE_TIMEOUT_MS", "1234")

	cfg := Load()
	if cfg.OpenLineageEmitterEnabled != "true" {
		t.Fatalf("unexpected OpenLineageEmitterEnabled: %q", cfg.OpenLineageEmitterEnabled)
	}
	if cfg.OpenLineageEndpoint != "http://marquez.example/api/v1/lineage" {
		t.Fatalf("unexpected OpenLineageEndpoint: %q", cfg.OpenLineageEndpoint)
	}
	if cfg.OpenLineageSubscription != "asset-events-openlineage" {
		t.Fatalf("unexpected OpenLineageSubscription: %q", cfg.OpenLineageSubscription)
	}
	if cfg.OpenLineageNamespace != "cyber-databrew-dev" || cfg.OpenLineageProducer != "test-producer" {
		t.Fatalf("unexpected namespace/producer: %q %q", cfg.OpenLineageNamespace, cfg.OpenLineageProducer)
	}
	if cfg.OpenLineageTimeoutMs != "1234" {
		t.Fatalf("unexpected OpenLineageTimeoutMs: %q", cfg.OpenLineageTimeoutMs)
	}
}
