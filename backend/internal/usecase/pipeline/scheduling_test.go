package pipeline

import (
	"context"
	"strings"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

func TestDeploy_AppliesExecutionTargetSchedulingDefaults(t *testing.T) {
	ctx := context.Background()
	target := &models.ExecutionTarget{
		ID:             "video-proc-dev",
		Name:           "video-proc-dev",
		Namespace:      "video-proc-dev",
		ServiceAccount: "workflow-runner",
		Enabled:        true,
		Status:         "available",
		ResourceDefaults: map[string]interface{}{
			"scheduling": map[string]interface{}{
				"templateTolerations": []interface{}{
					map[string]interface{}{
						"key":      "environment",
						"operator": "Equal",
						"value":    "dev",
						"effect":   "NoSchedule",
					},
				},
			},
		},
	}
	targetRepo := &mockTargetRepo{byID: map[string]*models.ExecutionTarget{target.ID: target}}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, newMockAssetRepo(), nil, "default")
	uc.SetRunRepositories(targetRepo, nil, nil)

	pipe := map[string]interface{}{
		"name": "gpu-target",
		"nodes": []interface{}{map[string]interface{}{
			"id": "gpu-node",
			"component": map[string]interface{}{
				"name":    "gpu",
				"image":   "nvidia/cuda:12.4.1-base-ubuntu22.04",
				"command": []interface{}{"sh", "-c"},
				"args": []interface{}{map[string]interface{}{
					"name":  "script",
					"value": "nvidia-smi",
				}},
				"resources": map[string]interface{}{
					"cpu":         "1",
					"memory":      "1Gi",
					"gpu":         "1",
					"computeTier": "gpu-l4",
				},
			},
		}},
	}
	dep, err := uc.Deploy(ctx, pipe, "", nil, DeployOptions{DryRun: true, TargetID: target.ID})
	if err != nil {
		t.Fatalf("Deploy dry-run: %v", err)
	}
	if dep == nil || dep.Manifest == nil {
		t.Fatal("expected dry-run manifest")
	}
	manifest := *dep.Manifest
	for _, want := range []string{
		"serviceAccountName: workflow-runner",
		"namespace: video-proc-dev",
		"cloud.google.com/gke-accelerator: nvidia-l4",
		"key: nvidia.com/gpu",
		"value: present",
		"key: environment",
		"value: dev",
	} {
		if !strings.Contains(manifest, want) {
			t.Fatalf("expected manifest to contain %q, got %s", want, manifest)
		}
	}
}

func TestExecutionTargetGpuStepNodeSelector_ParsesResourceDefaults(t *testing.T) {
	target := &models.ExecutionTarget{
		ResourceDefaults: map[string]interface{}{
			"gpuStepNodeSelector": map[string]interface{}{
				"cloud.google.com/gke-nodepool": "g2-l4-dev-pool",
			},
		},
	}
	got := executionTargetGpuStepNodeSelector(target)
	if got["cloud.google.com/gke-nodepool"] != "g2-l4-dev-pool" {
		t.Fatalf("expected gpuStepNodeSelector parsed, got %#v", got)
	}
}

// The alias "gpuNodeSelector" is accepted so a caller doesn't have to guess
// which spelling — consistent with how templateNodeSelector accepts alternate
// keys elsewhere in this file.
func TestExecutionTargetGpuStepNodeSelector_AcceptsAlias(t *testing.T) {
	target := &models.ExecutionTarget{
		ResourceDefaults: map[string]interface{}{
			"gpuNodeSelector": map[string]interface{}{
				"key": "value",
			},
		},
	}
	got := executionTargetGpuStepNodeSelector(target)
	if got["key"] != "value" {
		t.Fatalf("expected gpuNodeSelector alias parsed, got %#v", got)
	}
}

func TestExecutionTargetGpuStepNodeSelector_EmptyReturnsNil(t *testing.T) {
	target := &models.ExecutionTarget{ResourceDefaults: map[string]interface{}{}}
	if got := executionTargetGpuStepNodeSelector(target); got != nil {
		t.Fatalf("expected nil for empty target, got %#v", got)
	}
}

// PIPELINE_GPU_STEP_NODE_SELECTOR_JSON is the escape hatch for setting the
// GPU-only nodeSelector at the process level, e.g. when the target's
// resource_defaults haven't been updated yet. A nil target must not crash and
// must still fall through to the env value.
func TestExecutionTargetGpuStepNodeSelector_IncludesEnvAndHandlesNilTarget(t *testing.T) {
	t.Setenv("PIPELINE_GPU_STEP_NODE_SELECTOR_JSON", `{"cloud.google.com/gke-nodepool":"g2-l4-dev-pool"}`)
	got := executionTargetGpuStepNodeSelector(nil)
	if got["cloud.google.com/gke-nodepool"] != "g2-l4-dev-pool" {
		t.Fatalf("expected env selector to fall through with nil target, got %#v", got)
	}
}

func TestExecutionTargetSchedulingDefaultsIncludeEnv(t *testing.T) {
	t.Setenv("PIPELINE_TEMPLATE_NODE_SELECTOR_JSON", `{"cloud.google.com/gke-accelerator":"nvidia-l4"}`)
	t.Setenv("PIPELINE_TEMPLATE_TOLERATIONS_JSON", `[{"key":"environment","operator":"Equal","value":"dev","effect":"NoSchedule"}]`)

	target := &models.ExecutionTarget{
		ResourceDefaults: map[string]interface{}{
			"templateNodeSelector": map[string]interface{}{
				"workload.cyberorigin.ai/tier": "gpu",
			},
			"templateTolerations": []interface{}{
				map[string]interface{}{
					"key":      "nvidia.com/gpu",
					"operator": "Equal",
					"value":    "present",
					"effect":   "NoSchedule",
				},
			},
		},
	}

	nodeSelector := executionTargetTemplateNodeSelector(target)
	if nodeSelector["cloud.google.com/gke-accelerator"] != "nvidia-l4" {
		t.Fatalf("expected env node selector, got %#v", nodeSelector)
	}
	if nodeSelector["workload.cyberorigin.ai/tier"] != "gpu" {
		t.Fatalf("expected target node selector, got %#v", nodeSelector)
	}

	tolerations := executionTargetTemplateTolerations(target)
	if len(tolerations) != 2 {
		t.Fatalf("expected env and target tolerations, got %#v", tolerations)
	}
	if tolerations[0].Key != "environment" || tolerations[0].Value != "dev" {
		t.Fatalf("expected env toleration first, got %#v", tolerations)
	}
	if tolerations[1].Key != "nvidia.com/gpu" || tolerations[1].Value != "present" {
		t.Fatalf("expected target toleration second, got %#v", tolerations)
	}
}

func TestResourceGuardForTargetSkipsGlobalCeilingsForNonDefaultTargetWithoutQuota(t *testing.T) {
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, newMockAssetRepo(), nil, "default")
	uc.SetResourceGuardConfig(ResourceGuardConfig{
		MaxCPU:    "8",
		MaxMemory: "28Gi",
		MaxDisk:   "250Gi",
		MaxGPU:    "1",
	})

	videoTargetGuard := uc.resourceGuardForTarget(&models.ExecutionTarget{
		ID:        "a03ad932-397f-4a3b-a3d8-1cfb13a6dd54",
		Name:      "video-proc-dev",
		Namespace: "video-proc-dev",
	})
	if videoTargetGuard.MaxCPU != "" || videoTargetGuard.MaxMemory != "" || videoTargetGuard.MaxDisk != "" || videoTargetGuard.MaxGPU != "" {
		t.Fatalf("expected non-default target without quota to skip global ceilings, got %#v", videoTargetGuard)
	}

	defaultGuard := uc.resourceGuardForTarget(&models.ExecutionTarget{Name: "default", IsDefault: true})
	if defaultGuard.MaxCPU != "8" || defaultGuard.MaxMemory != "28Gi" || defaultGuard.MaxDisk != "250Gi" {
		t.Fatalf("expected default ceilings to remain unchanged, got %#v", defaultGuard)
	}
}

func TestResourceGuardForTargetAppliesExplicitQuotaPolicy(t *testing.T) {
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, newMockAssetRepo(), nil, "default")
	uc.SetResourceGuardConfig(ResourceGuardConfig{
		MaxCPU:    "8",
		MaxMemory: "28Gi",
		MaxDisk:   "250Gi",
		MaxGPU:    "1",
	})

	targetGuard := uc.resourceGuardForTarget(&models.ExecutionTarget{
		Name: "video-proc-dev",
		QuotaPolicy: map[string]interface{}{
			"resourceCeilings": map[string]interface{}{
				"maxCpu":    "16",
				"maxMemory": "60Gi",
				"maxDisk":   "45Gi",
				"maxGpu":    "1",
			},
		},
	})
	if targetGuard.MaxCPU != "16" || targetGuard.MaxMemory != "60Gi" || targetGuard.MaxDisk != "45Gi" || targetGuard.MaxGPU != "1" {
		t.Fatalf("expected explicit target quota ceilings, got %#v", targetGuard)
	}
}
