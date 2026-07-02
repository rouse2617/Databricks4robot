package config

import (
	"strings"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/transpiler"
)

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

func TestLoadArgoWorkflowTTLConfig(t *testing.T) {
	t.Setenv("ARGO_WORKFLOW_TTL_SECONDS_AFTER_COMPLETION", "7200")
	cfg := Load()
	if cfg.ArgoWorkflowTTLSecondsAfterCompletion != 7200 {
		t.Fatalf("unexpected ttl: %d", cfg.ArgoWorkflowTTLSecondsAfterCompletion)
	}

	t.Setenv("ARGO_WORKFLOW_TTL_SECONDS_AFTER_COMPLETION", "not-a-number")
	cfg = Load()
	if cfg.ArgoWorkflowTTLSecondsAfterCompletion != transpiler.DefaultTTLSecondsAfterCompletion {
		t.Fatalf("expected default ttl fallback, got %d", cfg.ArgoWorkflowTTLSecondsAfterCompletion)
	}
}

func TestLoadPipelineResourceGuardConfig(t *testing.T) {
	t.Setenv("PIPELINE_RESOURCE_MAX_CPU", "8")
	t.Setenv("PIPELINE_RESOURCE_MAX_MEMORY", "28Gi")
	t.Setenv("PIPELINE_RESOURCE_MAX_DISK", "250Gi")
	t.Setenv("PIPELINE_RESOURCE_MAX_GPU", "1")
	t.Setenv("PIPELINE_UNSCHEDULABLE_PENDING_THRESHOLD", "7m")

	cfg := Load()
	if cfg.PipelineResourceMaxCPU != "8" ||
		cfg.PipelineResourceMaxMemory != "28Gi" ||
		cfg.PipelineResourceMaxDisk != "250Gi" ||
		cfg.PipelineResourceMaxGPU != "1" {
		t.Fatalf("unexpected resource guard config: %#v", cfg)
	}
	if got := cfg.PipelineUnschedulablePendingThresholdDuration(); got != 7*time.Minute {
		t.Fatalf("unexpected pending threshold: %s", got)
	}

	t.Setenv("PIPELINE_UNSCHEDULABLE_PENDING_THRESHOLD", "invalid")
	cfg = Load()
	if got := cfg.PipelineUnschedulablePendingThresholdDuration(); got != 15*time.Minute {
		t.Fatalf("expected default pending threshold, got %s", got)
	}
}

func TestValidate_NonProductionSkipsChecks(t *testing.T) {
	for _, env := range []string{"development", "staging", "test", ""} {
		cfg := &Config{Env: env, DatabrewToken: devDefaultDatabrewToken, JWTSecret: devDefaultJWTSecret}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("env=%q: expected nil error for non-production, got %v", env, err)
		}
	}
}

func TestValidate_ProductionRejectsDevDefaults(t *testing.T) {
	cfg := &Config{
		Env:           "production",
		DatabrewToken: devDefaultDatabrewToken,
		JWTSecret:     devDefaultJWTSecret,
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for production with dev-default secrets, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "DATABREW_TOKEN") {
		t.Errorf("expected error to mention DATABREW_TOKEN, got: %s", msg)
	}
	if !strings.Contains(msg, "JWT_SECRET") {
		t.Errorf("expected error to mention JWT_SECRET, got: %s", msg)
	}
}

func TestValidate_ProductionRejectsEmptySecrets(t *testing.T) {
	cfg := &Config{Env: "production", DatabrewToken: "", JWTSecret: ""}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for production with empty secrets, got nil")
	}
}

func TestValidate_ProductionAcceptsRealSecrets(t *testing.T) {
	cfg := &Config{
		Env:           "production",
		DatabrewToken: "a-real-random-token-value",
		JWTSecret:     "a-real-random-jwt-secret-value",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected nil error for production with real secrets, got %v", err)
	}
}

func TestValidate_NilConfig(t *testing.T) {
	var cfg *Config
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for nil config, got nil")
	}
}
