package pipeline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/transpiler"
)

var (
	ErrPromotionNotReady = errors.New("pipeline promotion is not ready")
)

func digestJSON(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func (uc *Usecase) BuildPromotionPlan(
	ctx context.Context,
	templateID string,
	mappings []models.PipelinePromotionMappingRequirement,
) (*models.PipelinePromotionPlan, error) {
	t, err := uc.templateRepo.FindByID(ctx, strings.TrimSpace(templateID))
	if err != nil {
		return nil, fmt.Errorf("find promotion source: %w", err)
	}
	if t == nil {
		return nil, ErrTemplateNotFound
	}
	if t.Scope == "prod" {
		return nil, fmt.Errorf("%w: source must be a dev pipeline", ErrInvalidArgument)
	}
	raw, err := json.Marshal(t.Pipeline)
	if err != nil {
		return nil, fmt.Errorf("marshal promotion source: %w", err)
	}
	var pipe transpiler.Pipeline
	if err := json.Unmarshal(raw, &pipe); err != nil {
		return nil, fmt.Errorf("%w: parse promotion source: %v", ErrInvalidArgument, err)
	}

	dependencies := make([]models.PipelinePromotionDependency, 0)
	requirements := make([]models.PipelinePromotionMappingRequirement, 0)
	blockers := make([]string, 0)
	warnings := make([]string, 0)
	mappingIndex := make(map[string]string, len(mappings))
	for _, mapping := range mappings {
		mappingIndex[mapping.Kind+"\x00"+mapping.SourceID] = strings.TrimSpace(mapping.TargetID)
	}

	var visit func([]transpiler.Node)
	visit = func(nodes []transpiler.Node) {
		for i := range nodes {
			node := &nodes[i]
			component := node.Component
			dependencies = append(dependencies, models.PipelinePromotionDependency{
				NodeName:              node.ID,
				ComponentID:           component.ComponentID,
				ReleaseID:             component.ReleaseID,
				ComponentVersionLabel: component.ComponentVersionLabel,
				RuntimeImage:          component.Image,
			})
			if !strings.Contains(component.Image, "@sha256:") {
				blockers = append(blockers, fmt.Sprintf("node %q image must be pinned by sha256 digest", node.ID))
			}
			if component.ReleaseID == "" {
				warnings = append(warnings, fmt.Sprintf("node %q has no immutable releaseId", node.ID))
			}
			for _, env := range component.Env {
				upper := strings.ToUpper(env.Name)
				if env.Value != "" && containsSensitiveName(upper) {
					blockers = append(blockers, fmt.Sprintf("node %q environment %q must use a prod Secret mapping", node.ID, env.Name))
				}
			}
			if node.RuntimeConfig != nil && node.RuntimeConfig.ConfigID != "" {
				requirements = append(requirements, promotionRequirement(
					"config", node.RuntimeConfig.ConfigID, node.ID, node.RuntimeConfig.MountPath, mappingIndex,
				))
			}
			for _, secret := range node.RuntimeSecrets { // pragma: allowlist secret
				requirements = append(requirements, promotionRequirement(
					"secret", secret.ResourceID, node.ID, secret.MountPath, mappingIndex,
				))
			}
			for _, storage := range node.StorageMounts {
				requirements = append(requirements, promotionRequirement(
					"storage", storage.ResourceID, node.ID, storage.MountPath, mappingIndex,
				))
			}
			visit(node.SubNodes)
		}
	}
	visit(pipe.Nodes)

	sort.Slice(dependencies, func(i, j int) bool { return dependencies[i].NodeName < dependencies[j].NodeName })
	sort.Slice(requirements, func(i, j int) bool {
		if requirements[i].Kind == requirements[j].Kind {
			if requirements[i].SourceID == requirements[j].SourceID {
				return requirements[i].NodeName < requirements[j].NodeName
			}
			return requirements[i].SourceID < requirements[j].SourceID
		}
		return requirements[i].Kind < requirements[j].Kind
	})
	for _, requirement := range requirements {
		if requirement.TargetID == "" {
			blockers = append(blockers, fmt.Sprintf("%s resource %q requires a prod mapping", requirement.Kind, requirement.SourceID))
		}
	}

	bundle := models.PipelinePromotionBundle{
		SourceEnvironment: "dev",
		SourceTemplateID:  t.ID,
		SourceVersion:     t.Version,
		Name:              t.Name,
		Owner:             t.Owner,
		Pipeline:          t.Pipeline,
		Dependencies:      dependencies,
	}
	bundle.BundleDigest, err = digestJSON(struct {
		SourceTemplateID string
		SourceVersion    int
		Pipeline         map[string]interface{}
		Dependencies     []models.PipelinePromotionDependency
	}{bundle.SourceTemplateID, bundle.SourceVersion, bundle.Pipeline, bundle.Dependencies})
	if err != nil {
		return nil, fmt.Errorf("digest promotion bundle: %w", err)
	}
	planDigest, err := digestJSON(struct {
		BundleDigest string
		Mappings     []models.PipelinePromotionMappingRequirement
	}{bundle.BundleDigest, requirements})
	if err != nil {
		return nil, fmt.Errorf("digest promotion plan: %w", err)
	}
	return &models.PipelinePromotionPlan{
		TargetEnvironment: "prod",
		PlanDigest:        planDigest,
		Bundle:            bundle,
		RequiredMappings:  requirements,
		Warnings:          warnings,
		Blockers:          blockers,
		Ready:             len(blockers) == 0,
	}, nil
}

func promotionRequirement(kind, sourceID, nodeName, mountPath string, mappings map[string]string) models.PipelinePromotionMappingRequirement {
	targetID := mappings[kind+"\x00"+sourceID]
	resolution := ""
	if targetID != "" {
		resolution = "explicit"
	}
	return models.PipelinePromotionMappingRequirement{
		Kind:       kind,
		SourceID:   sourceID,
		TargetID:   targetID,
		NodeName:   nodeName,
		MountPath:  mountPath,
		Required:   true,
		Resolution: resolution,
	}
}

func containsSensitiveName(name string) bool { // pragma: allowlist secret
	for _, marker := range []string{"TOKEN", "PASSWORD", "SECRET", "CREDENTIAL", "API_KEY", "PRIVATE_KEY"} {
		if strings.Contains(name, marker) {
			return true
		}
	}
	return false
}
