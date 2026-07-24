package pipeline

import (
	"context"
	"errors"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

func TestBuildPromotionPlan(t *testing.T) {
	ctx := context.Background()

	t.Run("source_must_be_dev", func(t *testing.T) {
		uc := newUsecase(newMockAssetRepo())
		uc.templateRepo.(*mockTemplateRepo).byID["t1"] = &models.PipelineTemplate{
			ID:    "t1",
			Name:  "prod-pipe",
			Scope: "prod",
		}
		_, err := uc.BuildPromotionPlan(ctx, "t1", nil)
		if !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("expected ErrInvalidArgument, got %v", err)
		}
	})

	t.Run("template_not_found", func(t *testing.T) {
		uc := newUsecase(newMockAssetRepo())
		_, err := uc.BuildPromotionPlan(ctx, "nonexistent", nil)
		if !errors.Is(err, ErrTemplateNotFound) {
			t.Fatalf("expected ErrTemplateNotFound, got %v", err)
		}
	})

	t.Run("blocker_missing_digest", func(t *testing.T) {
		uc := newUsecase(newMockAssetRepo())
		pipeline := map[string]interface{}{
			"nodes": []interface{}{
				map[string]interface{}{
					"id": "node-1",
					"component": map[string]interface{}{
						"name":  "test",
						"image": "gcr.io/test/my-image:latest",
					},
				},
			},
		}
		uc.templateRepo.(*mockTemplateRepo).byID["t1"] = &models.PipelineTemplate{
			ID:       "t1",
			Name:     "dev-pipe",
			Version:  1,
			Scope:    "dev",
			Pipeline: pipeline,
		}
		plan, err := uc.BuildPromotionPlan(ctx, "t1", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if plan.Ready {
			t.Fatal("expected plan not ready due to missing digest")
		}
		if len(plan.Blockers) == 0 {
			t.Fatal("expected at least one blocker for missing digest")
		}
	})

	t.Run("blocker_sensitive_env", func(t *testing.T) {
		uc := newUsecase(newMockAssetRepo())
		pipeline := map[string]interface{}{
			"nodes": []interface{}{
				map[string]interface{}{
					"id": "node-1",
					"component": map[string]interface{}{
						"name":  "test",
						"image": "gcr.io/test/my-image@sha256:abc123def456",
						"env": []interface{}{
							map[string]interface{}{
								"name":  "MY_API_KEY",
								"value": "sk-12345",
							},
						},
					},
				},
			},
		}
		uc.templateRepo.(*mockTemplateRepo).byID["t1"] = &models.PipelineTemplate{
			ID:       "t1",
			Name:     "dev-pipe",
			Version:  1,
			Scope:    "dev",
			Pipeline: pipeline,
		}
		plan, err := uc.BuildPromotionPlan(ctx, "t1", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if plan.Ready {
			t.Fatal("expected plan not ready due to sensitive env")
		}
	})

	t.Run("happy_path_ready", func(t *testing.T) {
		uc := newUsecase(newMockAssetRepo())
		pipeline := map[string]interface{}{
			"nodes": []interface{}{
				map[string]interface{}{
					"id": "node-1",
					"component": map[string]interface{}{
						"name":                  "test",
						"componentId":           "comp-1",
						"releaseId":             "release-1",
						"componentVersionLabel": "v1.0",
						"image":                 "gcr.io/test/my-image@sha256:abc123def456",
					},
				},
			},
		}
		uc.templateRepo.(*mockTemplateRepo).byID["t1"] = &models.PipelineTemplate{
			ID:       "t1",
			Name:     "dev-pipe",
			Version:  1,
			Owner:    "alice",
			Scope:    "dev",
			Pipeline: pipeline,
		}
		plan, err := uc.BuildPromotionPlan(ctx, "t1", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !plan.Ready {
			t.Fatalf("expected plan ready, got blockers=%v warnings=%v", plan.Blockers, plan.Warnings)
		}
		if plan.TargetEnvironment != "prod" {
			t.Fatalf("expected target=prod, got %s", plan.TargetEnvironment)
		}
		if plan.PlanDigest == "" {
			t.Fatal("expected non-empty plan digest")
		}
		if len(plan.Bundle.Dependencies) != 1 {
			t.Fatalf("expected 1 dependency, got %d", len(plan.Bundle.Dependencies))
		}
		if plan.Bundle.Dependencies[0].ComponentID != "comp-1" {
			t.Fatalf("expected componentId=comp-1, got %s", plan.Bundle.Dependencies[0].ComponentID)
		}
	})

	t.Run("missing_mapping_blocker", func(t *testing.T) {
		uc := newUsecase(newMockAssetRepo())
		pipeline := map[string]interface{}{
			"nodes": []interface{}{
				map[string]interface{}{
					"id": "node-1",
					"component": map[string]interface{}{
						"name":  "test",
						"image": "gcr.io/test/my-image@sha256:abc123def456",
					},
					"runtimeConfig": map[string]interface{}{
						"configId":  "cfg-1",
						"mountPath": "/etc/config",
					},
				},
			},
		}
		uc.templateRepo.(*mockTemplateRepo).byID["t1"] = &models.PipelineTemplate{
			ID:       "t1",
			Name:     "dev-pipe",
			Version:  1,
			Scope:    "dev",
			Pipeline: pipeline,
		}
		plan, err := uc.BuildPromotionPlan(ctx, "t1", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if plan.Ready {
			t.Fatal("expected plan not ready due to missing mapping")
		}
	})

	t.Run("with_mappings_resolves", func(t *testing.T) {
		uc := newUsecase(newMockAssetRepo())
		pipeline := map[string]interface{}{
			"nodes": []interface{}{
				map[string]interface{}{
					"id": "node-1",
					"component": map[string]interface{}{
						"name":  "test",
						"image": "gcr.io/test/my-image@sha256:abc123def456",
					},
					"storageMounts": []interface{}{
						map[string]interface{}{
							"resourceId": "pvc-1",
							"mountPath":  "/data",
						},
					},
				},
			},
		}
		uc.templateRepo.(*mockTemplateRepo).byID["t1"] = &models.PipelineTemplate{
			ID:       "t1",
			Name:     "dev-pipe",
			Version:  1,
			Scope:    "dev",
			Pipeline: pipeline,
		}
		mappings := []models.PipelinePromotionMappingRequirement{
			{Kind: "storage", SourceID: "pvc-1", TargetID: "prod-pvc-1"},
		}
		plan, err := uc.BuildPromotionPlan(ctx, "t1", mappings)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !plan.Ready {
			t.Fatalf("expected plan ready, got blockers=%v", plan.Blockers)
		}
		if len(plan.RequiredMappings) != 1 {
			t.Fatalf("expected 1 required mapping, got %d", len(plan.RequiredMappings))
		}
		if plan.RequiredMappings[0].TargetID != "prod-pvc-1" {
			t.Fatalf("expected targetId=prod-pvc-1, got %s", plan.RequiredMappings[0].TargetID)
		}
		if plan.RequiredMappings[0].Resolution != "explicit" {
			t.Fatalf("expected resolution=explicit, got %s", plan.RequiredMappings[0].Resolution)
		}
	})
}

func TestContainsSensitiveName(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  bool
	}{
		{"token detected", "MY_TOKEN", true},
		{"password detected", "DB_PASSWORD", true},
		{"secret detected", "SECRET_KEY", true},
		{"credential detected", "CREDENTIALS", true},
		{"api_key detected", "SERVICE_API_KEY", true},
		{"private_key detected", "PRIVATE_KEY", true},
		{"not sensitive", "MY_CONFIG", false},
		{"not sensitive lowercase", "my_config", false},
		{"not sensitive mixed", "ConfigPath", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := containsSensitiveName(tc.input)
			if got != tc.want {
				t.Fatalf("containsSensitiveName(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestPromote(t *testing.T) {
	ctx := context.Background()

	t.Run("not_found", func(t *testing.T) {
		uc := newUsecase(newMockAssetRepo())
		_, err := uc.Promote(ctx, "nonexistent")
		if !errors.Is(err, ErrTemplateNotFound) {
			t.Fatalf("expected ErrTemplateNotFound, got %v", err)
		}
	})

	t.Run("already_prod", func(t *testing.T) {
		uc := newUsecase(newMockAssetRepo())
		uc.templateRepo.(*mockTemplateRepo).byID["t1"] = &models.PipelineTemplate{
			ID:    "t1",
			Name:  "prod-pipe",
			Scope: "prod",
		}
		_, err := uc.Promote(ctx, "t1")
		if err == nil {
			t.Fatal("expected error for already-prod template")
		}
	})

	t.Run("promote_dev_to_prod", func(t *testing.T) {
		uc := newUsecase(newMockAssetRepo())
		pipeline := map[string]interface{}{"key": "value"}
		uc.templateRepo.(*mockTemplateRepo).byID["t1"] = &models.PipelineTemplate{
			ID:       "t1",
			Name:     "dev-pipe",
			Version:  1,
			Owner:    "alice",
			Scope:    "dev",
			Pipeline: pipeline,
		}
		result, err := uc.Promote(ctx, "t1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Scope != "prod" {
			t.Fatalf("expected scope=prod, got %s", result.Scope)
		}
		if result.Name != "dev-pipe" {
			t.Fatalf("expected name=dev-pipe, got %s", result.Name)
		}
	})
}
